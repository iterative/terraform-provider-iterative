packer {
  required_plugins {
    amazon = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variables {
  image_name        = "iterative-cml"
  image_description = "CML (Continuous Machine Learning) Ubuntu 18.04"
}

build {
  sources = ["source.amazon-ebs.source"]
  
  provisioner "shell" {
    script = "${path.root}/../provisioner/setup.sh"
  }
}
