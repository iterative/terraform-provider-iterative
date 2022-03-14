# Permissions

This directory contains reference permissions for TPI. 

### Authenticating for the first time

Follow these guides to learn how to authenticate with your cloud provider:
* [Amazon Web Services — `aws`](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration)
* [Microsoft Azure — `az`](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/azure_cli) 
* [Google Cloud — `gcp`](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started)
* [Kubernetes — `k8s`](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig)

### Creating permissions and credentials

Run `terraform init && terraform apply` to install the required providers and create the resources
User + Role + Policy —  replace `us-west-1` by the adequate region
Application + Service Principal + Role
Google Cloud — Service Account + Role — replace `task-project` below with the name of the project
 Kubernetes — Service Account + Role

### Using the created credentials

#### `aws`
* Set the [`AWS_ACCESS_KEY_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AWS_ACCESS_KEY_ID) environment variable to the value returned by  `terraform output --raw aws_access_key_id`
* Set the [`AWS_SECRET_ACCESS_KEY`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AWS_SECRET_ACCESS_KEY) environment variable to the value returned by  `terraform output --raw aws_secret_access_key`

#### `az`
* Set the [`AZURE_TENANT_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_TENANT_ID) environment variable to the value returned by  `terraform output --raw azure_tenant_id`
* Set the [`AZURE_SUBSCRIPTION_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_SUBSCRIPTION_ID) environment variable to the value returned by  `terraform output --raw azure_subscription_id`
* Set the [`AZURE_CLIENT_ID`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_CLIENT_ID) environment variable to the value returned by  `terraform output --raw azure_client_id`
* Set the [`AZURE_CLIENT_SECRET`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#AZURE_CLIENT_SECRET) environment variable to the value returned by  `terraform output --raw azure_client_secret`

#### `gcp`
* Set the [`GOOGLE_APPLICATION_CREDENTIALS_DATA`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#GOOGLE_APPLICATION_CREDENTIALS) environment variable to the value returned by  `terraform output --raw google_application_credentials_data`

> :bulb: Use the following oneliner to set and export all the variables at once:
> ```
> eval "$(terraform output --json | jq --raw-output 'to_entries[]|"export \(.key|ascii_upcase)=\(.value.value|@sh)"')"
> ```

## Kubernetes

If required, authentication can be configured through a narrowly scoped service account inside an ad-hoc namespace. Applying the following definitions will create a new namespace and an equally named service account, along with the required roles and role bindings:

```yaml


```


>💡 *You can just save the configuration above in a file and apply it by running `kubectl apply --filename <file>` on your computer.*

### Generating a `kubeconfig` file for the created service account

After applying the above configuration, you can generate the required `kubeconfig`data by running the following script:

```bash
SERVER="$(
  kubectl get endpoints --output \
    jsonpath="{.items[0].subsets[0].addresses[0].ip}"
)"

AUTHORITY="$(
  kubectl config view --raw --minify --flatten --output \
    jsonpath='{.clusters[].cluster.certificate-authority-data}'
)"

SECRET="$(
  kubectl --namespace=task get serviceaccount task --output \
    jsonpath="{.secrets[0].name}"
)"

TOKEN="$(
  kubectl get secret "$SECRET" --namespace=task --output \
    jsonpath="{.data.token}" | base64 --decode
)"

(
  export KUBECONFIG="$(mktemp)"
  {
    kubectl config set-cluster cluster --server="https://$SERVER"
    kubectl config set clusters.cluster.certificate-authority-data "$AUTHORITY"
    kubectl config set-credentials task --token="$TOKEN"
    kubectl config set-context cluster --cluster=cluster --namespace=task --user=task
    kubectl config use-context cluster
  } > /dev/null
  cat "$KUBECONFIG" && rm "$_"
)

```

> 📖 *The code above is just an example and might fail for some managed Kubernetes providers; it's known to work on GKE, but your mileage may vary.*

7. Set the [`KUBECONFIG_DATA`](https://registry.terraform.io/providers/iterative/iterative/latest/docs#KUBECONFIG_DATA) environment variable to the value returned by  `terraform output --raw kubeconfig_data`
