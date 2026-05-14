<!-- Source: https://www.theregister.com/2024/05/14/nix_forked_but_over_politics/ -->
<!-- Retrieved: 2026-03-20 -->

# Nix forked, but over politics instead of progress

**Source:** The Register
**Date:** May 14, 2024

## Timeline of Events

### 2024 - Community Crisis Period
- Anonymous open letter published calling to "save Nix together" via save-nix-together.org
- Eelco Dolstra (Nix creator and foundation head) responded with "On community in Nix" post
- Response poorly received by the community
- Fork announcement: Auxolotl (aux.computer) created as alternative
- Dolstra subsequently stepped down from leadership

## Key Players

### Leadership
- **Eelco Dolstra**: Nix creator and original foundation head; resigned following backlash
- **Michael Brantley**: CTO of Flox (Nix vendor); praised Nix's packaging approach

### Controversial Sponsor
- **Palmer Luckey's Anduril Industries**: VR and combat-drone manufacturer that offered to sponsor NixCon 2023, generating significant controversy due to military weaponry connections

## What Caused the Fork
The split centered on moderation, governance, and funding decisions rather than technical disagreements. The anonymous letter criticized community management approaches, while the response from leadership proved divisive enough to trigger a formal fork under the Auxolotl name.

## Impact Assessment

### Adoption Concerns
Nix remains niche within Linux packaging despite financial backing. Mainstream adoption favors containerized alternatives like Red Hat's Flatpak and Canonical's Snappy.

### Community Fragmentation
The fork fragments the already-small user base without addressing underlying technical objections about filesystem usability.

## Technical Criticism
The article identifies Nix's most significant user-facing problem: unreadable filesystem hierarchies. The /nix/store/ directory uses cryptographic hashes like nqi39ksavkfrxkrz3d0797n5wmzi9r30-go-1.16.15, forcing users to trust automated systems without comprehension -- problematic if software fails.

## Overall Ecosystem Assessment
The fork addressed political grievances but missed a critical opportunity to solve Nix's core usability barrier. Until filesystem readability improves, adoption will likely remain confined to specialized communities despite the technology's genuine packaging advantages.
