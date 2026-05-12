# SLSA Build Track Specification
- **Source**: https://slsa.dev/spec/v1.2/build-track-basics
- **Retrieved**: 2026-05-12

## Build L0: No Guarantees
**Requirements:** None
**Use case:** Development or test builds on single machines
No security protections; represents baseline lack of SLSA compliance.

## Build L1: Provenance Exists
**Key requirement:** "Package has provenance showing how it was built"

**Software producer requirements:**
- Maintain consistent build processes
- Use L1-compliant build platforms
- Distribute provenance to consumers

**Build platform requirements:**
- Automatically generate provenance documenting the builder, build process, and top-level inputs

**Benefits:** Enables debugging, patch management, and software inventory tracking; prevents release process mistakes through verification

## Build L2: Hosted Build Platform
**Key requirement:** Builds run on hosted platforms that "generate and sign the provenance itself"

**Additional requirements beyond L1:**
- Use hosted build platforms meeting L2 standards
- Consumers must validate provenance authenticity

**Benefits:** Prevents post-build tampering through digital signatures; deters adversaries facing legal/financial consequences; enables large-scale team migration

## Build L3: Hardened Builds
**Focus:** "Prevents tampering during the build — by insider threats, compromised credentials, or other tenants"

**Additional requirements beyond L2:**
- Implement controls preventing runs from influencing one another
- Isolate secret material from user-defined build steps

**Benefits:** Comprehensive tamper protection during build execution; reduces impact of compromised credentials
