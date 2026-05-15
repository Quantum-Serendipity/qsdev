# GitHub Releases Are Where SBOMs Go to Die

- **Source**: https://sbom-insights.dev/posts/github-releases-are-where-sboms-goto-die/
- **Retrieved**: 2026-05-15

## Core Problem

The article identifies a critical inefficiency in Software Bill of Materials (SBOM) management. Organizations generate SBOMs and store them in GitHub releases, but retrieving and integrating these documents into security platforms requires manual intervention at multiple stages.

**Key Pain Points:**
- Time-consuming manual downloads and uploads
- Error-prone human handling
- Inability to scale alongside accelerating release cycles
- Lack of automated workflow integration

## The Manual Workflow Challenge

Security teams typically must:
1. Search GitHub releases for SBOM artifacts
2. Download files locally
3. Re-upload to SBOM management platforms like Dependency-Track
4. Repeat this process across multiple repositories and releases

As the article states: "Manual workflow is widespread across open-source projects, enterprises, and regulated industries, where software security and compliance are critical."

## Proposed Solution: sbommv

Interlynk developed **sbommv**, a tool leveraging modular input/output adapters to automate SBOM transfers. The tool supports:

### Three Primary Use Cases

**1. GitHub API Method**
- Automatically fetches SBOMs from repository default branches
- Converts formats to CycloneDX
- Auto-creates projects in Dependency-Track with metadata

**2. Pre-existing Local SBOMs**
- Transfers SBOM collections from local folders
- Automatically upgrades SPDX 2.2 to 2.3 format
- Generates project names from component metadata

**3. Dry-Run Preview Mode**
- Validates transfers before execution
- Lists detected SBOMs and format details
- Prevents errors through pre-execution verification

## Installation & Command Examples

```bash
brew tap interlynk-io/interlynk
brew install sbommv
```

**GitHub to Dependency-Track transfer:**
```bash
sbommv transfer \
--input-adapter=github \
--in-github-url="https://github.com/interlynk-io/sbommv" \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081"
```

**Folder to Dependency-Track transfer:**
```bash
sbommv transfer \
--input-adapter=folder \
--in-folder-path="demo" \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081"
```

**Dry-run validation:**
```bash
sbommv transfer \
--input-adapter=github \
--in-github-url="[repository]" \
--output-adapter=dtrack \
--out-dtrack-url="http://localhost:8081" \
--dry-run
```

## Regulatory Context

The article contextualizes SBOM importance through recent policy developments: the 2021 U.S. Cybersecurity Executive Order and subsequent regulations from the EU, Germany, and India establishing SBOM requirements as mandatory compliance standards.

## Future Roadmap

Upcoming features include:
- Automated folder monitoring with continuous SBOM uploads
- Expanded support for S3 buckets and additional security tools
- Enhanced SBOM format conversions and validation logging

## Key Recommendation

Organizations should transition from manual SBOM handling to automated workflows. The article emphasizes: "The shift towards automated SBOM management isn't just a convenience—it's a necessity."
