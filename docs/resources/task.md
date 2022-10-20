# Task Resource

This resource will:

1. Create cloud resources (machines and storage) for the task.
2. Upload the given `storage.workdir` to the cloud storage.
3. Run the given `script` on the cloud machine until completion or `timeout`.
4. Download results to the given `storage.output`.

## Example Usage

```hcl
resource "iterative_task" "example" {
  cloud       = "aws"     # or any of: gcp, az, k8s
  machine     = "m"       # medium. Or any of: l, xl, m+k80, xl+v100, ...
  image       = "ubuntu"  # or "nvidia", ...
  region      = "us-west" # or "us-east", "eu-west", ...
  disk_size   = -1        # GB. Default -1 for automatic
  spot        = 0         # auto-price. Default -1 to disable, or >0 for hourly USD limit
  parallelism = 1
  timeout     = 86400     # max 24h before forced termination

  environment = { GREETING = "Hello, world!" }
  storage {
    workdir = "."         # default blank (don't upload)
    output  = "results"   # default blank (don't download). Relative to workdir
    exclude = ["/.dvc/cache", "/results/tempfile", "*.pyc"]
  }
  script = <<-END
    #!/bin/bash

    # create output directory if needed
    mkdir -p results
    echo "$GREETING" | tee results/$(uuidgen)
    # read last result (in case of spot/preemptible instance recovery)
    if test -f results/epoch.txt; then EPOCH="$(cat results/epoch.txt)"; fi
    EPOCH=$${EPOCH:-1}  # start from 1 if last result not found

    echo "(re)starting training loop from $EPOCH up to 1337 epochs"
    for epoch in $(seq $EPOCH 1337); do
      sleep 1
      echo "$epoch" | tee results/epoch.txt
    done
  END
  # or: script = file("example.sh")
}
```

## Argument Reference

### Required

