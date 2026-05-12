# PEP 751: pylock.toml Format Overview

- **Source**: https://peps.python.org/pep-0751/
- **Retrieved**: 2026-05-12

## File Format Specification

**Format**: TOML (human-readable and machine-generated)

**File Naming**:
- Primary: `pylock.toml`
- Named variants: `pylock.{name}.toml` (e.g., `pylock.dev.toml`)

**Key Metadata Fields**:
- `lock-version`: Required string ("1.0")
- `created-by`: Required string identifying the tool that created the file
- `requires-python`: Optional; specifies minimum Python version compatibility
- `environments`: Optional array of environment markers
- `extras`: Optional array of supported extras
- `dependency-groups`: Optional array of dependency groups
- `[[packages]]`: Required array containing all packages that may be installed

## Hash Requirements and Storage

**Hash Implementation**:
- Hashes are stored in `[packages.archive.hashes]`, `[packages.sdist.hashes]`, and `[packages.wheels.hashes]` as tables
- Each hash entry uses the algorithm name as the key (e.g., `sha256`) and the hash value as the value
- "At least one secure algorithm from `hashlib.algorithms_guaranteed` SHOULD always be included (at time of writing, sha256 specifically is recommended)"

**Mandatory Hashing**: Hashes are required for archives, sdists, and wheels, establishing security as a default rather than an opt-in feature.

## Security Properties

**Security-First Design**:
- The format promotes "good security defaults" through mandatory hash inclusion
- File size recording enables validation against tampering
- Upload timestamps provide audit trails
- Attestation identity support for provenance verification

**Limitations Acknowledged**:
- The format does not prevent typosquatting or name confusion attacks
- Lock file tampering requires external protections (signing via external mechanisms or `[tool]` entries)
- The specification focuses on supply chain verification at the file level, not upstream validation

## Reproducible Installation

**No Resolver Required**: "Installers consuming the file should be able to calculate what to install without the need for dependency resolution at install-time," enabling deterministic, offline installations.

**Installation Steps**: The specification provides a prescriptive process where each package's marker conditions are evaluated, hashes are validated, and files are retrieved from recorded locations (URLs or paths).

**Multi-Use Capability**: Supports extras and dependency groups through enhanced marker syntax, allowing a single lock file to serve multiple use cases while maintaining reproducibility.
