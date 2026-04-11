package models

import (
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusDeadLetter JobStatus = "dead_letter"
)

type Job struct {
	ID          string            `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Payload     string            `json:"payload" db:"payload"`
	Priority    int               `json:"priority" db:"priority"` // 1 (low) - 10 (high)
	Status      JobStatus         `json:"status" db:"status"`
	MaxRetries  int               `json:"max_retries" db:"max_retries"`
	Attempts    int               `json:"attempts" db:"attempts"`
	WorkerID    string            `json:"worker_id,omitempty" db:"worker_id"`
	Error       string            `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	Duration    *float64          `json:"duration_ms,omitempty" db:"duration_ms"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type CreateJobRequest struct {
	Name       string            `json:"name"`
	Payload    string            `json:"payload"`
	Priority   int               `json:"priority"`
	MaxRetries int               `json:"max_retries"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type JobStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Running    int `json:"running"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	DeadLetter int `json:"dead_letter"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type AuditLog struct {
	ID        int64     `json:"id" db:"id"`
	JobID     string    `json:"job_id" db:"job_id"`
	Event     string    `json:"event" db:"event"`
	Details   string    `json:"details" db:"details"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
