package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saikrishnans/job-scheduler/internal/models"
)

const (
	// jobsZSet is the priority queue (sorted set, score = priority * -1 + timestamp fraction).
	jobsZSet = "jobs:queue"
	// jobsHash stores job data by ID.
	jobsHash = "jobs:data"
	// deadLetterList holds dead-letter job IDs.
	deadLetterList = "jobs:dead_letter"
	// processingSet holds job IDs currently being processed.
	processingSet = "jobs:processing"
)

// Queue wraps Redis for priority-based job queuing.
type Queue struct {
	rdb *redis.Client
}

func New(addr, password string, db int) *Queue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &Queue{rdb: rdb}
}

func (q *Queue) Ping(ctx context.Context) error {
	return q.rdb.Ping(ctx).Err()
}

func (q *Queue) Close() error {
	return q.rdb.Close()
}

// Enqueue adds a job to the priority sorted set.
// Score = -(priority * 1e9) + epoch_ns_fraction ensures:
//   - Higher priority jobs come first (lower score wins in ZPOPMIN).
//   - Within same priority, FIFO by insertion time.
func (q *Queue) Enqueue(ctx context.Context, job *models.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	pipe := q.rdb.TxPipeline()

	// Store full job JSON in hash.
	pipe.HSet(ctx, jobsHash, job.ID, data)

	// Score: negate priority so higher priority = lower score (ZPOPMIN).
	score := float64(-job.Priority)*1e9 + float64(time.Now().UnixNano()%int64(1e9))
	pipe.ZAdd(ctx, jobsZSet, redis.Z{Score: score, Member: job.ID})

	_, err = pipe.Exec(ctx)
	return err
}

// Dequeue atomically pops the highest-priority job ID and marks it processing.
// Returns nil, nil when the queue is empty.
func (q *Queue) Dequeue(ctx context.Context) (*models.Job, error) {
	// ZPOPMIN returns the member with the smallest score (= highest priority).
	result, err := q.rdb.ZPopMin(ctx, jobsZSet, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("zpopmin: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	jobID := result[0].Member.(string)

	// Fetch job data.
	data, err := q.rdb.HGet(ctx, jobsHash, jobID).Bytes()
	if err != nil {
		return nil, fmt.Errorf("hget job %s: %w", jobID, err)
	}

	var job models.Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}

	// Track in processing set.
	q.rdb.SAdd(ctx, processingSet, jobID)

	return &job, nil
}

// Complete removes a job from the processing set and hash.
func (q *Queue) Complete(ctx context.Context, jobID string) error {
	pipe := q.rdb.TxPipeline()
	pipe.SRem(ctx, processingSet, jobID)
	pipe.HDel(ctx, jobsHash, jobID)
	_, err := pipe.Exec(ctx)
	return err
}

// Requeue re-adds a failed job with the same priority (retry).
func (q *Queue) Requeue(ctx context.Context, job *models.Job) error {
	pipe := q.rdb.TxPipeline()
	pipe.SRem(ctx, processingSet, job.ID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return q.Enqueue(ctx, job)
}

// MoveToDeadLetter puts a job ID onto the dead-letter list.
func (q *Queue) MoveToDeadLetter(ctx context.Context, jobID string) error {
	pipe := q.rdb.TxPipeline()
	pipe.SRem(ctx, processingSet, jobID)
	pipe.HDel(ctx, jobsHash, jobID)
	pipe.LPush(ctx, deadLetterList, jobID)
	_, err := pipe.Exec(ctx)
	return err
}

// QueueDepth returns the number of jobs waiting in the sorted set.
func (q *Queue) QueueDepth(ctx context.Context) (int64, error) {
	return q.rdb.ZCard(ctx, jobsZSet).Result()
}

// ProcessingCount returns how many jobs are currently being processed.
func (q *Queue) ProcessingCount(ctx context.Context) (int64, error) {
	return q.rdb.SCard(ctx, processingSet).Result()
}

// DeadLetterCount returns the size of the dead-letter list.
func (q *Queue) DeadLetterCount(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, deadLetterList).Result()
}

// UpdateJobData updates the cached job JSON in Redis (e.g., after status change).
func (q *Queue) UpdateJobData(ctx context.Context, job *models.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.rdb.HSet(ctx, jobsHash, job.ID, data).Err()
}
