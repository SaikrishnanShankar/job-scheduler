variable "kubeconfig_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Path to kubeconfig file."
}

variable "kube_context" {
  type        = string
  default     = "minikube"
  description = "Kubernetes context to use."
}

variable "namespace" {
  type    = string
  default = "job-scheduler"
}

variable "postgres_user" {
  type    = string
  default = "postgres"
}

variable "postgres_password" {
  type      = string
  default   = "postgres"
  sensitive = true
}

variable "postgres_db" {
  type    = string
  default = "jobscheduler"
}

variable "image_registry" {
  type    = string
  default = "ghcr.io/saikrishnans"
}

variable "image_tag" {
  type    = string
  default = "latest"
}

variable "server_replicas" {
  type    = number
  default = 2
}

variable "worker_replicas" {
  type    = number
  default = 3
}
