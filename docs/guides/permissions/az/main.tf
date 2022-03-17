terraform {
  required_providers {
    azuread = { source = "hashicorp/azuread", version = "~> 2.18.0" }
    azurerm = { source = "hashicorp/azurerm", version = "~> 2.98.0" }
  }
}

provider "azuread" {}
provider "azurerm" {
  features {}
}

data "azuread_client_config" "current" {}
data "azurerm_subscription" "current" {}

resource "azuread_application" "task" {
  display_name = "task"
  owners       = [data.azuread_client_config.current.object_id]
}
resource "azuread_application_password" "task" {
  application_object_id = azuread_application.task.object_id
}
resource "azuread_service_principal" "task" {
  application_id               = azuread_application.task.application_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}
resource "azurerm_role_definition" "task" {
  name  = azuread_application.task.display_name
  scope = data.azurerm_subscription.current.id
  permissions {
    actions = [
      "Microsoft.Compute/virtualMachineScaleSets/delete",
      "Microsoft.Compute/virtualMachineScaleSets/delete/action",
      "Microsoft.Compute/virtualMachineScaleSets/instanceView/read",
      "Microsoft.Compute/virtualMachineScaleSets/networkInterfaces/read",
      "Microsoft.Compute/virtualMachineScaleSets/publicIPAddresses/read",
      "Microsoft.Compute/virtualMachineScaleSets/read",
      "Microsoft.Compute/virtualMachineScaleSets/scale/action",
      "Microsoft.Compute/virtualMachineScaleSets/skus/read",
      "Microsoft.Compute/virtualMachineScaleSets/start/action",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/delete",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/networkInterfaces/ipConfigurations/publicIPAddresses/read",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/networkInterfaces/ipConfigurations/read",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/networkInterfaces/read",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/read",
      "Microsoft.Compute/virtualMachineScaleSets/virtualMachines/write",
      "Microsoft.Compute/virtualMachineScaleSets/vmSizes/read",
      "Microsoft.Compute/virtualMachineScaleSets/write",
      "Microsoft.Compute/virtualMachines/delete",
      "Microsoft.Compute/virtualMachines/read",
      "Microsoft.Compute/virtualMachines/write",
      "Microsoft.Network/networkInterfaces/delete",
      "Microsoft.Network/networkInterfaces/join/action",
      "Microsoft.Network/networkInterfaces/read",
      "Microsoft.Network/networkInterfaces/write",
      "Microsoft.Network/networkSecurityGroups/delete",
      "Microsoft.Network/networkSecurityGroups/join/action",
      "Microsoft.Network/networkSecurityGroups/read",
      "Microsoft.Network/networkSecurityGroups/write",
      "Microsoft.Network/publicIPAddresses/delete",
      "Microsoft.Network/publicIPAddresses/join/action",
      "Microsoft.Network/publicIPAddresses/read",
      "Microsoft.Network/publicIPAddresses/write",
      "Microsoft.Network/virtualNetworks/delete",
      "Microsoft.Network/virtualNetworks/read",
      "Microsoft.Network/virtualNetworks/subnets/delete",
      "Microsoft.Network/virtualNetworks/subnets/join/action",
      "Microsoft.Network/virtualNetworks/subnets/read",
      "Microsoft.Network/virtualNetworks/subnets/write",
      "Microsoft.Network/virtualNetworks/write",
      "Microsoft.Resources/subscriptions/resourceGroups/delete",
      "Microsoft.Resources/subscriptions/resourceGroups/read",
      "Microsoft.Resources/subscriptions/resourceGroups/write",
      "Microsoft.Storage/storageAccounts/blobServices/containers/delete",
      "Microsoft.Storage/storageAccounts/blobServices/containers/read",
      "Microsoft.Storage/storageAccounts/blobServices/containers/write",
      "Microsoft.Storage/storageAccounts/delete",
      "Microsoft.Storage/storageAccounts/listKeys/action",
      "Microsoft.Storage/storageAccounts/read",
      "Microsoft.Storage/storageAccounts/write",
    ]
  }
}
resource "azurerm_role_assignment" "task" {
  name               = azurerm_role_definition.task.name
  principal_id       = azuread_service_principal.task.object_id
  role_definition_id = azurerm_role_definition.task.role_definition_resource_id
  scope              = data.azurerm_subscription.current.id
}

output "azure_subscription_id" {
  value = basename(data.azurerm_subscription.current.id)
}
output "azure_tenant_id" {
  value = data.azurerm_subscription.current.tenant_id
}
output "azure_client_id" {
  value = azuread_application.task.application_id
}
output "azure_client_secret" {
  value     = azuread_application_password.task.value
  sensitive = true
}
