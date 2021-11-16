---
page_title: Getting Started
---

# Getting Started

## Installing Terraform

See [this guide](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform) for more information.

## Defining a Task

In an empty directory:

1. Create a directory named `shared` to store input data and output artifacts.
2. Create a file named `main.tf` with the following contents:

```hcl
terraform {
  required_providers {
    iterative = {
      source  = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_task" "example" {
  name  = "example"
  cloud = "aws" # or any of: gcp, az, k8s

  directory = "${path.root}/shared"

  script = <<-END
    #!/bin/bash
    echo "Hello World!" > greeting.txt
  END
}
```

## Initializing Terraform

```console
$ terraform init
```

This command will:

1. Download and install the Iterative Provider.
2. Initialize Terraform in the current directory.

~> **Note:** None of the subsequent commands will work without first setting some [authentication environment variables](https://registry.terraform.io/providers/iterative/iterative/latest/docs#authentication).

## Launching Tasks

```console
$ terraform apply
···
```

This command will:

1. Create all the required cloud resources.
2. Upload the specified shared `directory` to the cloud.
3. Launch the task `script`.

## Viewing Task Statuses

```console
$ terraform refresh && terraform show
resource "iterative_task" "example" {
  ···
}
```

This command will:

1. Query the task status from the cloud.
2. Display the task status.

## Deleting Tasks

```console
terraform destroy
```

This command will:

1. Download the specified shared `directory` from the cloud.
2. Delete all the created cloud resources.

## Viewing Task Results

After running `terraform destroy`, the `shared/` directory should contain a file named `greeting.txt` with the text `Hello, World!`
