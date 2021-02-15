![Terraform Provider Iterative](https://user-images.githubusercontent.com/414967/98701372-7f60d700-2379-11eb-90d0-47b5eeb22658.png)

# Terraform Provider Iterative

The Terraform Iterative provider is a plugin for Terraform that allows for the
full lifecycle management of GPU or non GPU cloud resources and with your
favourite [vendor](#supported-vendors). There are two types of resources
available:

- iterative_machine
- iterative_cml_runner

# Usage

### CML runner

A CI self hosted runner based on a thin wrapper over the GL and GH runner:

- same spec:
  - name
  - labels
  - idle-timeout
  - repo
  - token
  - driver
- Unified logging
- Easy to launch
- Auto provision of cloud resources
- Auto unregister and removal of cloud resources

#### 1- Setup your provider credentials as ENV variables

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

<details>
<summary>Kubernetes</summary>
<p>

```sh
export KUBERNETES_CONFIGURATION=KUBECONFIG_CONTENTS
export CML_TOKEN=YOUR_REPO_TOKEN
```

</p>
</details>

#### 2- Save your terraform file main.tf

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
    repo           = "https://github.com/iterative/cml"
    driver         = "github"
    labels         = "tf"

    cloud          = "aws"
    region         = "us-west"
    instance_type  = "m"
    # Uncomment it if GPU is needed:
    # instance_gpu = "tesla"
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
    repo           = "https://github.com/iterative/cml"
    driver         = "github"
    labels         = "tf"

    cloud          = "azure"
    region         = "us-west"
    instance_type  = "m"
    # Uncomment it if GPU is needed:
    # instance_gpu = "tesla"
}
```

</p>
</details>

<details>
<summary>Kubernetes</summary>
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
    repo           = "https://github.com/iterative/cml"
    driver         = "github"
    labels         = "tf"

    cloud          = "kubernetes"
    instance_type  = "m"
    # Uncomment it if GPU is needed:
    # instance_gpu = "tesla"
}
```

</p>
</details>

#### 3- Launch it!

```
terraform init
terraform apply --auto-approve
```

#### Argument reference

| Variable            | Values                                   | Default                                                                   |                                                                                                                                                                                                                                                                                               |
| ------------------- | ---------------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `driver`            | `gitlab` `github`                        |                                                                           | The kind of runner that you are setting                                                                                                                                                                                                                                                       |
| `repo`              |                                          |                                                                           | The repo to subscribe to.                                                                                                                                                                                                                                                                     |
| `token`             |                                          |                                                                           | The repository token. It must have Workflow permissions in Github. If not specified tries to read it from the env variable CML_REPO                                                                                                                                                           |
| `labels`            |                                          | `cml`                                                                     | The runner labels for your CI workflow to be waiting for                                                                                                                                                                                                                                      |
| `idle-timeout`      |                                          | 5min                                                                      | The max time for the runner to be waiting for jobs. If the timeout happens the runner will unregister automatically from the repo and cleanup all the cloud resources. If set to `0` it will wait forever.                                                                                    |
| `cloud`             | `aws` `azure`                            |                                                                           | Sets cloud vendor.                                                                                                                                                                                                                                                                            |
| `region`            | `us-west` `us-east` `eu-west` `eu-north` | `us-west`                                                                 | Sets the collocation region. AWS or Azure regions are also accepted.                                                                                                                                                                                                                          |
| `image`             |                                          | `iterative-cml` in AWS `Canonical:UbuntuServer:18.04-LTS:latest` in Azure | Sets the image to be used. On AWS the provider does a search in the cloud provider by image name not by id, taking the lastest version in case there are many with the same name. Defaults to [iterative-cml image](#iterative-cml-image). On Azure uses the form Publisher:Offer:SKU:Version |
| `spot`| boolean | false | If true launch a spot instance  |
| `spot_price`| float with 5 decimals at most | -1 | Sets the max price that you are willing to pay by the hour. If not specified it takes current spot bidding pricing |
| `name`              |                                          | iterative\_{UID}                                                          | Sets the instance name and related resources based on that name. In Azure groups everything under a resource group with that name.                                                                                                                                                            |
| `instance_hdd_size` |                                          | 10                                                                        | Sets the instance hard disk size in gb                                                                                                                                                                                                                                                        |
| `instance_type`     | `m`, `l`, `xl`                           | `m`                                                                       | Sets thee instance computing size. You can also specify vendor specific machines in AWS i.e. `t2.micro`. [See equivalences](#Supported-vendors) table below.                                                                                                                                  |
| `instance_gpu`      | ``, `testla`, `k80`                      | ``                                                                        | Sets the desired GPU if the `instance_type` is one of our types.                                                                                                                                                                                                                              |
| `ssh_private`       |                                          |                                                                           | SSH private in PEM format. If not provided one private and public key wll be automatically generated and returned in terraform.tfstate                                                                                                                                                        |

### Machine

#### 1- Setup your provider credentials as ENV variables

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

<details>
<summary>Kubernetes</summary>
<p>

```sh
export KUBERNETES_CONFIGURATION="$(cat ~/.kube/config)"
```

</p>
</details>

#### 2- Save your terraform file main.tf

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
  cloud             = "aws"
  region            = "us-west"
  name              = "machine"
  instance_hdd_size = 10
  instance_type     = "m"
  # Uncomment it if GPU is needed:
  # instance_gpu    = "tesla"
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
  # instance_gpu = "tesla"
}
```

</p>
</details>

#### 3- Launch it!

```
terraform init
terraform apply --auto-approve
```

#### 4- Stop it

Run it to destroy your instance

```
terraform destroy --auto-approve
```

#### Argument reference

| Variable            | Values                                   | Default                                                                   |                                                                                                                                                                                                                                                                                               |
| ------------------- | ---------------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cloud`             | `aws` `azure`                            |                                                                           | Sets cloud vendor.                                                                                                                                                                                                                                                                            |
| `region`            | `us-west` `us-east` `eu-west` `eu-north` | `us-west`                                                                 | Sets the collocation region. AWS or Azure regions are also accepted.                                                                                                                                                                                                                          |
| `image`             |                                          | `iterative-cml` in AWS `Canonical:UbuntuServer:18.04-LTS:latest` in Azure | Sets the image to be used. On AWS the provider does a search in the cloud provider by image name not by id, taking the lastest version in case there are many with the same name. Defaults to [iterative-cml image](#iterative-cml-image). On Azure uses the form Publisher:Offer:SKU:Version |
| `name`              |                                          | iterative\_{UID}                                                          | Sets the instance name and related resources based on that name. In Azure groups everything under a resource group with that name.                                                                                                                                                            |
| `spot`| boolean | false | If true launch a spot instance  |
| `spot_price`| float with 5 decimals at most | -1 | Sets the max price that you are willing to pay by the hour. If not specified it takes current spot bidding pricing |
| `instance_hdd_size` |                                          | 10                                                                        | Sets the instance hard disk size in gb                                                                                                                                                                                                                                                        |
| `instance_type`     | `m`, `l`, `xl`                           | `m`                                                                       | Sets thee instance computing size. You can also specify vendor specific machines in AWS i.e. `t2.micro`. [See equivalences](#Supported-vendors) table below.                                                                                                                                  |
| `instance_gpu`      | ``, `testla`, `k80`                      | ``                                                                        | Sets the desired GPU if the `instance_type` is one of our types.                                                                                                                                                                                                                              |
| `ssh_private`       |                                          |                                                                           | SSH private in PEM format. If not provided one private and public key wll be automatically generated and returned in terraform.tfstate                                                                                                                                                        |
| `startup_script`    |                                          |                                                                           | Startup script also known as userData on AWS and customData in Azure. It can be expressed as multiline text using [TF heredoc syntax ](https://www.terraform.io/docs/configuration-0-11/variables.html)                                                                                       |

# Pitfalls

To be able to use the `instance_type` and `instance_gpu` you will need also to
be allowed to launch [such instances](#Supported-vendors) within you cloud
provider. Normally all the GPU instances need to be approved prior to be used by
your vendor. You can always try with an already approved instance type by your
vendor just setting it i.e. `t2.micro`

<details>
<summary>Example with native AWS instace type and region</summary>
<p>

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
      version = "0.5.1"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west-1"
  ami = "iterative-cml"
  instance_name = "machine"
  instance_hdd_size = "10"
  instance_type = "t2.micro"
}
```

</p>
</details>

# Supported vendors

- AWS
- Azure

<details>
<summary>AWS instance equivalences</summary>
<p>

The instance type in AWS is calculated joining the `instance_type` and
`instance_gpu`

| type | gpu   | aws         |
| ---- | ----- | ----------- |
| m    |       | m5.2xlarge  |
| l    |       | m5.8xlarge  |
| xl   |       | m5.16xlarge |
| m    | k80   | p2.xlarge   |
| l    | k80   | p2.8xlarge  |
| xl   | k80   | p2.16xlarge |
| m    | tesla | p3.xlarge   |
| l    | tesla | p3.8xlarge  |
| xl   | tesla | p3.16xlarge |

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

The instance type in Azure is calculated joining the `instance_type` and
`instance_gpu`

| type | gpu   | azure             |
| ---- | ----- | ----------------- |
| m    |       | Standard_F8s_v2   |
| l    |       | Standard_F32s_v2  |
| xl   |       | Standard_F64s_v2  |
| m    | k80   | Standard_NC6      |
| l    | k80   | Standard_NC12     |
| xl   | k80   | Standard_NC24     |
| m    | tesla | Standard_NC6s_v3  |
| l    | tesla | Standard_NC12s_v3 |
| xl   | tesla | Standard_NC24s_v3 |

| region   | azure       |
| -------- | ----------- |
| us-west  | westus2     |
| us-east  | eastus      |
| eu-north | northeurope |
| eu-west  | westeurope  |

</p>
</details>

# iterative-cml image

It's a GPU ready image based on Ubuntu 18.04. It has the following stack already
installed:

- nvidia drivers
- docker
- nvidia-docker
