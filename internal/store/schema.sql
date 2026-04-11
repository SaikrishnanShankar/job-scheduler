-- Job Scheduler PostgreSQL Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE job_status AS ENUM (
    'pending',
    'running',
    'completed',
    'failed',
    'dead_letter'
);

CREATE TABLE IF NOT EXISTS jobs (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name         VARCHAR(255) NOT NULL,
    payload      TEXT NOT NULL DEFAULT '{}',
    priority     SMALLINT NOT NULL DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    status       job_status NOT NULL DEFAULT 'pending',
    max_retries  SMALLINT NOT NULL DEFAULT 3,
    attempts     SMALLINT NOT NULL DEFAULT 0,
    worker_id    VARCHAR(255),
    error        TEXT,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms  DOUBLE PRECISION
);

CREATE INDEX idx_jobs_status     ON jobs(status);
CREATE INDEX idx_jobs_priority   ON jobs(priority DESC);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_worker_id  ON jobs(worker_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id         BIGSERIAL PRIMARY KEY,
    job_id     UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    event      VARCHAR(100) NOT NULL,
    details    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_job_id    ON audit_logs(job_id);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at DESC);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
