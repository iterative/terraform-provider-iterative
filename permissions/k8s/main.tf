 resource "kubernetes_service_account" "task" {
  metadata {
    name = "task"
  }

  secret {
    name = "${kubernetes_secret.task.metadata.0.name}"
  }
}

resource "kubernetes_secret" "task" {
  metadata {
    name = "task"
  }
}

resource "kubernetes_role" "task" {
  metadata {
    name = "task"
  }

  rule {
    api_groups     = ["", "apps", "batch"]
    resources      = ["jobs", "pods"]
    resource_names = ["tpi-*"]
    verbs          = ["*"]
  }
}

resource "kubernetes_role_binding" "task" {
  metadata {
    name      = "task"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "task"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "task"
  }
}