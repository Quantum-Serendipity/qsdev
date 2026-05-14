---
source: https://raw.githubusercontent.com/openzim/sotoki/main/README.md
retrieved: 2026-05-14
type: github-readme
---

# Sotoki README

**Project Overview:**
"Sotoki" (*Stack Overflow to Kiwix*) is an openZIM scraper to create offline versions of Stack Exchange websites.

**Core Functionality:**
The tool operates using Stack Exchange Data Dumps from The Internet Archive, allowing users to generate offline-accessible ZIM files for platforms like Stack Overflow.

**Key Usage Requirements:**
Users must provide three essential parameters:
- `--mirror`: URL pointing to the Stack Exchange dump location
- `--domain`: The specific Stack Exchange site to process
- `--title`: ZIM file title (maximum 30 characters)
- `--description`: ZIM file description (maximum 80 characters)

**Installation Methods:**
Docker deployment: `docker run -v my_dir:/output ghcr.io/openzim/sotoki sotoki --help`

For Python-based installation, users should establish a virtual environment and install via pip, with the package available at PyPI.

**Development:**
Contributions are welcomed, with guidelines documented in a CONTRIBUTING.md file.

**Pre-built Resources:**
Rather than creating custom files, users may access regularly updated ZIM files at the Kiwix library's Stack Exchange category.
