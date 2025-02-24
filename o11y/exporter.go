package o11y

import (
	"context"
	"fmt"
	"time"

	"github.com/mikeblum/atproto-graph-viz/conf"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
)

const (
	ServiceName = "atproto-graph-viz"
)

func NewO11y(ctx context.Context, log *conf.Log) (*otlpmetricgrpc.Exporter, error) {
	// configure global error log handler
	otel.SetErrorHandler(NewOTELErrorHandler(log.Logger))

	cfg := NewConf()

	var res *resource.Resource
	var err error
	if res, err = resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(ServiceName),
			semconv.DeploymentEnvironment(cfg.env()),
		),
	); err != nil {
		return nil, err
	}
	var exporter *otlpmetricgrpc.Exporter
	if exporter, err = otlpmetricgrpc.New(ctx,
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
					log.ErrorContext(ctx, "OTLP transport error",
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
		return nil, err
	}

	provider := sdk.NewMeterProvider(
		sdk.WithResource(res),
		sdk.WithReader(
			sdk.NewPeriodicReader(exporter,
				sdk.WithInterval(10*time.Second),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	// Start runtime metrics collection
	if err := runtime.Start(
		runtime.WithMinimumReadMemStatsInterval(time.Second),
		runtime.WithMeterProvider(provider),
	); err != nil {
		log.WithError(err).Error("failed to start runtime metrics collection")
		return nil, err
	}

	return exporter, nil
}

func Cleanup(ctx context.Context) error {
	if provider, ok := otel.GetMeterProvider().(*sdk.MeterProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}
	return nil
}
