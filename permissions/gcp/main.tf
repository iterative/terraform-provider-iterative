terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.12.0"
    }
  }
}

provider "google" {
  project = "cml-gcp-test"
}

data "google_project" "current" {}

resource "google_service_account" "task" {
  account_id = "task-service-account"
}

resource "google_service_account_key" "task" {
  service_account_id = google_service_account.task.email
}

resource "google_project_iam_binding" "task" {
  project = data.google_project.current.project_id
  role    = "projects/${data.google_project.current.project_id}/roles/${google_project_iam_custom_role.task.role_id}"
  members = ["serviceAccount:${google_service_account.task.email}"]
}

resource "google_project_iam_custom_role" "task" {
  role_id = replace("${google_service_account.task.account_id}-role", "-", "_")
  title   = replace("${google_service_account.task.account_id}-role", "-", "_")
  permissions = [
    "compute.acceleratorTypes.get",

    "compute.disks.create",
    "compute.diskTypes.get",

    "compute.firewalls.create",
    "compute.firewalls.delete",
    "compute.firewalls.get",

    "compute.globalOperations.get",

    "compute.instanceGroupManagers.create", # remove for iterative_machine
    "compute.instanceGroupManagers.delete", # remove for iterative_machine
    "compute.instanceGroupManagers.get",    # remove for iterative_machine
    "compute.instanceGroupManagers.update", # remove for iterative_machine

    "compute.instanceGroups.create", # remove for iterative_machine
    "compute.instanceGroups.delete", # remove for iterative_machine
    "compute.instanceGroups.get",    # remove for iterative_machine

    "compute.instances.create",
    "compute.instances.delete",
    "compute.instances.get",

    "compute.instances.setMetadata",
    "compute.instances.setServiceAccount",
    "compute.instances.setTags",

    "compute.instanceTemplates.create",      # remove for iterative_machine
    "compute.instanceTemplates.delete",      # remove for iterative_machine
    "compute.instanceTemplates.get",         # remove for iterative_machine
    "compute.instanceTemplates.useReadOnly", # remove for iterative_machine

    "compute.machineTypes.get",

    "compute.networks.get",
    "compute.networks.updatePolicy",

    "compute.subnetworks.use",
    "compute.subnetworks.useExternalIp",

    "compute.zoneOperations.get",

    "iam.serviceAccounts.actAs",

    "storage.buckets.create", # remove for iterative_machine
    "storage.buckets.delete", # remove for iterative_machine
    "storage.buckets.get",    # remove for iterative_machine

    "storage.multipartUploads.abort",     # remove for iterative_machine
    "storage.multipartUploads.create",    # remove for iterative_machine
    "storage.multipartUploads.list",      # remove for iterative_machine
    "storage.multipartUploads.listParts", # remove for iterative_machine

    "storage.objects.create", # remove for iterative_machine
    "storage.objects.delete", # remove for iterative_machine
    "storage.objects.get",    # remove for iterative_machine
    "storage.objects.list",   # remove for iterative_machine
    "storage.objects.update", # remove for iterative_machine
  ]
}

output "google_application_credentials_data" {
  value     = base64decode(google_service_account_key.task.private_key)
  sensitive = true
}