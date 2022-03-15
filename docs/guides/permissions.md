---
page_title: Permissions
subcategory: Development
---

# Permissions

The [docs/permissions](https://github.com/iterative/terraform-provider-iterative/tree/master/docs/permissions) directory contains sample roles and permissions to use TPI in all the supported cloud providers.

## Authenticating for the first time

Follow these guides to learn how to authenticate with your cloud provider:

- [`aws` (Amazon Web Services)](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration)
- [`az` (Microsoft Azure)](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/azure_cli)
- [`gcp` (Google Cloud)](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started)
- [`k8s` (Kubernetes)](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig)

## Setting up permissions, credentials and environment variables

### `aws`

- Run `terraform init` and `terraform apply` in the `aws` directory
- Set the [`AWS_ACCESS_KEY_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AWS_ACCESS_KEY_ID) environment variable to the value returned by `terraform output --raw aws_access_key_id`
- Set the [`AWS_SECRET_ACCESS_KEY`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AWS_SECRET_ACCESS_KEY) environment variable to the value returned by `terraform output --raw aws_secret_access_key`

### `az`

- Run `terraform init` and `terraform apply` in the `az` directory
- Set the [`AZURE_TENANT_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_TENANT_ID) environment variable to the value returned by `terraform output --raw azure_tenant_id`
- Set the [`AZURE_SUBSCRIPTION_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_SUBSCRIPTION_ID) environment variable to the value returned by `terraform output --raw azure_subscription_id`
- Set the [`AZURE_CLIENT_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_CLIENT_ID) environment variable to the value returned by `terraform output --raw azure_client_id`
- Set the [`AZURE_CLIENT_SECRET`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_CLIENT_SECRET) environment variable to the value returned by `terraform output --raw azure_client_secret`

### `gcp`

- Run `terraform init` and `terraform apply` in the `gcp` directory
- Set the [`GOOGLE_APPLICATION_CREDENTIALS_DATA`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#GOOGLE_APPLICATION_CREDENTIALS) environment variable to the value returned by `terraform output --raw google_application_credentials_data`

### `k8s`

- Run `kubectl apply --filename main.yml` in the `k8s` directory
- Set the [`KUBECONFIG_DATA`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#KUBECONFIG_DATA) environment variable to the value returned by the `kubeconfig.sh` script
