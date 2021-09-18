variables {
  azure_resource_group  = "cml-packer-resource-group"
  azure_storage_account = "iterative"
  azure_storage_region  = "eastus" # see [3]
}

locals {
  tags = {
    Env   = "Prod"
    Owner = "packer"
  }
}

resource "azurerm_resource_group" "group" {
  name     = var.azure_resource_group
  location = var.azure_storage_region
  tags     = local.tags
}

resource "azurerm_storage_account" "account" {
  name                     = var.azure_storage_account
  resource_group_name      = azurerm_resource_group.group.name
  location                 = azurerm_resource_group.group.location
  allow_blob_public_access = true
  account_tier             = "Standard"
  account_replication_type = "GRS"
  tags                     = local.tags
}

resource "azurerm_storage_container" "container" {
  name                  = "system"
  storage_account_name  = azurerm_storage_account.account.name
  container_access_type = "blob"
}

# [1]: https://www.spectrocloud.com/blog/custom-vm-images-on-azure
# [2]: https://github.com/hashicorp/packer-plugin-azure/blob/ca28f01/builder/azure/arm/builder.go#L34
# [3]: https://github.com/hashicorp/packer-plugin-azure/issues/20

# https://github.com/hashicorp/packer/issues/9647
# https://www.spectrocloud.com/blog/custom-vm-images-on-azure/
# https://cloud.google.com/compute/docs/images/managing-access-custom-images#share-images-publicly
