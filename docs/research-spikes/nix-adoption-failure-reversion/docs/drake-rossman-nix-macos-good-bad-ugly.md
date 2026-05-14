# Nix on MacOS - The Good, the Bad and the Ugly
- **Source URL**: https://drakerossman.com/blog/nix-on-macos-the-good-the-bad-and-the-ugly
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Drake Rossman

## Abandonment Status
**No clear abandonment stated.** The article concludes that Nix is "the worst package manager on MacOS except for all the other package managers," suggesting continued reluctant use rather than switching away.

## Usage Context
**Both personal and team use.** Rossman notes that "Nix development environments have proven invaluable for the engineers on my team" while also describing personal MacBook experiences (corporate Ventura machine and personal Sonoma device).

## Duration
Not explicitly stated.

## Key Pain Points

### System instability
- "Applications do not show up in the launcher menu" and "Applications may work or stop working randomly"
- Software randomly becomes non-functional
- "MacOS still occasionally deletes files"

### Configuration limitations
- "Many configurations...are not exposed in the Nix-darwin Module System"

### Specific broken tools
- Karabiner Elements "barely works as a packaged option"
- Firefox marked as "badPlatforms" due to compilation failures
- Deploy-rs doesn't function on macOS

### Apple-imposed restrictions
- Hostname configuration requires "pinning" rather than assignment
- Permission prompts create "fragile" system states with "virtually non-existent" rollback capability

## What They Switched To
No alternative mentioned. The author criticizes all competitors equally, stating "brew sucks. Everything else sucks."
