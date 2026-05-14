# Claude Code Security Documentation
- **Source**: https://code.claude.com/docs/en/security
- **Retrieved**: 2026-03-27

## How we approach security

### Security foundation

Your code's security is paramount. Claude Code is built with security at its core, developed according to Anthropic's comprehensive security program. Learn more and access resources (SOC 2 Type 2 report, ISO 27001 certificate, etc.) at Anthropic Trust Center (https://trust.anthropic.com).

### Permission-based architecture

Claude Code uses strict read-only permissions by default. When additional actions are needed (editing files, running tests, executing commands), Claude Code requests explicit permission. Users control whether to approve actions once or allow them automatically.

### Built-in protections

To mitigate risks in agentic systems:

* **Sandboxed bash tool**: Sandbox bash commands with filesystem and network isolation, reducing permission prompts while maintaining security.
* **Write access restriction**: Claude Code can only write to the folder where it was started and its subfolders — it cannot modify files in parent directories without explicit permission. While Claude Code can read files outside the working directory (useful for accessing system libraries and dependencies), write operations are strictly confined to the project scope, creating a clear security boundary.
* **Prompt fatigue mitigation**: Support for allowlisting frequently used safe commands per-user, per-codebase, or per-organization.
* **Accept Edits mode**: Batch accept multiple edits while maintaining permission prompts for commands with side effects.

### User responsibility

Claude Code only has the permissions you grant it. You're responsible for reviewing proposed code and commands for safety before approval.

## Privacy safeguards

We have implemented several safeguards to protect your data, including:

* Limited retention periods for sensitive information
* Restricted access to user session data
* User control over data training preferences. Consumer users can change their privacy settings at any time.

For full details, please review the Commercial Terms of Service (for Team, Enterprise, and API users) or Consumer Terms (for Free, Pro, and Max users) and Privacy Policy.

### Key privacy distinctions by plan:

- **Enterprise plans**: Do not train Claude on your Enterprise data — your code, conversations, and proprietary information remain private and are not used to improve models.
- **Consumer tiers**: Require users to manually opt out of training data collection and lack the zero data retention guarantees that enterprise plans provide.
- **Zero-Data-Retention (ZDR)**: Optional addendum for organizations handling regulated or sensitive data that eliminates stored records entirely.

## Cloud execution security

When using Claude Code on the web, additional security controls are in place:

* **Isolated virtual machines**: Each cloud session runs in an isolated, Anthropic-managed VM
* **Network access controls**: Network access is limited by default and can be configured to be disabled or allow only specific domains
* **Credential protection**: Authentication is handled through a secure proxy
* **Branch restrictions**: Git push operations are restricted to the current working branch
* **Audit logging**: All operations in cloud environments are logged for compliance and audit purposes
* **Automatic cleanup**: Cloud environments are automatically terminated after session completion

## Security best practices

### Working with sensitive code

* Review all suggested changes before approval
* Use project-specific permission settings for sensitive repositories
* Consider using devcontainers for additional isolation
* Regularly audit your permission settings

### Team security

* Use managed settings to enforce organizational standards
* Share approved permission configurations through version control
* Train team members on security best practices
* Monitor Claude Code usage through OpenTelemetry metrics
* Audit or block settings changes during sessions with ConfigChange hooks

## Compliance

Anthropic maintains SOC 2 Type 2 and ISO 27001 certifications. Resources available at Anthropic Trust Center (https://trust.anthropic.com).
