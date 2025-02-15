package bsky

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/mikeblum/atproto-graph-viz/conf"
)

const (
	ReadOperationStr  = "READ_OP"
	WriteOperationStr = "WRITE_OP"
)

type OperationType int

func (op OperationType) String() string {
	switch op {
	case WriteOperation:
		return WriteOperationStr
	default:
		return ReadOperationStr
	}
}

const (
	ReadOperation OperationType = iota
	WriteOperation
)

type RateLimitHandler struct {
	client            *xrpc.Client
	log               *conf.Log
	maxRetries        int
	maxWaitTime       time.Duration
	readBaseWaitTime  time.Duration
	writeBaseWaitTime time.Duration
}

// NewRateLimitHandler - default rate limt
func NewRateLimitHandler(client *xrpc.Client) *RateLimitHandler {
	return &RateLimitHandler{
		client:            client,
		log:               conf.NewLog(),
		maxRetries:        maxRetries(),
		maxWaitTime:       30 * time.Second,       // 30s retry deadline
		readBaseWaitTime:  500 * time.Millisecond, // 500ms for read operations
		writeBaseWaitTime: 1 * time.Second,        // 1s for write operations
	}
}

// executeWithRetry executes an API call with rate limit handling
func (h *RateLimitHandler) executeWithRetry(ctx context.Context, opType OperationType, operation func() error) error {
	var waitTime time.Duration
	var err error
	var attempt int
	for {
		if err = operation(); err == nil {
			return nil
		}

		var apiErr *xrpc.Error
		var ok bool
		// short circuit if no atproto error
		if apiErr, ok = err.(*xrpc.Error); !ok {
			return nil
		}

		switch apiErr.StatusCode {
		case http.StatusTooManyRequests:
			waitTime = h.calculateWaitTime(apiErr, attempt, opType)
			if waitTime >= h.maxWaitTime {
				break
			}
			h.log.With("action", "retry", "op", opType, "wait", waitTime, "attempt", attempt+1, "max-retry", h.maxRetries, "max-wait", h.maxWaitTime).Warn(fmt.Sprintf("Rate limit exceeded. Waiting %v", waitTime))
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for rate limit: %w", ctx.Err())
			// wait alloted cooldown period
			case <-time.After(waitTime):
			}
		}
		attempt++
		if (h.maxRetries > 0 && attempt >= h.maxRetries) || waitTime >= h.maxWaitTime {
			break
		}
	}
	var retryErr error
	if h.maxRetries > 0 {
		retryErr = fmt.Errorf("operation failed after %d retries: %w", h.maxRetries, err)
	} else {
		retryErr = fmt.Errorf("operation failed after %v: %w", h.maxWaitTime, err)
	}
	h.log.WithError(retryErr, "Retry Exhausted", "action", "retry", "op", opType, "max-retry", h.maxRetries, "max-wait", h.maxWaitTime)
	return retryErr
}

// calculateWaitTime determines how long to wait before retrying upto maxWaitTime
func (h *RateLimitHandler) calculateWaitTime(apiErr *xrpc.Error, attempt int, opType OperationType) time.Duration {
	// use specified rate limit TTL if specififed
	if apiErr != nil && apiErr.Ratelimit != nil {
		return time.Since(apiErr.Ratelimit.Reset).Abs()
	}

	// otherwise fall back to exponential backoff based on read vs write op
	baseWait := h.readBaseWaitTime
	if opType == WriteOperation {
		baseWait = h.writeBaseWaitTime
	}

	// 2^n expoential backoff in seconds
	// 1st retry: 500ms
	// 2nd retry: 1s
	// 3rd retry: 2s
	// 4th retry: 4s
	backoff := baseWait * time.Duration(1<<uint(attempt))
	if backoff > h.maxWaitTime {
		backoff = h.maxWaitTime
	}
	return backoff
}

func maxRetries() int {
	var maxRetries int
	var err error
	if maxRetries, err = strconv.Atoi(conf.GetEnv(ENV_BSKY_MAX_RETRY_COUNT, strconv.Itoa(DEFAULT_MAX_RETRIES))); err != nil {
		maxRetries = DEFAULT_MAX_RETRIES
	}
	return maxRetries
}
