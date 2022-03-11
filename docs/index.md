# Terraform Provider Iterative (TPI)

![TPI](https://static.iterative.ai/img/cml/banner-terraform.png)

[![tests](https://img.shields.io/github/workflow/status/iterative/terraform-provider-iterative/Test?label=tests&logo=GitHub)](https://github.com/iterative/terraform-provider-iterative/actions/workflows/test.yml)
[![Apache-2.0](https://img.shields.io/badge/licence-Apache%202.0-blue)](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE)

TPI is a [Terraform](https://terraform.io) plugin built with machine learning in mind. Full lifecycle management of computing resources (including GPUs and respawning spot instances) from several cloud vendors (AWS, Azure, GCP, K8s)... without needing to be a cloud expert.

- **Provision Resources**: create cloud compute & storage resources without reading pages of documentation
- **Sync & Execute**: easily sync & run local data & code in the cloud
- **Low cost**: transparent auto-recovery from interrupted low-cost spot/preemptible instances
- **No waste**: auto-cleanup unused resources (terminate compute instances upon job completion/failure & remove storage upon download of results)
- **No lock-in**: switch between several cloud vendors with ease due to concise unified configuration

Supported cloud vendors include:

- Amazon Web Services (AWS)
- Microsoft Azure
- Google Cloud Platform (GCP)
- Kubernetes (K8s)

## Links

- [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started)
  + [Authentication](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication)
- [Full reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task)
