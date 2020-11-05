
terraform {
  required_providers {
    iterative = {
      #source = "DavidGOrtega/iterative"
      #version = "0.4.0"
      versions = ["0.3"]
      source = "github.com/davidgortega/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west"
  instance_type = "t2.micro"
  instance_gpu = "tesla"
}