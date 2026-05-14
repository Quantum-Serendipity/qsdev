# Per-Project Development Environments with Nix
- **Source URL**: https://mtlynch.io/notes/nix-dev-environment/
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Michael Lynch

## Project Scope
Personal projects (not team-based)

## Key Pain Points with Nix Dev Environments

### Performance/Load Times
"the environment load times are slow. `cd`ing into a directory is normally something that happens in milliseconds, but if I need to load my Nix environment, it can take 5-10 seconds." Adding dependencies compounds this problem, as Nix maintains separate `nixpkgs` instances per dependency.

### Version Pinning Complexity
The version-pinning process is "honestly, a huge pain," requiring users to look up git commit hashes corresponding to desired package versions rather than specifying versions directly.

### CI Integration Challenges
"Nix takes 60-180 seconds to initialize its environment for the first time, usually downloading multiple gigs of data from package servers." Attempted Cachix but found initialization times remained around 90 seconds minimum per CI step.

### Go Compilation Issues
When building CGO-dependent Go projects, encountered linking failures requiring the `-tags=netgo,osusergo` workaround with no clear explanation for why it worked.

### Environment Variable Conflicts
A misconfigured `GOROOT` variable caused version mismatches requiring system reboots to resolve.

## Conclusion
Lynch **continued using Nix** for personal projects despite limitations, migrating from Ansible because Nix provided lighter-weight dependency management (2 minutes versus 20 minutes for upgrades).

## Alternatives Considered
- **Docker:** Rejected due to incompatibility with his VS Code over SSH development workflow
- **Ansible:** Previously used for six years but abandoned due to overhead for experimental work
