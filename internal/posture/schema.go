package posture

// SchemaChangeLog documents the evolution of the PostureReport schema. Minor
// versions are backward compatible: readers accept any report with the same
// major version and ignore fields they do not know.
const SchemaChangeLog = `Schema Version History:
  1.0.0 (initial) - PostureReport with score, conformance, defense, config,
                     dependencies, drift, tools, ecosystems sections.
  1.1.0           - Optional policyPosture, packageRiskPosture and
                     mcpTrustPosture sections; optional repository field.
  1.2.0           - dependencies.status ("scanned", "unscanned",
                     "scan-failed", "not-applicable"); dependencies.score and
                     score.depHealth are null when the dependencies were not
                     scanned, and score.total then weighs defense and config
                     only; conformance levels and checks carry a status
                     ("pass", "fail", "unknown"), with pass true only for
                     "pass". Readers of older reports derive the statuses from
                     the pass and scanned/scanFailed flags.`
