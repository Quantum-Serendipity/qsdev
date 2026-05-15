# Consulting-Specific Lifecycle Needs

## Research Question

What consulting-specific lifecycle capabilities does gdev need? How should it handle project teardown, archival, client-specific profiles, and compliance evidence generation?

## The Consulting Context

A consulting firm's relationship with developer tooling is fundamentally different from a product company:

- **Frequent project rotation:** Engineers move between client projects every 3-12 months
- **Client-specific constraints:** Different clients have different security requirements, registry proxies, compliance standards
- **Clean separation required:** Client A's code, credentials, and tools must not leak to Client B
- **Re-engagement possible:** A project "ended" 6 months ago may restart; the environment must be reproducible
- **Compliance evidence:** Clients may require proof that security controls were in place during development
- **Intellectual property boundaries:** Consulting firm's internal tools vs client's proprietary code

## Feature 1: Client-Specific Profiles

### Design

The `.qsdev.yaml` `client` block encodes client-specific configuration:

```yaml
# .qsdev.yaml
client:
  name: acme-corp
  compliance:
    - soc2
    - hipaa
  registry_proxy: https://nexus.acme-corp.internal
  nix_cache: https://cache.acme-corp.internal
  security_level: strict    # baseline | enhanced | strict
  allowed_mcp_servers:
    - github
    - context7
  blocked_mcp_servers:
    - "*"                   # deny-by-default for HIPAA client
  data_classification: confidential
```

### Compliance Level Mapping

Different compliance requirements map to different security configurations:

| Setting | Baseline | Enhanced (SOC2) | Strict (HIPAA/FedRAMP) |
|---------|----------|-----------------|----------------------|
| Age-gating threshold | 72 hours | 168 hours (1 week) | 336 hours (2 weeks) |
| Install script blocking | Enabled | Enabled | Enabled + audit log |
| Vulnerability scanning | osv-scanner | osv-scanner + Semgrep | osv-scanner + Semgrep + daily |
| MCP servers | allow-list | allow-list | explicit allow-list only |
| Pre-commit hooks | ripsecrets, gitleaks | + semgrep, conventional commits | + license scanning |
| Claude Code permissions | standard | restricted | restricted + audit log |
| SBOM generation | off | on release | every build |

### Profile Inheritance

Client profiles compose with language/project profiles:

```
Org Defaults (compiled)
  -> Client Profile (from .qsdev.yaml client section)
    -> Project Profile (from .qsdev.yaml profile field)
      -> Project Overrides (from .qsdev.yaml overrides section)
        -> Local Overrides (.qsdev.local.yaml)
```

The client profile applies security constraints that cannot be loosened by lower layers. If the client requires `security_level: strict`, no project-level override can set it to `baseline`.

### Implementation: Security Level as a Floor

```go
type SecurityLevel int
const (
    SecurityBaseline SecurityLevel = iota
    SecurityEnhanced
    SecurityStrict
)

func mergeConfig(org, client, project, local Config) Config {
    result := deepMerge(org, client, project, local)
    
    // Client security level is a floor -- never lowered by project/local
    if client.SecurityLevel > result.SecurityLevel {
        result.SecurityLevel = client.SecurityLevel
    }
    
    // Client blocked_mcp_servers cannot be unblocked
    result.BlockedMCPServers = union(
        client.BlockedMCPServers,
        result.BlockedMCPServers,
    )
    
    return result
}
```

## Feature 2: Project Teardown (`qsdev teardown`)

When an engineer leaves a client project, a clean teardown ensures no client data lingers.

### Command Design

