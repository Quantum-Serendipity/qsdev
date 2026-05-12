<!-- Source: https://flyingcircus.io/en/about-us/blog-news/details-view/introducing-vulnix-a-vulnerability-scanner-for-nixos -->
<!-- Retrieved: 2026-05-12 -->

# Introducing Vulnix - Flying Circus Blog Post

## Design Philosophy
Vulnix reverses the approach of traditional monitoring tools like nixpkgs-monitor. Rather than helping maintainers track vulnerabilities across packages, it "answers the question, which of my current active garbage collecting roots is affected by a known CVE" for individual running systems.

## Matching Packages to CVEs

Two-step matching process:

1. **Store Analysis**: Queries the nix-store for live garbage collection roots, converting derivation files (*.drv) into Python objects with properties like name and builder.

2. **CVE Matching**: The NVD component downloads gzipped CVE files from NIST, translating XML to Common Platform Enumeration (CPE) format — "a naming scheme used by NIST to structure affected software products in categories like vendor, version and the likes."

## Handling False Positives
Includes a whitelist mechanism to exclude known false matches. Example: "the access derivation in Nix which lead to a match with Microsoft Access related entries." Filters can target CVE IDs, specific package versions, derivation properties, or vendor/product combinations.

## Production Experience
No quantitative data on false positive rates, accuracy metrics, or performance benchmarks provided. Single example output showing expat-2.1.0 matching CVE-2015-1283.