- `cloud` - (Required) Cloud provider to run the task on; valid values are `aws`, `gcp`, `az` and `k8s`.
- `script` - (Required) Script to run (relative to `storage.workdir`); must begin with a valid [shebang](<https://en.wikipedia.org/wiki/Shebang_(Unix)>). Can use a string, including a [heredoc](https://www.terraform.io/docs/language/expressions/strings.html#heredoc-strings), or the contents of a file returned by the [`file`](https://www.terraform.io/docs/language/functions/file.html) function.

### Optional

- `region` - (Optional) [Cloud region/zone](#cloud-region) to run the task on, or node selector labels for Kubernetes.
- `machine` - (Optional) See [Machine Types](#machine-type) below.
- `disk_size` - (Optional) Size of the ephemeral machine storage in GB. `-1`: automatic based on `image`.
- `spot` - (Optional) Spot instance price. `-1`: disabled, `0`: automatic price, any other positive number: maximum bidding price in USD per hour (above which the instance is terminated until the price drops).
- `image` - (Optional) [Machine image](#machine-image) to run the task with.
- `permission_set` - (Optional) See [Permission Set](#permission-set) below.
- `parallelism` - (Optional) Number of machines to be launched in parallel.
- `storage.workdir` - (Optional) Local working directory to upload and use as the `script` working directory.
- `storage.output` - (Optional) Results directory (**relative to `workdir`**) to download (default: no download).
- `storage.exclude` - (Optional) List of files and globs to exclude from transfering. Excluded files are neither uploaded to cloud storage nor downloaded from it. Exclusions are defined relative to `storage.workdir`.
- `environment` - (Optional) Map of environment variable names and values for the task script. Empty string values are replaced with local environment values. Empty values may also be combined with a [glob](<https://en.wikipedia.org/wiki/Glob_(programming)>) name to import all matching variables.
- `timeout` - (Optional) Maximum number of seconds to run before instances are force-terminated. The countdown is reset each time TPI auto-respawns a spot instance.
- `tags` - (Optional) Map of tags for the created cloud resources.
- `name` - (Optional) _Discouraged and may be removed in future - change the resource name instead, i.e. `resource "iterative_task" "some_other_example_name"`._ Deterministic task name (e.g. `name="Hello, World!"` always produces `id="tpi-hello-world-5kz6ldls-57wo7rsp"`).

-> **Note:** `output` is relative to `workdir`, so `storage { workdir = "foo", output = "bar" }` means "upload `./foo/`, change working directory to the uploaded folder, run `script`, and download `bar` (i.e. `./foo/bar`)".

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - Task identifier, `tpi-{name}-{random_hash_1}-{random_hash_2}`. Either the full `{id}` or (if too long), the shorter `{random_hash_1}{random_hash_2}` is used as the name for all cloud resources.
- `ssh_public_key` - Used to access the created machines.
- `ssh_private_key` - Used to access the created machines.
- `addresses` - IP addresses of the currently active machines.
- `status` - Status of the machine orchestrator.
- `events` - List of events for the machine orchestrator.
- `logs` - List with task logs; one for each machine.

~> **Warning:** `events` have different formats across cloud providers and cannot be relied on for programmatic consumption/automation.

After `terraform apply`, these attributes may be obtained using `terraform console`. For example:

```console
$ echo 'try(join("\n", iterative_task.example.logs), "")' | terraform console
```

For convenience, this can placed in an `output` block in `main.tf`:

```hcl
resource "iterative_task" "example" {
  ...
}
output "logs" {
  value = try(join("\n", iterative_task.example.logs), "")
}
```

The above would allow:

```console
$ terraform output --raw logs
```

Finally, JSON output can be parsed using `terraform show --json` and `jq` like this:

```console
$ terraform show --json | jq --raw-output '
  .values.root_module.resources[] |
  select(.address == "iterative_task.example") |
  .values.logs[]'
```

## Machine Type

### Generic

The Iterative Provider offers some common machine types (medium, large, and extra large) which are roughly the same for all supported clouds.

| Type      | Minimum CPU cores | Minimum RAM | GPU                 |
| :-------- | ----------------: | ----------: | :------------------ |
| `s`       |                 1 |        1 GB | -                   |
| `m`       |                 8 |       16 GB | -                   |
| `l`       |                32 |       64 GB | -                   |
| `xl`      |                64 |      128 GB | -                   |
| `m+t4`    |                 4 |       16 GB | 1 NVIDIA Tesla T4   |
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

## Machine Image

### Generic

The Iterative Provider offers some common machine images which are roughly the same for all supported clouds.

- `ubuntu` - Official [Ubuntu LTS](https://wiki.ubuntu.com/LTS) image (currently 20.04).
- `nvidia` - Official Ubuntu LTS with NVIDIA GPU drivers and CUDA toolkit (currently 11.3).

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

## Cloud Region

### Generic

The Iterative Provider offers some common cloud regions which are roughly the same for all supported clouds.

- `us-west` - United States of America, West.
- `us-east` - United States of America, East.
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

For Kubernetes, the `region` attribute can be used to set the [`nodeSelector`](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) field of the job specification. The value of `region` should be a comma-delimited list of `key=value` nodel label pairs.

For example:
```tf
resource "iterative_task" "example" {
  cloud  = "k8s"
  region = "foo=bar,goo=baz"
}
```

This will create a pod with a `nodeSelector` value like the following:
```yaml
apiVersion: V1
kind: Pod
spec:
  nodeSelector:
    foo: bar
    goo: baz
```

## Permission Set

A set of "permissions" assigned to the `task` instance, format depends on the cloud provider

#### Amazon Web Services

An [instance profile `arn`](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html), e.g.:
`permission_set = "arn:aws:iam:1234567890:instance-profile/rolename"`

#### Google Cloud Platform

A service account email and a [list of scopes](https://cloud.google.com/sdk/gcloud/reference/alpha/compute/instances/set-scopes#--scopes), e.g.:
`permission_set = "sa-name@project_id.iam.gserviceaccount.com,scopes=storage-rw"`

#### Microsoft Azure
A comma-separated list of [user-assigned identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview) ARM resource ids, e.g.:
`permission_set = "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}"`

#### Kubernetes

The name of a service account in the current namespace.

## Known Issues

### Kubernetes

#### Directory storage

Unlike public cloud providers, Kubernetes does not offer any portable way of persisting and sharing storage between pods. When specified, the `storage.workdir` attribute will create a `PersistentVolumeClaim` of the default `StorageClass`, with the same lifecycle as the task and the specified `disk_size`.

~> **Warning:** Access mode will be `ReadWriteOnce` if `parallelism=1` or `ReadWriteMany` otherwise.

-> **Note:** Rancher's [Local Path Provisioner](https://github.com/rancher/local-path-provisioner) might be the easiest way of deploying a quick `ReadWriteOnce` dynamically allocated storage solution for testing: just run `kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml`.
