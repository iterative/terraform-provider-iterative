# Task Resource

Task, consisting of one or more cloud machines that run a given `script` upon
completion or `timeout`, sharing a common network and a storage `directory`.

## Example Usage

```hcl
resource "iterative_task" "task" {
  name  = "example"
  cloud = "aws"

  environment = {GREETING = "Hello, world!"}
  directory   = path.root

  script = <<-END
    #!/bin/bash
    echo "$GREETING" | tee result-$(uuidgen)
  END
}
```

## Argument Reference

The following arguments are required:

- `name` - (Required) Name of the task.
- `cloud` - (Required) Cloud provider to run the task on; valid values are `aws`, `gcp`, `az` and `k8s`.
- `script` - (Required) Script to run; must begin with a valid [shebang](<https://en.wikipedia.org/wiki/Shebang_(Unix)>).

The following arguments are optional:

- `region` - (Optional) Cloud region / zone to run the task on.
- `machine` - (Optional) Machine type; see the Machine Types section below.
- `disk_size` - (Optional) Size of the ephemeral machine storage.
- `spot` - (Optional) Spot configuration: `-1` means disabled, `0` means enabled with automatic price, and any other positive number sets a fixed price.
- `image` - (Optional) Machine image to run the task with.
- `parallelism` - (Optional) Number of machines to be launched in parallel.
- `directory` - (Optional) Local directory to synchronize.
- `environment` - (Optional) Environment variables
- `timeout` - (Optional) Timeout for the task, in seconds.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `ssh_public_key` - SSH public key to access the created machines.
- `ssh_private_key` - SSH private key to access the created machines.
- `addresses` - IP addresses of the currently active machines.
- `status` - Status of the machine orchestrator.
- `events` - List of events for the machine orchestrator.
- `logs` - List with task logs, one for each machine.

~> **Note** Status and events don't produce a stable output between cloud providers and are intended for human consumption only.

## Machine Sizes

### Generic

This provider offers some common machine sizes that are roughly the same for all
the supported cloud providers.

#### Without GPU

- `m` - Medium, with roughly 8 CPU cores and 32 GB of RAM.
- `l` - Large, with roughly 32 CPU cores and 128 GB of RAM.
- `xl` - Extra large, with roughly 64 CPU cores and 256 GB of RAM.

#### With NVIDIA Tesla K80 GPU

- `m+k80` - Medium, with roughly 8 CPU cores, 32 GB of RAM and 1 GPU device.
- `l+k80` - Large, with roughly 32 CPU cores, 128 GB of RAM and 8 GPU devices.
- `xl+k80` - Extra large, with roughly 64 CPU cores, 512 GB of RAM and 16 GPU devices.

#### With NVIDIA Tesla V100 GPU

- `m+v100` - Medium, with roughly 8 CPU cores, 32 GB of RAM and 1 GPU device.
- `l+v100` - Large, with roughly 32 CPU cores, 128 GB of RAM and 8 GPU devices.
- `xl+v100` - Extra large, with roughly 64 CPU cores, 512 GB of RAM and 16 GPU devices.

### Cloud-specific

In addition to generic sizes, it's possible to specify any machine type
supported by the underlying cloud provider.

#### Amazon Web Services

- `{machine}` - Any cloud-specific machine type, like `p3.16xlarge`.

See https://aws.amazon.com/ec2/instance-explorer for more information.

#### Google Cloud Platform

- `{machine}` - Any cloud-specific machine type, like `n2-custom-64-262144`.
- `{machine}+{accelerator}*{count}` - Any machine and accelerator combination, like `custom-8-53248+nvidia-tesla-k80*1`.

See https://cloud.google.com/compute/docs/machine-types for more information.

#### Microsoft Azure

- `{machine}` - Any cloud-specific machine type, like `Standard_F8s_v2`.

See https://azure.microsoft.com/en-us/pricing/vm-selector for more information.

### Kubernetes

- `{cpu}-{memory}` - Any CPU + memory combination, like `64-256000`.
- `{cpu}-{memory}+{accelerator}*{count}` - Any CPU + memory + accelerator combination, like `64-256000+nvidia-tesla-k80*1`.

See https://kubernetes.io/docs/concepts/configuration/manage-resources-containers and https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus for more information.

-> **Note** Specified resource amounts will be considered as **limits** and not as **requests**.

-> **Note** `{accelerator}` will be transformed into a node selector requesting `accelerator={accelerator}` and `{count}` will be configured as the **limits** count for `kubernetes.io/gpu`.

## Machine Images

### Generic

This provider offers some common machine images that are roughly the same for
all the supported cloud providers.

- `ubuntu` - Official Ubuntu LTS image, currently 20.04.

### Cloud-specific

In addition to generic images, it's possible to specify any machine image
supported by the underlying cloud provider.

#### Amazon Web Services

`{user}@{architecture}:{owner}:{name}`

Fields:

- `{user}` - User name; e.g. `ubuntu`.
- `{architecture}` — Image architecture; e.g. `x86_64` or `*` for any.
- `{owner}` - Account number of the image owner; e.g. `099720109477` or `*` for any.
- `{name}` - Name of the image; e.g. `*ubuntu/images/hvm-ssd/ubuntu-focal-20.04*`.

See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html for more information.

#### Google Cloud Platform

`{user}@{project}/{family}`

Fields:

- `{user}` - User name; e.g. `ubuntu`.
- `{project}` - Project name; e.g. `ubuntu-os-cloud`.
- `{family}` — Image architecture; e.g. `ubuntu-2004-lts`.

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

This provider offers some common cloud regions that are roughly the same for all
the supported cloud providers.

- `us-east` - United States of America, East.
- `us-west` - United States of America, West.
- `eu-north` - Europe, North.
- `eu-west` - Europe, West.

### Cloud-specific

In addition to generic sizes, it's possible to specify any cloud region
supported by the underlying cloud provider.

#### Amazon Web Services

- `{region}` - Any cloud-specific region, like `us-east-1`.

See https://aws.amazon.com/about-aws/global-infrastructure/regions_az for more information.

#### Google Cloud Platform

- `{zone}` - Any cloud-specific **zone** — not region — like `us-east1-a`.

See https://cloud.google.com/compute/docs/regions-zones for more information.

#### Microsoft Azure

- `{region}` - Any cloud-specific region, like `eastus`.

See https://azure.microsoft.com/en-us/global-infrastructure/geographies for more information.

### Kubernetes

The `region` attribute will be ignored.

## Quirks

### Kubernetes

#### Region attribute

Setting the `region` attribute is considered undefined behavior.

#### Directory string format

Unlike public cloud providers, Kubernetes does not offer any portable way of persisting and sharing storage between pods. When specified, the `directory` attribute will create a `PersistentVolumeClaim` with the same lifecycle as the task.

- `{storage_class}:{size}` or
- `{storage_class}:{size}:{path}`

Fields:

- `{storage_class}` - Name of the storage class; e.g. `local-path`.
- `{size}` - Size in gigabytes.
- `{path}` - Local path to synchronize; equivalent to the whole `directory` attribute on public cloud providers. When unspecified, persistent volumes will just be used as a cache and local file synchronization will be disabled.

~> **Warning** Access mode will be `ReadWriteOnce` for `parallelism` equal to 1 or `ReadWriteMany` otherwise.

-> **Note** Rancher's [Local Path Provisioner](https://github.com/rancher/local-path-provisioner) might be the easiest way of deploying a quick `ReadWriteOnce` dynamically allocated storage solution for testing: just run `kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml`
