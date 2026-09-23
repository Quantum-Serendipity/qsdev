package posture

// SchemaChangeLog documents the evolution of the PostureReport schema. Minor
// versions are backward compatible: readers accept any report with the same
// major version and ignore fields they do not know.
const SchemaChangeLog = `Schema Version History:
  1.0.0 (initial) - PostureReport with score, conformance, defense, config,
                     dependencies, drift, tools, ecosystems sections.
  1.1.0           - Optional policyPosture, packageRiskPosture and
                     mcpTrustPosture sections; optional repository field.`
