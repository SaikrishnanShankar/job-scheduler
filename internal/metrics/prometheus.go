package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsSubmitted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_submitted_total",
		Help: "Total number of jobs submitted.",
	})

	JobsCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_completed_total",
		Help: "Total number of jobs completed successfully.",
	})

	JobsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_failed_total",
		Help: "Total number of jobs that failed (including retries exhausted).",
	})

	JobsDeadLettered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_dead_lettered_total",
		Help: "Total number of jobs moved to dead-letter queue.",
	})

	JobsRetried = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_retried_total",
		Help: "Total number of job retry attempts.",
	})

	WorkerProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "worker_processing_duration_seconds",
		Help:    "Time taken by a worker to process a job.",
		Buckets: prometheus.DefBuckets,
	})

	WorkerProcessingDurationByPriority = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_processing_duration_by_priority_seconds",
			Help:    "Processing duration broken down by job priority.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"priority"},
	)

	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "worker_active_count",
		Help: "Number of workers currently processing a job.",
	})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Number of jobs currently waiting in the Redis priority queue.",
	})

	DeadLetterQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dead_letter_queue_depth",
		Help: "Number of jobs in the dead-letter queue.",
	})

	WebSocketConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_active_connections",
		Help: "Number of active WebSocket connections to the dashboard.",
	})

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)
)
