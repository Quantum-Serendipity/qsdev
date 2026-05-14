<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/admin/security/secrets.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Secrets Management Overview

## Core Principles

Coder supports flexible secret management approaches. The documentation emphasizes starting with local workflows: users receive secrets in advance and write them to persistent files after workspace creation.

## SSH Keys

Coder automatically generates SSH key pairs for each user, enabling authentication with git providers and other tools. A key security feature is that "SSH keys are never stored in Coder workspaces, and are fetched only when SSH is invoked. The keys are held in-memory and never written to disk." Users can access their public keys through account settings.

## Dynamic Secrets

Dynamic secrets automatically inject into workspaces during their lifecycle. These are provisioned through Terraform providers and attached to workspace resources. Example implementation involves creating API keys (like Twilio credentials) and exposing them via environment variables:

```
TWILIO_API_SECRET = "${twilio_iam_api_key.api_key.secret}"
```

An alternative pattern involves provisioning cloud service accounts (GCP, AWS) per workspace and accessing secrets through the cloud provider's native secret management system.

## Secret Display in UI

Administrators can surface secrets in the Workspace UI using `coder_metadata` resources with sensitive value flagging to control visibility.

## Advanced Integration

For sophisticated scenarios, external tools like HashiCorp Vault integrate with Coder to store and retrieve secrets within workspaces.

## Security Warning

The documentation strongly warns against using template parameters for secrets, noting that parameters display in cleartext and remain accessible to anyone with workspace view permissions.
