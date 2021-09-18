packer {
  required_plugins {
    amazon = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/amazon"
    }
    azure = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/azure"
    }
    googlecompute = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/googlecompute"
    }
  }
}

variable "test" {
  type    = bool
  default = false
}

variables {
  image_name        = "iterative-cml"
  image_description = "CML (Continuous Machine Learning) Ubuntu 18.04"
}

locals {
  gcp_publish_script = <<-END
    cat manifest.json | jq --raw-output '.builds | .[].artifact_id' | while read image; do
      gcloud compute images add-iam-policy-binding "$image" \
        --member='allAuthenticatedUsers' --role='roles/compute.imageUser'
    done
  END
  gcp_delete_old_images_script = <<-END
    cat manifest.json | jq --raw-output '.builds | sort_by(.build_time) | .[:-1] | .[].artifact_id' | while read image; do
      gcloud compute images delete "$image"
    done
  END
}

build {
  sources = [
    #"source.amazon-ebs.source",
    #"source.azure-arm.source",
    "sources.googlecompute.source"
  ]

  # provisioner "shell" {
  #  script = "${path.root}/../provisioner/setup.sh"
  # }

  provisioner "ansible" {
    playbook_file = "${path.root}/../ansible/playbook.yml"
    galaxy_file   = "${path.root}/../ansible/requirements.yml"
  }

  post-processor "manifest" {
    output = "manifest.json"
    strip_path = true
  }

  post-processor "shell-local" {
    inline = [
      local.gcp_publish_script,
      local.gcp_delete_old_images_script
    ]
  }
}
