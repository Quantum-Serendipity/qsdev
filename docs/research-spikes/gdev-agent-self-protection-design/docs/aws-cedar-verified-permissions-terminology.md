# Amazon Verified Permissions and Cedar Policy Language: Authorization Model

- **Source URL**: https://docs.aws.amazon.com/verifiedpermissions/latest/userguide/terminology.html
- **Retrieved**: 2026-05-15

## Cedar Authorization Decision Algorithm

Cedar uses a **forbid-overrides-permit** model:

1. If any `forbid` policy matches the request, the decision is **DENY**
2. If no `forbid` policy matches but at least one `permit` policy matches, the decision is **ALLOW**
3. If neither forbid nor permit policies match, the decision is **DENY** (default-deny)

This is order-independent: policies can be evaluated in any order and reach the same decision.

## Key Concepts

### Determining Policies
The policies that determine the authorization response. If two satisfied policies exist (one deny, one allow), the deny policy is the determining policy. If multiple satisfied permit policies exist with no forbid policies, all are determining policies. If no policies match, the default-deny has no determining policies.

### Satisfied Policies
Policies that match the parameters of the authorization request.

### Considered Policies
The full set of policies selected for inclusion when evaluating a request. Relevance is determined by policy scope (principal, resource, action).

## Authorization Model

Cedar supports both RBAC (via principal groups) and ABAC (via policy conditions on attributes), and can combine them in a single policy.

## gdev Relevance

Cedar's forbid-overrides-permit model maps directly to gdev's verdict system:
- Cedar `forbid` = gdev `deny`
- Cedar `permit` = gdev `allow`
- Cedar default (no match) = gdev `deny` (fail-closed)
- gdev adds `ask` between permit and forbid, which Cedar does not have

Cedar's order-independence is a design goal gdev should adopt: the verdict should not depend on which hook happens to run first.
