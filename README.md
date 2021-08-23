![Terraform Provider Iterative](https://static.iterative.ai/img/cml/banner-terraform.png)

# Terraform Iterative provider

The Iterative Provider is a Terraform plugin that enables full lifecycle
management of cloud computing resources, including GPUs, from your favorite
[vendors](#supported-vendors). Two types of resources are available:

- Runner (`iterative_cml_runner`)
- Machine (`iterative_machine`)

The Provider is designed for benefits like:

- Unified logging for workflows run in cloud resources
- Automatic provision of cloud resources
- Automatic unregister and removal of cloud resources (never forget to turn your
  GPU off again)
- Arguments inherited from the GitHub/GitLab runner for ease of integration
  (`name`,`labels`,`idle-timeout`,`repo`,`token`, and `driver`)

## Usage

### Runner

A self hosted runner based on a thin wrapper over the GitLab and GitHub
self-hosted [runners](https://github.com/actions/runner), abstracting their
functionality to a common specification that allows adjusting the main runner
settings, like idle timeouts, or custom runner labels.

The runner resource also provides features like unified logging and automated
cloud resource provisioning and management through various vendors.

#### Configuring the vendor credentials

This provider requires a repository token for registering and unregistering
self-hosted runners during the cloud resource lifecycle. Depending on the
platform you use, the instructions to get that token may vary; please refer to
your platform documentation:

- [GitHub](https://docs.github.com/es/github-ae@latest/github/authenticating-to-github/creating-a-personal-access-token)
- [GitLab](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)

This token can be passed to the provider through the `CML_TOKEN` or environment
variable, like in the following example:

```sh
export CML_TOKEN=···
```

Additionally, you need to provide credentials for the cloud provider where the
computing resources should be allocated. Follow the steps below to get started.

#### Basic usage

1. - Setup your provider credentials as ENV variables

<details>
<summary>AWS</summary>
<p>

```sh
export AWS_SECRET_ACCESS_KEY=YOUR_KEY
export AWS_ACCESS_KEY_ID=YOUR_ID
export CML_TOKEN=YOUR_REPO_TOKEN
```

</p>
</details>

<details>
<summary>Azure</summary>
<p>

```sh
export AZURE_CLIENT_ID=YOUR_ID
export AZURE_CLIENT_SECRET=YOUR_SECRET
export AZURE_SUBSCRIPTION_ID=YOUR_SUBSCRIPTION_ID
export AZURE_TENANT_ID=YOUR_TENANT_ID
export CML_TOKEN=YOUR_REPO_TOKEN
```

</p>
</details>

2. Save your terraform file `main.tf`.

<details>
<summary>AWS</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
    repo = "https://github.com/iterative/cml"
    driver = "github"
    labels = "tf"

    cloud = "aws"
    region = "us-west"
    instance_type = "m"
    # Uncomment it if GPU is needed:
    # instance_gpu = "v100"
}
```

</p>
</details>

<details>
<summary>Azure</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
   repo = "https://github.com/iterative/cml"
    driver = "github"
    labels = "tf"

    cloud = "azure"
    region = "us-west"
    instance_type = "m"
    # Uncomment it if GPU is needed:
    # instance_gpu = "v100"
}
```

</p>
</details>

💡 _Alternatively, you can use the [JSON Terraform Configuration Syntax](https://www.terraform.io/docs/language/syntax/json.html) instead of the default [HCL](https://www.terraform.io/docs/language/syntax/configuration.html) syntax._

3. Launch it!

```sh
terraform init
terraform apply --auto-approve
```

#### Argument reference

| Variable            | Values                                   | Default                                                                   |                                                                                                                                                                                                                                                                                                  |
| ------------------- | ---------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `driver`            | `gitlab` `github`                        |                                                                           | The kind of runner that you are setting                                                                                                                                                                                                                                                          |
| `repo`              |                                          |                                                                           | The Git repository to subscribe to.                                                                                                                                                                                                                                                              |
| `token`             |                                          |                                                                           | A personal access token. In GitHub, your token must have Workflow and Repository permissions. If not specified, the Iterative Provider looks for the environmental variable CML_REPO                                                                                                             |
| `labels`            |                                          | `cml`                                                                     | Your runner will listen for workflows tagged with this label. Ideal for assigning workflows to select runners.                                                                                                                                                                                   |
| `idle-timeout`      |                                          | 5min                                                                      | The maximum time for the runner to wait for jobs. After timeout, the runner will unregister automatically from the repository and clean up all cloud resources. If set to `0`, the runner will never time out (be warned if you've got a cloud GPU).                                             |
| `cloud`             | `aws` `azure`                            |                                                                           | Sets cloud vendor.                                                                                                                                                                                                                                                                               |
| `region`            | `us-west` `us-east` `eu-west` `eu-north` | `us-west`                                                                 | Sets the collocation region. AWS or Azure regions are also accepted.                                                                                                                                                                                                                             |
| `image`             |                                          | `iterative-cml` in AWS `Canonical:UbuntuServer:18.04-LTS:latest` in Azure | Sets the image to be used. On AWS, the provider searches the cloud provider by image name (not by id), taking the lastest version if multiple images with the same name are found. Defaults to [iterative-cml image](#iterative-cml-image). On Azure uses the form `Publisher:Offer:SKU:Version` |
| `spot`              | boolean                                  | false                                                                     | If true, launch a spot instance                                                                                                                                                                                                                                                                  |
| `spot_price`        | float with 5 decimals at most            | -1                                                                        | Sets the maximum price that you are willing to pay by the hour. If not specified, the current spot bidding pricing will be used                                                                                                                                                                  |
| `name`              |                                          | iterative\_{UID}                                                          | Sets the instance name and related resources based on that name. In Azure, groups everything under a resource group with that name.                                                                                                                                                              |
| `instance_hdd_size` |                                          | 10                                                                        | Sets the instance hard disk size in GB                                                                                                                                                                                                                                                           |
| `instance_type`     | `m`, `l`, `xl`                           | `m`                                                                       | Sets the instance CPU size. You can also specify vendor specific machines in AWS i.e. `t2.micro`. [See equivalences](#Supported-vendors) table below.                                                                                                                                            |
| `instance_gpu`      | ``, `testla`, `k80`                      | ``                                                                        | Selects the desired GPU for supported `instance_types`.                                                                                                                                                                                                                                          |
| `ssh_private`       |                                          |                                                                           | An SSH private key in PEM format. If not provided, one private and public key wll be automatically generated and returned in `terraform.tfstate`                                                                                                                                                 |

### Machine

Setup instructions:

1. Setup your provider credentials as ENV variables

<details>
<summary>AWS</summary>
<p>

```sh
export AWS_SECRET_ACCESS_KEY=YOUR_KEY
export AWS_ACCESS_KEY_ID=YOUR_ID
```

</p>
</details>

<details>
<summary>Azure</summary>
<p>

```sh
export AZURE_CLIENT_ID=YOUR_ID
export AZURE_CLIENT_SECRET=YOUR_SECRET
export AZURE_SUBSCRIPTION_ID=YOUR_SUBSCRIPTION_ID
export AZURE_TENANT_ID=YOUR_TENANT_ID
```

</p>
</details>

2. Save your terraform file `main.tf`

<details>
<summary>AWS</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  cloud = "aws"
  region = "us-west"
  name = "machine"
  instance_hdd_size = "10"
  instance_type = "m"
  # Uncomment it if GPU is needed:
  # instance_gpu = "v100"
}
```

</p>
</details>

<details>
<summary>Azure</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  cloud = "azure"
  region = "us-west"
  name = "machine"
  instance_hdd_size = "10"
  instance_type = "m"
  ## Uncomment it if GPU is needed:
  # instance_gpu = "v100"
}
```

</p>
</details>

3. Launch your instance

```sh
terraform init
terraform apply --auto-approve
```

4. Stop the instance

Run to destroy your instance:

```sh
terraform destroy --auto-approve
```

#### Argument reference

| Variable            | Values                                   | Default                                                                   |                                                                                                                                                                                                                                                                                               |
| ------------------- | ---------------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cloud`             | `aws` `azure`                            |                                                                           | Sets cloud vendor.                                                                                                                                                                                                                                                                            |
| `region`            | `us-west` `us-east` `eu-west` `eu-north` | `us-west`                                                                 | Sets the collocation region. AWS or Azure regions are also accepted.                                                                                                                                                                                                                          |
| `image`             |                                          | `iterative-cml` in AWS `Canonical:UbuntuServer:18.04-LTS:latest` in Azure | Sets the image to be used. On AWS the provider does a search in the cloud provider by image name not by id, taking the lastest version in case there are many with the same name. Defaults to [iterative-cml image](#iterative-cml-image). On Azure uses the form Publisher:Offer:SKU:Version |
| `name`              |                                          | iterative\_{UID}                                                          | Sets the instance name and related resources based on that name. In Azure, groups everything under a resource group with that name.                                                                                                                                                           |
| `spot`              | boolean                                  | false                                                                     | If true launch a spot instance                                                                                                                                                                                                                                                                |
| `spot_price`        | float with 5 decimals at most            | -1                                                                        | Sets the max price that you are willing to pay by the hour. If not specified, the current spot bidding price will be used.                                                                                                                                                                    |
| `instance_hdd_size` |                                          | 10                                                                        | Sets the instance hard disk size in GB                                                                                                                                                                                                                                                        |
| `instance_type`     | `m`, `l`, `xl`                           | `m`                                                                       | Sets the instance CPU size. You can also specify vendor specific machines in AWS i.e. `t2.micro`. [See equivalences](#Supported-vendors) table below.                                                                                                                                         |
| `instance_gpu`      | ``, `testla`, `k80`                      | ``                                                                        | Sets the desired GPU for supported `instance_types`.                                                                                                                                                                                                                                          |
| `ssh_private`       |                                          |                                                                           | SSH private key in PEM format. If not provided, one private and public key wll be automatically generated and returned in terraform.tfstate                                                                                                                                                   |
| `startup_script`    |                                          |                                                                           | Startup script also known as userData on AWS and customData in Azure. It can be expressed as multiline text using [TF heredoc syntax ](https://www.terraform.io/docs/configuration-0-11/variables.html)                                                                                       |

## Requirements

To be able to use `instance_type` and `instance_gpu`, you'll need access to
launch [instances from supported cloud vendors](#Supported-vendors). Please
ensure that you have sufficient quotas with your cloud provider for the
instances you intend to provision with Iterative Provider. If you're just
starting out with a new account with a vendor, we recommend trying Iterative
Provider with approved instances, such as the `t2.micro` instance for AWS.

<details>
<summary>Example with native AWS instace type and region</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west-1"
  image = "iterative-cml"
  name = "machine"
  instance_hdd_size = "10"
  instance_type = "t2.micro"
}
```

</p>
</details>

## Supported vendors

The Iterative Provider currently supports AWS and Azure. Google Cloud Platform
is not currently supported.

<details>
<summary>AWS instance equivalences</summary>
<p>

The instance type in AWS is calculated by joining the `instance_type` and
`instance_gpu` values.

| type | gpu  | aws         |
| ---- | ---- | ----------- |
| m    |      | m5.2xlarge  |
| l    |      | m5.8xlarge  |
| xl   |      | m5.16xlarge |
| m    | k80  | p2.xlarge   |
| l    | k80  | p2.8xlarge  |
| xl   | k80  | p2.16xlarge |
| m    | v100 | p3.xlarge   |
| l    | v100 | p3.8xlarge  |
| xl   | v100 | p3.16xlarge |

| region   | aws        |
| -------- | ---------- |
| us-west  | us-west-1  |
| us-east  | us-east-1  |
| eu-north | us-north-1 |
| eu-west  | us-west-1  |

</p>
</details>

<details>
<summary>Azure instance equivalences</summary>
<p>

The instance type in Azure is calculated by joining the `instance_type` and
`instance_gpu`

| type | gpu  | azure             |
| ---- | ---- | ----------------- |
| m    |      | Standard_F8s_v2   |
| l    |      | Standard_F32s_v2  |
| xl   |      | Standard_F64s_v2  |
| m    | k80  | Standard_NC6      |
| l    | k80  | Standard_NC12     |
| xl   | k80  | Standard_NC24     |
| m    | v100 | Standard_NC6s_v3  |
| l    | v100 | Standard_NC12s_v3 |
| xl   | v100 | Standard_NC24s_v3 |

| region   | azure       |
| -------- | ----------- |
| us-west  | westus2     |
| us-east  | eastus      |
| eu-north | northeurope |
| eu-west  | westeurope  |

</p>
</details>

## The iterative-cml image

We've created a GPU-ready image based on Ubuntu 18.04. It comes with the
following stack already installed:

- Nvidia drivers
- Docker
- Nvidia-docker
