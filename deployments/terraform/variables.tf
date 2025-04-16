variable "do_token" {
  description = "DigitalOcean API Token"
  type        = string
}

variable "do_region" {
  description = "DigitalOcean region for the cluster"
  type        = string
  default     = "nyc1"
}

variable "do_k8s_version" {
  description = "Kubernetes version for the cluster"
  type        = string
  default     = "1.28.2-do.0"
}

variable "do_node_size" {
  description = "Size of the worker nodes"
  type        = string
  default     = "s-2vcpu-4gb"
}

variable "namespace" {
  description = "Kubernetes namespace for OHLC service"
  type        = string
  default     = "ohlc"
}

variable "replicas" {
  description = "Number of OHLC service replicas"
  type        = number
  default     = 1
}

variable "image" {
  description = "Docker image for OHLC service"
  type        = string
  default     = "ohlc:latest"
}

variable "storage_size" {
  description = "Storage size for OHLC data"
  type        = string
  default     = "1Gi"
}

variable "grpc_port" {
  description = "gRPC port for OHLC service"
  type        = number
  default     = 50051
}

variable "postgres_password" {
  description = "PostgreSQL database password"
  type        = string
  sensitive   = true
}

variable "postgres_user" {
  description = "PostgreSQL database user"
  type        = string
  default     = "ohlc"
}

variable "postgres_db" {
  description = "PostgreSQL database name"
  type        = string
  default     = "ohlc"
}

variable "postgres_port" {
  description = "PostgreSQL database port"
  type        = number
  default     = 5432
}