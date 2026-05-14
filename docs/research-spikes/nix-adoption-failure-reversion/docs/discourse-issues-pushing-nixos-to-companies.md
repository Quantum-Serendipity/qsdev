# Discourse Thread: Issues When Suggesting NixOS Within Corporate Environments
- **Source URL**: https://discourse.nixos.org/t/my-issues-when-pushing-nixos-to-companies/28629
- **Retrieved**: 2026-03-20
- **Type**: NixOS Discourse discussion

## TLATER
**Company Context:** Smallish consultancy filled with Debian and GNOME maintainers
**What They Tried:** Attempted to suggest NixOS internally for infrastructure
**Objections Encountered:**
- Operations lead stated: "it's too different, I don't have the time to learn this"
- Team continued using Ansible instead
- When customers were mentioned, suggesting a "relatively obscure" technology with a "smaller community" was met with indifference
**Outcome:** NixOS rejected; company maintained Ansible-based infrastructure. Only one internal project eventually added a flake.nix, which became mostly unused after enthusiasts left the company.

## Melkor333
**Company Context:** Previous employer (size not specified)
**What They Tried:** Attempted to advocate for NixOS adoption
**Blockers Identified:**
- Lack of proper LTS release and "way too short update time"
- No company backing with open-source presence comparable to RHEL/SLES
- Flakes remain experimental, deterring enterprise recommendation
- CVE handling concerns and stopped vulnerability roundups
**Outcome:** Remained "extremely hesitant" in pushing NixOS; advocacy efforts faced cultural resistance around stability expectations.

## NobbZ
**Company Context:** International firm with large customers (one customer has 800x more employees)
**Team Size:** Operations team of 3 managing customer hosting, internal infrastructure, and employee equipment
**What They Tried:** Presented Nix(OS) to Ops team and selected staff
**Objections Encountered:**
- Service and compliance agreements with customers mandate specific platforms
- Customers only allow Ubuntu LTS across all hostings
- "completely different beast to maintain" perception
- Ubuntu LTS schedule already causes problems; "5 years seems to be too fast paced"
- Company cannot support hard switches every 6 months on all hostings
**Alternatives Used:** Ubuntu LTS; employees restricted to Mac or Windows devices
**Outcome:** NixOS usage prohibited for hosting; Nix permitted only on Mac or WSL for development.

## Nebucatnetzer
**Company Context:** 20-employee company with non-stringent support requirements
**What They Tried:** Considering NixOS adoption for company servers; exploring alternatives to Ansible
**Challenges Acknowledged:**
- Only DevOps person on team, making risky technology choices difficult
- Conventional tools allow bugs; new tools create expectation of personal problem-solving
- Hesitant about introducing another "completely different beast" to maintain
**Current Approach:** Using Nix for dev environments; deploying to Ubuntu servers instead of direct NixOS deployment
**Note:** This company appears positioned for potential adoption but faces organizational risk barriers.

## Summary
All documented failures involved resistance rooted in: stability/LTS requirements, compliance constraints, operational risk aversion, small ops teams unable to absorb learning curves, and lack of corporate backing/guarantees that enterprise clients demand.
