# gdev Integration Research: Local Documentation MCP Servers

## Executive Summary

Local documentation MCP servers integrate cleanly into the gdev-secure-devenv-bootstrap implementation plan as a new tool category within the existing tool lifecycle system (Phase 12). The recommended deployment path uses devenv's native `claude.code.mcpServers` configuration to generate `.mcp.json` entries, with Nix-managed Python environments for openzim-mcp and man-mcp-server, and Nix-managed Node.js for a DevDocs MCP server. Documentation data (ZIM files, DevDocs JSON) lives in `~/.local/share/gdev/docs/` with Nix hash-pinned downloads for supply chain integrity. The devenv ecosystem already has first-class MCP server support (devenv 2.0's `claude.code.mcpServers` options), and two Nix flake projects (mcps.nix, mcp-servers-nix) provide patterns for declarative MCP server packaging that gdev should follow.

---

## 1. Phase Placement in the Implementation Plan

### Recommendation: Extend Phase 12 + New Validation in Phase 22

Local documentation MCP servers fit as **additional tools within the Phase 12 tool lifecycle system**, not a separate phase. The reasoning:

**Phase 12 is the right home** because:
- Phase 12 already implements `gdev enable/disable <tool>` with shared-file surgery for `.mcp.json`, `CLAUDE.md`, `devenv.nix`, and `settings.json` — exactly the files doc MCP servers need to modify
- Phase 12 already integrates Context7 MCP as a lifecycle-managed tool (Unit 12.9 in the plan)
- The tool registry pattern (`Tool` struct with `Name`, `Category`, `Default`, `DetectFunc`, `OwnedFiles`) applies directly to doc MCP servers
- The `FileOwnership` system with `Exclusive` and `Shared` ownership types handles both dedicated config files and `.mcp.json` entries

**Why not Phase 4 (Claude Code addon core)?** Phase 4 generates the base `.mcp.json` structure and Socket.dev MCP entry. Doc MCP servers are optional enhancements, not core security infrastructure. Adding them to Phase 4 would bloat the critical path.

**Why not Phase 11 (AI agent tooling)?** Phase 11 is for tools that enhance the AI agent's reasoning (postmortem verification, version guardrails, semantic search). Doc MCP servers enhance the agent's knowledge base — a different concern.

**Why not a new phase?** The tool lifecycle infrastructure from Phase 12 Unit 12.1 is a prerequisite. Creating a new phase would either duplicate lifecycle code or create a dependency that delays the work unnecessarily.

### Dependencies

| Dependency | Phase | What's Needed |
|-----------|-------|---------------|
| Tool lifecycle system | Phase 12 (Unit 12.1) | `gdev enable/disable` commands, file ownership registry, shared-file surgery |
| `.mcp.json` generation | Phase 4 (Unit 3.6) | Base `.mcp.json` struct marshaling infrastructure |
| Wizard infrastructure | Phase 6 | Tool selection form groups, detection engine |
| devenv addon | Phase 3 | `devenv.nix` generation with `claude.code.mcpServers` |
| Profile system | Phase 6 / Phase 13 | Profile-driven doc set selection |
| `gdev outdated`/`gdev update` | Phase 16 | Doc update integration |

### Proposed Units (added to Phase 12)

- **Unit 12.10: OpenZIM MCP Server Integration** — Tool registration, Nix packaging, ZIM file management, `.mcp.json` generation
- **Unit 12.11: DevDocs MCP Server Integration** — Tool registration, Nix packaging, data download, `.mcp.json` generation
- **Unit 12.12: System Documentation Servers** — man-mcp-server and MCP-NixOS registration and configuration
- **Unit 12.13: Documentation Corpus Management** — Storage, download orchestration, update checking, disk space management
- **Unit 12.14: Wizard Integration — Documentation MCP Servers** — Detection-driven doc selection, disk space warnings, profile mapping

Validation scenarios for these tools would be added to **Phase 22** (Agentic Skills, Compliance & DX Validation), covering enable/disable round-trips, shared-file integrity after toggling doc MCP servers, and update mechanism testing.

---

## 2. .mcp.json Generation

### Generation Strategy: Two Paths

gdev generates MCP server configuration through two complementary paths:

**Path A: devenv.nix `claude.code.mcpServers`** (preferred when project uses devenv)

devenv 2.0 natively supports MCP server configuration. When `claude.code.enable = true` in `devenv.nix`, devenv generates `.mcp.json` automatically. gdev should generate the `devenv.nix` entries and let devenv handle `.mcp.json` generation.

**Path B: Direct `.mcp.json` generation** (fallback when devenv is not in use)

For projects not using devenv (or for user-level configuration), gdev generates `.mcp.json` entries directly using struct marshaling, following the existing Phase 4 pattern.

### Concrete .mcp.json Snippets

#### openzim-mcp (Python, ZIM file reader)

