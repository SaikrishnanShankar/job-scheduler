# Distributed Job Scheduler with Real-time Dashboard

A production-grade distributed job scheduling system built with Go, React, Redis, and PostgreSQL — deployable to Kubernetes via Helm + Terraform.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Browser                               │
│                   React Dashboard (port 3000)                        │
│         ┌──────────────────┬──────────────────┐                     │
│         │   REST API calls │  WebSocket /ws   │                     │
└─────────┼──────────────────┼──────────────────┼─────────────────────┘
          │                  │                  │
          ▼                  ▼                  │
┌─────────────────────────────────────┐         │
│         Go API Server (port 8080)   │◄────────┘
│  ┌────────────┐  ┌────────────────┐ │
│  │ REST Routes│  │  WebSocket Hub │ │
│  │ chi router │  │ (broadcast)    │ │
│  └─────┬──────┘  └────────┬───────┘ │
│        │                  │         │
└────────┼──────────────────┼─────────┘
         │                  │
    ┌────▼────┐        ┌────▼──────────────────────────────────┐
    │PostgreSQL│        │          Redis Priority Queue          │
    │(job      │        │  ZADD jobs:queue  score=−priority+ts  │
    │ history, │        │  HSET jobs:data   id → JSON           │
    │ audit)   │        │  SADD jobs:processing                 │
    └────▲────┘        └────────────────┬──────────────────────┘
         │                              │
         │             ┌────────────────▼──────────────────────┐
         │             │          Go Worker Pool                │
         │             │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ │
         └─────────────│  │  W1  │ │  W2  │ │  W3  │ │  WN  │ │
                       │  └──────┘ └──────┘ └──────┘ └──────┘ │
                       │  • ZPOPMIN (highest priority first)    │
                       │  • Exponential backoff retries         │
                       │  • Dead-letter after MaxRetries        │
                       │  • EventCallback → WS broadcast        │
                       └───────────────────────────────────────┘
                                        │
                            ┌───────────▼──────────┐
                            │   Prometheus /metrics │
                            │   + Grafana Dashboard │
                            └──────────────────────┘
```

## Tech Stack

| Layer        | Technology                              |
|------------- |-----------------------------------------|
| Backend      | Go 1.22, chi router, gorilla/websocket  |
| Queue        | Redis 7 (sorted set priority queue)     |
| Database     | PostgreSQL 16 (pgx/v5 pool)             |
| Frontend     | React 18, TypeScript, Recharts, Vite    |
| Observability| Prometheus + Grafana                    |
| Container    | Docker multi-stage, distroless runtime  |
| Orchestration| Kubernetes (minikube), Helm chart       |
| IaC          | Terraform (kubernetes + helm providers) |
| CI/CD        | GitHub Actions                          |

## Features

- **Priority queue** — jobs scored by `−priority + timestamp`, ensuring high-priority FIFO
- **Configurable worker pool** — N concurrent workers per replica, auto-scales via HPA
- **Retry + exponential backoff** — `min(2^attempt * 1s + jitter, 60s)` with dead-letter after MaxRetries
- **WebSocket live updates** — React dashboard subscribes to real-time job state changes
- **Prometheus metrics** — `jobs_submitted_total`, `jobs_completed_total`, `jobs_failed_total`, `worker_processing_duration_seconds` histogram, queue depth gauges
- **Pre-wired Grafana dashboard** — 10-panel dashboard provisioned automatically
- **Helm chart + Terraform** — full IaC for minikube or any CNCF-compliant cluster
- **GitHub Actions** — build → test → docker push → helm lint → terraform validate

## Quick Start

### Docker Compose (recommended for local dev)

```bash
cp .env.example .env
make dev
# API:       http://localhost:8080
# Dashboard: http://localhost:3000
# Grafana:   http://localhost:3001  (admin/admin)
# Prometheus: http://localhost:9090
```

### Run locally (no Docker)

```bash
# Start dependencies
docker compose up -d postgres redis

# Apply schema
make migrate

# Terminal 1 — API server
make run-server

# Terminal 2 — Worker pool
make run-worker

# Terminal 3 — Frontend
make frontend-install
make frontend-dev
```

### Kubernetes (minikube)

```bash
make minikube-start
make docker-build
make minikube-load
make helm-install

