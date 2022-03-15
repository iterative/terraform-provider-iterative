---
page_title: Authentication
---

# Authentication

Environment variables are the only supported authentication method. They should be present when running any of the `terraform` commands. For example:

```bash
$ export GOOGLE_APPLICATION_CREDENTIALS_DATA="$(cat service_account.json)"
$ terraform apply
```

## Amazon Web Services

- `AWS_ACCESS_KEY_ID` - Access key identifier.
- `AWS_SECRET_ACCESS_KEY` - Secret access key.
- `AWS_SESSION_TOKEN` - (Optional) Session token.

See the [AWS documentation](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html) for more information.

## Microsoft Azure

- `AZURE_CLIENT_ID` - Client identifier.
- `AZURE_CLIENT_SECRET` - Client secret.
- `AZURE_SUBSCRIPTION_ID` - Subscription identifier.
- `AZURE_TENANT_ID` - Tenant identifier.

See the [Azure documentation](https://docs.microsoft.com/en-us/python/api/azure-identity/azure.identity.environmentcredential) for more information.

## Google Cloud Platform

- `GOOGLE_APPLICATION_CREDENTIALS` - Path to (or contents of) a service account JSON key file.

See the [GCP documentation](https://cloud.google.com/docs/authentication/getting-started#creating_a_service_account) for more information.

## Kubernetes

Either one of:

- `KUBECONFIG` - Path to a [`kubeconfig` file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable).
- `KUBECONFIG_DATA` - Alternatively, the **contents** of a `kubeconfig` file.
