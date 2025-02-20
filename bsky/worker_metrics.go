package bsky

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type WorkerMetrics struct {
	jobsQueued  metric.Int64Gauge
	itemsQueued metric.Int64Gauge
}

func NewWorkerMetrics(ctx context.Context) (*WorkerMetrics, error) {
	meter := otel.GetMeterProvider().Meter(
		"atproto_worker",
		metric.WithInstrumentationVersion("0.1.0"),
	)

	jobsQueued, err := meter.Int64Gauge(
		"atproto.worker.jobs_queued",
		metric.WithDescription("Current number of repo jobs queued"),
		metric.WithUnit("{job}"),
	)

	if err != nil {
		return nil, err
	}

	itemsQueued, err := meter.Int64Gauge(
		"atproto.worker.items_queued",
		metric.WithDescription("Current number of items queued for ingest"),
		metric.WithUnit("{item}"),
	)

	if err != nil {
		return nil, err
	}

	return &WorkerMetrics{
		jobsQueued:  jobsQueued,
		itemsQueued: itemsQueued,
	}, nil
}
