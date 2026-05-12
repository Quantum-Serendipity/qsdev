<!-- Source: https://discourse.nixos.org/t/how-to-do-vulnerability-scanning-with-nix-sboms/66161 -->
<!-- Retrieved: 2026-05-12 -->

# Vulnerability Scanning with Nix SBOMs - NixOS Discourse Discussion

## Original Problem
A user sought to introduce Nix-based Docker image building into their organization's container pipeline, which includes vulnerability scanning using tools like Grype. The challenge was that SBOM generators weren't detecting vulnerabilities in Python packages when integrated with standard vulnerability scanners.

Test case: `python-multipart` version 0.0.9, which contains known vulnerability GHSA-59g5-xgcq-4qw3.

## Tools Evaluated

### Bombon
Generated SBOM with Nix PURL format, but Grype found zero vulnerabilities. Issue: "Nix not yet being defined as part of the PURL specification," preventing vulnerability matching.

### Syft
Running Syft directly against Nix store path discovered zero packages — no actionable SBOM data.

### Sbomnix
Generated SBOMs and detected system-level vulnerabilities (OpenSSL, SQLite) but failed to identify Python package vulnerabilities, despite including CPE metadata that Grype doesn't utilize.

### Vulnix and Vulnxscan
Both Nix-native scanners focused exclusively on system dependencies, completely omitting Python packages from their analysis.

## Working Solution
Embedding the Python environment within a Docker container:

```nix
container = pkgs.dockerTools.buildLayeredImage {
  name = "python-multipart-container";
  contents = [python];
};
```

Running Syft against this container successfully identified Python packages. When piped through Grype, it correctly detected the target vulnerability.

## Key Insights
- Syft can discover Python packages when scanning containerized Nix environments (recognizes standard Python metadata files dist-info/METADATA)
- Direct store path scanning misses language-level packages
- The PURL specification gap for Nix is a fundamental blocker for SBOM-based vulnerability matching

## Recommendations
- Build Docker images using `dockerTools.buildLayeredImage` rather than scanning raw Nix store paths
- Run vulnerability scanners against the resulting container images
- This leverages existing SBOM tooling (Syft, Grype) without requiring specialized Nix integrations
