![Terraform Provider Iterative](https://static.iterative.ai/img/cml/banner-terraform.png)

# Iterative Provider [![](https://img.shields.io/badge/-documentation-5c4ee5?logo=terraform)](https://registry.terraform.io/providers/iterative/iterative/latest/docs)

The Iterative Provider is a Terraform plugin that enables full lifecycle
management of computing resources for machine learning pipelines, including GPUs, from your favorite cloud vendors.

The Iterative Provider makes it easy to:

- Rapidly move local machine learning experiments to a cloud infrastructure
- Take advantage of training models on spot instances without losing any progress
- Unify configuration of various cloud compute providers
- Automatically destroy unused cloud resources (never forget to turn your GPU off again)

The Iterative Provider can provision resources with the following cloud providers and orchestrators:

- Amazon Web Services
- Microsoft Azure
- Google Cloud Platform
- Kubernetes

## Documentation

See the [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started) guide to learn how to use the Iterative Provider. More details on configuring and using the Iterative Provider are in the [documentation](https://registry.terraform.io/providers/iterative/iterative/latest/docs).

## Support

Have a feature request or found a bug? Let us know via [GitHub issues](https://github.com/iterative/terraform-provider-iterative/issues). Have questions? Join our [community on Discord](https://discord.gg/CDEsr8t9Nj); we'll be happy to help you get started!

## License

Iterative Provider is released under the [Apache 2.0 License](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE).

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
  required_providers { iterative = { source = "iterative/iterative" } }
}
provider "iterative" {}
# ... other resource blocks ...
```

**Note:** to use your local build, specify `source = "github.com/iterative/iterative"` (`source = "iterative/iterative"` will download the latest stable release instead).

### Initialize the provider

Run this command after every `make install` to use the new build:

```console
terraform init --upgrade
```

### Test the provider

```console
terraform apply
```
