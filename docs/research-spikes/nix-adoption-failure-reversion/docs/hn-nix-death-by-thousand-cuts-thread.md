# HN Thread: Nix - Death by a Thousand Cuts
- **Source URL**: https://news.ycombinator.com/item?id=42666851
- **Retrieved**: 2026-03-20
- **Type**: Hacker News discussion thread

## Complete Abandonment Cases

**foundart**
- **Duration**: ~30 minutes
- **Context**: Heard about Nix from someone who found the tools interesting
- **Reason**: "After about 30 minutes I decided it was not for me"
- **Current Status**: Hasn't touched it since, monitors Nix stories occasionally

**kstenerud**
- **Duration**: Multiple forays over a decade
- **Context**: Used various Linux distros over 30 years, attempted NixOS multiple times
- **Reasons**: Language complexity, scattered documentation, code fragments that don't fit use cases, architectural confusion
- **Quote**: "I'm done with Nix - burned one time too many"
- **Switched to**: Debian (reverted one server), considering Guix

**nothrabannosir**
- **Context**: "On the inside" with substantial Nix experience
- **Main Issue**: Language design prevents proper LSP tooling ("go to definition"), perpetuates entry barriers
- **Assessment**: Language is "close to the number one threat to wider adoption"

**cge**
- **Duration**: ~3 years
- **Use Case**: Making research code repositories reproducible
- **Result**: "everything broke within around three years"
- **Issues**: Poor documentation for channel revision pinning, opaque commit hash system

**indrora**
- **Duration**: Several hours at NixconfNA
- **Context**: Attempted on Chromebook; very limited setup
- **Result**: Couldn't even complete basic "hello world" build
- **Quote**: "The ergonomics are just that of a hiltless double bladed sword"
- **Note**: Acknowledged difficulty even though experienced developer

**_w1tm**
- **Use Case**: Rust development
- **Issues**: Couldn't resolve linker errors with macOS standard libraries; Nix fixes created different problems
- **Result**: "Everything just worked outside of the Nix environment so ended up dropping it"

## Partial Abandonment/Switching Away

**Zambyte**
- **Duration**: Several months on NixOS
- **Switched to**: GNU Guix (~4 years as of writing)
- **Reasons**: Language underdocumented, "compounding frustration" over months
- **Quote**: "Nix was a breath of fresh air, using a language with decades of academic backing"
- **Context**: Familiar with Haskell before Nix; had never used Lisp before NixOS

**tombert**
- **Current**: Still daily-drives NixOS on personal laptop (6+ years)
- **Server Context**: "I really have no desire to ever use anything but NixOS" for servers
- **Desktop Limitations**: Can't run "generic Linux programs" without workarounds (FHS environments, Flakes)
- **Complaint**: "I didn't really want to know how to make my own Nix package"
- **Not Recommended**: Too complex for non-technical users like parents, unlike Ubuntu

**emarthinsen**
- **Current**: Daily driver but reluctant
- **Statement**: "I wouldn't recommend it for most people (even for me)"
- **Alternative Preference**: "I'd probably just go Arch if I were to do it over again"
- **Issues**: Builds non-reproducible (needs multiple `nixos-rebuild switch` runs), NVIDIA complexity, Wayland issues

**yoyohello13**
- **Switched to**: Arch Linux with Nix + Home Manager
- **Reason**: Waiting for Flakes to become official; unwilling to commit to NixOS "until the experimental flag comes off"
- **Current Setup**: Syncs dotfiles across machines with Home Manager

**3836293648**
- **Issue**: NixOS broke webcam drivers on Alder Lake, wake-from-sleep functionality during security update
- **Mitigation**: Uses rollback feature rather than staying current
- **Implicit**: Will tolerate known vulnerabilities rather than risk broken state

## Switched from Nix Package Manager (not OS)

**_huayra_**
- **Status**: On fence about Nix
- **Concerns**: "Waiting-for-Godot situation for flakes," weird language, community infighting
- **Current**: Still on Tumbleweed (5+ years)
- **Considering**: Universal Blue or Guix instead

**rasmus-kirk**
- **Recommendation**: "Just use Nix/Home Manager on Ubuntu or something instead of NixOS"
- **Quote**: "NixOS feels more like a great server environment, but not that good of a DE"

## Summary Statistics
- **Total accounts documenting abandonment/avoidance**: 12
- **Time before abandonment**: 30 minutes to several years
- **Primary complaint category**: Language design/documentation issues
- **Secondary complaint**: Reproducibility claims not met
- **Tertiary complaint**: NixOS-specific (dynamic linking, generic binary incompatibility)
