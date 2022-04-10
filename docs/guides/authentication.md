---
page_title: Authentication
---

# Authentication

Environment variables are the only supported authentication method, and should be present when running any `terraform` command. For example:

```bash
export GOOGLE_APPLICATION_CREDENTIALS_DATA="$(cat service_account.json)"
TF_LOG_PROVIDER=INFO terraform apply
```

## Amazon Web Services

[Create an AWS account](https://aws.amazon.com/premiumsupport/knowledge-center/create-and-activate-aws-account/) if needed, and then set these environment variables:

- `AWS_ACCESS_KEY_ID` - Access key identifier.
- `AWS_SECRET_ACCESS_KEY` - Secret access key.
- `AWS_SESSION_TOKEN` - (Optional) Session token.

See the [AWS documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) to obtain these variables directly.

Alternatively, for more idiomatic or advanced use cases, follow the [Terraform AWS provider documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration) and run the following commands in the [`permissions/aws`](https://github.com/iterative/terraform-provider-iterative/tree/master/docs/guides/permissions/aws) directory:

```bash
terraform init && terraform apply
export AWS_ACCESS_KEY_ID="$(terraform output --raw aws_access_key_id)"
export AWS_SECRET_ACCESS_KEY="$(terraform output --raw aws_secret_access_key)"
```

## Microsoft Azure

[Create an Azure account](https://docs.microsoft.com/en-us/learn/modules/create-an-azure-account/) if needed, and then set these environment variables:

- `AZURE_CLIENT_ID` - Client identifier.
- `AZURE_CLIENT_SECRET` - Client secret.
- `AZURE_SUBSCRIPTION_ID` - Subscription identifier.
- `AZURE_TENANT_ID` - Tenant identifier.

See the [Azure documentation](https://docs.microsoft.com/en-us/python/api/azure-identity/azure.identity.environmentcredential) to obtain these variables directly.

Alternatively, for more idiomatic or advanced use cases, follow the [Terraform Azure provider documentation](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/azure_cli) and run the following commands in the [`permissions/az`](https://github.com/iterative/terraform-provider-iterative/tree/master/docs/guides/permissions/az) directory:

```bash
terraform init && terraform apply
export AZURE_TENANT_ID="$(terraform output --raw azure_tenant_id)"
export AZURE_SUBSCRIPTION_ID="$(terraform output --raw azure_subscription_id)"
export AZURE_CLIENT_ID="$(terraform output --raw azure_client_id)"
export AZURE_CLIENT_SECRET="$(terraform output --raw azure_client_secret)"
```

## Google Cloud Platform

[Create a GCP account](https://cloud.google.com/free) if needed, and then set the environment variable:

- `GOOGLE_APPLICATION_CREDENTIALS` - Path to (or contents of) a service account JSON key file.

See the [GCP documentation](https://cloud.google.com/docs/authentication/getting-started#creating_a_service_account) to obtain these variables directly.

Alternatively, for more idiomatic or advanced use cases, follow the [Terraform GCP provider documentation](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started) and run the following commands in the [`permissions/gcp`](https://github.com/iterative/terraform-provider-iterative/tree/master/docs/guides/permissions/gcp) directory:

```bash
terraform init && terraform apply
export GOOGLE_APPLICATION_CREDENTIALS_DATA="$(terraform output --raw google_application_credentials_data)"
```

## Kubernetes

Either one of:

- `KUBECONFIG` - Path to a [`kubeconfig` file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable).
- `KUBECONFIG_DATA` - Alternatively, the **contents** of a `kubeconfig` file.

Alternatively, authenticate with a local `kubeconfig` file and run the following commands in the [`permissions/k8s`](https://github.com/iterative/terraform-provider-iterative/tree/master/docs/guides/permissions/k8s) directory:

```bash
kubectl apply --filename main.yml
export KUBECONFIG_DATA="$(bash kubeconfig.sh)"
```
