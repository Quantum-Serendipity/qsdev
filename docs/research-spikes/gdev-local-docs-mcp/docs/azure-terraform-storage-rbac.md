<!-- Source: https://codewithme.cloud/posts/2023/09/improve-storage-account-security-with-azure-rbac-and-terraform/ -->
<!-- Retrieved: 2026-05-14 -->

# Azure Storage Account Security with RBAC and Terraform

## Key Configuration: Disable Shared Key Access

```hcl
resource "azurerm_storage_account" "example" {
  name                      = "storageaccountname"
  resource_group_name       = "rgname"
  location                  = "norwayeast"
  account_tier              = "Standard"
  account_replication_type  = "GRS"
  shared_access_key_enabled = false
}
```

## Provider Configuration for Entra ID Auth

```hcl
provider "azurerm" {
  features {}
  use_oidc            = true
  subscription_id     = "00000000-0000-0000-0000-000000000000"
  tenant_id           = "00000000-0000-0000-0000-000000000000"
  storage_use_azuread = true
}
```

## Backend Configuration with Entra ID

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "abcd1234"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"
    use_azuread_auth     = true
    subscription_id      = "00000000-0000-0000-0000-000000000000"
    tenant_id            = "00000000-0000-0000-0000-000000000000"
    use_oidc             = true
  }
}
```

## Key Takeaways

- Set `shared_access_key_enabled = false` to force Entra ID authentication
- Set `storage_use_azuread = true` in the provider to use Entra ID for Terraform operations
- Use RBAC roles (Storage Blob Data Reader, Storage Blob Data Contributor) instead of access keys
- Private endpoints disable data plane access unless you have network line-of-sight via private network
