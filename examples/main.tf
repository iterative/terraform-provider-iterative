terraform {
  required_providers {
    iterative = {
      versions = ["0.6"]
      source = "github.com/iterative/iterative"
    }

    # iterative = {
    #   source = "iterative/iterative"
    #   version = "0.5.7"
    # }
  }
}

provider "iterative" {}


resource "iterative_machine" "machine" {
    cloud = "aws"
    region = "us-west"
    instance_type = "t2.micro"
    spot = true
    #spot_price = 0.09
} 

resource "iterative_cml_runner" "runner-gh-aws" {
    token = "c38c69db3cdb81d2878fcc9c0556ed972249f930"
    repo = "https://github.com/DavidGOrtega/cml"
    driver = "github"
    labels = "tf"

    idle_timeout = 0
    cloud = "aws"
    region = "us-west"
    instance_type = "t2.micro"
    spot = true
    #spot_price = 0.09
} 

# resource "iterative_cml_runner" "runner-gh-aws" {
#     name = "githubrunner2"
#     token = "a0b56d03294f0908f4ee7290be2ea53051d227d3"
#     repo = "https://github.com/DavidGOrtega/3_tensorboard"
#     driver = "github"
#     labels = "tf"

#     cloud = "aws"
#     region = "us-east-2"
#     instance_type = "t2.micro"
# } 
