package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/saikrishnans/job-scheduler/internal/metrics"
	"github.com/saikrishnans/job-scheduler/internal/models"
	"github.com/saikrishnans/job-scheduler/internal/queue"
	"github.com/saikrishnans/job-scheduler/internal/store"
)

type Handler struct {
	store *store.Store
	queue *queue.Queue
	hub   *Hub
}

func NewHandler(s *store.Store, q *queue.Queue, hub *Hub) *Handler {
	return &Handler{store: s, queue: q, hub: hub}
}

// POST /jobs
func (h *Handler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Defaults and validation.
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Priority < 1 || req.Priority > 10 {
		req.Priority = 5
	}
	if req.MaxRetries <= 0 {
		req.MaxRetries = 3
	}
	if req.Payload == "" {
		req.Payload = "{}"
	}

	job, err := h.store.CreateJob(r.Context(), req)
	if err != nil {
		log.Printf("create job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	if err := h.queue.Enqueue(r.Context(), job); err != nil {
		log.Printf("enqueue job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	metrics.JobsSubmitted.Inc()

	// Broadcast to WebSocket clients.
	h.hub.Broadcast(models.WSMessage{Type: "job_created", Payload: job})

	writeJSON(w, http.StatusCreated, job)
}

// GET /jobs
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	jobs, err := h.store.ListJobs(r.Context(), limit, offset)
	if err != nil {
		log.Printf("list jobs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	if jobs == nil {
		jobs = []*models.Job{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

// GET /jobs/{id}
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// GET /jobs/{id}/audit
func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logs, err := h.store.GetAuditLogs(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch audit logs")
		return
	}
	if logs == nil {
		logs = []*models.AuditLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}

// GET /stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
