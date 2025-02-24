package bsky

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type RateLimitFailureType string

const (
	FailureMaxAttempts  RateLimitFailureType = "max_attempts"
	FailureMaxWaitTime  RateLimitFailureType = "max_wait_time"
	FailureTimeout      RateLimitFailureType = "timeout"
	FailureNetworkError RateLimitFailureType = "network_error"

	MetricRetryAttempts     = "atproto.rate_limit.retry_attempts"
	MetricWaitRemaining     = "atproto.rate_limit.wait_remaining"
	MetricRateLimitHits     = "atproto.rate_limit.hits"
	MetricRequestsRemaining = "atproto.rate_limit.requests_remaining"
	MetricRequestsLimit     = "atproto.rate_limit.requests_limit"
	MetricFailures          = "atproto.rate_limit.failures"
	MetricStatusCodes       = "atproto.rate_limit.status_codes"
)

type RateLimitMetrics struct {
	meter                      metric.Meter
	retryAttempts              metric.Int64UpDownCounter
	rateLimitHits              metric.Int64Counter
	rateLimitRequestsRemaining metric.Int64Gauge
	rateLimitRequestsLimit     metric.Int64Gauge
	rateLimitRequestsReset     metric.Int64Gauge
	failures                   metric.Int64Counter
	statusCodes                metric.Int64Counter
}

func NewRateLimitMetrics(ctx context.Context) (*RateLimitMetrics, error) {
	r := &RateLimitMetrics{}
	r.meter = otel.GetMeterProvider().Meter(
		"atproto_rate_limit",
		metric.WithInstrumentationVersion("0.1.0"),
	)
	return r.Reset()
}

func (r *RateLimitMetrics) Reset() (*RateLimitMetrics, error) {
	for _, metric := range []string{
		MetricRetryAttempts,
		MetricWaitRemaining,
		MetricRateLimitHits,
		MetricRequestsRemaining,
		MetricRequestsLimit,
		MetricFailures,
		MetricStatusCodes,
	} {
		if err := r.ResetMetric(metric); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *RateLimitMetrics) ResetMetric(metricName string) error {
	var err error
	switch metricName {
	case MetricRateLimitHits:
		r.rateLimitHits, err = r.meter.Int64Counter(
			MetricRateLimitHits,
			metric.WithDescription("Number of rate limits encountered"),
			metric.WithUnit("{hit}"),
		)
	case MetricRequestsRemaining:
		r.rateLimitRequestsRemaining, err = r.meter.Int64Gauge(
			MetricRequestsRemaining,
			metric.WithDescription("Number of requests remaining per 5 minutes, rate limited by IP"),
			metric.WithUnit("{request}"),
		)
	case MetricRequestsLimit:
		r.rateLimitRequestsLimit, err = r.meter.Int64Gauge(
			MetricRequestsLimit,
			metric.WithDescription("Number of requests allowed / 3,000 requests per 5 minutes, rate limited by IP"),
			metric.WithUnit("{request}"),
		)
	case MetricRetryAttempts:
		r.retryAttempts, err = r.meter.Int64UpDownCounter(
			MetricRetryAttempts,
			metric.WithDescription("Current number of retry attempts"),
			metric.WithUnit("{attempt}"),
		)
	case MetricWaitRemaining:
		r.rateLimitRequestsReset, err = r.meter.Int64Gauge(
			MetricWaitRemaining,
			metric.WithDescription("Time spent waiting in seconds due to rate limits"),
			metric.WithUnit("s"),
		)
	case MetricFailures:
		r.failures, err = r.meter.Int64Counter(
			MetricFailures,
			metric.WithDescription("Rate limit failures by type"),
			metric.WithUnit("{failure}"),
		)
	case MetricStatusCodes:
		r.statusCodes, err = r.meter.Int64Counter(
			MetricStatusCodes,
			metric.WithDescription("Rate limit HTTP status codes"),
			metric.WithUnit("{status_code}"),
		)
	default:
		return fmt.Errorf("unknown metric: %s", metricName)
	}
	return err
}
