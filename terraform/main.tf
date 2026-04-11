terraform {
  required_version = ">= 1.7"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.13"
    }
  }
}

# ── Providers ─────────────────────────────────────────────────────────────────
provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = var.kube_context
}

provider "helm" {
  kubernetes {
    config_path    = var.kubeconfig_path
    config_context = var.kube_context
  }
}

# ── Namespace ─────────────────────────────────────────────────────────────────
resource "kubernetes_namespace" "job_scheduler" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
}

# ── Redis (in-cluster) ────────────────────────────────────────────────────────
resource "kubernetes_deployment" "redis" {
  metadata {
    name      = "redis"
    namespace = kubernetes_namespace.job_scheduler.metadata[0].name
  }
  spec {
    replicas = 1
    selector {
      match_labels = { app = "redis" }
    }
    template {
      metadata { labels = { app = "redis" } }
      spec {
        container {
          name  = "redis"
          image = "redis:7-alpine"
          port { container_port = 6379 }
          resources {
            requests = { cpu = "50m", memory = "64Mi" }
            limits   = { cpu = "200m", memory = "128Mi" }
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "redis" {
  metadata {
    name      = "redis-service"
    namespace = kubernetes_namespace.job_scheduler.metadata[0].name
  }
  spec {
    selector = { app = "redis" }
    port {
      port        = 6379
      target_port = 6379
    }
  }
}

# ── PostgreSQL (in-cluster) ───────────────────────────────────────────────────
resource "kubernetes_persistent_volume_claim" "postgres" {
  metadata {
    name      = "postgres-pvc"
    namespace = kubernetes_namespace.job_scheduler.metadata[0].name
  }
  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = { storage = "5Gi" }
    }
  }
}

resource "kubernetes_deployment" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace.job_scheduler.metadata[0].name
  }
  spec {
    replicas = 1
    selector {
      match_labels = { app = "postgres" }
    }
    template {
      metadata { labels = { app = "postgres" } }
      spec {
        container {
          name  = "postgres"
          image = "postgres:16-alpine"
          env {
            name  = "POSTGRES_USER"
            value = var.postgres_user
          }
          env {
            name  = "POSTGRES_PASSWORD"
            value = var.postgres_password
          }
          env {
            name  = "POSTGRES_DB"
            value = var.postgres_db
          }
          port { container_port = 5432 }
          volume_mount {
            name       = "postgres-data"
            mount_path = "/var/lib/postgresql/data"
          }
          resources {
            requests = { cpu = "100m", memory = "256Mi" }
            limits   = { cpu = "500m", memory = "512Mi" }
          }
        }
        volume {
          name = "postgres-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.postgres.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "postgres" {
  metadata {
    name      = "postgres-service"
    namespace = kubernetes_namespace.job_scheduler.metadata[0].name
  }
  spec {
    selector = { app = "postgres" }
    port {
      port        = 5432
      target_port = 5432
    }
  }
}

# ── Job Scheduler Helm Release ────────────────────────────────────────────────
resource "helm_release" "job_scheduler" {
  name       = "job-scheduler"
  chart      = "${path.module}/../helm/job-scheduler"
  namespace  = kubernetes_namespace.job_scheduler.metadata[0].name
  depends_on = [kubernetes_deployment.redis, kubernetes_deployment.postgres]

  set {
    name  = "image.registry"
    value = var.image_registry
  }
  set {
    name  = "image.server.tag"
    value = var.image_tag
  }
  set {
    name  = "image.worker.tag"
    value = var.image_tag
  }
  set {
    name  = "image.frontend.tag"
    value = var.image_tag
  }
  set {
    name  = "replicaCount.server"
    value = var.server_replicas
  }
  set {
    name  = "replicaCount.worker"
    value = var.worker_replicas
  }
}
