terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.8.0"
    }
  }
}

provider "kubernetes" {
  config_path = pathexpand("~/.kube/config")
}

resource "kubernetes_service_account" "task1" {
  metadata {
    name = "task1"
  }

  secret {
    name = kubernetes_secret.task1.metadata.0.name
  }
}

resource "kubernetes_secret" "task1" {
  metadata {
    name = "task1"
  }
}

resource "kubernetes_role" "task1" {
  metadata {
    name = "task1"
  }

  rule {
    api_groups     = ["", "apps", "batch"]
    resources      = ["jobs", "pods"]
    resource_names = ["tpi-*"]
    verbs          = ["*"]
  }
}

resource "kubernetes_role_binding" "task1" {
  metadata {
    name = "task1"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "task1"
  }

  subject {
    kind = "ServiceAccount"
    name = "task1"
  }
}

output "kubeconfig_data" {
  sensitive = true

  value = yamlencode({
    apiVersion = "v1"
    clusters = [{
      cluster = {
        "certificate-authority-data" = lookup(kubernetes_secret.task1.data, "ca.crt", null)
        server                       = var.server
      }
      name = "cluster"
    }]
    contexts = [{
      context = {
        cluster   = "cluster"
        namespace = lookup(kubernetes_secret.task1.data, "namespace", null)
        user      = kubernetes_service_account.task1.metadata.name
      }
      name = "cluster"
    }]
    "current-context" = "cluster"
    kind              = "Config"
    preferences       = {}
    users = [{
      name = kubernetes_service_account.task1.metadata.name
      user = {
        token = lookup(kubernetes_secret.task1.data, "token", null)
      }
    }]
  })
}