# Access
minikube service job-scheduler-frontend -n job-scheduler
```

### Terraform

```bash
make tf-init
make tf-plan
make tf-apply
```

## API Reference

| Method | Path                  | Description                  |
|--------|-----------------------|------------------------------|
| POST   | `/api/v1/jobs`        | Submit a new job             |
| GET    | `/api/v1/jobs`        | List jobs (limit/offset)     |
| GET    | `/api/v1/jobs/{id}`   | Get job by ID                |
| GET    | `/api/v1/jobs/{id}/audit` | Audit log for a job     |
| GET    | `/api/v1/stats`       | Aggregate job counts         |
| GET    | `/health`             | Health check                 |
| GET    | `/metrics`            | Prometheus metrics           |
| WS     | `/ws`                 | WebSocket live updates       |

### Submit Job (example)

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "process-wafer-batch",
    "payload": "{\"batch_id\": \"W-2024-001\", \"units\": 50}",
    "priority": 9,
    "max_retries": 5
  }'
```

## Job Lifecycle

```
submit → pending ──► running ──► completed
                        │
                        └──► failed ──► (retry w/ backoff) ──► pending
                                                │
                                    (attempts >= max_retries)
                                                │
                                                ▼
                                          dead_letter
```

## Prometheus Metrics

| Metric                                      | Type      | Description                          |
|---------------------------------------------|-----------|--------------------------------------|
| `jobs_submitted_total`                      | Counter   | Jobs submitted via API               |
| `jobs_completed_total`                      | Counter   | Jobs completed successfully          |
| `jobs_failed_total`                         | Counter   | Job failures (all attempts)          |
| `jobs_retried_total`                        | Counter   | Retry attempts                       |
| `jobs_dead_lettered_total`                  | Counter   | Jobs moved to dead-letter            |
| `worker_processing_duration_seconds`        | Histogram | End-to-end job processing time       |
| `worker_processing_duration_by_priority_seconds` | Histogram | Processing time by priority     |
| `worker_active_count`                       | Gauge     | Workers currently processing         |
| `queue_depth`                               | Gauge     | Jobs waiting in Redis queue          |
| `dead_letter_queue_depth`                   | Gauge     | Jobs in dead-letter queue            |
| `websocket_active_connections`              | Gauge     | Active WebSocket dashboard clients   |
| `http_request_duration_seconds`             | Histogram | HTTP latency by method/route/status  |

## Project Structure

```
job-scheduler/
├── cmd/
│   ├── server/          # REST API + WebSocket binary
│   └── worker/          # Worker pool binary
├── internal/
│   ├── api/             # chi routes, HTTP handlers, WS hub
│   ├── metrics/         # Prometheus counters, histograms, gauges
│   ├── models/          # Shared domain types (Job, Stats, etc.)
│   ├── queue/           # Redis sorted-set priority queue
│   ├── store/           # PostgreSQL layer (pgx pool)
│   └── worker/          # Worker pool + exponential backoff
├── frontend/            # React 18 + TypeScript + Vite
│   └── src/
│       ├── components/  # StatsBar, JobTable, SubmitJobForm, StatusChart
│       ├── hooks/       # useJobs, useWebSocket
│       └── types/       # Shared TypeScript types
├── k8s/                 # Raw Kubernetes manifests
├── helm/job-scheduler/  # Helm chart (values, templates)
├── terraform/           # IaC (K8s + Helm providers)
├── monitoring/
│   ├── prometheus.yml
│   └── grafana/         # Provisioned datasource + dashboard JSON
├── .github/workflows/   # CI/CD pipeline
├── Dockerfile.server    # Multi-stage Go server image
├── Dockerfile.worker    # Multi-stage Go worker image
├── Dockerfile.frontend  # Multi-stage React + nginx image
├── docker-compose.yml
├── Makefile
└── README.md
```

## CI/CD Pipeline

```
push to main
    │
    ├── go-test          (go vet, go build, go test -race)
    ├── frontend-build   (tsc --noEmit, npm run build)
    ├── helm-lint        (helm lint + helm template)
    └── terraform-validate
            │
    (all pass + branch=main)
            │
    docker-push (matrix: server, worker, frontend)
        └── GHCR: ghcr.io/<owner>/job-scheduler-{server,worker,frontend}:sha-<sha>
```
