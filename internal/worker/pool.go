package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saikrishnans/job-scheduler/internal/metrics"
	"github.com/saikrishnans/job-scheduler/internal/models"
	"github.com/saikrishnans/job-scheduler/internal/queue"
	"github.com/saikrishnans/job-scheduler/internal/store"
)

// EventCallback is called whenever a job changes state (for WebSocket broadcast).
type EventCallback func(job *models.Job)

// Pool manages a fixed number of concurrent job workers.
type Pool struct {
	id          string
	concurrency int
	q           *queue.Queue
	store       *store.Store
	onEvent     EventCallback
	wg          sync.WaitGroup
	stopCh      chan struct{}
}

func NewPool(concurrency int, q *queue.Queue, s *store.Store, onEvent EventCallback) *Pool {
	return &Pool{
		id:          uuid.New().String()[:8],
		concurrency: concurrency,
		q:           q,
		store:       s,
		onEvent:     onEvent,
		stopCh:      make(chan struct{}),
	}
}

// Start launches worker goroutines and a metrics ticker.
func (p *Pool) Start(ctx context.Context) {
	log.Printf("[pool] starting %d workers", p.concurrency)
	for i := range p.concurrency {
		workerID := fmt.Sprintf("worker-%s-%d", p.id, i)
		p.wg.Add(1)
		go p.run(ctx, workerID)
	}

	// Update queue-depth gauge every 5s.
	p.wg.Add(1)
	go p.metricsTicker(ctx)
}

// Stop signals all workers to stop and waits for them.
func (p *Pool) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	log.Println("[pool] all workers stopped")
}

// run is the main loop for a single worker goroutine.
func (p *Pool) run(ctx context.Context, workerID string) {
	defer p.wg.Done()
	log.Printf("[%s] started", workerID)

	for {
		select {
		case <-p.stopCh:
			log.Printf("[%s] shutting down", workerID)
			return
		default:
		}

		job, err := p.q.Dequeue(ctx)
		if err != nil {
			log.Printf("[%s] dequeue error: %v", workerID, err)
			select {
			case <-p.stopCh:
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if job == nil {
			// Queue empty — back off briefly.
			select {
			case <-p.stopCh:
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}

		p.process(ctx, workerID, job)
	}
}

// process executes a single job with timing and status tracking.
func (p *Pool) process(ctx context.Context, workerID string, job *models.Job) {
	metrics.ActiveWorkers.Inc()
	defer metrics.ActiveWorkers.Dec()

	start := time.Now()
	log.Printf("[%s] processing job %s (priority=%d attempt=%d)", workerID, job.ID, job.Priority, job.Attempts+1)

	// Mark running in DB.
	if err := p.store.UpdateJobStatus(ctx, job.ID, models.StatusRunning, workerID, ""); err != nil {
		log.Printf("[%s] failed to mark running: %v", workerID, err)
	}
	job.Status = models.StatusRunning
	job.WorkerID = workerID
	p.broadcast(job)
	p.auditLog(ctx, job.ID, "running", fmt.Sprintf("picked up by %s", workerID))

	// Simulate job work (replace with real dispatch logic).
	err := executeJob(ctx, job)

	elapsed := time.Since(start).Seconds()
	metrics.WorkerProcessingDuration.Observe(elapsed)
	metrics.WorkerProcessingDurationByPriority.WithLabelValues(strconv.Itoa(job.Priority)).Observe(elapsed)

	if err == nil {
		// Success path.
		metrics.JobsCompleted.Inc()
		if dbErr := p.store.UpdateJobStatus(ctx, job.ID, models.StatusCompleted, workerID, ""); dbErr != nil {
			log.Printf("[%s] failed to mark completed: %v", workerID, dbErr)
		}
		p.q.Complete(ctx, job.ID)
		job.Status = models.StatusCompleted
		p.broadcast(job)
		p.auditLog(ctx, job.ID, "completed", fmt.Sprintf("duration=%.2fms", elapsed*1000))
		log.Printf("[%s] job %s completed in %.2fms", workerID, job.ID, elapsed*1000)
		return
	}

	// Failure path.
	log.Printf("[%s] job %s failed (attempt %d/%d): %v", workerID, job.ID, job.Attempts+1, job.MaxRetries, err)
	metrics.JobsFailed.Inc()

	if job.Attempts+1 >= job.MaxRetries {
		// Exhausted retries → dead letter.
		metrics.JobsDeadLettered.Inc()
		if dbErr := p.store.MarkDeadLetter(ctx, job.ID, err.Error()); dbErr != nil {
			log.Printf("[%s] failed to mark dead_letter: %v", workerID, dbErr)
		}
		p.q.MoveToDeadLetter(ctx, job.ID)
		job.Status = models.StatusDeadLetter
		p.broadcast(job)
		p.auditLog(ctx, job.ID, "dead_letter", fmt.Sprintf("retries exhausted: %v", err))
		return
	}

	// Schedule retry with exponential backoff.
	job.Attempts++
	job.Status = models.StatusFailed
	metrics.JobsRetried.Inc()

	backoff := exponentialBackoff(job.Attempts)
	p.auditLog(ctx, job.ID, "retry", fmt.Sprintf("attempt %d, backoff %s, err: %v", job.Attempts, backoff, err))

	if dbErr := p.store.UpdateJobStatus(ctx, job.ID, models.StatusFailed, workerID, err.Error()); dbErr != nil {
		log.Printf("[%s] failed to mark failed: %v", workerID, dbErr)
	}
	p.broadcast(job)

	time.Sleep(backoff)
	p.q.Requeue(ctx, job)
}

// exponentialBackoff returns min(2^attempt * 1s + jitter, 60s).
func exponentialBackoff(attempt int) time.Duration {
	base := math.Pow(2, float64(attempt)) * float64(time.Second)
	jitter := rand.Float64() * float64(time.Second)
	d := time.Duration(base + jitter)
	if d > 60*time.Second {
		d = 60 * time.Second
	}
	return d
}

// executeJob simulates job execution (replace with real dispatch logic).
func executeJob(ctx context.Context, job *models.Job) error {
	// Simulate variable duration based on payload size.
	workDuration := time.Duration(50+rand.Intn(450)) * time.Millisecond

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(workDuration):
	}

	// Simulate ~15% failure rate for demo purposes.
	if rand.Float64() < 0.15 {
		return fmt.Errorf("simulated job failure")
	}
	return nil
}

func (p *Pool) broadcast(job *models.Job) {
	if p.onEvent != nil {
		p.onEvent(job)
	}
}

func (p *Pool) auditLog(ctx context.Context, jobID, event, details string) {
	if err := p.store.AddAuditLog(ctx, jobID, event, details); err != nil {
		log.Printf("[pool] audit log error: %v", err)
	}
}

func (p *Pool) metricsTicker(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-t.C:
			if depth, err := p.q.QueueDepth(ctx); err == nil {
				metrics.QueueDepth.Set(float64(depth))
			}
			if dl, err := p.q.DeadLetterCount(ctx); err == nil {
				metrics.DeadLetterQueueDepth.Set(float64(dl))
			}
		}
	}
}
