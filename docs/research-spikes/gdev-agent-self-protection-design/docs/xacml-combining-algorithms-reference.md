# XACML Policy Combining Algorithms Reference

- **Source**: Web search synthesis from Axiomatics, WSO2, and OASIS XACML documentation
- **Retrieved**: 2026-05-15

## Combining Algorithms

### Deny-Overrides
Combines decisions such that if any decision is Deny, Deny wins. One of the safest combining algorithms since it favors denial. If no children return Deny, the algorithm will never produce Deny.

### Permit-Overrides
Combines decisions such that if any decision is Permit, Permit wins. Intended for cases where a permit decision should have priority over deny.

### First-Applicable
Returns the first produced decision of Permit or Deny. Useful to shortcut policy evaluation. Requires ordered evaluation.

### Only-One-Applicable
Returns the decision only if exactly one policy applies. Returns Indeterminate if more than one policy is applicable (prevents ambiguous outcomes).

### Ordered Variants
Ordered-deny-overrides and ordered-permit-overrides bring the guarantee that policies are considered in definition order. The XACML specification does not mandate ordering in the unordered variants.

## Decision Values

XACML defines four possible outcomes:
1. **Permit** -- access granted
2. **Deny** -- access denied
3. **NotApplicable** -- no policy matches the request
4. **Indeterminate** -- an error occurred during evaluation

## gdev Mapping

| XACML | gdev Equivalent |
|-------|----------------|
| Permit | allow |
| Deny | deny |
| NotApplicable | allow (default for non-security) or deny (default for security) |
| Indeterminate | depends on failure policy tier (fail-closed for security, fail-open for advisory) |

gdev's `ask` verdict has no XACML equivalent -- it is a human-in-the-loop escalation that XACML's machine-to-machine model does not contemplate. The closest concept is XACML's "obligations" -- side-effects that must be fulfilled alongside a Permit decision, which could include requiring additional confirmation.

## Recommended Algorithm for gdev

**Deny-overrides** is the correct combining algorithm for gdev's self-protection rules:
- Any deny from any rule wins (most conservative)
- Multiple allow verdicts are collapsed to a single allow
- Ask escalates to the user only when no rule denies
- Order-independent (hooks run in parallel)
