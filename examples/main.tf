terraform {
  required_providers {
    iterative = {
      versions = ["0.6"]
      source = "github.com/iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west"
  instance_type = "m" //fallback to known instance type
}