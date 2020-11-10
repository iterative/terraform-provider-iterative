![Terraform Provider Iterative](https://user-images.githubusercontent.com/414967/98701372-7f60d700-2379-11eb-90d0-47b5eeb22658.png)

# Terraform Provider Iterative

The Terraform Iterative provider is a plugin for Terraform that allows for the full lifecycle management of GPU or non GPU cloud resources with your favourite [vendor](#supported-vendors). The provider offers a simple and homogeneous way to deploy a GPU or a cluster of them reducing the complexity.

# Usage

```tf
terraform {
  required_providers {
    iterative = {
      source = "iterative/iterative"
      version = "0.5.0"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west"
  instance_name = "machine"
  instance_hdd_size = "20"
  instance_type = "m"
  instance_gpu = "tesla"
}
```

## Argument reference

| Variable | Values | Default | |
| ------- | ------ | -------- | ------------- |
| ```region``` | ```us-west``` ```us-east``` ```eu-west``` ```eu-north``` | ```us-west``` | Sets the collocation region |
| ```instance_name``` |  | cml_{UID} | Sets the instance name and related resources like AWS key pair. |
| ```instance_hdd_size``` | | 10 | Sets the instance hard disk size in gb |
| ```instance_type``` | ```m```, ```l```, ```xl``` | ```m``` | Sets thee instance computing size. You can also specify vendor specific machines in AWS i.e. ```t2.micro``` |
| ```instance_gpu``` | ``` ```, ```testla```, ```k80``` | ``` ``` | Sets the desired GPU  if the ```instance_type``` is one of our types. |
| ```key_public``` | | | Set up ssh access with your OpenSSH public key. If not provided one be automatically generated and returned in terraform.tfstate  |
| aws_security_group | | ```cml``` | AWS specific variable to setup an specific security group. If specified the instance will be launched in with that sg within the vpc managed by the specified sg. If not a new sg called ```cml``` will be created under the default vpc |
 

# Supported vendors

 - AWS
 
### AWS instance equivalences.
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
