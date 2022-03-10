![TPI](https://static.iterative.ai/img/cml/banner-terraform.png)

# Terraform Provider Iterative (TPI)

[![docs](https://img.shields.io/badge/-docs-5c4ee5?logo=terraform)](https://registry.terraform.io/providers/iterative/iterative/latest/docs)
[![tests](https://img.shields.io/github/workflow/status/iterative/terraform-provider-iterative/Test?label=tests&logo=GitHub)](https://github.com/iterative/terraform-provider-iterative/actions/workflows/test.yml)
[![Apache-2.0](https://img.shields.io/badge/licence-Apache%202.0-blue)](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE)

- **Provision Resources**: create cloud compute & storage resources without reading pages of documentation
- **Sync & Execute**: easily sync & run local data & code in the cloud
- **Low cost**: transparent auto-recovery from interrupted low-cost spot/preemptible instances
- **No waste**: auto-cleanup unused resources (terminate compute instances upon job completion/failure & remove storage upon download of results)
- **No lock-in**: switch between several cloud vendors with ease due to concise unified configuration

Supported cloud vendors include:

- Amazon Web Services (AWS)
- Microsoft Azure
- Google Cloud Platform (GCP)
- Kubernetes (K8s)

## Usage

See the [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started) guide for a more detailed guide.

### Requirements

- [Install Terraform 1.0 or greater](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform)
- Create an account with any supported cloud vendor and expose its [authentication credentials via environment variables](https://registry.terraform.io/providers/iterative/iterative/latest/docs#authentication)

### Create a test file

Create a file named `main.tf` in an empty directory with the following contents:

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

See the [Reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task) for the full list of options for `main.tf`.

### Initialize the provider

Run this once to pull the latest TPI plugin:

```console
terraform init --upgrade
```

### Test the provider

```console
terraform apply
```

## Help

Have a feature request or found a bug? Let us know via [GitHub issues](https://github.com/iterative/terraform-provider-iterative/issues). Have questions? Join our [community on Discord](https://discord.gg/bzA6uY7); we'll be happy to help you get started!

## Contributing

Instead of using the latest stable release, a local copy of the repository must be used.

1. [Install Go 1.17+](https://golang.org/doc/install)
2. Clone the repository
   ```console
   git clone https://github.com/iterative/terraform-provider-iterative
   cd terraform-provider-iterative
   ```
3. Use `source = "github.com/iterative/iterative"` in your `main.tf` to use the local repository (`source = "iterative/iterative"` will download the latest release instead)
