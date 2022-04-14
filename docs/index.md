# Terraform Provider Iterative (TPI)

![TPI](https://static.iterative.ai/img/tpi/banner.svg)

[![tests](https://img.shields.io/github/workflow/status/iterative/terraform-provider-iterative/Test?label=tests&logo=GitHub)](https://github.com/iterative/terraform-provider-iterative/actions/workflows/test.yml)
[![Apache-2.0](https://img.shields.io/badge/licence-Apache%202.0-blue)](https://github.com/iterative/terraform-provider-iterative/blob/master/LICENSE)

TPI is a [Terraform](https://terraform.io) plugin built with machine learning in mind. Full lifecycle management of computing resources (including GPUs and respawning spot instances) from several cloud vendors (AWS, Azure, GCP, K8s)... without needing to be a cloud expert.

- **Easy to use**: create cloud compute (CPU, GPU, RAM) & storage resources without reading pages of documentation
- **Low cost**: transparent auto-recovery from interrupted low-cost spot/preemptible instances
- **No cloud vendor lock-in**: switch between several cloud vendors with ease due to concise unified configuration
- **Seamless developer experience**: easily sync & run data & code in the cloud as easily as on a local laptop
- **No waste**: auto-cleanup unused resources (terminate compute instances upon job completion/failure & remove storage upon download of results)

Supported cloud vendors [include][auth]:

- Amazon Web Services (AWS)
- Microsoft Azure
- Google Cloud Platform (GCP)
- Kubernetes (K8s)

[auth]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication

## Links

- [Getting Started](https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/getting-started)
  - [Authentication][auth]
- [Full reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task)
