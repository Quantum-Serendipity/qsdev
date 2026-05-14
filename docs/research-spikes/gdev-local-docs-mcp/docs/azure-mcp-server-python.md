<!-- Source: https://learn.microsoft.com/en-us/azure/developer/azure-mcp-server/get-started/languages/python -->
<!-- Retrieved: 2026-05-14 -->

# Azure MCP Server with Python

The Azure MCP Server uses the Model Context Protocol (MCP) to standardize integrations between AI apps and external tools and data sources, allowing AI systems to perform operations that are context-aware of Azure resources.

## Authentication

Azure MCP Server authenticates to Microsoft Entra ID using the Azure Identity library for .NET. Two modes:

1. **Broker mode**: Uses OS native authentication (like Windows WAM) with InteractiveBrowserCredential
2. **Credential chain mode**: Tries multiple methods in sequence: environment variables, VS Code, Visual Studio, Azure CLI, Azure PowerShell, Azure Developer CLI, and interactive browser

Supported sign-in methods: VS Code Azure extension, Visual Studio, `az login`, `Connect-AzAccount`, `azd auth login`

## Architecture

- MCP client connects to Azure MCP Server as a local process running MCP protocol (stdio transport)
- Server started via: `npx -y @azure/mcp@latest server start`
- Azure resources must already exist in subscription
- User must have necessary RBAC roles assigned

## Python Integration Pattern

```python
from azure.identity import DefaultAzureCredential, get_bearer_token_provider
from openai import AzureOpenAI
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

# Initialize Azure credentials
token_provider = get_bearer_token_provider(
    DefaultAzureCredential(), "https://cognitiveservices.azure.com/.default"
)

# MCP client configurations
server_params = StdioServerParameters(
    command="npx",
    args=["-y", "@azure/mcp@latest", "server", "start"],
    env=None
)
```

## Key Takeaway

This demonstrates that Azure MCP Server + DefaultAzureCredential + az login is a proven, documented pattern for MCP servers accessing Azure resources. The same pattern can be adapted for our documentation MCP servers accessing Azure Storage.
