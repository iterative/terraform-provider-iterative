terraform {
  required_providers {
    iterative = {
      versions = ["0.6"]
      source = "github.com/iterative/iterative"
    }
  }
}

provider "iterative" {}

/*
resource "iterative_machine" "machine-az" {
  cloud = "azure"
  region = "us-east"
  instance_type = "m"
}
*/

resource "iterative_runner" "runner-az" {
    token = "arszDpb3xtNdKaXmQ6vN"
    repo = "https://gitlab.com/DavidGOrtega/3_tensorboard"
    driver = "gitlab"
    labels = "tf"
    machine {
      cloud = "azure"
      region = "us-west"
      instance_type = "m"
    }
} 

/*
 resource "iterative_machine" "machine-aws" {
    cloud = "aws"
    region = "us-west"
    instance_type = "t2.micro" //fallback to known instance type
} 
*/


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