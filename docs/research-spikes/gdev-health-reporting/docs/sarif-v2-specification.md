<!-- Source: https://docs.oasis-open.org/sarif/sarif/v2.0/sarif-v2.0.html -->
<!-- Retrieved: 2026-05-12 -->

# SARIF 2.0 Core Schema Structure

## Top-Level Document Structure

The SARIF format centers on a **sarifLog object** serving as the root element. "A SARIF log file SHALL contain a serialization of the SARIF object model into the JSON format." This root object contains a runs array, which aggregates analysis results from one or more tool executions.

## Run Object Composition

Each run represents a single invocation of an analysis tool. The run object encompasses:

- **tool**: Contains driver and extension information identifying the analysis tool
- **results**: An array of detected findings
- **invocations**: Execution details about how the tool ran
- **artifacts**: Artifacts examined during analysis
- **logicalLocations**: Programmatic constructs referenced in results

## Result Object Fields

Results constitute the primary output. Key fields include:

- **ruleId**: Identifies the rule that produced the result
- **level**: Severity classification (see severity section below)
- **message**: Human-readable description of the finding
- **locations**: Physical and logical positions where issues occur
- **codeFlows**: Execution paths demonstrating the issue
- **kind**: Classification of the reporting item (problem, pass, notification, etc.)

## Severity Levels

SARIF supports a structured severity model. The specification references a level property that "SHALL" be present in result objects, supporting values that indicate problem severity and urgency. Values: error, warning, note, none.

## Location Specification

Locations employ a dual approach:

**Physical locations** reference artifacts through URI-based artifactLocation objects, optionally combined with region objects specifying precise character or line ranges within files.

**Logical locations** reference programmatic constructs (classes, methods, functions) through hierarchical naming, without specifying containing artifacts.

## Message Formatting

The message object supports both plain text and formatted presentations. Messages can incorporate placeholders for dynamic content and embedded links referencing result locations, enabling rich contextual information within structured output.
