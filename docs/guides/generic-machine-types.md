---
page_title: Generic Machine Types
subcategory: Development
---

# Generic Machine Types

The table below is a more detailed version of the common choices summarised in [Task Machine Types](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#machine-type).

| Type      | [`aws`]       | [`az`]              | [`gcp`]                                         | [`k8s`]                                                |
| :-------- | :------------ | :------------------ | :---------------------------------------------- | :----------------------------------------------------- |
| `m`       | `m5.2xlarge`  | `Standard_F8s_v2`   | `e2-custom-8-32768`                             | `cpu: 8`<br>`memory: 32G`                              |
| `l`       | `m5.8xlarge`  | `Standard_F32s_v2`  | `e2-custom-32-131072`                           | `cpu: 32`<br>`memory: 128G`                            |
| `xl`      | `m5.16xlarge` | `Standard_F64s_v2`  | `n2-custom-64-262144`                           | `cpu: 64`<br>`memory: 256G`                            |
| `m+k80`   | `p2.xlarge`   | `Standard_NC6`      | `custom-8-53248`<br>1 `nvidia-tesla-k80`        | `cpu: 4`<br>`memory: 64G`<br>1 `nvidia-tesla-k80`      |
| `l+k80`   | `p2.8xlarge`  | `Standard_NC12`     | `custom-32-131072`<br>4 `nvidia-tesla-k80`      | `cpu: 32`<br>`memory: 512G`<br>8 `nvidia-tesla-k80`    |
| `xl+k80`  | `p2.16xlarge` | `Standard_NC24`     | `custom-64-212992-ext`<br>8 `nvidia-tesla-k80`  | `cpu: 64`<br>`memory: 768G`<br>16 × `nvidia-tesla-k80` |
| `m+v100`  | `p3.xlarge`   | `Standard_NC6s_v3`  | `custom-8-65536-ext`<br>1 `nvidia-tesla-v100`   | `cpu: 8`<br>`memory: 64G`<br>1 `nvidia-tesla-v100`     |
| `l+v100`  | `p3.8xlarge`  | `Standard_NC12s_v3` | `custom-32-262144-ext`<br>4 `nvidia-tesla-v100` | `cpu: 32`<br>`memory: 256G`<br>4 `nvidia-tesla-v100`   |
| `xl+v100` | `p3.16xlarge` | `Standard_NC24s_v3` | `custom-64-524288-ext`<br>8 `nvidia-tesla-v100` | `cpu: 64`<br>`memory: 512G`<br>8 `nvidia-tesla-v100`   |

[`aws`]: https://aws.amazon.com/ec2/instance-explorer
[`az`]: https://azure.microsoft.com/en-us/pricing/vm-selector
[`gcp`]: https://cloud.google.com/compute/docs/machine-types
[`k8s`]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
