# Multi-Project Environment Switching Research

## Research Question

For consultants working across multiple client projects, are there gaps in the devenv + direnv workflow that gdev should address?

## Current State: devenv 2.0+ Environment Switching

### devenv hook (Native, Recommended)

As of devenv 2.0, the recommended approach is `devenv hook` -- native auto-activation that replaces direnv entirely:

```bash
# Add to ~/.bashrc or ~/.zshrc
eval "$(devenv hook bash)"  # or zsh, fish, nu
```

How it works:
- Activates on `cd` into a directory containing `devenv.nix`
- Deactivates on `cd` out
- Trust managed via `devenv allow` / `devenv revoke`
- No `.envrc` file needed
- No external dependencies

### devenv 2.1 (May 2026)

devenv 2.1 adds shell support via libghostty -- zsh, fish, and nushell work natively, not just bash. This eliminates the main remaining friction point with devenv shell.

### direnv (Legacy but Still Supported)

direnv remains supported for "in-place environment modification without a subshell." Setup requires an `.envrc` file with `eval "$(devenv direnvrc)" && use devenv`. Still useful for teams that prefer the direnv model or need direnv for non-devenv env vars.

## Gap Analysis: What Breaks for Multi-Project Consultants

### Gap 1: First-Activation Latency

**Problem**: When a consultant `cd`s into a new client project for the first time, devenv must evaluate the full Nix expression and potentially download/build derivations. This can take 30 seconds to several minutes.

**Impact**: High for consultants switching between 3-5 active projects daily.

**Mitigation in devenv 2.0**: Incremental evaluation cache returns cached results in milliseconds when nothing has changed. First activation is still slow, but subsequent activations are near-instant.

**What gdev can do**: Nothing beyond ensuring the generated devenv.nix is well-structured (minimal unnecessary dependencies). The caching is devenv's responsibility. However, gdev could generate a `gdev warmup` command that pre-evaluates all project environments in parallel (background).

**Recommendation: Consider `gdev warmup` for pre-caching**, but it is low priority. The real fix is devenv's caching, which is already good.

### Gap 2: Environment Variable Conflicts

**Problem**: Client A needs `DATABASE_URL=postgres://clientA/...` and Client B needs `DATABASE_URL=postgres://clientB/...`. When switching, stale env vars can cause data to go to the wrong database.

**Impact**: Critical -- data leaks between client environments are a security/compliance failure.

**Mitigation**: devenv hook and direnv both handle this correctly by design -- they unload the previous environment before loading the new one. The danger is when developers use manual `export` commands outside of devenv's management.

**What gdev can do**: Generate a `.envrc.example` or `devenv.nix` template that encourages all env vars to be defined in devenv rather than manually exported. Document the anti-pattern of manual exports.

**Recommendation: Document the anti-pattern, don't build tooling.** The protection already exists in devenv's activation/deactivation lifecycle.

### Gap 3: Nix Store Disk Usage

**Problem**: Each client project's devenv environment consumes Nix store space. With 5-10 active projects using different Node versions, Python versions, database servers, etc., the Nix store can grow to 20-50GB.

**Impact**: Medium -- disk space is cheap but SSD space on laptops is finite.

**Mitigation**: `nix store gc` cleans unused paths. `nix-collect-garbage -d` removes old generations. devenv's caching reduces rebuilds.

**What gdev can do**: Include `nix store gc` as a documented maintenance task. A `gdev gc` alias would add convenience but minimal value.

**Recommendation: Document, don't build.** The Nix store GC is a well-understood operation.

### Gap 4: Project Discovery and Status

**Problem**: "Which client projects do I have set up? Which are stale? Which have pending updates?" There is no cross-project view.

**Impact**: Medium for consultants juggling many engagements.

**What gdev can do**: A `gdev projects` command that scans for directories containing gdev-managed configs (`.gdev.yaml` or similar marker file) and reports status:
- Project name, path, last activated
- devenv health (valid config? packages cached?)
- gdev config version (outdated? needs update?)

**Recommendation: Include in a future `gdev status` / `gdev projects` command.** This overlaps with the gdev-health-reporting spike but is specifically about the multi-project view. Low implementation cost, high utility for consultants.

### Gap 5: Credential Management Across Projects

**Problem**: Different clients use different cloud providers, registries, and credential stores. Switching projects means switching AWS profiles, Docker registries, npm tokens, etc.

**Impact**: High -- credential mistakes can cause cross-client data exposure.

**Mitigation**: devenv 2.0 integrates SecretSpec 0.7.2 for declarative secret management with provider abstraction (keyring, dotenv, env, 1Password). This is the right tool for this problem.

**What gdev can do**: Generate SecretSpec configuration as part of the devenv addon, with per-ecosystem credential templates (npm token, Docker registry, cloud provider).

**Recommendation: Include SecretSpec configuration in devenv addon templates.** This naturally fits the existing generation pipeline and addresses a real pain point.

## Summary: Genuine Gaps

| Gap | Severity | gdev Action | Priority |
|-----|----------|-------------|----------|
| First-activation latency | Medium | `gdev warmup` (optional) | Low |
| Env var conflicts | Critical | Document anti-pattern | Low (already solved by devenv) |
| Nix store disk usage | Low | Document GC | Low |
| Cross-project status view | Medium | `gdev projects` command | Medium |
| Credential switching | High | Generate SecretSpec config | High |

**Key finding: devenv 2.0 already solves multi-project switching well.** The gaps are at the edges (cross-project visibility, credential management) rather than in the core activation/deactivation flow.

## Depth Checklist

- [x] Underlying mechanism explained -- devenv hook, direnv, activation lifecycle
- [x] Key tradeoffs -- native hook vs direnv, latency vs caching
- [x] Compared to alternatives -- devenv hook vs direnv vs manual
- [x] Failure modes -- stale env vars, disk pressure, credential leaks, first-activation latency
- [x] Concrete examples -- devenv hook setup, SecretSpec integration, project discovery
- [x] Standalone-readable -- yes
