---
page_title: Getting Started
---

# Getting Started

## Requirements

- [Install Terraform 1.0+](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform), e.g.:
  + Brew (Homebrew/Mac OS): `brew tap hashicorp/tap && brew install hashicorp/tap/terraform`
  + Choco (Chocolatey/Windows): `choco install terraform`
  + Conda (Anaconda): `conda install -c conda-forge terraform`
  + Debian (Ubuntu/Linux):
    ```
    sudo apt-get update && sudo apt-get install -y gnupg software-properties-common curl
    curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
    sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
    sudo apt-get update && sudo apt-get install terraform
    ```
- Create an account with any supported cloud vendor and expose its [authentication credentials via environment variables][authentication]

[authentication]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication

## Defining a Task

In a project root directory, create a file named `main.tf` with the following contents:

```hcl
terraform {
  required_providers { iterative = { source = "iterative/iterative" } }
}
provider "iterative" {}
resource "iterative_task" "example" {
  cloud   = "aws" # or any of: gcp, az, k8s
  machine = "m"   # medium, or any of: l, xl, m+k80, xl+v100, ...

  storage {
    workdir = "."
    output  = "results"
  }
  script = <<-END
    #!/bin/bash
    mkdir results
    echo "Hello World!" > results/greeting.txt
  END
}
```

See [the reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#argument-reference) for the full list of options for `main.tf` -- including more information on [`machine` types](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#machine-type) with and without GPUs.

-> **Note:** The `script` argument must begin with a valid [shebang](<https://en.wikipedia.org/wiki/Shebang_(Unix)>), and can take the form of a [heredoc string](https://www.terraform.io/docs/language/expressions/strings.html#heredoc-strings) or [a `file()` function](https://www.terraform.io/docs/language/functions/file.html) function (e.g. `file("task_run.sh")`).

The project layout should look similar to this:

```
project/
├── main.tf
└── results/
    └── greeting.txt (created in the cloud and downloaded locally)
```

## Initialise Terraform

```console
$ terraform init
```

This command will check `main.tf` and download the required TPI plugin.

~> **Warning:** None of the subsequent commands will work without first setting some [authentication environment variables][authentication].

## Launch Task

```console
$ terraform apply
```

This command will:

1. Create all the required cloud resources.
2. Upload the specified working directory (`workdir`) to the cloud.
3. Launch the task `script`.

## View Task Status

```console
$ terraform refresh && terraform show
```

This command will:

1. Query the task status from the cloud.
2. Display the task status.

## Stop Task

```console
$ terraform destroy
```

This command will:

1. Download the specified output directory from the cloud.
2. Delete all the cloud resources created by `terraform apply`.

## View Task Results

After running `terraform destroy`, the `results` directory should contain a file named `greeting.txt` with the text `Hello, World!`
