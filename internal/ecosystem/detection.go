package ecosystem

import "fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"

// DetectionSummary bundles per-module detection results with an aggregated
// DetectedProject that the wizard and generators consume.
type DetectionSummary struct {
	Project types.DetectedProject  `yaml:"project" json:"project"`
	Results map[string]DetectionResult `yaml:"results" json:"results"`
}

// aggregateDetections maps individual module DetectionResults into the
// well-known fields of types.DetectedProject. Modules whose names do not
// correspond to a dedicated field are recorded in the Ecosystems map.
func aggregateDetections(results map[string]DetectionResult) types.DetectedProject {
	p := types.DetectedProject{
		Ecosystems: make(map[string]bool),
	}

	for name, dr := range results {
		if !dr.Detected {
			continue
		}

		// Record every detected ecosystem in the extensible map.
		p.Ecosystems[name] = true

		// Populate well-known fields for modules that have dedicated struct fields.
		switch name {
		case "go":
			p.HasGoMod = true
			if dr.SuggestedConfig.Version != "" {
				p.GoVersion = dr.SuggestedConfig.Version
			}

		case "javascript":
			p.HasPackageJSON = true
			if dr.SuggestedConfig.Version != "" {
				p.NodeVersion = dr.SuggestedConfig.Version
			}
			if dr.SuggestedConfig.PackageManager != "" {
				p.PackageManager = dr.SuggestedConfig.PackageManager
			}

		case "python":
			p.HasPyProject = true
			if dr.SuggestedConfig.Version != "" {
				p.PythonVersion = dr.SuggestedConfig.Version
			}

		case "rust":
			p.HasCargoToml = true

		case "java":
			// Java detection may set extras to indicate the build tool.
			if dr.SuggestedConfig.Extras != nil {
				if _, ok := dr.SuggestedConfig.Extras["build_tool"]; ok {
					switch dr.SuggestedConfig.Extras["build_tool"] {
					case "maven":
						p.HasPomXML = true
					case "gradle":
						p.HasBuildGradle = true
					case "both":
						p.HasPomXML = true
						p.HasBuildGradle = true
					}
				}
			}
			// If no build tool extra is set, default to both false (just Ecosystems map).

		case "dotnet":
			p.HasCsproj = true

		case "docker":
			p.HasDockerfile = true

		case "terraform":
			p.HasTerraform = true
		}
	}

	return p
}
