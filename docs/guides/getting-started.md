---
page_title: "Getting Started"
---

# Getting Started

## Installing Terraform

See [this guide](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform) for more information.

## Declaring a Task

On an empty directory:

1. Create a directory named `directory` to save the task results.
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

  directory = "${path.root}/directory"

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

1. Download and install this provider.
2. Initialize Terraform on the current directory.

~> **Note** None of the subsequent commands will work without setting first some [authentication environment variables](https://registry.terraform.io/providers/iterative/iterative/latest/docs#authentication)

## Launching Tasks

```console
$ terraform apply
···
```

This command will:

1. Create all the required cloud resources.
2. Upload the specified `directory` to the cloud.
3. Launch the task.

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

1. Download the specified `directory` from the cloud.
2. Delete all the created cloud resources.

## Viewing Task Results

The specified `directory` should contain a file named `greeting.txt` with the text `Hello, World!` inside.