```json
{
  "mcpServers": {
    "local-docs-zim": {
      "command": "openzim-mcp",
      "args": [],
      "env": {
        "OPENZIM_MCP_ZIM_DIR": "${HOME}/.local/share/gdev/docs/zim",
        "OPENZIM_MCP_TOOL_MODE": "simple",
        "OPENZIM_MCP_CACHE_TTL": "3600",
        "OPENZIM_MCP_LOG_LEVEL": "WARNING"
      }
    }
  }
}
```

The equivalent devenv.nix configuration:

```nix
claude.code.mcpServers.local-docs-zim = {
  type = "stdio";
  command = "${pkgs.openzim-mcp}/bin/openzim-mcp";
  env = {
    OPENZIM_MCP_ZIM_DIR = "${config.home.homeDirectory}/.local/share/gdev/docs/zim";
    OPENZIM_MCP_TOOL_MODE = "simple";
    OPENZIM_MCP_CACHE_TTL = "3600";
    OPENZIM_MCP_LOG_LEVEL = "WARNING";
  };
};
```

#### DevDocs MCP (TypeScript/Node, JSON file reader)

For the direct-file-access approach (following jiegec/devdocs-mcp-server pattern):

```json
{
  "mcpServers": {
    "local-docs-devdocs": {
      "command": "npx",
      "args": ["devdocs-mcp-server"],
      "env": {
        "DEVDOCS_DATA_DIR": "${HOME}/.local/share/gdev/docs/devdocs",
        "DEVDOCS_DOC_SETS": "typescript,node,react,python~3.12,go"
      }
    }
  }
}
```

For madhan-g-p/DevDocs-MCP (NestJS, version-pinning):

```json
{
  "mcpServers": {
    "local-docs-devdocs": {
      "command": "npx",
      "args": ["-y", "devdocs-mcp"],
      "env": {
        "DEVDOCS_DATA_PATH": "${HOME}/.local/share/gdev/docs/devdocs"
      }
    }
  }
}
```

#### man-mcp-server (Python, system man pages)

```json
{
  "mcpServers": {
    "man-pages": {
      "command": "uvx",
      "args": ["man-mcp-server"],
      "env": {}
    }
  }
}
```

devenv.nix equivalent:

```nix
claude.code.mcpServers.man-pages = {
  type = "stdio";
  command = "${pkgs.man-mcp-server}/bin/man-mcp-server";
};
```

#### MCP-NixOS (Python, NixOS ecosystem queries)

```json
{
  "mcpServers": {
    "nixos": {
      "command": "uvx",
      "args": ["mcp-nixos"],
      "env": {}
    }
  }
}
```

Note: MCP-NixOS requires internet access (queries search.nixos.org, FlakeHub, noogle.dev). This is acceptable because the sources are first-party NixOS infrastructure with low prompt injection risk. It should be clearly labeled as an online-dependent server in documentation.

#### Context7 (web fallback, clearly labeled)

```json
{
  "mcpServers": {
    "context7": {
      "command": "npx",
      "args": ["-y", "@upstash/context7-mcp@latest"],
      "env": {}
    }
  }
}
```

This is the web fallback — labeled in CLAUDE.md as lower-trust than local sources. Already planned for Phase 12 Unit 12.9.

### .mcp.json Merge Strategy

Documentation MCP servers contribute entries to `.mcp.json` as `Shared` ownership (tool contributes server key). On `gdev enable local-docs-zim`, the entry is added via JSON parse/insert/marshal. On `gdev disable local-docs-zim`, the entry is removed. This follows the identical pattern used for semble, context7, and github-mcp in the existing plan.

---

## 3. Nix Packaging

### 3.1 openzim-mcp: Python Package with Native libzim

**Challenge:** openzim-mcp requires Python >=3.12, depends on `libzim` (Python package that bundles native C++ libzim via Cython wheels), plus beautifulsoup4, html2text, mcp[cli], pydantic, pydantic-settings, and tiktoken.

**Approach: Use uv/pip in a Nix-managed Python environment, not nixpkgs libzim.**

