terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "=2.76.0"
    }
  }
}

provider "azurerm" {
  #client_id= env("AZURE_CLIENT_ID")
  #client_secret = env("AZURE_CLIENT_SECRET")
  #subscription_id = env("AZURE_SUBSCRIPTION_ID")
  #tenant_id = env("AZURE_TENANT_ID")

  features {}
}
