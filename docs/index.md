# Iterative Provider

Use the Iterative provider to launch resource–intensive tasks in popular cloud
providers with just a simple Terraform file.

## Example Usage

```hcl
terraform {
  required_providers {
    iterative = {
      source  = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_task" "task" {
  name  = "example"
  cloud = "aws"

  script = <<-END
    #!/bin/bash
    echo "hello!"
  END
}
```

## Authentication

Environment variables are the only supported authentication method. They should
be present when running any of the `terraform` commands.

### Example

```bash
$ export GOOGLE_APPLICATION_CREDENTIALS_DATA="$(cat service_account.json)"
$ terraform apply
```

### Environment Variables

#### Amazon Web Services

* `AWS_ACCESS_KEY_ID` - Access key identifier.
* `AWS_SECRET_ACCESS_KEY` - Secret access key.
* `AWS_SESSION_TOKEN` - (Optional) Session token.

#### Google Cloud Platform

* `GOOGLE_APPLICATION_CREDENTIALS` - Path to a service account JSON key file.

-> **Note** you can also use `GOOGLE_APPLICATION_CREDENTIALS_DATA` with the
**contents** of the service account JSON key file.

#### Microsoft Azure

* `AZURE_CLIENT_ID` - Client identifier.
* `AZURE_CLIENT_SECRET` - Client secret.
* `AZURE_SUBSCRIPTION_ID` - Subscription identifier.
* `AZURE_TENANT_ID` - Tenant identifier.

## Argument Reference

This module doesn't have any top-level arguments. See the [task resource](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task) for more information.
