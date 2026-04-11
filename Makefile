.PHONY: all build test lint docker-build docker-up docker-down \
        k8s-apply k8s-delete helm-install helm-uninstall \
        tf-init tf-plan tf-apply tf-destroy \
        frontend-install frontend-dev frontend-build \
        migrate run-server run-worker

# ── Config ────────────────────────────────────────────────────────────────────
REGISTRY      ?= ghcr.io/saikrishnans
TAG           ?= latest
NAMESPACE     ?= job-scheduler
HELM_RELEASE  ?= job-scheduler
DATABASE_URL  ?= postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable
REDIS_ADDR    ?= localhost:6379

# ── Go ────────────────────────────────────────────────────────────────────────
all: build test

build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server
	go build -ldflags="-s -w" -o bin/worker ./cmd/worker

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not installed"

run-server:
	DATABASE_URL=$(DATABASE_URL) REDIS_ADDR=$(REDIS_ADDR) go run ./cmd/server

run-worker:
	DATABASE_URL=$(DATABASE_URL) REDIS_ADDR=$(REDIS_ADDR) WORKER_CONCURRENCY=5 go run ./cmd/worker

# ── Database ──────────────────────────────────────────────────────────────────
migrate:
	psql "$(DATABASE_URL)" -f internal/store/schema.sql

# ── Frontend ──────────────────────────────────────────────────────────────────
frontend-install:
	cd frontend && npm ci

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

# ── Docker ────────────────────────────────────────────────────────────────────
docker-build:
	docker build -f Dockerfile.server   -t $(REGISTRY)/job-scheduler-server:$(TAG)   .
	docker build -f Dockerfile.worker   -t $(REGISTRY)/job-scheduler-worker:$(TAG)   .
	docker build -f Dockerfile.frontend -t $(REGISTRY)/job-scheduler-frontend:$(TAG) .

docker-push: docker-build
	docker push $(REGISTRY)/job-scheduler-server:$(TAG)
	docker push $(REGISTRY)/job-scheduler-worker:$(TAG)
	docker push $(REGISTRY)/job-scheduler-frontend:$(TAG)

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f server worker

# ── Kubernetes (raw manifests) ────────────────────────────────────────────────
k8s-apply:
	kubectl apply -f k8s/

k8s-delete:
	kubectl delete -f k8s/

# ── Helm ──────────────────────────────────────────────────────────────────────
helm-lint:
	helm lint helm/job-scheduler/

helm-install:
	helm upgrade --install $(HELM_RELEASE) helm/job-scheduler/ \
	  --namespace $(NAMESPACE) --create-namespace \
	  --set image.registry=$(REGISTRY) \
	  --set image.server.tag=$(TAG) \
	  --set image.worker.tag=$(TAG) \
	  --set image.frontend.tag=$(TAG)

helm-uninstall:
	helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE)

# ── Terraform ─────────────────────────────────────────────────────────────────
tf-init:
	cd terraform && terraform init

tf-plan:
	cd terraform && terraform plan -var="image_tag=$(TAG)"

tf-apply:
	cd terraform && terraform apply -var="image_tag=$(TAG)" -auto-approve

tf-destroy:
	cd terraform && terraform destroy -auto-approve

# ── Minikube quick-start ──────────────────────────────────────────────────────
minikube-start:
	minikube start --driver=docker --cpus=4 --memory=4096
	minikube addons enable ingress
	minikube addons enable metrics-server

minikube-load:
	minikube image load $(REGISTRY)/job-scheduler-server:$(TAG)
	minikube image load $(REGISTRY)/job-scheduler-worker:$(TAG)
	minikube image load $(REGISTRY)/job-scheduler-frontend:$(TAG)

# ── Dev quick-start ───────────────────────────────────────────────────────────
dev: docker-up
	@echo "Services running:"
	@echo "  API:      http://localhost:8080"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Grafana:  http://localhost:3001  (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
