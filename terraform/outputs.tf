output "namespace" {
  value       = kubernetes_namespace.job_scheduler.metadata[0].name
  description = "Kubernetes namespace for the job scheduler."
}

output "helm_release_status" {
  value       = helm_release.job_scheduler.status
  description = "Helm release status."
}

output "postgres_service" {
  value       = kubernetes_service.postgres.metadata[0].name
  description = "PostgreSQL service name."
}

output "redis_service" {
  value       = kubernetes_service.redis.metadata[0].name
  description = "Redis service name."
}
