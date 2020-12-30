terraform {
  required_providers {
    iterative = {
      versions = ["0.6"]
      source = "github.com/iterative/iterative"
    }

    # iterative = {
    #   source = "iterative/iterative"
    #  // version = "0.5.7"
    # }
  }
}

provider "iterative" {}


resource "iterative_cml_runner" "runner-aws" {
    name = "holy-moly57"
    repo = "https://github.com/DavidGOrtega/3_tensorboard"
    driver = "github"
    labels = "tf"

    cloud = "aws"
    region = "us-west"
    instance_type = "t2.micro"
} 


resource "iterative_cml_runner" "runner-az" {
    name = "holy-moly56"
    repo = "https://github.com/DavidGOrtega/3_tensorboard"
    driver = "github"
    labels = "tf"

    cloud = "azure"
    region = "us-west"
    instance_type = "m"
}


/*
resource "iterative_machine" "machine-az" {
  cloud = "azure"
  region = "us-east"
  instance_type = "m"

  provisioner "remote-exec" {
    scripts = [
      "provision.sh",
    ]

    connection {
      user        = "ubuntu"
      private_key = "${self.ssh_private}"
      host        = "${self.instance_ip}"
    }
  }
}
*/


/* resource "iterative_machine" "machine-azure-gpu" {
  cloud = "azure"
  region = "us-east"
  instance_type = "m"
  instance_gpu = "k80"

  provisioner "remote-exec" {
    inline = [
      "echo 'hello azure'",
      "nvidia-smi"
    ]

    connection {
      user        = "ubuntu"
      private_key = "${self.key_private}"
      host        = "${self.instance_ip}"
    }
  }
} */
/* 
resource "iterative_machine" "machine-aws" {
  name = "cml-aws4"
  cloud = "aws"
  region = "us-west"
  instance_type = "t2.micro" //fallback to known instance type
}  */

/* resource "iterative_machine" "machine-az" {
  name = "cml-azure"
  cloud = "azure"
  region = "us-west"
  instance_type = "m"
} */