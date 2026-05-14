<!-- Source: https://learn.microsoft.com/en-us/azure/container-apps/authentication-entra -->
<!-- Retrieved: 2026-05-14 -->

# Azure Container Apps Entra ID Authentication

## Overview
Container Apps authentication can automatically create an Entra ID app registration. Also supports using an existing registration.

## Option 1: Automatic Registration
1. Go to Container App > Security > Authentication > Add identity provider
2. Select Microsoft
3. A client secret is created and stored as a Container Apps secret
4. Configure authentication settings (redirect unauthenticated requests to sign in)

## Option 2: Existing Registration
Required settings:
- Application (client) ID
- Client secret (optional - without it, implicit flow is used)
- Issuer URL: `https://login.microsoftonline.com/<TENANT-ID>/v2.0`
- Allowed token audiences

## Daemon/Service-to-Service Authentication
For non-interactive apps (like MCP servers accessing on behalf of themselves):
1. Register a daemon app in Entra ID (no redirect URI needed)
2. Create client secret
3. Request access token using client credentials flow
4. Set resource parameter to Application ID URI of target app
5. Present token via standard OAuth 2.0 Authorization header

## Key Takeaway for gdev
Container Apps with built-in Entra ID authentication provides a turnkey solution for protecting kiwix-serve or DevDocs behind SSO. The daemon client pattern enables MCP servers to authenticate programmatically using service principals.
