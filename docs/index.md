# Iterative Provider

![TPI](https://static.iterative.ai/img/cml/banner-terraform.png)

[![tests](https://img.shields.io/github/workflow/status/iterative/terraform-provider-iterative/Test?label=tests&logo=GitHub)](https://github.com/iterative/terraform-provider-iterative/actions/workflows/test.yml)
[![Apache-2.0](https://img.shields.io/badge/licence-Apache%202.0-blue)](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE)

- **Orchestrate Resources**: create cloud compute & storage resources without reading pages of documentation
- **Sync & Execute**: move data & run code in the cloud with minimal configuration
- **Low cost**: auto-recovery from spot/preemptible instances to vastly reduce cost
- **No waste**: auto-cleanup unused resources
- **No lock-in**: switch between cloud vendors with ease

Iterative's Provider is a [Terraform](https://terraform.io) plugin built with machine learning pipelines in mind. It enables full lifecycle management of computing resources (including GPUs) from several cloud vendors:

- Amazon Web Services (AWS)
- Microsoft Azure
- Google Cloud Platform (GCP)
- Kubernetes (K8s)

With a minimal configuration unified across cloud vendors, the aim is to easily move local experiments to the cloud, transparently resume from interrupted low-cost spot instances, and avoid being charged for unused cloud resources (terminate compute instances upon job completion/failure, and remove storage upon download of results).

## Example Usage

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

-> **Note:** See [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started) for more information.

## Authentication

Environment variables are the only supported authentication method. They should be present when running any of the `terraform` commands.

### Example

```bash
$ export GOOGLE_APPLICATION_CREDENTIALS_DATA="$(cat service_account.json)"
$ terraform apply
```

### Amazon Web Services

- `AWS_ACCESS_KEY_ID` - Access key identifier.
- `AWS_SECRET_ACCESS_KEY` - Secret access key.
- `AWS_SESSION_TOKEN` - (Optional) Session token.

See the [AWS documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) for more information.

### Microsoft Azure

- `AZURE_CLIENT_ID` - Client identifier.
- `AZURE_CLIENT_SECRET` - Client secret.
- `AZURE_SUBSCRIPTION_ID` - Subscription identifier.
- `AZURE_TENANT_ID` - Tenant identifier.

See the [Azure documentation](https://docs.microsoft.com/en-us/python/api/azure-identity/azure.identity.environmentcredential) for more information.

### Google Cloud Platform

- `GOOGLE_APPLICATION_CREDENTIALS` - Path to a service account JSON key file.

-> **Note:** you can also use `GOOGLE_APPLICATION_CREDENTIALS_DATA` with the **contents** of the service account JSON key file.

See the [GCP documentation](https://cloud.google.com/docs/authentication/getting-started#creating_a_service_account) for more information.

### Kubernetes

- `KUBECONFIG` - Path to a [`kubeconfig` file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable).

-> **Note:** You can use `KUBECONFIG_DATA` instead, with the **contents** of the `kubeconfig` file.
