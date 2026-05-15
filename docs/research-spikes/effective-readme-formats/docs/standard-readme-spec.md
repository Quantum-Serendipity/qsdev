# Standard Readme Specification
- **Source**: https://github.com/RichardLitt/standard-readme
- **Retrieved**: 2026-05-15

## Overview
The specification defines requirements for compliant README files in open source libraries across multiple languages and package managers.

## Mandatory Sections
1. **Title** — Required
2. **Short Description** — Required
3. **Table of Contents** — Required (optional for READMEs under 100 lines)
4. **Install** — Required (optional for documentation repositories)
5. **Usage** — Required (optional for documentation repositories)
6. **Contributing** — Required
7. **License** — Required

## Optional Sections
- Banner
- Badges
- Long Description
- Security
- Background
- Extra Sections
- API
- Maintainer(s)
- Thanks

## Key Requirements by Section

**Title:**
- Must match repository, folder, and package manager names, or include the actual name in italics and parentheses
- Should be self-evident

**Short Description:**
- Less than 120 characters
- Must not start with `> `
- Must be on its own line
- Must match package manager and GitHub descriptions
- No independent heading

**Long Description:**
- No independent heading
- Should explain naming discrepancies between folder/repository/package manager if they exist

**Table of Contents:**
- Must link to all sections
- Start after title/ToC, not before
- Minimum one-depth (all level-two headings required)
- May include third and fourth-level headings

**Install:**
- Code block showing installation process
- Optional "Dependencies" subsection if needed
- May include system-specific information

**Usage:**
- Code block with common usage examples
- Required CLI subsection if applicable
- Code blocks for import functionality if relevant

**Contributing:**
- State where users can ask questions
- Clarify PR acceptance policy
- List contribution requirements

**License:**
- Must state license using SPDX identifier or "UNLICENSED"
- Must state license owner
- Must be the final section

## File Naming Requirements
- Filename: `README` with appropriate extension (`.md`, `.org`, `.html`)
- For internationalization: `README.de.md` (using BCP 47 language tags)
- Non-regional language subtags prioritized
- When multiple languages exist, `README.md` reserved for English

## General Rules
- All links must function properly
- Code examples must follow project's linting standards
- Must be valid in selected format
- Section titles must appear in specified order
- Titles must be translated when README is in another language
