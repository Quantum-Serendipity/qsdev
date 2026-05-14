# Module Composition — Terraform HashiCorp Developer Docs
- **Source**: https://developer.hashicorp.com/terraform/language/modules/develop/composition
- **Retrieved**: 2026-05-14

## Core Philosophy

Terraform deliberately avoids abstracting over similar services from different vendors. Instead, you can compose modules to create your own lightweight multi-cloud abstractions by making strategic tradeoffs about which platform features matter for your use case.

## Creating Abstraction Layers

The recommended approach uses object types to define common concepts across vendors. For example, a DNS recordset abstraction might look like:

```hcl
variable "recordsets" {
  type = list(object({
    name    = string
    type    = string
    ttl     = number
    records = list(string)
  }))
}
```

This allows you to define Terraform object types representing the concepts involved and then use these object types for module input variables.

## Provider-Specific Implementations

Keep implementations vendor-specific while exposing a common interface. For instance, you could have separate modules for Route53 and other DNS providers, all accepting the same recordsets input variable. This means all of your implementations would have the same variable declared identically, enabling easy swaps.

## Dependency Inversion Pattern

Rather than embedding dependencies, modules should receive them from the root module. This approach:
- Keeps modules small and focused
- Improves flexibility for future refactoring
- Allows the calling module to control how dependencies are sourced

## When to Split vs. Compose

Prefer flat module hierarchies with only one level of child modules. Assemble them to produce a larger system using expressions that describe relationships between modules rather than nesting deeply.

## Cross-Platform Abstraction Example

The documentation provides Kubernetes as a complex example where multiple vendors offer implementations. You could design modules where each implementation exports a common output (e.g., hostname), then other modules expect only that common output and use them interchangeably.
