<!-- Source: https://github.com/OWASP/ASVS -->
<!-- Retrieved: 2026-05-12 -->

# OWASP ASVS: Key Information

## Verification Levels

The standard organizes requirements by chapter and section for progressive security assessment. Three levels: L1 (minimum), L2 (standard), L3 (advanced).

## Chapter Structure
Requirements are organized hierarchically. The identifier format is `<chapter>.<section>.<requirement>`. For example, all `1.#.#` requirements belong to the "Encoding and Sanitization" chapter, with subsections like "Injection Prevention" (section 2). A specific example given: requirement `1.2.5` addresses OS command injection protection through parameterized queries.

## Machine-Readable Formats
Version 5.0.0 is available in three formats:
- **PDF** (English and translations in Turkish, Russian, French, Korean)
- **Word document** (.docx)
- **CSV** format for programmatic access

Additionally available in JSON format for programmatic integration.

## Requirement Numbering Scheme
The versioned reference format is `v<version>-<chapter>.<section>.<requirement>`. The page notes: "As the standard grows and changes this becomes problematic, which is why writers or developers should include the version element." This allows precise historical reference across ASVS iterations.

## Compliance & Reporting Use
The standard serves as "an open application security standard for web apps and web services," enabling organizations to verify security controls during design, development, and testing phases.

**Current Version:** 5.0.0 (released May 2025)
