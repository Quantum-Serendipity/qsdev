<!-- Source: https://developer.hashicorp.com/terraform/mcp-server -->
<!-- Retrieved: 2026-05-14 -->

# Terraform MCP Server Overview (HashiCorp Developer)

## Official Status
Official HashiCorp product, currently in beta status. "Do not use beta functionality in production environments."

## Key Tools & Capabilities

The server provides AI models with access to:

- **Provider Documentation**: Search and retrieve current provider documentation
- **Module Information**: Access inputs, outputs, and examples
- **Sentinel Policies**: Find governance and compliance policies
- **Workspace Management**: List organizations, projects, and workspaces
- **Workspace Operations**: Create, update, delete workspaces with support for variables, tags, and run management

## How It Works

The MCP server integrates with AI models to provide real-time access to current Terraform provider documentation, modules, and policies from the Terraform registry rather than relying on potentially outdated training data.

## Architecture Components

- AI model and MCP host (e.g., Claude Desktop)
- MCP client (discovers tools and translates prompts)
- MCP server (executes tools via JSON-RPC 2.0)
- Transport methods: stdio pipes or HTTP endpoints

## Security & Access

References HCP Terraform & Terraform Enterprise Support with private registry access. No specific details on credentials or authentication configuration in the overview.

## Limitations

- Beta software unsuitable for production use
- Only exposes Terraform registry and Terraform Cloud metadata
- Cannot list deployed resources, inspect configurations, or identify unmanaged infrastructure
