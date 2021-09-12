variables {
  azure_build_region       = "eastus"
  azure_build_instance     = "Standard_NC6"
  azure_build_ubuntu_image = "16.04-LTS"
}

variables {
  azure_resource_group_name = "cml-packer-resource-group"
  azure_gallery_name        = "IterativeImages"
}

variables {
  azure_client_id = env("AZURE_CLIENT_ID")
  azure_client_secret = env("AZURE_CLIENT_SECRET")
  azure_subscription_id = env("AZURE_SUBSCRIPTION_ID")
}

locals {
  tags = {
    Env   = var.test ? "Test" : "Prod"
    Owner = "packer"
  }
  release_regions = [
    # "asia",
    # "asiapacific",
    # "australia",
    # "australiacentral",
    # "australiacentral2",
    # "australiaeast",
    # "australiasoutheast",
    # "brazil",
    # "brazilsouth",
    # "brazilsoutheast",
    # "canada",
    # "canadacentral",
    # "canadaeast",
    # "centralindia",
    # "centralus",
    # "centraluseuap",
    # "centralusstage",
    # "eastasia",
    # "eastasiastage",
    "eastus",
    # "eastus2",
    # "eastus2euap",
    # "eastus2stage",
    # "eastusstage",
    # "europe",
    # "france",
    # "francecentral",
    # "francesouth",
    # "germany",
    # "germanynorth",
    # "germanywestcentral",
    # "global",
    # "india",
    # "japan",
    # "japaneast",
    # "japanwest",
    # "jioindiacentral",
    # "jioindiawest",
    # "korea",
    # "koreacentral",
    # "koreasouth",
    # "northcentralus",
    # "northcentralusstage",
    # "northeurope",
    # "norway",
    # "norwayeast",
    # "norwaywest",
    # "southafrica",
    # "southafricanorth",
    # "southafricawest",
    # "southcentralus",
    # "southcentralusstage",
    # "southeastasia",
    # "southeastasiastage",
    # "southindia",
    # "swedencentral",
    # "swedensouth",
    # "switzerland",
    # "switzerlandnorth",
    # "switzerlandwest",
    # "uae",
    # "uaecentral",
    # "uaenorth",
    # "uk",
    # "uksouth",
    # "ukwest",
    # "unitedstates",
    # "westcentralus",
    # "westeurope",
    # "westindia",
    # "westus",
    # "westus2",
    # "westus2stage",
    # "westus3",
    # "westusstage"
  ]
}

source "azure-arm" "source" {
  client_id = var.azure_client_id
  client_secret = var.azure_client_secret
  subscription_id = var.azure_subscription_id

  managed_image_name                = var.test ? var.image_name : "${var.image_name}-test"
  managed_image_resource_group_name = var.azure_resource_group_name

  location = var.azure_build_region
  vm_size  = var.azure_build_instance

  os_type         = "Linux"
  image_offer     = "UbuntuServer"
  image_publisher = "Canonical"
  image_sku       = var.azure_build_ubuntu_image

  azure_tags = local.tags

   shared_image_gallery_destination {
     resource_group       = var.azure_resource_group_name
     gallery_name         = var.azure_gallery_name
     image_name           = var.test ? var.image_name : "${var.image_name}-test"
     image_version        = "latest"
     replication_regions  = local.release_regions
     storage_account_type = "Standard_LRS"
   }
}
