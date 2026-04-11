package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/saikrishnans/job-scheduler/internal/api"
	"github.com/saikrishnans/job-scheduler/internal/queue"
	"github.com/saikrishnans/job-scheduler/internal/store"
)

func main() {
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL.
	dbDSN := mustEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable")
	db, err := store.New(ctx, dbDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()
	log.Println("connected to PostgreSQL")

	// Redis.
	redisAddr := mustEnv("REDIS_ADDR", "localhost:6379")
	redisPass := os.Getenv("REDIS_PASSWORD")
	q := queue.New(redisAddr, redisPass, 0)
	if err := q.Ping(ctx); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer q.Close()
	log.Println("connected to Redis")

	// Wire up API.
	hub := api.NewHub()
	handler := api.NewHandler(db, q, hub)
	router := api.NewRouter(handler, hub)

	port := mustEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func mustEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
