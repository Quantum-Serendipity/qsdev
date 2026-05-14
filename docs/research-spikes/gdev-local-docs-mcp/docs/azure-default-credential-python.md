<!-- Source: https://learn.microsoft.com/en-us/python/api/azure-identity/azure.identity.defaultazurecredential?view=azure-python -->
<!-- Retrieved: 2026-05-14 -->

# DefaultAzureCredential (Python)

A credential capable of handling most Azure SDK authentication scenarios.

## Credential Chain Order

When an access token is needed, it requests one using these identities in turn, stopping when one provides a token:

1. **EnvironmentCredential** — Service principal configured by environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID)
2. **WorkloadIdentityCredential** — If environment variable configuration is set by the Azure workload identity webhook
3. **ManagedIdentityCredential** — Azure managed identity (system or user-assigned)
4. **SharedTokenCacheCredential** — (Windows only) User signed in with a Microsoft application like Visual Studio
5. **VisualStudioCodeCredential** — Identity logged in to VS Code with Azure Resources extension
6. **AzureCliCredential** — Identity currently logged in to the Azure CLI (`az login`)
7. **AzurePowerShellCredential** — Identity currently logged in to Azure PowerShell
8. **AzureDeveloperCliCredential** — Identity currently logged in to the Azure Developer CLI
9. **BrokerCredential** — (Windows and WSL only) Default account logged in via Web Account Manager (WAM)

## Key Constructor Parameters

- `authority` — Entra endpoint (default: 'login.microsoftonline.com')
- `managed_identity_client_id` — Client ID for user-assigned managed identity
- `exclude_cli_credential` — Disable Azure CLI credential (default: False)
- `exclude_managed_identity_credential` — Disable managed identity (default: False)
- `exclude_interactive_browser_credential` — Disable interactive browser (default: True)
- `process_timeout` — Timeout for subprocess-based credentials like AzureCLI (default: 10s)

## Token Behavior

- Tokens are requested with specific scopes (e.g., `https://storage.azure.com/.default`)
- Token caching handled by underlying credential implementations
- For Azure CLI credential, tokens are obtained by running `az account get-access-token` subprocess
- Token refresh is automatic — DefaultAzureCredential re-requests when tokens expire

## Usage Example

```python
from azure.identity import DefaultAzureCredential
credential = DefaultAzureCredential()
# Automatically picks up az login token on developer machines
# Automatically uses managed identity in Azure
```

## Important Notes

- Starting with Azure Identity library version 1.11.0, credentials are restricted from acquiring tokens for tenants other than the one they're configured for, unless explicitly permitted
- The `process_timeout` for CLI credentials defaults to 10 seconds
- On Linux (non-WSL), the chain effectively tries: EnvironmentCredential → WorkloadIdentity → ManagedIdentity → VS Code → Azure CLI → Azure PowerShell → Azure Developer CLI
