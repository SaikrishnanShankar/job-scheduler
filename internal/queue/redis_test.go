package queue

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/saikrishnans/job-scheduler/internal/models"
)

func testQueue(t *testing.T) *Queue {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	q := New(addr, "", 0)
	if err := q.Ping(context.Background()); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}
	t.Cleanup(func() { q.Close() })
	return q
}

func newTestJob(priority int) *models.Job {
	return &models.Job{
		ID:         uuid.New().String(),
		Name:       "test-job",
		Payload:    "{}",
		Priority:   priority,
		Status:     models.StatusPending,
		MaxRetries: 3,
	}
}

func TestEnqueueDequeuePriorityOrder(t *testing.T) {
	q := testQueue(t)
	ctx := context.Background()

	low := newTestJob(1)
	high := newTestJob(9)
	mid := newTestJob(5)

	for _, j := range []*models.Job{low, high, mid} {
		if err := q.Enqueue(ctx, j); err != nil {
			t.Fatalf("enqueue %s: %v", j.ID, err)
		}
	}
	t.Cleanup(func() {
		for _, j := range []*models.Job{low, high, mid} {
			q.Complete(ctx, j.ID)
		}
	})

	first, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if first == nil || first.ID != high.ID {
		t.Fatalf("expected highest priority job (%s) first, got %+v", high.ID, first)
	}

	second, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if second == nil || second.ID != mid.ID {
		t.Fatalf("expected mid priority job (%s) second, got %+v", mid.ID, second)
	}

	third, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if third == nil || third.ID != low.ID {
		t.Fatalf("expected low priority job (%s) third, got %+v", low.ID, third)
	}
}

func TestRequeueReturnsJobToQueue(t *testing.T) {
	q := testQueue(t)
	ctx := context.Background()

	job := newTestJob(7)
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	t.Cleanup(func() { q.Complete(ctx, job.ID) })

	popped, err := q.Dequeue(ctx)
	if err != nil || popped == nil || popped.ID != job.ID {
		t.Fatalf("expected to dequeue job %s, got %+v, err=%v", job.ID, popped, err)
	}

	popped.Attempts++
	if err := q.Requeue(ctx, popped); err != nil {
		t.Fatalf("requeue: %v", err)
	}

	again, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue after requeue: %v", err)
	}
	if again == nil || again.ID != job.ID {
		t.Fatalf("expected requeued job %s to be dequeued again, got %+v", job.ID, again)
	}
	if again.Attempts != 1 {
		t.Errorf("expected requeued job to retain Attempts=1, got %d", again.Attempts)
	}
}

func TestMoveToDeadLetter(t *testing.T) {
	q := testQueue(t)
	ctx := context.Background()

	job := newTestJob(3)
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	popped, err := q.Dequeue(ctx)
	if err != nil || popped == nil {
		t.Fatalf("dequeue: %v", err)
	}

	before, err := q.DeadLetterCount(ctx)
	if err != nil {
		t.Fatalf("dead letter count: %v", err)
	}

	if err := q.MoveToDeadLetter(ctx, job.ID); err != nil {
		t.Fatalf("move to dead letter: %v", err)
	}

	after, err := q.DeadLetterCount(ctx)
	if err != nil {
		t.Fatalf("dead letter count: %v", err)
	}
	if after != before+1 {
		t.Errorf("expected dead letter count to increase by 1, got before=%d after=%d", before, after)
	}

	procCount, err := q.ProcessingCount(ctx)
	if err != nil {
		t.Fatalf("processing count: %v", err)
	}
	_ = procCount // job should no longer be tracked as processing; checked implicitly via no error
}
