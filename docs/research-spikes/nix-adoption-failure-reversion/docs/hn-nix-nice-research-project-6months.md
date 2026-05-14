# HN Thread: "Nix is a nice research project, but after playing with it for ~6 months..."
- **Source URL**: https://news.ycombinator.com/item?id=25026661
- **Retrieved**: 2026-03-20
- **Type**: Hacker News comment thread

## Original Comment (k32)

k32 criticized Nix after 6 months of use, stating: "Nix is a nice research project, but after playing with it for ~6 months and contributing to nix-packages, I came to conclusion that (in my case) it's not suitable for production nor development."

**Key complaints:**
- Extensive patching requirements compared to traditional Linux distributions
- Security concerns about patch quality and review processes
- Erlang/Elixir integration issues where "They patched out the checks for rebar.lock files"
- Secrets management vulnerabilities due to world-readable /nix directories
- Personal use friction with opam packages and GOG game compatibility

**Resolution:** Switched to managing machines with Ansible playbooks.

## Major Counterarguments

**soraminazuki** disputed k32's claims systematically, arguing:
- Most distros patch extensively; this isn't unique to Nix
- nixpkgs undergoes mandatory review processes
- The Erlang patch was misrepresented; it was designed for internal Nix packaging, not general use
- FHSUserenv provides workarounds for FHS-dependent software
- Nix avoids forced clean installs

**Ericson2314** noted Nix works well for many and acknowledged language ecosystem packaging remains challenging.

## Technical Nuances

The thread reveals disagreement about whether departing from FHS (Filesystem Hierarchy Standard) represents a design flaw or feature. k32 viewed FHS compatibility as essential; supporters argued modern software shouldn't assume specific filesystem layouts.

The secrets management debate centered on whether storing configuration in derivations inherently compromises security, with disagreement about practical mitigation approaches.
