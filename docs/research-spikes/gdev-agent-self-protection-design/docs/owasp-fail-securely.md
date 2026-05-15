# OWASP Fail Securely

- **Source URL**: https://owasp.org/www-community/Fail_securely
- **Retrieved**: 2026-05-15

## Core Concept

The page defines secure error handling as essential to secure coding, emphasizing that security mechanisms must fail safely without inadvertently enabling unauthorized behavior.

## Key Principle

Security controls should handle three possible outcomes:
- Allow the operation
- Disallow the operation
- Exception

"a failure will follow the same execution path as disallowing the operation" is the recommended design pattern. Methods like `isAuthorized()` and `authenticate()` should return false when exceptions occur during processing.

## Two Critical Error Types

1. **Exceptions within security controls** - Must not bypass security mechanisms
2. **Exceptions in non-security code** - Can indirectly compromise security by preventing proper control invocation or corrupting initialization variables

## Code Example Analysis

The problematic pattern initializes `isAdmin = true`, then attempts role verification. If an exception occurs before the role check completes, the user remains authenticated as admin -- a serious vulnerability.

The corrected approach reverses this: "isAdmin = false" ensures that any exception defaults to denying access, implementing the least privilege principle by never granting more permissions than necessary.

## Practical Impact

This demonstrates that default-deny approaches prevent security controls from being bypassed by runtime failures, making exception handling architecture critical to application security.