```
$ gdev teardown
  Teardown: acme-widget-service (client: acme-corp)
  
  This will:
  1. Archive project configuration (to ~/.gdev/archives/)
  2. Remove devenv environment (Nix store GC for this project)
  3. Revoke MCP server tokens configured for this project
  4. Remove project from trusted paths
  5. Clear shell history entries containing project paths (optional)
  6. Generate teardown evidence report (optional)
  
  ⚠ This does NOT delete the git repository. Do that separately.
  
  Proceed? [y/N] y
  
  ✓ Configuration archived to ~/.gdev/archives/acme-widget-service-2026-05-12.tar.gz
  ✓ devenv environment removed (freed 2.3 GB)
  ✓ MCP tokens revoked (github, context7)
  ✓ Project removed from trusted paths
  ✓ Teardown evidence saved to ~/.gdev/archives/acme-widget-service-2026-05-12-teardown.json
  
  Teardown complete. Remember to:
  - Delete the git repository when ready
  - Revoke any personal access tokens created for this project
  - Remove any client VPN configurations
```

### Teardown Checklist

| Step | Action | Reversible? | Required |
|------|--------|-------------|----------|
| Archive config | Copy .qsdev.yaml + .qsdev.local.yaml to archive | Yes | Yes |
| Archive state | Copy internal GeneratedState to archive | Yes | Yes |
| Nix GC | `nix-collect-garbage` for project-specific derivations | No (re-downloadable) | Optional |
| Revoke MCP tokens | Delete tokens from local keyring/config | No | Yes |
| Remove trust | Remove project from trusted_config_paths | Yes | Yes |
| Clear direnv | `direnv deny` for project .envrc | Yes | Yes |
| Generate evidence | JSON report of teardown actions + timestamps | N/A | Optional |

### What Teardown Does NOT Do

- **Delete the git repository** -- that's the engineer's decision (they may need to reference old code)
- **Revoke cloud credentials** -- gdev doesn't manage cloud auth
- **Delete Nix store entries shared with other projects** -- Nix's store deduplication means cleaning up one project's derivations could affect another
- **Clear browser state** -- out of scope

## Feature 3: Project Archival

When a client engagement ends, preserve enough state to re-engage months or years later.

### Archive Contents

```
~/.gdev/archives/acme-widget-service-2026-05-12.tar.gz
├── .qsdev.yaml                    # Project configuration
├── .qsdev.local.yaml              # Local overrides (if not sensitive)
├── generated-state.yaml          # Internal state (hashes, versions)
├── gdev-check-report.json        # Last compliance check
├── tool-versions.txt             # Exact versions of all tools at archival time
├── devenv-lock.json              # devenv.lock (pinned Nix inputs)
└── metadata.json                 # Archive metadata
```

`metadata.json`:
```json
{
  "project": "acme-widget-service",
  "client": "acme-corp",
  "archived": "2026-05-12T16:00:00Z",
  "gdev_version": "0.16.0",
  "profile": "go-web-service",
  "compliance": ["soc2"],
  "archival_reason": "engagement-end",
  "git_commit": "abc123def456",
  "git_remote": "https://github.com/acme-corp/widget-service.git"
}
```

### Re-Engagement Workflow

When re-engaging with an archived project:

```
$ git clone <project-url>
$ cd <project>
$ qsdev init
  
  Found archive for this project: acme-widget-service (archived 2026-05-12)
  
  The archive was created with gdev v0.16.0 (current: v0.18.0).
  Config version migration needed: v2 -> v3.
  
  Options:
  1. Restore from archive and migrate (recommended)
  2. Start fresh with current defaults
  3. View archive contents first
  
  Choice: 1
  
  Migrating config v2 -> v3... done
  Restoring local overrides... done
  
  ⚠ Some archived tools may have newer versions available.
  Run `qsdev init --update` after setup to check for updates.
```

## Feature 4: Compliance Evidence Generation

For clients requiring compliance documentation, gdev generates evidence that security controls were in place.

### `qsdev evidence` Command

```
$ gdev evidence --format json --output compliance-evidence.json
  
  Generating compliance evidence for: acme-widget-service
  Client: acme-corp | Compliance: SOC2, HIPAA
  
  Evidence collected:
  ✓ Security configuration snapshot
  ✓ Pre-commit hook verification
  ✓ Dependency scanning results
  ✓ Tool version inventory
  ✓ qsdev check results
  ✓ Generated file integrity verification
```

### Evidence Report Structure

