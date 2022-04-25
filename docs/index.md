# Terraform Provider Iterative (TPI)

![TPI](https://static.iterative.ai/img/tpi/banner.svg)

[![tests](https://img.shields.io/github/workflow/status/iterative/terraform-provider-iterative/Test?label=tests&logo=GitHub)](https://github.com/iterative/terraform-provider-iterative/actions/workflows/test.yml)
[![Apache-2.0](https://img.shields.io/badge/licence-Apache%202.0-blue)](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE)

TPI is a [Terraform](https://terraform.io) plugin built with machine learning in mind. This CLI tool offers full lifecycle management of computing resources (including GPUs and respawning spot instances) from several cloud vendors (AWS, Azure, GCP, K8s)... without needing to be a cloud expert.

- **Lower cost with spot recovery**: transparent auto-recovery from interrupted low-cost spot/preemptible instances
- **No cloud vendor lock-in**: switch between clouds with just one line thanks to unified abstraction
- **No waste**: auto-cleanup unused resources (terminate compute instances upon task completion/failure & remove storage upon download of results), pay only for what you use
- **Developer-first experience**: one-command data sync & code execution with no external server, making the cloud feel like a laptop

Supported cloud vendors [include][auth]:

| [![Amazon Web Services (AWS)][aws-badge]][aws] | [![Microsoft Azure][azure-badge]][azure] | [![Google Cloud Platform (GCP)][gcp-badge]][gcp] | [![Kubernetes (K8s)][k8s-badge]][k8s] |
| ---------------------------------------------- | ---------------------------------------- | ------------------------------------------------ | ------------------------------------- |

[aws-badge]: https://img.shields.io/badge/AWS-Amazon_Web_Services-black?colorA=white&logoColor=232F3E&logo=amazonaws
[aws]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication#amazon-web-services
[azure-badge]: https://img.shields.io/badge/Azure-Microsoft_Azure-black?colorA=white&logoColor=0078D4&logo=microsoftazure
[azure]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication#microsoft-azure
[gcp-badge]: https://img.shields.io/badge/GCP-Google_Cloud_Platform-black?colorA=white&logoColor=4285F4&logo=googlecloud
[gcp]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication#google-cloud-platform
[k8s-badge]: https://img.shields.io/badge/K8s-Kubernetes-black?colorA=white&logoColor=326CE5&logo=kubernetes
[k8s]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication#kubernetes
[auth]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication

![](https://static.iterative.ai/img/tpi/high-level-light.png)

## What's Special

There are a several reasons to use TPI instead of other related solutions (custom scripts and/or cloud orchestrators):

1. **Reduced management overhead and infrastructure cost**:
   TPI is a CLI tool, not a running service. It requires no additional orchestrating machine (control plane/head nodes) to schedule/recover/terminate instances. Instead, TPI runs (spot) instances via cloud-native scaling groups ([AWS Auto Scaling Groups](https://docs.aws.amazon.com/autoscaling/ec2/userguide/what-is-amazon-ec2-auto-scaling.html), [Azure VM Scale Sets](https://azure.microsoft.com/en-us/services/virtual-machine-scale-sets), [GCP managed instance groups](https://cloud.google.com/compute/docs/instance-groups#managed_instance_groups), and [Kubernetes Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job)), taking care of recovery and termination automatically on the cloud provider's side. This design reduces management overhead & infrastructure costs. You can close your laptop while cloud tasks are running -- auto-recovery happens even if you are offline.
2. **Unified tool for data science and software development teams**:
   TPI provides consistent tooling for both data scientists and DevOps engineers, improving cross-team collaboration. This simplifies compute management to a single config file, and reduces time to deliver ML models into production.
3. **Reproducible, codified environments**:
   Store hardware requirements & pipelines in a single configuration file with the rest of your ML project code.

<img width=24px src="https://static.iterative.ai/logo/cml.svg"/> TPI is used to power [CML](https://cml.dev), bringing cloud providers to existing GitHub, GitLab & Bitbucket CI/CD workflows ([repository](https://github.com/iterative/cml)).

## Links

- [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started)
  - [Authentication][auth]
- [Full reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task)
