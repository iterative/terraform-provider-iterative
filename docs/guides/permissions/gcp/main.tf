terraform {
  required_providers {
    google = { source = "hashicorp/google", version = "~> 4.12.0" }
  }
}

variable "gcp_project" {
  description = "Name of the Google Cloud project to use"
}

provider "google" {
  project = var.gcp_project
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
    "compute.diskTypes.get",
    "compute.disks.create",
    "compute.firewalls.create",
    "compute.firewalls.delete",
    "compute.firewalls.get",
    "compute.globalOperations.get",
    "compute.instanceGroupManagers.create",
    "compute.instanceGroupManagers.delete",
    "compute.instanceGroupManagers.get",
    "compute.instanceGroupManagers.update",
    "compute.instanceGroups.create",
    "compute.instanceGroups.delete",
    "compute.instanceGroups.get",
    "compute.instanceTemplates.create",
    "compute.instanceTemplates.delete",
    "compute.instanceTemplates.get",
    "compute.instanceTemplates.useReadOnly",
    "compute.instances.create",
    "compute.instances.delete",
    "compute.instances.get",
    "compute.instances.setMetadata",
    "compute.instances.setServiceAccount",
    "compute.instances.setTags",
    "compute.machineTypes.get",
    "compute.networks.create",
    "compute.networks.get",
    "compute.networks.updatePolicy",
    "compute.subnetworks.use",
    "compute.subnetworks.useExternalIp",
    "compute.zoneOperations.get",
    "iam.serviceAccounts.actAs",
    "storage.buckets.create",
    "storage.buckets.delete",
    "storage.buckets.get",
    "storage.multipartUploads.abort",
    "storage.multipartUploads.create",
    "storage.multipartUploads.list",
    "storage.multipartUploads.listParts",
    "storage.objects.create",
    "storage.objects.delete",
    "storage.objects.get",
    "storage.objects.list",
    "storage.objects.update",
  ]
}

output "google_application_credentials_data" {
  value     = base64decode(google_service_account_key.task.private_key)
  sensitive = true
}