```json
{
  "report_version": 1,
  "generated": "2026-05-12T16:00:00Z",
  "project": {
    "name": "acme-widget-service",
    "client": "acme-corp",
    "compliance_standards": ["soc2", "hipaa"]
  },
  "tool_chain": {
    "gdev_version": "0.16.0",
    "devenv_version": "2.1.0",
    "nix_version": "2.28.0",
    "pre_commit_version": "4.0.0"
  },
  "security_controls": {
    "package_age_gating": {
      "enabled": true,
      "threshold_hours": 336,
      "ecosystems": ["npm", "pip", "go"]
    },
    "install_script_blocking": {
      "enabled": true,
      "ecosystems": ["npm", "pip"]
    },
    "vulnerability_scanning": {
      "tool": "osv-scanner",
      "last_scan": "2026-05-12T14:00:00Z",
      "findings": 0
    },
    "pre_commit_hooks": {
      "installed": ["ripsecrets", "gitleaks", "semgrep"],
      "last_verified": "2026-05-12T16:00:00Z"
    },
    "claude_code_guardrails": {
      "deny_rules_count": 48,
      "permission_level": "restricted",
      "pre_tool_use_hooks": true
    }
  },
  "file_integrity": {
    ".claude/settings.json": {
      "hash": "sha256:...",
      "last_generated": "2026-05-10T10:00:00Z",
      "user_modified": false
    }
  }
}
```

### How This Maps to Compliance Frameworks

| SOC2 Control | gdev Evidence | Field |
|---|---|---|
| CC6.1 (Logical access) | Claude Code permissions | `security_controls.claude_code_guardrails` |
| CC7.1 (System monitoring) | Pre-commit hooks, scanning | `security_controls.pre_commit_hooks`, `vulnerability_scanning` |
| CC8.1 (Change management) | Config versioning, hash tracking | `file_integrity` |
| CC6.6 (External software) | Age-gating, install script blocking | `security_controls.package_age_gating` |

The evidence report does not replace a full SOC2 audit -- it provides machine-generated evidence that specific technical controls were configured and active, which supports the auditor's assessment.

## Feature 5: Quick Teardown for Short Engagements

For spike/POC work (1-2 weeks), full archival is overkill:

```
$ gdev teardown --quick
  Quick teardown: poc-widget (client: acme-corp)
  
  ✓ devenv environment removed
  ✓ MCP tokens revoked
  ✓ Project removed from trusted paths
  
  Note: No archive created. The git repo contains all configuration.
```

### Teardown Profiles

| Profile | When | Archive? | Nix GC? | Token Revocation? | Evidence? |
|---------|------|----------|---------|-------------------|-----------|
| `--quick` | POC/spike end | No | Yes | Yes | No |
| (default) | Normal engagement end | Yes | Optional | Yes | Optional |
| `--compliance` | Regulated client end | Yes | Optional | Yes | Yes (required) |

## Edge Cases

1. **Client requires all data deleted, including archives:** `qsdev teardown --purge` deletes archives too. Confirmation required with client name typed out.

2. **Multiple projects for same client:** Archives are per-project, not per-client. `qsdev teardown --client acme-corp` tears down all projects for that client.

3. **Client rebrands / is acquired:** The `client.name` in `.qsdev.yaml` should be updated. Archives with the old name remain searchable by project name.

4. **Concurrent access from personal and client machines:** qsdev config is per-machine (not synced). Each machine has its own trust state, archives, and local overrides.

5. **Compliance evidence tampering:** The evidence report is signed with the gdev binary's embedded version hash. This is not cryptographic non-repudiation -- it's integrity verification that the report was generated by an authentic gdev binary.

## Depth Checklist

- [x] Underlying mechanism explained -- client profiles, teardown workflow, archival format, evidence generation
- [x] Key tradeoffs and limitations identified -- archive vs purge, evidence limitations, what teardown doesn't cover
- [x] Compared to alternatives -- compliance framework mapping, teardown profiles
- [x] Failure modes and edge cases described -- five scenarios
- [x] Concrete examples -- CLI output, JSON structures, YAML configs
- [x] Report is standalone-readable
