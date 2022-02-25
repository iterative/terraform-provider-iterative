# Iterative Provider

Use the Iterative Provider to launch resource-intensive tasks in popular cloud providers with a single Terraform file.

## Example Usage

```hcl
terraform {
  required_providers { iterative = { source = "iterative/iterative" } }
}
provider "iterative" {}
resource "iterative_task" "task" {
  cloud = "aws"

  script = <<-END
    #!/bin/bash
    echo "hello!"
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
