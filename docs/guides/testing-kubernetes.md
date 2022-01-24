---
page_title: Azure Kubernetes Service
subcategory: Development
---

# Azure Kubernetes Service

## Installing the Azure command-line interface tool

Install the `az` tool following the official documentation on the Microsoft documentation portal: [https://docs.microsoft.com/en-us/cli/azure/install-azure-cli](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

## Enabling experimental features

In order to automatically provision GPU nodes for the cluster, enable the following experimental features through the `aks-preview` extension:

```bash
az extension add --name aks-preview
az provider register --namespace Microsoft.ContainerService
az feature register \
  --namespace Microsoft.ContainerService \
  --name GPUDedicatedVHDPreview
```

## Creating a test cluster

The following commands will create an AKS cluster with a single node, keeping everything in a new resource group for easier deletion when done:

```bash
az group create \
  --name testKubernetesResourceGroup \
  --location eastus
```

```bash
az aks create \
  --resource-group testKubernetesResourceGroup \
  --name testKubernetesCluster \
  --node-vm-size Standard_NC6 \
  --node-count 1 \
  --aks-custom-headers UseGPUDedicatedVHD=true
```

<details><summary><i>Click to reveal a budget-friendly cluster configuration without GPU...</summary>

```bash
az aks create \
  --resource-group testKubernetesResourceGroup \
  --name testKubernetesCluster \
  --node-vm-size Standard_A2_v2 \
  --node-count 1
```

</details>

## Retrieving the credentials

Azure has some wrappers for Kubernetes authentication, and can generate the required credentials. The following command will produce a full-fledged `kubeconfig` string that can be directly stored in a `KUBERNETES_CONFIGURATION` CI/CD environment secret:

```bash
az aks get-credentials \
  --resource-group testKubernetesResourceGroup \
  --name testKubernetesCluster \
  --file -
```

-> **Note:** Omitting the `--file -` option causes settings to be stored in the local computer's `~/.kube/config` file (which is automatically used by `kubectl` should you ever need to run a manual sanity check on the cluster).

## Deleting the test cluster

Once you've finished testing you can run the following command to delete the entire resource group (including the cluster and all its nodes):

```bash
az group delete --name testKubernetesResourceGroup
```

~> **Warning:** Try to delete the entire resource group as soon as you finish using the cluster to avoid unnecessary expenses. 🔥💵
