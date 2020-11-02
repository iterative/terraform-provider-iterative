
terraform {
  required_providers {
    iterative = {
      source  = "DavidGOrtega/iterative"
      version = "0.4.0"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-east-1"



}