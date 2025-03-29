package o11y

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mikeblum/atgraph.dev/conf"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"google.golang.org/grpc"
)

const (
	ServiceName = "atgraph.dev"
)

func NewO11y(ctx context.Context, logger *conf.Log) error {
	// configure global error log handler
	otel.SetErrorHandler(NewOTELErrorHandler(logger.Logger))

	cfg := NewConf()

	var resources *resource.Resource
	var err error
	if resources, err = resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(cfg.GetEnv("OTEL_SERVICE_VERSION", "dev")),
			semconv.DeploymentEnvironmentName(cfg.env()),
		),
	); err != nil {
		return err
	}

	var tracesExporter *otlptrace.Exporter
	if tracesExporter, err = otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(cfg.o11yEndpoint()),
		),
	); err != nil {
		return err
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			// TODO: toggle sampling for production
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(tracesExporter),
			sdktrace.WithResource(resources),
		),
	)

	var metricsExporter *otlpmetricgrpc.Exporter
	if metricsExporter, err = otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpointURL(cfg.o11yEndpoint()),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     10 * time.Second,
			MaxElapsedTime:  1 * time.Minute,
		}),
		otlpmetricgrpc.WithDialOption(
			grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				start := time.Now()
				err := invoker(ctx, method, req, reply, cc, opts...)
				if err != nil {
					logger.ErrorContext(ctx, "OTLP transport error",
						"error", err,
						"engine", "otel",
						"method", method,
						"duration", time.Until(start),
					)
				}
				return err
			}),
		),
	); err != nil {
		return err
	}

	metricsProvider := sdk.NewMeterProvider(
		sdk.WithResource(resources),
		sdk.WithReader(
			sdk.NewPeriodicReader(metricsExporter,
				sdk.WithInterval(10*time.Second),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(metricsProvider)

	// Start runtime metrics collection
	if err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
		runtime.WithMeterProvider(metricsProvider),
	); err != nil {
		logger.WithError(err).Error("failed to start runtime metrics collection")
		return err
	}

	var logExporter *stdoutlog.Exporter
	if logExporter, err = stdoutlog.New(); err != nil {
		return err
	}

	// Create the logger provider with batch processor
	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)

	global.SetLoggerProvider(loggerProvider)

	otelLog := otelslog.NewLogger(
		ServiceName,
		otelslog.WithLoggerProvider(loggerProvider),
	)

	slog.SetDefault(otelLog)

	return nil
}

func Cleanup(ctx context.Context) error {
	if provider, ok := otel.GetMeterProvider().(*sdk.MeterProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}
	if provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown traces provider: %w", err)
		}
	}
	return nil
}
