# Sonatype Repository Firewall for Malicious Code Protection

- **Source URL**: https://www.sonatype.com/products/sonatype-repository-firewall
- **Retrieved**: 2026-05-12

## How It Works

Sonatype Firewall operates as a protective layer across your software supply chain. It uses "proprietary AI and the industry's best research" to identify threats before they reach development environments. The system functions at multiple points: blocking malicious code at the network edge, protecting repositories, and securing endpoints.

The firewall blocks threats automatically in real-time when developers or systems attempt downloads. For uncertain cases, it quarantines suspicious components and "automatically releases them if confirmed safe, reducing manual reviews."

## Supported Ecosystems

**Primary Support:**
- npm, Maven, PyPI, NuGet (Firewall Pro)
- Docker/OCI container images
- Go Modules, Rust/Cargo, PHP/Composer
- Ruby, JavaScript, Java, C#, C++, Python, Scala, Swift

**Emerging Support:**
- Hugging Face AI/ML models
- RPM packages

The platform supports "any repository" without requiring a dedicated repository manager, integrating with Nexus Repository, JFrog Artifactory, Cloudsmith, Azure Artifacts, and others.

## Threats Blocked

**Malicious Code:**
- Open source malware in packages and containers
- Supply chain attack components
- Suspicious or tampered AI/ML models

**Policy Violations:**
- Licensing compliance issues
- Vulnerable packages
- Components failing organizational standards

## Malware Detection Capabilities

- Proprietary AI analyzing component behavior
- Industry-leading open source intelligence research
- Identification of threats "others miss" through specialized malware analysis
- Real-time continuous database updates

Firewall specifically targets "intentional code crafted by attackers to cause harm" — addressing a gap where traditional SCA tools detect vulnerabilities only.

## Dependency Confusion Protection

Automated quarantine functionality and customized component controls through policy enforcement.

## Deployment Options

- Fully managed SaaS (fastest setup, minimal overhead)
- On-premises deployment
- Self-hosted configurations
- Air-gapped environments via SAGE (Sonatype Air-Gapped Environment)

## Two-Tier Approach

- **Firewall Pro:** Focuses on malicious package blocking with straightforward onboarding
- **Firewall Enterprise:** Adds full policy engine, governance workflows, waivers, and broader SDLC coverage
