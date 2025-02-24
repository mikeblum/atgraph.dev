package bsky

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type RateLimitFailureType string

const (
	FailureMaxAttempts  RateLimitFailureType = "max_attempts"
	FailureMaxWaitTime  RateLimitFailureType = "max_wait_time"
	FailureTimeout      RateLimitFailureType = "timeout"
	FailureNetworkError RateLimitFailureType = "network_error"
)

type RateLimitMetrics struct {
	retryAttempts              metric.Int64UpDownCounter
	rateLimitHits              metric.Int64Counter
	rateLimitRequestsRemaining metric.Int64Gauge
	rateLimitRequestsLimit     metric.Int64Gauge
	rateLimitRequestsReset     metric.Float64Histogram
	failures                   metric.Int64Counter
	statusCodes                metric.Int64Counter
}

func NewRateLimitMetrics(ctx context.Context) (*RateLimitMetrics, error) {
	meter := otel.GetMeterProvider().Meter(
		"atproto_rate_limit",
		metric.WithInstrumentationVersion("0.1.0"),
	)

	retryAttempts, err := meter.Int64UpDownCounter(
		"atproto.rate_limit.retry_attempts",
		metric.WithDescription("Current number of retry attempts"),
		metric.WithUnit("{attempt}"),
	)
	if err != nil {
		return nil, err
	}

	rateLimitRequestsReset, err := meter.Float64Histogram(
		"atproto.rate_limit.wait_remaining",
		metric.WithDescription("Time spent waiting in seconds due to rate limits"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	rateLimitHits, err := meter.Int64Counter(
		"atproto.rate_limit.hits",
		metric.WithDescription("Number of rate limits encountered"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, err
	}

	rateLimitRequestsRemaining, err := meter.Int64Gauge(
		"atproto.rate_limit.requests_remaining",
		metric.WithDescription("Number of requests remaining per 5 minutes, rate limited by IP"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	rateLimitRequestsLimit, err := meter.Int64Gauge(
		"atproto.rate_limit.requests_limit",
		metric.WithDescription("Number of requests allowed / 3,000 requests per 5 minutes, rate limited by IP"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	failures, err := meter.Int64Counter(
		"atproto.rate_limit.failures",
		metric.WithDescription("Rate limit failures by type"),
		metric.WithUnit("{failure}"),
	)
	if err != nil {
		return nil, err
	}

	statusCodes, err := meter.Int64Counter(
		"atproto.rate_limit.status_codes",
		metric.WithDescription("Rate limit HTTP status codes"),
		metric.WithUnit("{status_code}"),
	)
	if err != nil {
		return nil, err
	}

	return &RateLimitMetrics{
		retryAttempts:              retryAttempts,
		rateLimitHits:              rateLimitHits,
		rateLimitRequestsRemaining: rateLimitRequestsRemaining,
		rateLimitRequestsLimit:     rateLimitRequestsLimit,
		rateLimitRequestsReset:     rateLimitRequestsReset,
		failures:                   failures,
		statusCodes:                statusCodes,
	}, nil
}
