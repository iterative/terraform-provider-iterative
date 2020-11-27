![Terraform Provider Iterative](https://user-images.githubusercontent.com/414967/98701372-7f60d700-2379-11eb-90d0-47b5eeb22658.png)

# Terraform Provider Iterative

The Terraform Iterative provider is a plugin for Terraform that allows for the full lifecycle management of GPU or non GPU cloud resources with your favourite [vendor](#supported-vendors). The provider offers a simple and homogeneous way to deploy a GPU or a cluster of them reducing the complexity. 

# Usage

#### 1- Setup your provider credentials as ENV variables

```sh
export AWS_SECRET_ACCESS_KEY=YOUR_KEY
export AWS_ACCESS_KEY_ID=YOUR_ID
```

#### 2- Save your terraform file main.tf

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
  region = "us-west"
  ami = "iterative-cml"
  instance_name = "machine"
  instance_hdd_size = "10"
  instance_type = "m"
  ## Uncomment it if GPU is needed:
  # instance_gpu = "tesla" 
}
```

#### 3- Launch it!

```
terraform init
terraform apply --auto-approve

# run it to destroy your instance
# terraform destroy --auto-approve
```

## Pitfalls

To be able to use the ```instance_type``` and ```instance_gpu``` you will need also to be allowed to launch [such instances](#AWS-instance-equivalences) within you cloud provider. Normally all the GPU instances need to be approved prior to be used by your vendor.
You can always try with an already approved instance type by your vendor just setting it i.e. ```t2.micro```

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

## Argument reference

| Variable | Values | Default | |
| ------- | ------ | -------- | ------------- |
| ```region``` | ```us-west``` ```us-east``` ```eu-west``` ```eu-north``` | ```us-west``` | Sets the collocation region. AWS regions are also accepted. |
| ```ami``` | | ```iterative-cml``` | Sets the ami to be used. For that the provider does a search in the cloud provider by image name not by id, taking the lastest version in case there are many with the same name. Defaults to [iterative-cml image](#iterative-cml-image) |
| ```instance_name``` |  | cml_{UID} | Sets the instance name and related resources like AWS key pair. |
| ```instance_hdd_size``` | | 10 | Sets the instance hard disk size in gb |
| ```instance_type``` | ```m```, ```l```, ```xl``` | ```m``` | Sets thee instance computing size. You can also specify vendor specific machines in AWS i.e. ```t2.micro```. [See equivalences]((#AWS-instance-equivalences)) table below. |
| ```instance_gpu``` | ``` ```, ```testla```, ```k80``` | ``` ``` | Sets the desired GPU  if the ```instance_type``` is one of our types. |
| ```key_public``` | | | Set up ssh access with your OpenSSH public key. If not provided one be automatically generated and returned in terraform.tfstate  |
| aws_security_group | | ```cml``` | AWS specific variable to setup an specific security group. If specified the instance will be launched in with that sg within the vpc managed by the specified sg. If not a new sg called ```cml``` will be created under the default vpc |
 

# Supported vendors

 - AWS

### AWS instance equivalences
The instance type in AWS is calculated joining the ```instance_type``` and ```instance_gpu```

| type | gpu | aws |
| ------- | ------ | -------- |
| m |  | m5.2xlarge |
| l |  | m5.8xlarge |
| xl |  | m5.16xlarge |
| m | k80 | p2.xlarge |
| l | k80 | p2.8xlarge |
| xl | k80 | p2.16xlarge |
| m | tesla | p3.xlarge |
| l | tesla | p3.8xlarge |
| xl | tesla | p3.16xlarge |

| region | aws |
| ------- | ------ |
| us-west | us-west-1 |
| us-east | us-east-1 |
| eu-north | us-north-1 |
| eu-west | us-west-1 |

# iterative-cml image

It's a GPU ready image based on Ubuntu 18.04. It has the following stack already installed:

 - nvidia drivers
 - docker
 - nvidia-docker
