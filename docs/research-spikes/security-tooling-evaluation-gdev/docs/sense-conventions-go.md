# Sense internal/conventions/conventions.go
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/conventions/conventions.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code.

---

## Key Components

**Core Types:**
- `Convention`: Represents detected patterns with category, description, strength metrics, and examples
- `Example`: Individual instances showing pattern evidence
- `Category`: Enumeration of 9 detection categories (inheritance, naming, structure, composition, testing, design patterns, frameworks, architecture, key types)

**Main Detection Functions:**
The `Detect()` function orchestrates analysis by:
1. Loading symbols, edges, and files from a SQL database
2. Running specialized detectors for each convention category
3. Filtering by prevalence thresholds and strength
4. Sorting and returning results

**Pattern Detectors:**
- `detectInheritance()`: Finds classes extending common base classes
- `detectNaming()`: Identifies naming conventions (suffixes, file patterns)
- `detectStructure()`: Locates grouped symbol directories
- `detectComposition()`: Discovers mixin and serializer patterns
- `detectTesting()`: Recognizes test file naming conventions
- `detectDesignPatterns()`: Identifies service objects and similar patterns
- `detectFrameworkIdioms()`: Detects Rails concerns, callbacks, scopes; Go interfaces; React hooks; Go middleware
- `detectArchitectureLayers()`: Maps unidirectional dependencies between directories
- `detectKeyTypes()`: Highlights most-referenced domain types

**Helper Utilities:**
Database query functions, symbol indexing, example sorting/deduplication, strength calculation, and pluralization logic support the detection pipeline.
