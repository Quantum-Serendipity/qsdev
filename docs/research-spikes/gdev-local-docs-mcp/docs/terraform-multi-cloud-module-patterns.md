# How to Create Terraform Modules for Multi-Cloud
- **Source**: https://oneuptime.com/blog/post/2026-02-23-how-to-create-terraform-modules-for-multi-cloud/view
- **Retrieved**: 2026-05-14

## Module Structure Pattern

The recommended architecture organizes cloud-specific implementations under a common interface:

```
modules/compute/
  ├── interface/          # Shared definitions
  ├── aws/               # AWS implementation
  ├── azure/             # Azure implementation
  └── gcp/               # GCP implementation
```

This separation allows each cloud provider to have its own resource definitions while maintaining consistent input/output contracts.

## Common Interface Definition

Define shared variables that all cloud implementations accept:
- name: Instance identifier
- size: Abstract sizing (small, medium, large, xlarge) with validation
- image: OS image reference
- network_id: Network/VPC identifier
- subnet_id: Subnet reference
- tags: Resource labeling

Standardized outputs include instance_id, private_ip, and public_ip across all implementations.

## Provider-Specific Implementation

Each cloud provider maps abstract inputs to native resource types:
- AWS uses instance type mapping (t3.small → t3.small)
- Azure maps to VM sizes (Standard_B1ms)
- GCP uses machine types (e2-small)

Each implementation defines its own variables for cloud-specific requirements (resource groups, zones, locations).

## Conditional Resource Creation

The wrapper module pattern uses count to activate only the relevant cloud implementation:

```hcl
module "aws" {
  count = var.cloud_provider == "aws" ? 1 : 0
  # configuration
}
```

Outputs consolidate results using coalesce() to return values from whichever module is active.

## Key Principles

Do not abstract everything — focus on infrastructure primitives (compute, networking, storage). Avoid abstracting managed services without cloud equivalents, IAM mechanisms, or cloud-specific optimizations.

Test each cloud separately with dedicated test configurations rather than attempting unified testing across providers.
