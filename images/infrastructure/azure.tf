resource "azurerm_resource_group" "group" {
  name     = "cml-packer-resource-group"
  location = "eastus"
}

resource "azurerm_shared_image_gallery" "gallery" {
  name                = "IterativeImages"
  resource_group_name = azurerm_resource_group.group.name
  location            = azurerm_resource_group.group.location
  description         = "CML machine images"

  tags = {
    Env   = "Prod"
    Owner = "packer"
  }
}