Rationale:
- nixpkgs `libzim` has had build failures (ICU linkage issues, GitHub #384684, resolved but indicates fragility)
- python-libzim PyPI wheels bundle pre-built libzim binaries — no system libzim needed
- openzim-mcp's PyPI distribution is designed for `uv tool install` / `pip install`
- devenv's Python integration natively manages venvs with pip/uv

**Nix derivation approach:**

```nix
# In devenv.nix — lightweight approach via uv
languages.python = {
  enable = true;
  version = "3.12";
  uv.enable = true;
};

# Install openzim-mcp as a tool
enterShell = ''
  if ! command -v openzim-mcp &>/dev/null; then
    uv tool install openzim-mcp
  fi
'';
```

**Alternative: Fully Nix-packaged derivation** (for reproducibility purists):

```nix
openzim-mcp = pkgs.python312Packages.buildPythonApplication {
  pname = "openzim-mcp";
  version = "2.0.0a12";
  src = pkgs.fetchFromPyPI {
    pname = "openzim-mcp";
    version = "2.0.0a12";
    hash = "sha256-XXXX";
  };
  propagatedBuildInputs = with pkgs.python312Packages; [
    beautifulsoup4
    html2text
    libzim  # PyPI package, not nixpkgs libzim
    pydantic
    pydantic-settings
    tiktoken
  ];
  # The libzim PyPI wheel bundles native C++ libzim
  # Need to use wheel format to get pre-built binaries
  format = "wheel";
};
```

**Recommended approach for gdev:** Use `uv tool install` in the devenv enterShell, gated on availability check. This avoids complex Nix packaging of native Python wheels while still providing Nix-managed Python. The tool binary ends up in `~/.local/bin/` (or uv's tool directory) which is on PATH in the devenv shell.

### 3.2 DevDocs Data: Nix Derivation for Documentation JSON

DevDocs documentation is three JSON files per doc set. Two acquisition strategies:

**Strategy A: Extract from Docker image** (following jiegec model)

```nix
devdocs-data = pkgs.stdenv.mkDerivation {
  pname = "devdocs-data";
  version = "2026-05";
  src = pkgs.dockerTools.pullImage {
    imageName = "ghcr.io/freecodecamp/devdocs";
    imageDigest = "sha256:XXXX";  # Pin exact digest
    sha256 = "XXXX";
  };
  # Extract documentation JSON from image layers
  buildPhase = ''
    # Extract the docs layer and copy JSON files
    mkdir -p $out
    # ... layer extraction logic
  '';
};
```

**Strategy B: Direct download from devdocs.io** (simpler, per-doc-set)

```nix
# Per-doc-set fetcher
fetchDevDocs = { slug, hash }: pkgs.fetchurl {
  url = "https://documents.devdocs.io/${slug}/db.json";
  sha256 = hash;
};

# Usage:
devdocs-typescript = fetchDevDocs {
  slug = "typescript~5.5";
  hash = "sha256-XXXX";
};
```

**Note:** devdocs.io blocks automated access (403). Strategy A (Docker extraction) or self-hosted devdocs are the reliable paths. For gdev, the Docker extraction approach is preferred because the image is pinnable by digest hash, providing the same supply chain integrity as Nix fetchurl.

**Recommended approach for gdev:** Use `thor docs:download` in a Nix derivation or a gdev-managed download step (not Nix). The Docker image approach is clean for CI but adds Docker as a dependency. A simpler model: gdev's `enable local-docs-devdocs` triggers a download script that fetches documentation JSON files and stores them in `~/.local/share/gdev/docs/devdocs/`. The download uses checksums from a gdev-maintained manifest for integrity.

### 3.3 ZIM Files: Nix fetchurl with Hash Pinning

ZIM files are large static downloads — perfect candidates for Nix's content-addressed store.

```nix
# Per-ZIM-file fetcher with hash pinning
zim-unix-stackexchange = pkgs.fetchurl {
  url = "https://download.kiwix.org/zim/stack_exchange/unix.stackexchange.com_en_all_2026-02.zim";
  sha256 = "sha256-XXXX";  # Pin exact content hash
};

zim-serverfault = pkgs.fetchurl {
  url = "https://download.kiwix.org/zim/stack_exchange/serverfault.com_en_all_2026-02.zim";
  sha256 = "sha256-XXXX";
};
```

**Security benefit:** Nix hash pinning mitigates the supply chain risk identified in the prompt injection research. Neither ZIM nor DevDocs use cryptographic signing — Nix's content-addressed store provides the integrity verification that these formats lack natively. If the upstream file changes (even by one byte), the hash fails and the build aborts.

**Practical concern:** ZIM files are 1-4 GB each. Storing them in the Nix store means they consume store space and participate in garbage collection. For developer workstations, a symlink from `~/.local/share/gdev/docs/zim/` to the Nix store path is the right pattern — the Nix store provides integrity, and the symlink provides a stable path for openzim-mcp's `ZIM_DIR` configuration.

**Recommended approach:** Use Nix fetchurl for ZIM files in a dedicated derivation that creates a directory of symlinks. This gives hash-pinned integrity without duplicating the files on disk.

### 3.4 Existing Nix Packaging

| Component | In nixpkgs? | Notes |
|-----------|-------------|-------|
| libzim (C++) | Yes, but fragile | Had ICU linkage failures in 25.05. Resolved but indicates ongoing maintenance burden. |
| kiwix-tools | Yes | `pkgs.kiwix-tools` provides kiwix-serve, kiwix-manage. PR #206254 (init at 3.4.0). |
| python-libzim | No | PyPI wheels bundle native libzim. Use pip/uv, not nixpkgs. |
| openzim-mcp | No | PyPI package. Use `uv tool install`. |
| man-mcp-server | No | PyPI package. Use `uvx` or pip. |
| mcp-nixos | Partially | README mentions `pkgs.mcp-nixos` for NixOS/Home Manager. May exist in unstable. |
| DevDocs | No | Ruby app with Docker images. Not in nixpkgs. |
| Context7 | No | npm package, already handled in Phase 12 via `npx`. |
| mcps.nix | N/A (flake) | Nix flake providing MCP server presets for devenv and home-manager. 26 stars. |
| mcp-servers-nix | N/A (flake) | Nix flake providing 28+ MCP server packages for devenv/home-manager. 252 stars. |

**Key finding:** Two community Nix flake projects — `roman/mcps.nix` (26 stars) and `natsukium/mcp-servers-nix` (252 stars) — already provide declarative MCP server packaging patterns for devenv and home-manager. gdev should follow these patterns rather than inventing a new packaging approach. Both use devenv's native `claude.code.mcpServers` module, confirming this is the right integration point.

---

## 4. Wizard Integration

### 4.1 Detection: Project Signals to Doc Sets

The wizard's detection engine maps project files to recommended documentation sets:

| Signal | Doc Set | Source |
|--------|---------|--------|
| `package.json` | Node.js, JavaScript (DevDocs) | DevDocs |
| `tsconfig.json` | TypeScript (DevDocs) | DevDocs |
| `go.mod` | Go (DevDocs + godoc-mcp) | DevDocs |
| `Cargo.toml` | Rust (DevDocs) | DevDocs |
| `pyproject.toml` / `requirements.txt` | Python (DevDocs) | DevDocs |
| `pom.xml` / `build.gradle` | Java/Kotlin (DevDocs) | DevDocs |
| `*.csproj` | .NET/C# (DevDocs) | DevDocs |
| `docker-compose.yml` / `Dockerfile` | Docker (DevDocs) | DevDocs |
| `*.tf` | Terraform (DevDocs) | DevDocs |
| `flake.nix` / `devenv.nix` | NixOS docs | MCP-NixOS |
| Any project on Linux | Man pages | man-mcp-server |
| Any project | Unix & Linux SE, Software Engineering SE | OpenZIM (ZIM) |
| Web frontend (`react`, `vue`, `svelte` in deps) | MDN Web Docs, React/Vue/Svelte docs | DevDocs |
| `composer.json` | PHP (DevDocs) | DevDocs |
| `Gemfile` | Ruby (DevDocs) | DevDocs |

### 4.2 Selection: Auto-Include vs Opt-In

**Auto-included (zero cost, always useful):**
- man-mcp-server — purely local, zero setup, zero disk cost
- MCP-NixOS — for NixOS users (detected by `flake.nix` or `devenv.nix` presence), queries first-party APIs

**Auto-included when detected (low cost):**
- DevDocs for detected ecosystems — only download doc sets matching detected languages
- Context7 as labeled web fallback — already in Phase 12 plan

**Opt-in (significant disk cost):**
- OpenZIM with Stack Exchange ZIM files — 4-5 GB for curated set
- DevDocs "full corpus" — all 100+ doc sets, ~3.5 GB
- Additional ZIM files beyond the default set

### 4.3 Storage Location

```
~/.local/share/gdev/docs/
├── zim/                          # ZIM files (symlinks to Nix store or direct downloads)
│   ├── unix.stackexchange.com_en_all_2026-02.zim
│   ├── serverfault.com_en_all_2026-02.zim
│   └── softwareengineering.stackexchange.com_en_all_2026-02.zim
├── devdocs/                      # DevDocs JSON data
│   ├── typescript~5.5/
│   │   ├── index.json
│   │   ├── db.json
│   │   └── meta.json
│   ├── node~22/
│   ├── python~3.12/
│   └── react~18/
└── manifest.json                 # gdev tracking: installed docs, versions, hashes, sizes
```

**Why user-level, not project-level?**
- ZIM files are 1-4 GB each — sharing across projects saves 4-8 GB per developer
- DevDocs data is shared across all projects using the same language
- Content is not project-specific (it's upstream documentation, not project code)
- Project-level `.mcp.json` points to the user-level data directory via `$HOME` expansion

**Why not `/nix/store/`?**
- ZIM files are too large for the Nix store in practice (garbage collection, store optimization overhead)
- Better model: Nix verifies the download hash, then copies to `~/.local/share/gdev/docs/`
- The manifest.json tracks what's installed and its hash for integrity checking

### 4.4 Disk Space Handling in Wizard

```
┌─────────────────────────────────────────────────────────────┐
│ Local Documentation (offline docs for Claude Code)           │
│                                                              │
│ Auto-detected languages: TypeScript, Go, Python              │
│                                                              │
│ ☑ DevDocs for detected languages     ~450 MB                │
│   (TypeScript, Node.js, Go, Python)                          │
│ ☑ man-mcp-server (system man pages)  0 MB (uses system)     │
│ ☑ MCP-NixOS                          0 MB (online API)      │
│ ☐ Stack Exchange Q&A (ZIM files)     ~4.5 GB                │
│   (Unix & Linux, Server Fault, Software Engineering,         │
│    DevOps, Database Administrators)                          │
│ ☐ Full DevDocs (all 100+ doc sets)   ~3.5 GB                │
│                                                              │
│ Total selected: ~450 MB                                      │
│ Available disk: 142 GB                                       │
│                                                              │
│ Note: Documentation is downloaded on first use.              │
│ Run `gdev docs update` to refresh.                           │
└─────────────────────────────────────────────────────────────┘
```

Key UX decisions:
- Show disk cost per option — developers make informed choices
- Show available disk space — 4.5 GB is significant on a 256 GB SSD
- Default to lightweight options (DevDocs for detected languages only)
- Stack Exchange ZIM is opt-in due to size
- "Downloaded on first use" — don't block wizard completion on a multi-GB download

---

## 5. Profile-Driven Configuration

### 5.1 consulting-default Profile

The consulting-default profile is gdev's primary deployment profile for the consulting firm. Documentation selections should reflect the most commonly encountered client stacks:

```yaml
# In profile definition
docs:
  devdocs:
    auto_detect: true                    # Include docs matching detected ecosystems
    always_include:                       # Always include these regardless of detection
      - javascript
      - typescript
      - html
      - css
      - node
      - git
  man_pages: true                        # Always on
  mcp_nixos: true                        # Always on (NixOS shop)
  zim:
    enabled: false                       # Opt-in due to size
    default_sets:
      - unix.stackexchange.com
      - serverfault.com
      - softwareengineering.stackexchange.com
  context7: true                         # Web fallback, always on
```

### 5.2 Per-Ecosystem Profiles

Profiles can encode ecosystem-specific documentation choices:

```yaml
# go-service profile
docs:
  devdocs:
    always_include: [go, postgresql, redis, docker, terraform]
  zim:
    default_sets: [unix.stackexchange.com, serverfault.com, devops.stackexchange.com]

# ts-fullstack profile
docs:
  devdocs:
    always_include: [typescript, node, react, css, html, postgresql, redis]
  zim:
    default_sets: [unix.stackexchange.com]

# infrastructure profile
docs:
  devdocs:
    always_include: [terraform, ansible, docker, nginx, postgresql]
  zim:
    default_sets: [unix.stackexchange.com, serverfault.com, devops.stackexchange.com]
```

### 5.3 Custom Profiles (Team-Specified)

Teams add documentation corpus specifications to `.gdev.yaml`:

```yaml
# .gdev.yaml (project-level, committed to git)
docs:
  devdocs:
    sets:
      - typescript~5.5
      - react~18
      - node~22
      - postgresql~16
    auto_detect: false  # Explicit list, no auto-detection
  zim:
    enabled: true
    sets:
      - unix.stackexchange.com
      - dba.stackexchange.com
  context7: false  # Air-gapped environment, no web fallback
```

The three-layer config resolution applies: binary defaults (consulting-default) -> `.gdev.yaml` (project) -> `.gdev.local.yaml` (developer). A developer can disable ZIM files locally without affecting the team configuration.

---

## 6. Tool Lifecycle

### 6.1 Tool Registration

Each documentation MCP server is registered in the tool registry following Phase 12's pattern:

```go
// openzim-mcp tool registration
Tool{
    Name:        "local-docs-zim",
    DisplayName: "Local Docs — Stack Exchange (ZIM)",
    Category:    "documentation",
    Description: "Offline Stack Exchange Q&A via OpenZIM MCP server",
    Default:     OptIn,  // Significant disk cost
    DetectFunc:  func(os *OSInfo, proj *DetectedProject) bool {
        return false  // Always opt-in due to 4+ GB download
    },
    Prerequisites: []string{"python3"},
    OwnedFiles: []FileOwnership{
        {Path: ".mcp.json", Ownership: Shared, SectionID: "local-docs-zim"},
        {Path: "devenv.nix", Ownership: Shared, SectionID: "local-docs-zim"},
        {Path: "CLAUDE.md", Ownership: Shared, SectionID: "local-docs-zim"},
    },
}

// DevDocs MCP tool registration
Tool{
    Name:        "local-docs-devdocs",
    DisplayName: "Local Docs — DevDocs API Reference",
    Category:    "documentation",
    Description: "Offline language/framework API documentation via DevDocs",
    Default:     OnWhenDetected,
    DetectFunc:  func(os *OSInfo, proj *DetectedProject) bool {
        return len(proj.Languages) > 0  // Enable when any language detected
    },
    Prerequisites: []string{"node"},
    OwnedFiles: []FileOwnership{
        {Path: ".mcp.json", Ownership: Shared, SectionID: "local-docs-devdocs"},
        {Path: "devenv.nix", Ownership: Shared, SectionID: "local-docs-devdocs"},
        {Path: "CLAUDE.md", Ownership: Shared, SectionID: "local-docs-devdocs"},
    },
}

// man-mcp-server tool registration
Tool{
    Name:        "man-pages",
    DisplayName: "Local Docs — Man Pages",
    Category:    "documentation",
    Description: "System man page access via man-mcp-server",
    Default:     AlwaysOn,
    DetectFunc:  func(os *OSInfo, proj *DetectedProject) bool {
        return os.Family == "linux" || os.Family == "darwin"
    },
    Prerequisites: []string{"python3", "man"},
    OwnedFiles: []FileOwnership{
        {Path: ".mcp.json", Ownership: Shared, SectionID: "man-pages"},
        {Path: "CLAUDE.md", Ownership: Shared, SectionID: "man-pages"},
    },
}

// MCP-NixOS tool registration
Tool{
    Name:        "mcp-nixos",
    DisplayName: "MCP-NixOS",
    Category:    "documentation",
    Description: "NixOS packages, options, and Home Manager documentation",
    Default:     OnWhenDetected,
    DetectFunc:  func(os *OSInfo, proj *DetectedProject) bool {
        return proj.HasFile("flake.nix") || proj.HasFile("devenv.nix") || os.IsNixOS
    },
    Prerequisites: []string{"python3"},
    OwnedFiles: []FileOwnership{
        {Path: ".mcp.json", Ownership: Shared, SectionID: "mcp-nixos"},
        {Path: "CLAUDE.md", Ownership: Shared, SectionID: "mcp-nixos"},
    },
}
```

### 6.2 What Files Are Modified

**`gdev enable local-docs-zim`:**

| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"local-docs-zim"` server entry with ZIM_DIR env var |
| `devenv.nix` | Shared | Add `# --- local-docs-zim ---` section: openzim-mcp install in enterShell |
| `CLAUDE.md` | Shared | Add `<!-- gdev:local-docs-zim -->` section: usage docs, content scope, trust level |

**`gdev enable local-docs-devdocs`:**

| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"local-docs-devdocs"` server entry |
| `devenv.nix` | Shared | Add `# --- local-docs-devdocs ---` section: Node.js dependency |
| `CLAUDE.md` | Shared | Add `<!-- gdev:local-docs-devdocs -->` section: available doc sets, search tips |

**`gdev enable man-pages`:**

| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"man-pages"` server entry |
| `CLAUDE.md` | Shared | Add `<!-- gdev:man-pages -->` section: available sections, search syntax |

**`gdev disable local-docs-zim`:**

| File | Operation | Notes |
|------|-----------|-------|
| `.mcp.json` | Remove `"local-docs-zim"` key | JSON parse/remove/marshal |
| `devenv.nix` | Remove `# --- local-docs-zim ---` block | Validate Nix still parses |
| `CLAUDE.md` | Remove `<!-- gdev:local-docs-zim -->` block | |

### 6.3 Data Download Timing

**Not on enable — lazy on first use.** Rationale:

- `gdev enable local-docs-zim` should be fast (modify config files, done)
- Downloading 4+ GB of ZIM files during `gdev enable` would block the terminal
- The MCP server handles missing ZIM files gracefully (returns empty results, not errors)
- First actual Claude Code query to the server triggers a user-visible "no documentation found" response, which naturally prompts running `gdev docs download`

**Explicit download command:**

```bash
gdev docs download              # Download all enabled doc sets
gdev docs download --zim        # Download only ZIM files
gdev docs download --devdocs    # Download only DevDocs
gdev docs status                # Show what's installed, what's pending
```

This separates "configure" from "download" — configuration is instant and reversible, downloads are explicit and long-running.

### 6.4 Cleanup on Disable

**Keep cached data.** Removing 4 GB of ZIM files on `gdev disable` would be destructive and potentially slow. Instead:

- `gdev disable local-docs-zim` removes configuration only (fast, reversible)
- `gdev docs clean` removes downloaded data (explicit, destructive)
- `gdev docs clean --zim` removes only ZIM files
- `gdev docs clean --devdocs` removes only DevDocs data
- `gdev docs clean --all` removes everything

This follows the pattern of Docker image management: `docker rm` removes containers (fast), `docker rmi` removes images (separate, explicit).

---

## 7. Update Mechanism

### 7.1 ZIM File Updates

ZIM files follow a predictable naming convention: `<site>_<language>_<type>_<year>-<month>.zim`. Updates are detectable by comparing the year-month suffix against download.kiwix.org directory listings.

**Update detection:**

```bash
gdev docs outdated
# Output:
# ZIM files:
#   unix.stackexchange.com  installed: 2026-02  available: 2026-04  (2 months old)
#   serverfault.com         installed: 2026-02  available: 2026-05  (3 months old)
# DevDocs:
#   typescript~5.5          installed: 2026-03  available: 2026-05
#   node~22                 current
```

**Implementation:** Parse local ZIM filenames, query download.kiwix.org directory listings (same approach as jojo2357/kiwix-zim-updater), compare year-month components. Store installed version metadata in `~/.local/share/gdev/docs/manifest.json`.

**Update execution:**

```bash
gdev docs update              # Download newer versions of all installed docs
gdev docs update --zim        # Update only ZIM files
gdev docs update --devdocs    # Update only DevDocs
```

For Nix-pinned ZIM files, updating means updating the hash in the Nix derivation — which requires a gdev binary update or a manifest update mechanism.

### 7.2 DevDocs Updates

DevDocs data is updated when the Docker image is rebuilt (roughly monthly). For the direct-file approach, `gdev docs update --devdocs` would:

1. Check devdocs.io or the Docker image registry for newer versions
2. Download updated JSON files for installed doc sets
3. Verify checksums against gdev's manifest
4. Replace old files atomically

### 7.3 Integration with `gdev outdated` / `gdev update`

Phase 16 defines `gdev outdated` and `gdev update` for coordinated updates across all gdev-managed tools. Documentation freshness integrates naturally:

```bash
gdev outdated
# Output:
# Tools:
#   semgrep         1.91.0 → 1.94.0
#   gitleaks        8.22.0 → 8.24.1
# Documentation:
#   unix.stackexchange.com (ZIM)   2026-02 → 2026-04
#   typescript~5.5 (DevDocs)       2026-03 → 2026-05
# Dependencies:
#   react           18.2.0 → 18.3.1 (Renovate PR #42 pending)
```

`gdev update` with `--docs` flag triggers documentation updates alongside tool updates.

### 7.4 Update Cadence Recommendations

| Source | Release cadence | Recommended check | Notes |
|--------|----------------|-------------------|-------|
| ZIM (Stack Exchange) | ~2-3 months | Monthly | Smaller SE sites updated regularly; full SO stale since Nov 2023 |
| ZIM (Wikipedia, MDN) | ~1-3 months | Monthly | Kiwix updates vary by content type |
| DevDocs (Docker image) | ~Monthly | Monthly | freeCodeCamp rebuilds monthly |
| DevDocs (per-doc-set) | Per upstream release | On language version change | e.g., update Python docs when upgrading from 3.11 to 3.12 |
| MCP-NixOS | N/A (online) | N/A | Queries live APIs, always current |
| man-mcp-server | N/A (system) | On system update | Man pages update with Nix packages |

### 7.5 Nix Hash Update Mechanism

For Nix-pinned ZIM downloads, hash updates require one of:
1. **gdev binary update** — new release includes updated hashes in embedded manifest
2. **External manifest** — gdev fetches a manifest from a gdev-maintained URL with current hashes (adds network dependency)
3. **User-triggered rehash** — `gdev docs update` downloads new file, computes hash, updates local Nix expression

Option 3 is most practical: the local `manifest.json` stores current hashes, and `gdev docs update` refreshes both the files and the hashes. The Nix derivation reads hashes from the manifest rather than hardcoding them.

---

## 8. devenv Native Integration — The Key Enabler

### 8.1 devenv 2.0 claude.code Module

devenv 2.0 (March 2026) introduced native Claude Code integration via the `claude.code` module in `devenv.nix`. This is a game-changer for gdev's MCP server deployment:

- `claude.code.enable = true` activates the integration
- `claude.code.mcpServers.<name>` defines MCP servers declaratively
- devenv generates `.mcp.json` automatically from these definitions
- Servers can reference Nix store paths (`${pkgs.foo}/bin/foo`) for reproducible commands
- Environment variables, args, and HTTP headers are all configurable

**This means gdev should generate `devenv.nix` content, not `.mcp.json` directly.** The devenv module handles `.mcp.json` generation, and developers get the full benefit of Nix's reproducibility.

### 8.2 mcps.nix Pattern

The `roman/mcps.nix` project (26 stars) provides a pattern gdev should follow:
- MCP servers are Nix module options (`mcps.git.enable = true`)
- Credential handling reads from files, not environment variables
- The flake exports `devenvModules.claude` for devenv integration
- Each server preset defines its command, args, and environment

gdev doesn't need to depend on mcps.nix directly, but should follow its pattern of declarative MCP server modules that compose with devenv's `claude.code.mcpServers`.

### 8.3 Recommended devenv.nix Generation

When gdev generates `devenv.nix` with documentation MCP servers enabled:

```nix
{ pkgs, config, ... }:
{
  # --- local-docs-zim ---
  # OpenZIM MCP server for offline Stack Exchange Q&A
  enterShell = ''
    if ! command -v openzim-mcp &>/dev/null; then
      echo "Installing openzim-mcp..."
      uv tool install openzim-mcp 2>/dev/null || true
    fi
  '';

  claude.code.mcpServers.local-docs-zim = {
    type = "stdio";
    command = "openzim-mcp";
    env = {
      OPENZIM_MCP_ZIM_DIR = "${config.env.HOME}/.local/share/gdev/docs/zim";
      OPENZIM_MCP_TOOL_MODE = "simple";
      OPENZIM_MCP_LOG_LEVEL = "WARNING";
    };
  };
  # --- end local-docs-zim ---

  # --- local-docs-devdocs ---
  # DevDocs MCP server for offline API documentation
  claude.code.mcpServers.local-docs-devdocs = {
    type = "stdio";
    command = "npx";
    args = [ "devdocs-mcp-server" ];
    env = {
      DEVDOCS_DATA_DIR = "${config.env.HOME}/.local/share/gdev/docs/devdocs";
    };
  };
  # --- end local-docs-devdocs ---

  # --- man-pages ---
  claude.code.mcpServers.man-pages = {
    type = "stdio";
    command = "uvx";
    args = [ "man-mcp-server" ];
  };
  # --- end man-pages ---

  # --- mcp-nixos ---
  claude.code.mcpServers.nixos = {
    type = "stdio";
    command = "uvx";
    args = [ "mcp-nixos" ];
  };
  # --- end mcp-nixos ---
}
```

The section markers (`# --- tool-name ---` / `# --- end tool-name ---`) enable the tool lifecycle system to surgically add/remove each server's configuration.

---

## 9. CLAUDE.md Documentation Sections

Each documentation MCP server contributes a section to CLAUDE.md via section markers:

```markdown
<!-- gdev:local-docs-zim -->
## Local Documentation — Stack Exchange (ZIM)

Offline Stack Exchange Q&A is available via the `local-docs-zim` MCP server.
Query it for programming questions, system administration, and software engineering topics.

**Available sites:** Unix & Linux, Server Fault, Software Engineering, DevOps
**Trust level:** Community-generated content (lower trust than official documentation)
**Freshness:** Updated monthly. Run `gdev docs outdated` to check.

When answering questions, prefer official documentation (DevDocs) for API reference
and Stack Exchange for troubleshooting, debugging, and "how do I..." questions.
<!-- /gdev:local-docs-zim -->

<!-- gdev:local-docs-devdocs -->
## Local Documentation — DevDocs API Reference

Offline API documentation is available via the `local-docs-devdocs` MCP server.
Use it for language/framework API lookups before falling back to web search.

**Available doc sets:** TypeScript 5.5, Node.js 22, React 18, Python 3.12, Go
**Trust level:** Official upstream documentation (high trust)
**Freshness:** Run `gdev docs outdated` to check for updates.

Prefer local DevDocs over web fetches for API reference. The documentation is
curated from official sources and sanitized (scripts/styles removed).
<!-- /gdev:local-docs-devdocs -->

<!-- gdev:man-pages -->
## System Documentation — Man Pages

System man pages are accessible via the `man-pages` MCP server.
Use `search_man_pages` for discovery and `get_man_page` for full content.
<!-- /gdev:man-pages -->
```

---

## 10. MCP Server Count Budget

The implementation plan identifies 3-6 MCP servers as the sweet spot (more than 10 slows agents without proportional benefit). With documentation MCP servers, the default configuration would be:

| Server | Category | Default |
|--------|----------|---------|
| Socket.dev | Security | Always on (Phase 4) |
| GitHub | Integration | Always on (Phase 4) |
| semble | AI agent | On when Python detected (Phase 11) |
| Context7 | Docs (web) | Always on (Phase 12) |
| local-docs-devdocs | Docs (local) | On when languages detected |
| man-pages | Docs (local) | Always on (Linux/macOS) |
| mcp-nixos | Docs (online) | On when Nix detected |
| local-docs-zim | Docs (local) | Opt-in |

That's 5-7 servers with typical defaults, reaching the upper end of the sweet spot. If this proves too many, the recommendation is to drop Context7 (replaced by local docs) and make mcp-nixos opt-in, bringing the count to 4-5.

**Key tradeoff:** More documentation servers means richer context for Claude but slower tool discovery and potential confusion about which server to query. The CLAUDE.md sections (above) mitigate this by giving Claude explicit guidance on when to use each source.

---

## Sources

All source documents saved to `docs/`:
- `devenv-mcp-server-docs.md` — devenv MCP server documentation
- `devenv-claude-code-integration.md` — devenv Claude Code integration
- `devenv-claude-code-options-reference.md` — devenv.nix claude.code.* options
- `openzim-mcp-pyproject-toml.md` — openzim-mcp dependencies and requirements
- `mcps-nix-github.md` — mcps.nix MCP server presets for devenv/home-manager
- `mcp-servers-nix-github.md` — mcp-servers-nix Nix configuration framework
- `nixpkgs-libzim-build-failure-384684.md` — libzim build issues in nixpkgs
- `kiwix-zim-updater-github.md` — ZIM file update automation patterns
