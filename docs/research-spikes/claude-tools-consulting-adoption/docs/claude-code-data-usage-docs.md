# Claude Code Data Usage Documentation
- **Source**: https://code.claude.com/docs/en/data-usage
- **Retrieved**: 2026-03-27

## Data Training Policy

**Consumer users (Free, Pro, and Max plans)**:
We give you the choice to allow your data to be used to improve future Claude models. We will train new models using data from Free, Pro, and Max accounts when this setting is on (including when you use Claude Code from these accounts).

**Commercial users**: (Team and Enterprise plans, API, 3rd-party platforms, and Claude Gov) maintain existing policies: Anthropic does not train generative models using code or prompts sent to Claude Code under commercial terms, unless the customer has chosen to provide their data to us for model improvement (for example, the Developer Partner Program).

### Development Partner Program

If you explicitly opt in to methods to provide us with materials to train on, such as via the Development Partner Program, we may use those materials provided to train our models. An organization admin can expressly opt-in to the Development Partner Program for their organization. Note that this program is available only for Anthropic first-party API, and not for Bedrock or Vertex users.

### Feedback using the /feedback command

If you choose to send us feedback about Claude Code using the /feedback command, we may use your feedback to improve our products and services. Transcripts shared via /feedback are retained for 5 years.

### Session quality surveys

When you see the "How is Claude doing this session?" prompt, responding only records your numeric rating (1, 2, 3, or dismiss). No conversation transcripts, inputs, outputs, or other session data collected as part of this survey. Disable with CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1.

## Data Retention

**Consumer users (Free, Pro, and Max plans)**:
- Users who allow data use for model improvement: 5-year retention period
- Users who don't allow data use for model improvement: 30-day retention period
- Privacy settings can be changed at any time at claude.ai/settings/data-privacy-controls

**Commercial users (Team, Enterprise, and API)**:
- Standard: 30-day retention period
- Zero data retention: available for Claude Code on Claude for Enterprise. ZDR is enabled on a per-organization basis; each new organization must have ZDR enabled separately by your account team
- Local caching: Claude Code clients may store sessions locally for up to 30 days to enable session resumption (configurable)

## Data Flow — Local Claude Code

Claude Code is installed from NPM. Claude Code runs locally. In order to interact with the LLM, Claude Code sends data over the network. This data includes all user prompts and model outputs. The data is encrypted in transit via TLS and is not encrypted at rest. Claude Code is compatible with most popular VPNs and LLM proxies.

## Cloud Execution Data Flow

When using Claude Code on the web, sessions run in Anthropic-managed virtual machines:
- Code and data storage: Repository cloned to isolated VM, subject to retention/usage policies for account type
- Credentials: GitHub auth through secure proxy; credentials never enter sandbox
- Network traffic: All outbound through security proxy for audit logging
- Session data: Follows same data policies as local usage

## Telemetry Services

**Statsig**: Operational metrics (latency, reliability, usage patterns). Does NOT include code or file paths. Opt out: DISABLE_TELEMETRY=1

**Sentry**: Operational error logging. Opt out: DISABLE_ERROR_REPORTING=1

**/feedback command**: Sends FULL conversation history including code to Anthropic. Retained 5 years. Opt out: DISABLE_FEEDBACK_COMMAND=1

## Default Behaviors by API Provider

| Service | Claude API | Vertex API | Bedrock API | Foundry API |
|---------|-----------|------------|-------------|-------------|
| Statsig (Metrics) | Default ON | Default OFF | Default OFF | Default OFF |
| Sentry (Errors) | Default ON | Default OFF | Default OFF | Default OFF |
| /feedback reports | Default ON | Default OFF | Default OFF | Default OFF |
| Session quality surveys | Default ON | Default ON | Default ON | Default ON |

All can be disabled via environment variables. Bedrock/Vertex/Foundry default to OFF for telemetry.

Key env var: CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC disables all non-essential traffic including surveys.
