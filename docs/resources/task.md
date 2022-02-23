# Task Resource

This resource will:

1. Create cloud resources (machines and storage) for the task.
2. Upload the given `workdir.input` to the cloud storage.
3. Run the given `script` on the cloud machine until completion or `timeout`.
4. Download results to the given `workdir.output`.

## Example Usage

```hcl
resource "iterative_task" "task" {
  cloud = "aws"

  environment = { GREETING = "Hello, world!" }
  workdir {
    input  = "${path.root}/shared"
    output = "${path.root}/results"
  }
  script = <<-END
    #!/bin/bash
    echo "$GREETING" | tee $(uuidgen)
  END
}
```

## Argument Reference

### Required

- `cloud` - (Required) Cloud provider to run the task on; valid values are `aws`, `gcp`, `az` and `k8s`.
- `script` - (Required) Script to run; must begin with a valid [shebang](<https://en.wikipedia.org/wiki/Shebang_(Unix)>).

### Optional

- `name` - (Optional) Deterministic task name.
- `region` - (Optional) [Cloud region/zone](#cloud-regions) to run the task on.
- `machine` - (Optional) See [Machine Types](#machine-types) below.
- `disk_size` - (Optional) Size of the ephemeral machine storage.
- `spot` - (Optional) Spot instance price. `-1`: disabled, `0`: automatic price, any other positive number: fixed price.
- `image` - (Optional) [Machine image](#machine-images) to run the task with.
- `parallelism` - (Optional) Number of machines to be launched in parallel.
- `workdir.input` - (Optional) Local working directory to upload.
- `workdir.output` - (Optional) Local directory to download results to (default: no download).
- `environment` - (Optional) Map of environment variable names and values for the task script. Empty string values are replaced with local environment values. Empty values may also be combined with a [glob](<https://en.wikipedia.org/wiki/Glob_(programming)>) name to import all matching variables.
- `timeout` - (Optional) Maximum number of seconds to run before termination.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `ssh_public_key` - Used to access the created machines.
- `ssh_private_key` - Used to access the created machines.
- `addresses` - IP addresses of the currently active machines.
- `status` - Status of the machine orchestrator.
- `events` - List of events for the machine orchestrator.
- `logs` - List with task logs; one for each machine.

~> **Note:** Status and events don't produce a stable output between cloud providers and are intended for human consumption only.

## Machine Type

### Generic

The Iterative Provider offers some common machine types (medium, large, and extra large) which are roughly the same for all supported clouds.

| Type      | Minimum CPU cores | Minimum RAM | GPU                 |
| :-------- | ----------------: | ----------: | :------------------ |
| `m`       |                 8 |       16 GB | -                   |
| `l`       |                32 |       64 GB | -                   |
| `xl`      |                64 |      128 GB | -                   |
| `m+k80`   |                 4 |       53 GB | 1 NVIDIA Tesla K80  |
| `l+k80`   |                12 |      112 GB | 2 NVIDIA Tesla K80  |
| `xl+k80`  |                24 |      212 GB | 4 NVIDIA Tesla K80  |
| `m+v100`  |                 4 |       61 GB | 1 NVIDIA Tesla V100 |
| `l+v100`  |                12 |      224 GB | 2 NVIDIA Tesla V100 |
| `xl+v100` |                24 |      448 GB | 4 NVIDIA Tesla V100 |

See [Generic Machine Types](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/generic-machine-types) for exact specifications for each cloud.

### Cloud-specific

In addition to generic types, it's possible to specify any machine type supported by the underlying cloud provider.

#### Amazon Web Services

- `{machine}` - Any [EC2 instance type](https://aws.amazon.com/ec2/instance-explorer) (e.g. `g4dn.xlarge`).

#### Microsoft Azure

- `{machine}` - Any [Azure VM](https://azure.microsoft.com/en-us/pricing/vm-selector) (e.g. `Standard_F8s_v2`).

#### Google Cloud Platform

- `{machine}` - Any [GCP machine type](https://cloud.google.com/compute/docs/machine-types) (e.g. `n2-custom-64-262144`).
- `{machine}+{accelerator}*{count}` - Any machine and accelerator combination (e.g. `custom-8-53248+nvidia-tesla-k80*1`).

#### Kubernetes

- `{cpu}-{memory}` - Any [CPU & memory combination](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers) (e.g. `64-256000`).
- `{cpu}-{memory}+{accelerator}*{count}` - Any CPU, memory, & [accelerator](https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus) combination (e.g. `64-256000+nvidia-tesla-k80*1`).

-> **Note:** Specified resource amounts are considered **limits** (rather than **requests**).

-> **Note:** `{accelerator}` will be transformed into a node selector requesting `accelerator={accelerator}` and `{count}` will be configured as the **limits** count for `kubernetes.io/gpu`.

## Machine Images

### Generic

The Iterative Provider offers some common machine images which are roughly the same for all supported clouds.

- `ubuntu` - Official [Ubuntu LTS](https://wiki.ubuntu.com/LTS) image (currently 20.04).

### Cloud-specific

In addition to generic images, it's possible to specify any machine image supported by the underlying cloud provider.

Images should include, at least:

- Linux
- cloud-init
- systemd
- curl
- unzip, python3 or python

#### Amazon Web Services

`{user}@{owner}:{architecture}:{name}`

Fields:

- `{user}` - User name; e.g. `ubuntu`.
- `{architecture}` - Image architecture; e.g. `x86_64` or `*` for any.
- `{owner}` - Account number of the image owner; e.g. `099720109477` or `*` for any.
- `{name}` - Name of the image; e.g. `*ubuntu/images/hvm-ssd/ubuntu-focal-20.04*`.

See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html for more information.

#### Google Cloud Platform

`{user}@{project}/{family}`

Fields:

- `{user}` - User name; e.g. `ubuntu`.
- `{project}` - Project name; e.g. `ubuntu-os-cloud`.
- `{family}` - Image architecture; e.g. `ubuntu-2004-lts`.

See https://cloud.google.com/compute/docs/images/os-details for more information.

#### Microsoft Azure

`{user}@{publisher}:{offer}:{sku}:{version}`

Fields:

- `{user}` - User name; e.g. `ubuntu`.
- `{publisher}` - Image publisher; e.g. `Canonical`.
- `{offer}` - Image offer; e.g. `UbuntuServer`.
- `{sku}` - Image SKU; e.g. `18.04-LTS`.
- `{version}` - Image version; e.g. `latest`.

See https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage for more information.

### Kubernetes

- `{image}` - Any [container image](https://kubernetes.io/docs/concepts/containers/images/#image-names).

## Cloud Regions

### Generic

The Iterative Provider offers some common cloud regions which are roughly the same for all supported clouds.

- `us-east` - United States of America, East.
- `us-west` - United States of America, West.
- `eu-north` - Europe, North.
- `eu-west` - Europe, West.

### Cloud-specific

In addition to generic regions, it's possible to specify any cloud region supported by the underlying cloud provider.

#### Amazon Web Services

- `{region}` - Any [AWS region](https://aws.amazon.com/about-aws/global-infrastructure/regions_az) (e.g. `us-east-1`).

#### Google Cloud Platform

- `{zone}` - Any [GCP **zone**](https://cloud.google.com/compute/docs/regions-zones) (not region) (e.g. `us-east1-a`).

#### Microsoft Azure

- `{region}` - Any [Azure region](https://azure.microsoft.com/en-us/global-infrastructure/geographies) (e.g. `eastus`).

### Kubernetes

The `region` attribute is ignored.

## Known Issues

### Kubernetes

#### Region attribute

Setting the `region` attribute results in undefined behaviour.

#### Directory storage

Unlike public cloud providers, Kubernetes does not offer any portable way of persisting and sharing storage between pods. When specified, the `workdir.input` attribute will create a `PersistentVolumeClaim` of the default `StorageClass`, with the same lifecycle as the task and the specified `disk_size`.

~> **Warning:** Access mode will be `ReadWriteOnce` if `parallelism=1` or `ReadWriteMany` otherwise.

-> **Note:** Rancher's [Local Path Provisioner](https://github.com/rancher/local-path-provisioner) might be the easiest way of deploying a quick `ReadWriteOnce` dynamically allocated storage solution for testing: just run `kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml`.
