variables {
  gcp_build_region            = "us-west1-b"
  gcp_build_instance          = "custom-8-16384"
  gcp_build_accelerator_type  = "nvidia-tesla-k80"
  gcp_build_accelerator_count = 1
  gcp_build_ubuntu_image      = "ubuntu-1804-lts"
}

variables {
  gcp_project = "cml-gcp-test"
}

locals {
  tags = {
    environment = var.test ? "test" : "production"
  }
  release_regions = ["us"]
}

source "googlecompute" "source" {
  image_family            = var.test ? var.image_name : "${var.image_name}-test"
  image_name              = "${var.image_name}-{{timestamp}}"
  image_description       = var.image_description
  image_storage_locations = local.release_regions
  image_labels            = local.tags

  zone         = var.gcp_build_region
  machine_type = var.gcp_build_instance

  accelerator_count = var.gcp_build_accelerator_count
  accelerator_type = join("/", [
    "projects",
    var.gcp_project,
    "zones",
    var.gcp_build_region,
    "acceleratorTypes",
    var.gcp_build_accelerator_type,
  ])

  on_host_maintenance = "TERMINATE"

  source_image_family = "${var.gcp_build_ubuntu_image}"
  ssh_username        = "ubuntu"

  project_id = var.gcp_project
}

# https://cloud.google.com/compute/docs/images/managing-access-custom-images#share-images-publicly