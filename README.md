![TPI](https://static.iterative.ai/img/cml/banner-terraform.png)

# Terraform Provider Iterative

[![](https://img.shields.io/badge/-documentation-5c4ee5?logo=terraform)](https://registry.terraform.io/providers/iterative/iterative/latest/docs)

- **Orchestrate Resources**: create cloud compute & storage resources without reading pages of documentation
- **Sync & Execute**: move data & run code in the cloud with minimal configuration
- **Low cost**: auto-recovery from spot/preemptible instances to vastly reduce cost
- **No waste**: auto-cleanup unused resources
- **No lock-in**: switch between cloud vendors with minimal configuration

Iterative's Provider is a [Terraform](https://terraform.io) plugin built with machine learning pipelines in mind. It enables full lifecycle management of computing resources (including GPUs) from several cloud vendors:

- Amazon Web Services (AWS)
- Microsoft Azure
- Google Cloud Platform (GCP)
- Kubernetes (K8s)

The aim is to easily (minimal configuration unified across cloud vendors) move local experiments to the cloud, use low-cost spot instances without losing progress, and avoid being charged for unused cloud resources (terminate compute instances upon job completion/failure, and remove storage upon download of results).

## Documentation

See the [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started) guide and detailed configuration information in the [Documentation](https://registry.terraform.io/providers/iterative/iterative/latest/docs).

## Support

Have a feature request or found a bug? Let us know via [GitHub issues](https://github.com/iterative/terraform-provider-iterative/issues). Have questions? Join our [community on Discord](https://discord.gg/bzA6uY7); we'll be happy to help you get started!

## Usage

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

### Initialize the provider

Run this command after every `make install` to use the new build:

```console
terraform init --upgrade
```

### Test the provider

```console
terraform apply
```

## Contributing

Instead of using the latest stable release, a local copy of the repository must be used.

### Install Go 1.17+

Refer to the [official documentation](https://golang.org/doc/install) for specific instructions.

### Clone the repository

```console
git clone https://github.com/iterative/terraform-provider-iterative
cd terraform-provider-iterative
```

### Modify test file

Specify `source = "github.com/iterative/iterative"` to use the local repository.

**Note:** `source = "iterative/iterative"` will download the latest release instead.

## License

[Apache 2.0](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE).
