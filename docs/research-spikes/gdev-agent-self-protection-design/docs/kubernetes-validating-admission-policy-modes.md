<!-- Source: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/ -->
<!-- Retrieved: 2026-05-15 -->

# Kubernetes ValidatingAdmissionPolicy Validation Actions

## Overview

Each ValidatingAdmissionPolicyBinding must specify one or more validationActions to declare how validation failures are enforced.

## Validation Actions

| Action | Behavior |
|--------|----------|
| **Deny** | Validation failure results in a denied request (request is rejected) |
| **Warn** | Validation failure is reported to the request client as a warning |
| **Audit** | Validation failure is included in the audit event for the API request |

## Important Constraints

- **Deny and Warn cannot be used together** — this combination needlessly duplicates the validation failure both in the API response body and HTTP warning headers
- A validation that evaluates to false is always enforced according to these actions
- Failures defined by failurePolicy are enforced according to these actions only if failurePolicy is set to Fail (or not specified)

## Configuration Examples

### Warn Mode

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "demo-binding-warn.example.com"
spec:
  policyName: "demo-policy.example.com"
  validationActions: [Warn]
  matchResources:
    namespaceSelector:
      matchLabels:
        environment: test
```

User experience: Client receives a warning in the response but the request succeeds.

### Audit Mode

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "demo-binding-audit.example.com"
spec:
  policyName: "demo-policy.example.com"
  validationActions: [Audit]
  matchResources:
    namespaceSelector:
      matchLabels:
        environment: test
```

User experience: Request succeeds, failure recorded in audit event for later review.

### Combined Warn and Audit

```yaml
validationActions: [Warn, Audit]
```

Captures both immediate feedback to the user and persistent audit records.

### Per-Scope Configuration

Same policy, different bindings = different actions for different resource scopes:

```yaml
# Binding 1: Deny in production
spec:
  policyName: "demo-policy.example.com"
  validationActions: [Deny]
  matchResources:
    namespaceSelector:
      matchLabels:
        environment: production
---
# Binding 2: Warn in staging
spec:
  policyName: "demo-policy.example.com"
  validationActions: [Warn, Audit]
  matchResources:
    namespaceSelector:
      matchLabels:
        environment: staging
```

## Failure Policy Interaction

The failurePolicy field controls what happens when the policy itself fails (not when validation fails):

- If failurePolicy: Fail (default), failures are enforced according to validationActions
- If failurePolicy: Ignore, policy failures are ignored regardless of validationActions
