package bsky

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/bluesky-social/indigo/xrpc"
	log "github.com/mikeblum/atgraph.dev/conf"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
)

type WorkerPool struct {
	client       *Client
	log          *log.Log
	jobs         chan RepoJob
	items        chan RepoItem
	results      chan error
	jobsInflight atomic.Int64
	poolReady    chan bool
	ingestReady  chan bool
	done         chan bool
	rateLimiter  *RateLimitHandler
	metrics      *WorkerMetrics
	ingest       func(context.Context, int, RepoItem) error
	workerCount  int
}

func NewWorkerPool(ctx context.Context, client *Client, conf *Conf) (*WorkerPool, error) {
	rateLimit, err := NewRateLimitHandler(ctx, client.atproto)
	if err != nil {
		return nil, err
	}
	metrics, err := NewWorkerMetrics(ctx)
	if err != nil {
		return nil, err
	}
	return &WorkerPool{
		client:      client,
		log:         log.NewLog(),
		jobs:        make(chan RepoJob, conf.WorkerCount()*2),
		items:       make(chan RepoItem, conf.WorkerCount()*2),
		results:     make(chan error, conf.WorkerCount()*2),
		poolReady:   make(chan bool),
		ingestReady: make(chan bool),
		done:        make(chan bool),
		rateLimiter: rateLimit,
		metrics:     metrics,
		workerCount: conf.WorkerCount(),
	}, nil
}

func (p *WorkerPool) StartMonitor(ctx context.Context) *WorkerPool {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.metrics.jobsQueued.Record(ctx, int64(len(p.jobs)))
				p.metrics.itemsQueued.Record(ctx, int64(len(p.items)))
				p.metrics.resultsQueued.Record(ctx, int64(len(p.results)))
			}
		}
	}()
	return p
}

func (p *WorkerPool) PoolReady() chan bool {
	return p.poolReady
}

func (p *WorkerPool) IngestReady() chan bool {
	return p.ingestReady
}

func (p *WorkerPool) Size() int {
	return len(p.jobs)
}

func (p *WorkerPool) WithIngest(ingest func(context.Context, int, RepoItem) error) *WorkerPool {
	p.ingest = ingest
	return p
}

// Start - step #1: start worker pool
func (p *WorkerPool) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	p.log.Info("Starting worker pool", "worker-count", p.workerCount)

	// Start repo workers
	for i := 0; i < p.workerCount; i++ {
		workerID := i + 1
		g.Go(func() error {
			return p.repoWorker(ctx, workerID)
		})
	}

	// Start ingest workers
	for i := 0; i < p.workerCount; i++ {
		workerID := i + 1
		g.Go(func() error {
			return p.ingestWorker(ctx, workerID)
		})
	}

	go func() {
		<-ctx.Done()
		close(p.done)
	}()

	// Signal pool is ready
	close(p.poolReady)
	// Signal ingest is ready
	close(p.ingestReady)

	return g.Wait()
}

// Submit - step #2: submit repo jobs for processing
func (p *WorkerPool) Submit(ctx context.Context, job RepoJob) error {
	if job.repo == nil {
		return fmt.Errorf("error submitting RepoJob: missing repo")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return fmt.Errorf("worker pool is shutting down")
	case p.jobs <- job: // block until more work can be processed
		return nil
	}
}

func (p *WorkerPool) ingestWorker(ctx context.Context, workerID int) error {
	p.log.Info("Worker started", "type", "ingest", "worker-id", workerID)
	defer p.log.Info("Worker shutting down", "type", "ingest", "worker-id", workerID)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Context cancelled", "type", "ingest", "worker-id", workerID)
			return ctx.Err()
		case <-p.done:
			p.log.Info("Done channel closed", "type", "ingest", "worker-id", workerID)
			return nil
		case item, ok := <-p.items:
			if !ok {
				p.log.Info("Ingest channel closed", "type", "ingest", "worker-id", workerID)
				return nil
			}

			p.log.Info("Processing ingest",
				"action", "ingest",
				"worker-id", workerID,
				"did", item.repo.RepoDid())

			err := p.rateLimiter.WithRetry(ctx, WriteOperation, "ingest", func() error {
				return p.ingest(ctx, workerID, item)
			})

			status := "ok"
			if err != nil {
				status = "err"
				p.log.WithErrorMsg(err, "Retries exhausted",
					"action", "ingest",
					"worker-id", workerID,
					"did", item.repo.RepoDid())
			}
			p.metrics.itemsCount.Add(ctx, 1, metric.WithAttributes(
				attribute.Int("worker_id", workerID),
				attribute.String("status", status),
				attribute.String("action", "ingest"),
			))
			p.results <- err
		}
	}
}

func (p *WorkerPool) repoWorker(ctx context.Context, workerID int) error {
	p.log.Info("Worker started", "type", "repo", "worker-id", workerID)
	defer p.log.Info("Worker shutting down", "type", "repo", "worker-id", workerID)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Context cancelled", "type", "repo", "worker-id", workerID)
			return ctx.Err()
		case <-p.done:
			p.log.Info("Done channel closed", "type", "repo", "worker-id", workerID)
			return nil
		case job, ok := <-p.jobs:
			if !ok {
				p.log.Info("Jobs channel closed", "type", "repo", "worker-id", workerID)
				return nil
			}

			p.log.Info("Processing job",
				"action", "get-repo",
				"type", "repo",
				"worker-id", workerID,
				"did", job.repo.Did)

			err := p.rateLimiter.WithRetry(ctx, ReadOperation, "getRepo", func() error {
				if err := p.getRepo(ctx, job); err != nil {
					if !suppressATProtoErr(err) {
						p.log.WithErrorMsg(err, "Error getting repo",
							"worker-id", workerID,
							"did", job.repo.Did)
					}
					return err
				}
				return nil
			})

			if err != nil {
				p.log.WithErrorMsg(err, "Retries exhausted",
					"action", "get-repo",
					"type", "repo",
					"worker-id", workerID,
					"did", job.repo.Did)
			}
		}
	}
}

func (p *WorkerPool) getRepo(ctx context.Context, job RepoJob) error {
	var repoData []byte
	var err error
	var ident *identity.Identity
	var atid *syntax.AtIdentifier
	if atid, err = syntax.ParseAtIdentifier(job.repo.Did); err != nil {
		return err
	}
	if ident, err = identity.DefaultDirectory().Lookup(ctx, *atid); err != nil {
		return err
	}
	xrpcc := xrpc.Client{
		Client: NewHTTPClient(),
		Host:   ident.PDSEndpoint(),
	}
	if xrpcc.Host == "" {
		return fmt.Errorf("no PDS endpoint for identity: %s", atid)
	}
	if repoData, err = atproto.SyncGetRepo(ctx, &xrpcc, ident.DID.String(), ""); err != nil {
		if !suppressATProtoErr(err) {
			p.log.WithErrorMsg(err, "Error fetching bsky repo")
		}
		return err
	}
	var r *repo.Repo
	if r, err = repo.ReadRepoFromCar(context.Background(), bytes.NewReader(repoData)); err != nil {
		p.log.WithErrorMsg(err, "Error reading bsky repo")
		return err
	}

	if err = resolveLexicon(ctx, ident, r, p.items); err != nil {
		// Unwrap error to check if it's a LexiconError
		if unwrappedErr := errors.Unwrap(err); unwrappedErr != nil {
			var lexErr *LexiconError
			if errors.As(unwrappedErr, &lexErr) {
				p.log.With(
					"did", job.repo.Did,
					"lexicon-type", lexErr.nsid.Name()).Debug("Skipping due to lexicon error")
			}
		}
	}

	return nil
}
