terraform {
  required_providers {
    iterative = {
      versions = ["0.6"]
      source = "github.com/iterative/iterative"
    }
  }
}

provider "iterative" {}

/* resource "iterative_machine" "machine-aws" {
  driver = "aws"
  region = "us-west"
  instance_type = "t2.micro" //fallback to known instance type
} */

resource "iterative_machine" "machine-azure" {
  driver = "azure"
  region = "us-west"
  instance_type = "m"

  provisioner "remote-exec" {
    inline = [
      "ls",
    ]

    connection {
      user     = "ubuntu"
      private_key = "${self.key_private}"
      host     = "${self.instance_ip}"
    }
  }
}