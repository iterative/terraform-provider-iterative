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
  enable_aws   = false
  enable_azure = false
  enable_gcp   = false
}

variables {
  image_name        = "iterative-cml"
  image_description = "CML (Continuous Machine Learning) Ubuntu 18.04"
}

build {
  sources = concat([
    var.enable_aws ? ["source.amazon-ebs.source"] : [],
    var.enable_azure ? ["source.azure-arm.source"] : [],
    var.enable_gcp ? ["sources.googlecompute.source"] : []
  ])

  provisioner "shell" {
    script = "${dirname(path.module)}/../provisioner/setup.sh"
  }
}
