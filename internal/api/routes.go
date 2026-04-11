package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saikrishnans/job-scheduler/internal/metrics"
)

func NewRouter(h *Handler, hub *Hub) http.Handler {
	r := chi.NewRouter()

	// Middleware stack.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))
	r.Use(prometheusMiddleware)

	// REST routes.
	r.Get("/health", h.Health)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/jobs", h.SubmitJob)
		r.Get("/jobs", h.ListJobs)
		r.Get("/jobs/{id}", h.GetJob)
		r.Get("/jobs/{id}/audit", h.GetAuditLog)
		r.Get("/stats", h.GetStats)
	})

	// WebSocket.
	r.Get("/ws", hub.ServeWS)

	return r
}

// prometheusMiddleware records HTTP request duration.
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		metrics.HTTPRequestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(ww.Status()),
		).Observe(time.Since(start).Seconds())
	})
}
