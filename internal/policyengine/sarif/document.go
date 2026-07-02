package sarif

import "strconv"

// SchemaURL is the canonical SARIF 2.1.0 JSON schema URL.
const SchemaURL = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"

// Version is the SARIF specification version emitted by this package.
const Version = "2.1.0"

// Log is the root of a SARIF 2.1.0 document.
type Log struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

// Run is a single analysis run within a SARIF log.
type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

// Tool identifies the analysis tool that produced a run.
type Tool struct {
	Driver Driver `json:"driver"`
}

// Driver describes the analysis tool component and the rules it can report.
type Driver struct {
	Name           string                `json:"name"`
	Version        string                `json:"version,omitempty"`
	InformationURI string                `json:"informationUri,omitempty"`
	Rules          []ReportingDescriptor `json:"rules"`
}

// ReportingDescriptor is a single rule definition in the driver's rule catalog.
type ReportingDescriptor struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name,omitempty"`
	ShortDescription     Message             `json:"shortDescription"`
	DefaultConfiguration Configuration       `json:"defaultConfiguration"`
	Properties           *DescriptorProperty `json:"properties,omitempty"`
}

// DescriptorProperty carries rule-level SARIF property-bag fields.
type DescriptorProperty struct {
	Tags             []string `json:"tags,omitempty"`
	SecuritySeverity string   `json:"security-severity,omitempty"`
}

// Configuration holds a rule's default reporting configuration.
type Configuration struct {
	Level string `json:"level"`
}

// Result is a single finding within a run.
type Result struct {
	RuleID              string            `json:"ruleId"`
	Level               string            `json:"level"`
	Message             Message           `json:"message"`
	Locations           []Location        `json:"locations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          *ResultProperty   `json:"properties,omitempty"`
}

// ResultProperty carries result-level SARIF property-bag fields.
type ResultProperty struct {
	SecuritySeverity string `json:"security-severity,omitempty"`
}

// Message is a SARIF multiformat message string.
type Message struct {
	Text string `json:"text"`
}

// Location points at an artifact affected by a result.
type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

// PhysicalLocation is the concrete artifact location of a result.
type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
}

// ArtifactLocation identifies an artifact by URI.
type ArtifactLocation struct {
	URI string `json:"uri"`
}

// BuildLog assembles a valid SARIF 2.1.0 document. The driver advertises the
// full branded rule catalog (AllRules) and the supplied findings become SARIF
// results. results is always a non-nil array so the document conforms even when
// there are no findings.
func BuildLog(driverName, driverVersion, informationURI string, findings []SarifResult) Log {
	rules := make([]ReportingDescriptor, 0, len(AllRules))
	for _, r := range AllRules {
		rules = append(rules, ReportingDescriptor{
			ID:                   r.ID,
			Name:                 r.Name,
			ShortDescription:     Message{Text: r.ShortDescription},
			DefaultConfiguration: Configuration{Level: r.DefaultLevel},
			Properties: &DescriptorProperty{
				Tags:             r.Tags,
				SecuritySeverity: formatSecuritySeverity(r.SecuritySeverity),
			},
		})
	}

	results := make([]Result, 0, len(findings))
	for _, f := range findings {
		res := Result{
			RuleID:              f.RuleID,
			Level:               f.Level,
			Message:             Message{Text: f.Message},
			PartialFingerprints: f.PartialFingerprints,
			Properties:          &ResultProperty{SecuritySeverity: formatSecuritySeverity(f.SecuritySeverity)},
		}
		if f.ArtifactURI != "" {
			res.Locations = []Location{{
				PhysicalLocation: PhysicalLocation{
					ArtifactLocation: ArtifactLocation{URI: f.ArtifactURI},
				},
			}}
		}
		results = append(results, res)
	}

	return Log{
		Schema:  SchemaURL,
		Version: Version,
		Runs: []Run{{
			Tool: Tool{Driver: Driver{
				Name:           driverName,
				Version:        driverVersion,
				InformationURI: informationURI,
				Rules:          rules,
			}},
			Results: results,
		}},
	}
}

func formatSecuritySeverity(sev float64) string {
	if sev <= 0 {
		return ""
	}
	return strconv.FormatFloat(sev, 'f', -1, 64)
}
