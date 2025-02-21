package bsky

import (
	"context"
	"runtime/debug"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type WorkerMetrics struct {
	jobsQueued    metric.Int64Gauge
	itemsQueued   metric.Int64Gauge
	resultsQueued metric.Int64Gauge
	jobsInflight  metric.Int64UpDownCounter
	itemsCount    metric.Int64Counter
}

func NewWorkerMetrics(ctx context.Context) (*WorkerMetrics, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	version := "unknown"
	if ok {
		version = buildInfo.Main.Version
	}

	meter := otel.GetMeterProvider().Meter(
		"bsky.worker",
		metric.WithInstrumentationVersion(version),
	)

	jobsQueued, err := meter.Int64Gauge(
		"bsky.worker.jobs_queued",
		metric.WithDescription("Number of repo jobs currently queued"),
		metric.WithUnit("{jobs}"),
	)
	if err != nil {
		return nil, err
	}

	itemsQueued, err := meter.Int64Gauge(
		"bsky.worker.items_queued",
		metric.WithDescription("Number of items currently queued for ingest"),
		metric.WithUnit("{items}"),
	)
	if err != nil {
		return nil, err
	}

	resultsQueued, err := meter.Int64Gauge(
		"bsky.worker.results_queued",
		metric.WithDescription("Number of results currently queued for processing"),
		metric.WithUnit("{results}"),
	)
	if err != nil {
		return nil, err
	}

	jobsInflight, err := meter.Int64UpDownCounter(
		"bsky.worker.jobs_inflight",
		metric.WithDescription("Inflight jobs count"),
		metric.WithUnit("{jobs}"),
	)
	if err != nil {
		return nil, err
	}

	itemsCount, err := meter.Int64Counter(
		"bsky.worker.items_processed",
		metric.WithDescription("Completed items count"),
		metric.WithUnit("{items}"),
	)
	if err != nil {
		return nil, err
	}

	return &WorkerMetrics{
		jobsQueued:    jobsQueued,
		itemsQueued:   itemsQueued,
		resultsQueued: resultsQueued,
		jobsInflight:  jobsInflight,
		itemsCount:    itemsCount,
	}, nil
}
