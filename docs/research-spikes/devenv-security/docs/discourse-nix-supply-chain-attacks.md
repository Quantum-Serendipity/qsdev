# NixOS Discourse: Is Nix Vulnerable to Supply Chain Attacks?
- **Source**: https://discourse.nixos.org/t/is-nix-vulnerable-to-supply-chain-attacks/72411
- **Retrieved**: 2026-05-12

## Core Question
User dynamite88 asks whether someone with write access to nixpkgs could inject malicious code into the build process and distribute it via Hydra, comparing this to Debian + Docker as a supposedly safer alternative.

## Key Arguments

**Supply Chain Risk is Universal**
Multiple respondents emphasize that all package distribution systems require trust. As tejing notes: "In any distro, you always have to trust the distro's creators. They're the ones who package and distribute the core software." The assumption that Docker containers from developers eliminate this concern is challenged.

**Misconception About Docker**
TLATER corrects the original premise: "many (most?) docker containers (on dockerhub) are maintained by third parties." They further note that Debian deliberately prevents developers from being sole maintainers, creating additional review layers rather than removing trust requirements.

**Review Process Matters**
Arianvp explains that nixpkgs changes require review before merging, making malicious injection a matter of "do you trust our review process more than Debian's" — a social rather than technical question.

**Mitigation Options**
A significant safeguard exists: users can remove the binary cache (`https://cache.nixos.org`) and compile everything from source. Arianvp notes this compliance option distinguishes Nix from other distributions, though building from source is reportedly more straightforward in nixpkgs than Debian.

## Consensus
All distribution methods carry similar supply chain risks; the primary differentiator is review rigor rather than fundamental architecture.
