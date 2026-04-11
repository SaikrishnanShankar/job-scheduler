package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/saikrishnans/job-scheduler/internal/queue"
	"github.com/saikrishnans/job-scheduler/internal/store"
	"github.com/saikrishnans/job-scheduler/internal/worker"
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

	// Redis.
	redisAddr := mustEnv("REDIS_ADDR", "localhost:6379")
	redisPass := os.Getenv("REDIS_PASSWORD")
	q := queue.New(redisAddr, redisPass, 0)
	if err := q.Ping(ctx); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer q.Close()

	concurrency := 5
	if v := os.Getenv("WORKER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			concurrency = n
		}
	}

	pool := worker.NewPool(concurrency, q, db, nil) // no WS broadcast from worker binary
	pool.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("received shutdown signal")
	pool.Stop()
	log.Println("worker pool stopped")
}

func mustEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
