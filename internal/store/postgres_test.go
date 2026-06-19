package store

import (
	"context"
	"os"
	"testing"

	"github.com/saikrishnans/job-scheduler/internal/models"
)

func testStore(t *testing.T) *Store {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable"
	}
	s, err := New(context.Background(), dsn)
	if err != nil {
		t.Skipf("postgres not available at %s: %v", dsn, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetJob(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	req := models.CreateJobRequest{
		Name:       "test-create-job",
		Payload:    `{"k":"v"}`,
		Priority:   7,
		MaxRetries: 4,
	}

	created, err := s.CreateJob(ctx, req)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated ID, got empty string")
	}
	if created.Status != models.StatusPending {
		t.Errorf("expected status pending, got %s", created.Status)
	}
	if created.Priority != 7 {
		t.Errorf("expected priority 7, got %d", created.Priority)
	}

	fetched, err := s.GetJob(ctx, created.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if fetched.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, fetched.Name)
	}
}

func TestUpdateJobStatusTransitions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	job, err := s.CreateJob(ctx, models.CreateJobRequest{
		Name: "test-status-transition", Payload: "{}", Priority: 5, MaxRetries: 3,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := s.UpdateJobStatus(ctx, job.ID, models.StatusRunning, "worker-1", ""); err != nil {
		t.Fatalf("update to running: %v", err)
	}
	running, err := s.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if running.Status != models.StatusRunning {
		t.Errorf("expected status running, got %s", running.Status)
	}
	if running.StartedAt == nil {
		t.Error("expected started_at to be set when transitioning to running")
	}
	if running.Attempts != 1 {
		t.Errorf("expected attempts incremented to 1, got %d", running.Attempts)
	}

	if err := s.UpdateJobStatus(ctx, job.ID, models.StatusCompleted, "worker-1", ""); err != nil {
		t.Fatalf("update to completed: %v", err)
	}
	completed, err := s.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if completed.Status != models.StatusCompleted {
		t.Errorf("expected status completed, got %s", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("expected completed_at to be set when transitioning to completed")
	}
}

func TestMarkDeadLetter(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	job, err := s.CreateJob(ctx, models.CreateJobRequest{
		Name: "test-dead-letter", Payload: "{}", Priority: 5, MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := s.MarkDeadLetter(ctx, job.ID, "retries exhausted"); err != nil {
		t.Fatalf("mark dead letter: %v", err)
	}

	dl, err := s.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if dl.Status != models.StatusDeadLetter {
		t.Errorf("expected status dead_letter, got %s", dl.Status)
	}
	if dl.Error != "retries exhausted" {
		t.Errorf("expected error message to be recorded, got %q", dl.Error)
	}
}

func TestAuditLogAppendsHistory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	job, err := s.CreateJob(ctx, models.CreateJobRequest{
		Name: "test-audit-log", Payload: "{}", Priority: 5, MaxRetries: 3,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	events := []string{"running", "retry", "completed"}
	for _, e := range events {
		if err := s.AddAuditLog(ctx, job.ID, e, "detail-"+e); err != nil {
			t.Fatalf("add audit log %s: %v", e, err)
		}
	}

	logs, err := s.GetAuditLogs(ctx, job.ID)
	if err != nil {
		t.Fatalf("get audit logs: %v", err)
	}
	if len(logs) != len(events) {
		t.Fatalf("expected %d audit log entries, got %d", len(events), len(logs))
	}
	for i, e := range events {
		if logs[i].Event != e {
			t.Errorf("expected event[%d] = %q, got %q", i, e, logs[i].Event)
		}
	}
}

func TestGetStatsCountsByStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	before, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	job, err := s.CreateJob(ctx, models.CreateJobRequest{
		Name: "test-stats", Payload: "{}", Priority: 5, MaxRetries: 3,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	_ = job

	after, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if after.Total != before.Total+1 {
		t.Errorf("expected total to increase by 1, before=%d after=%d", before.Total, after.Total)
	}
	if after.Pending != before.Pending+1 {
		t.Errorf("expected pending to increase by 1, before=%d after=%d", before.Pending, after.Pending)
	}
}
