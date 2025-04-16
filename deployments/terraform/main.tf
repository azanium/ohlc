resource "digitalocean_kubernetes_cluster" "ohlc" {
  name    = "ohlc-cluster"
  region  = var.do_region
  version = var.do_k8s_version

  node_pool {
    name       = "worker-pool"
    size       = var.do_node_size
    node_count = var.replicas
  }
}

resource "kubernetes_namespace" "ohlc" {
  metadata {
    name = var.namespace
  }
}

resource "kubernetes_secret" "postgres" {
  metadata {
    name      = "postgres-secret"
    namespace = kubernetes_namespace.ohlc.metadata[0].name
  }

  data = {
    POSTGRES_PASSWORD = var.postgres_password
    POSTGRES_USER     = var.postgres_user
    POSTGRES_DB       = var.postgres_db
  }
}

resource "kubernetes_persistent_volume_claim" "postgres" {
  metadata {
    name      = "postgres-pvc"
    namespace = kubernetes_namespace.ohlc.metadata[0].name
  }
  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }
}

resource "kubernetes_deployment" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace.ohlc.metadata[0].name
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "postgres"
      }
    }

    template {
      metadata {
        labels = {
          app = "postgres"
        }
      }

      spec {
        container {
          name  = "postgres"
          image = "postgres:15"

          port {
            container_port = var.postgres_port
          }

          env_from {
            secret_ref {
              name = kubernetes_secret.postgres.metadata[0].name
            }
          }

          env {
            name  = "PGDATA"
            value = "/var/lib/postgresql/data/pgdata"
          }

          volume_mount {
            name       = "postgres-storage"
            mount_path = "/var/lib/postgresql/data"
          }
        }

        volume {
          name = "postgres-storage"
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
    name      = "postgres"
    namespace = kubernetes_namespace.ohlc.metadata[0].name
  }

  spec {
    selector = {
      app = "postgres"
    }

    port {
      port        = var.postgres_port
      target_port = var.postgres_port
    }
  }
}

# resource "helm_release" "ohlc" {
#   name       = "ohlc"
#   namespace  = kubernetes_namespace.ohlc.metadata[0].name
#   chart      = "${path.module}/../helm/ohlc"

#   set {
#     name  = "replicaCount"
#     value = var.replicas
#   }

#   set {
#     name  = "image.repository"
#     value = split(":", var.image)[0]
#   }

#   set {
#     name  = "image.tag"
#     value = split(":", var.image)[1]
#   }

#   set {
#     name  = "grpc.port"
#     value = var.grpc_port
#   }

#   set {
#     name  = "postgresql.host"
#     value = "${kubernetes_service.postgres.metadata[0].name}"
#   }

#   set {
#     name  = "postgresql.port"
#     value = var.postgres_port
#   }

#   set {
#     name  = "postgresql.database"
#     value = var.postgres_db
#   }

#   set {
#     name  = "postgresql.user"
#     value = var.postgres_user
#   }

#   set {
#     name  = "postgresql.password"
#     value = var.postgres_password
#   }
# }