package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saikrishnans/job-scheduler/internal/models"
)

type Store struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{db: pool}, nil
}

func (s *Store) Close() {
	s.db.Close()
}

// CreateJob inserts a new job and returns it with generated fields.
func (s *Store) CreateJob(ctx context.Context, req models.CreateJobRequest) (*models.Job, error) {
	const q = `
		INSERT INTO jobs (name, payload, priority, max_retries)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, payload, priority, status, max_retries,
		          attempts, worker_id, error, created_at, updated_at,
		          started_at, completed_at, duration_ms`

	row := s.db.QueryRow(ctx, q, req.Name, req.Payload, req.Priority, req.MaxRetries)
	return scanJob(row)
}

// GetJob fetches a single job by ID.
func (s *Store) GetJob(ctx context.Context, id string) (*models.Job, error) {
	const q = `
		SELECT id, name, payload, priority, status, max_retries,
		       attempts, worker_id, error, created_at, updated_at,
		       started_at, completed_at, duration_ms
		FROM jobs WHERE id = $1`

	row := s.db.QueryRow(ctx, q, id)
	return scanJob(row)
}

// ListJobs returns paginated jobs, newest first.
func (s *Store) ListJobs(ctx context.Context, limit, offset int) ([]*models.Job, error) {
	const q = `
		SELECT id, name, payload, priority, status, max_retries,
		       attempts, worker_id, error, created_at, updated_at,
		       started_at, completed_at, duration_ms
		FROM jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// UpdateJobStatus transitions a job's status and related fields.
func (s *Store) UpdateJobStatus(ctx context.Context, id string, status models.JobStatus, workerID, errMsg string) error {
	const q = `
		UPDATE jobs SET
			status    = $2::job_status,
			worker_id = NULLIF($3, ''),
			error     = NULLIF($4, ''),
			started_at   = CASE WHEN $2::job_status = 'running'   THEN NOW() ELSE started_at   END,
			completed_at = CASE WHEN $2::job_status IN ('completed','failed','dead_letter') THEN NOW() ELSE completed_at END,
			duration_ms  = CASE WHEN $2::job_status IN ('completed','failed','dead_letter') AND started_at IS NOT NULL
			                    THEN EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000
			                    ELSE duration_ms END,
			attempts  = CASE WHEN $2::job_status = 'running' THEN attempts + 1 ELSE attempts END
		WHERE id = $1`
	_, err := s.db.Exec(ctx, q, id, status, workerID, errMsg)
	return err
}

// MarkDeadLetter moves a job to dead_letter queue.
func (s *Store) MarkDeadLetter(ctx context.Context, id, reason string) error {
	return s.UpdateJobStatus(ctx, id, models.StatusDeadLetter, "", reason)
}

// GetStats returns aggregate counts per status.
func (s *Store) GetStats(ctx context.Context) (*models.JobStats, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'pending')     AS pending,
			COUNT(*) FILTER (WHERE status = 'running')     AS running,
			COUNT(*) FILTER (WHERE status = 'completed')   AS completed,
			COUNT(*) FILTER (WHERE status = 'failed')      AS failed,
			COUNT(*) FILTER (WHERE status = 'dead_letter') AS dead_letter
		FROM jobs`

	var s2 models.JobStats
	err := s.db.QueryRow(ctx, q).Scan(
		&s2.Total, &s2.Pending, &s2.Running,
		&s2.Completed, &s2.Failed, &s2.DeadLetter,
	)
	return &s2, err
}

// AddAuditLog appends an audit entry.
func (s *Store) AddAuditLog(ctx context.Context, jobID, event, details string) error {
	const q = `INSERT INTO audit_logs (job_id, event, details) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(ctx, q, jobID, event, details)
	return err
}

// GetAuditLogs returns audit history for a job.
func (s *Store) GetAuditLogs(ctx context.Context, jobID string) ([]*models.AuditLog, error) {
	const q = `
		SELECT id, job_id, event, details, created_at
		FROM audit_logs WHERE job_id = $1 ORDER BY created_at ASC`

	rows, err := s.db.Query(ctx, q, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.JobID, &l.Event, &l.Details, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

// scanner abstracts pgx Row/Rows for scanJob.
type scanner interface {
	Scan(dest ...any) error
}

func scanJob(s scanner) (*models.Job, error) {
	var j models.Job
	var workerID, errMsg *string
	err := s.Scan(
		&j.ID, &j.Name, &j.Payload, &j.Priority,
		&j.Status, &j.MaxRetries, &j.Attempts,
		&workerID, &errMsg,
		&j.CreatedAt, &j.UpdatedAt,
		&j.StartedAt, &j.CompletedAt, &j.Duration,
	)
	if err != nil {
		return nil, err
	}
	if workerID != nil {
		j.WorkerID = *workerID
	}
	if errMsg != nil {
		j.Error = *errMsg
	}
	return &j, nil
}
