---
page_title: Setting up Azure Kubernetes Service
subcategory: Development
---

# Setting up Azure Kubernetes Service

## Installing the Azure command-line interface tool

First of all, we need to install the `az` tool following the official documentation on the Microsoft documentation portal: [https://docs.microsoft.com/en-us/cli/azure/install-azure-cli](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)

## Enabling experimental features

In order to automatically provision GPU nodes for our cluster, we'll also need to enable the following experimental features through the `aks-preview` extension:

```bash
az extension add \
  --name aks-preview

az provider register \
  --namespace Microsoft.ContainerService

az feature register \
  --namespace Microsoft.ContainerService \
  --name GPUDedicatedVHDPreview
```

## Creating a test cluster

The following commands will create an AKS cluster with a single node, keeping everything into a a new resource group for easier deletion of the resources:

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

Azure has some wrappers for Kubernetes authentication and will generate for us the required credentials. The following command will produce a full-fledged `kubeconfig` string that can be directly stored in the `KUBERNETES_CONFIGURATION` secret of your continous integration system of choice:

```bash
az aks get-credentials \
  --resource-group testKubernetesResourceGroup \
  --name testKubernetesCluster \
  --file -
```

-> **Note:** If you skip the `--file` option, settings will be saved to your computer's `~/.kube/config` file, that will be automatically used by `kubectl` if you ever need to run any sanity check manually on the cluster.

## Deleting the test cluster

Once you've finished testing you can run the following command to delete the entire resource group, which includes the cluster and all its nodes:

```bash
az group delete \
  --name testKubernetesResourceGroup
```

~> **Warning:** Please delete the entire resource group as soon as you finish using the cluster. They are really pricey, and you may end spending a lot of money just on testing. 🔥💵
