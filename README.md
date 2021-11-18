![Terraform Provider Iterative](https://static.iterative.ai/img/cml/banner-terraform.png)

The Iterative Provider is a Terraform plugin that enables full lifecycle
management of computing resources for machine learning pipelines, including GPUs, from your favorite cloud vendors.

The Iterative Provider makes it easy to:

- Rapidly move local machine learning experiments to a cloud infrastructure
- Take advantage of training models on spot instances without losing any progress
- Configure provisioning of compute resources from any of the supported vendors in a unified manner
- Automatically unregister and remove cloud resources (never forget to turn your GPU off again)

## Prerequisites

To use Iterative Provider you will need to:

- [Install](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform) Terraform 1.0+.
- Make sure you have an account with a cloud vendor of choice, and have the respective authentication credentials set as environment variables. Check out cloud-specific authentication method details in the [docs](<(https://registry.terraform.io/providers/iterative/iterative/latest)>)

Iterative Provider can provision resources with the following cloud providers and orchestrators:

- Amazon Web Services
- Google Cloud Platform
- Microsoft Azure
- Kubernetes

## Example usage

```
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_task" "example" {
  name  = "example"
  cloud = "aws"

  script = <<-END
    #!/bin/bash
    echo "hello!"
  END
}
```

## Documentation

You can find detailed documentation on how to configure and use the Iterative Provider [here](https://registry.terraform.io/providers/iterative/iterative/latest).

## Support

Have you found any issues or missing features? Let us know via [GitHub issues](https://github.com/iterative/terraform-provider-iterative/issues). Have questions? Join our [community on Discord](https://discord.com/invite/dvwXA2N), we'll be happy to help you get started!

## License

Iterative Provider is licensed under the [Apache 2.0 License](LICENSE).

## Development

### Install Go 1.17+

Refer to the [official documentation](https://golang.org/doc/install) for specific instructions.

### Clone the repository

```console
git clone https://github.com/iterative/terraform-provider-iterative
cd terraform-provider-iterative
```

### Install the provider

Build the provider and install the resulting binary to the [local mirror directory](https://www.terraform.io/docs/cli/config/config-file.html#implied-local-mirror-directories):

```console
make install
```

### Create a test file

Create a file named `main.tf` in an empty directory with the following contents:

```hcl
terraform {
  required_providers {
    iterative = {
      source = "github.com/iterative/iterative"
    }
  }
}

provider "iterative" {}

# ... paste resource blocks from
# ... the documentation examples
```

**Note:** specify `source = "github.com/iterative/iterative"` instead of `source = "iterative/iterative"` in order to use your local build, as opposed to downloading the latest stable release.

### Initialize the provider

Run this command after every `make install` to use the new build:

```console
terraform init --upgrade
```

### Test the provider

```console
terraform apply
```
