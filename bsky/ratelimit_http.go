package bsky

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/util"
	"github.com/bluesky-social/indigo/xrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	HeaderRateLimitPolicy    = "RateLimit-Policy"
	HeaderRateLimitRemaining = "RateLimit-Remaining"
	HeaderRateLimitLimit     = "RateLimit-Limit"
	HeaderRateLimitReset     = "RateLimit-Reset"
)

type RateLimitInterceptor struct {
	metrics   *RateLimitMetrics
	transport http.RoundTripper
}

func NewHTTPClient() *http.Client {
	metrics, _ := NewRateLimitMetrics(context.Background())
	client := util.RobustHTTPClient()
	client.Transport = &RateLimitInterceptor{
		metrics:   metrics,
		transport: http.DefaultTransport,
	}
	return client
}

func NewRateLimit(resp *http.Response) (*xrpc.RatelimitInfo, error) {
	info := &xrpc.RatelimitInfo{
		Policy: resp.Header.Get(HeaderRateLimitPolicy),
	}
	if n, err := strconv.ParseInt(resp.Header.Get(HeaderRateLimitReset), 10, 64); err == nil {
		info.Reset = time.Unix(n, 0)
	}
	if n, err := strconv.ParseInt(resp.Header.Get(HeaderRateLimitLimit), 10, 64); err == nil {
		info.Limit = int(n)
	}
	if n, err := strconv.ParseInt(resp.Header.Get(HeaderRateLimitRemaining), 10, 64); err == nil {
		info.Remaining = int(n)
	}
	return info, nil
}

func (r *RateLimitInterceptor) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	baseAttrs := []attribute.KeyValue{
		attribute.String("method", resp.Request.Method),
		attribute.String("path", resp.Request.URL.Path),
	}
	info, err := NewRateLimit(resp)
	if err != nil {
		return nil, err
	}

	// emit metrics for rate limits
	ctx := resp.Request.Context()
	r.metrics.rateLimitRequestsRemaining.Record(ctx, int64(info.Remaining), metric.WithAttributes(baseAttrs...))
	r.metrics.rateLimitRequestsLimit.Record(ctx, int64(info.Limit), metric.WithAttributes(baseAttrs...))
	r.metrics.rateLimitRequestsReset.Record(ctx, time.Until(info.Reset).Seconds(), metric.WithAttributes(baseAttrs...))

	return resp, nil
}
