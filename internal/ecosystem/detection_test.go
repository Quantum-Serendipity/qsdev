package ecosystem_test

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
)

func TestAggregateDetections_Go(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "go",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Version: "1.22",
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	p := summary.Project

	if !p.HasGoMod {
		t.Error("HasGoMod should be true")
	}
	if p.GoVersion != "1.22" {
		t.Errorf("GoVersion = %q, want %q", p.GoVersion, "1.22")
	}
	if !p.Ecosystems["go"] {
		t.Error("Ecosystems[go] should be true")
	}
}

func TestAggregateDetections_JavaScript(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "javascript",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Version:        "22",
					PackageManager: "pnpm",
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	p := summary.Project

	if !p.HasPackageJSON {
		t.Error("HasPackageJSON should be true")
	}
	if p.NodeVersion != "22" {
		t.Errorf("NodeVersion = %q, want %q", p.NodeVersion, "22")
	}
	if p.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q", p.PackageManager, "pnpm")
	}
}

func TestAggregateDetections_Python(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "python",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Version: "3.12",
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	p := summary.Project

	if !p.HasPyProject {
		t.Error("HasPyProject should be true")
	}
	if p.PythonVersion != "3.12" {
		t.Errorf("PythonVersion = %q, want %q", p.PythonVersion, "3.12")
	}
}

func TestAggregateDetections_Rust(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "rust",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasCargoToml {
		t.Error("HasCargoToml should be true")
	}
}

func TestAggregateDetections_JavaMaven(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "java",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Extras: map[string]string{"build_tool": "maven"},
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasPomXML {
		t.Error("HasPomXML should be true for maven")
	}
	if summary.Project.HasBuildGradle {
		t.Error("HasBuildGradle should be false for maven")
	}
}

func TestAggregateDetections_JavaGradle(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "java",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Extras: map[string]string{"build_tool": "gradle"},
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if summary.Project.HasPomXML {
		t.Error("HasPomXML should be false for gradle")
	}
	if !summary.Project.HasBuildGradle {
		t.Error("HasBuildGradle should be true for gradle")
	}
}

func TestAggregateDetections_JavaBoth(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "java",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Extras: map[string]string{"build_tool": "both"},
				},
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasPomXML {
		t.Error("HasPomXML should be true for both")
	}
	if !summary.Project.HasBuildGradle {
		t.Error("HasBuildGradle should be true for both")
	}
}

func TestAggregateDetections_Dotnet(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "dotnet",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasCsproj {
		t.Error("HasCsproj should be true")
	}
}

func TestAggregateDetections_Docker(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "docker",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasDockerfile {
		t.Error("HasDockerfile should be true")
	}
}

func TestAggregateDetections_Terraform(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "terraform",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.HasTerraform {
		t.Error("HasTerraform should be true")
	}
}

func TestAggregateDetections_UnknownModuleGoesToEcosystemsMap(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "elixir",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if !summary.Project.Ecosystems["elixir"] {
		t.Error("Ecosystems[elixir] should be true for unknown module")
	}
	// Known fields should be unaffected.
	if summary.Project.HasGoMod {
		t.Error("HasGoMod should be false")
	}
}

func TestAggregateDetections_NotDetectedOmitted(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "go",
			DetectResult: ecosystem.DetectionResult{
				Detected: false,
			},
		},
		&ecosystem.MockModule{
			NameVal: "python",
			DetectResult: ecosystem.DetectionResult{
				Detected: false,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	if summary.Project.HasGoMod {
		t.Error("HasGoMod should be false when not detected")
	}
	if summary.Project.HasPyProject {
		t.Error("HasPyProject should be false when not detected")
	}
	if len(summary.Project.Ecosystems) != 0 {
		t.Errorf("Ecosystems should be empty, got %v", summary.Project.Ecosystems)
	}
}

func TestAggregateDetections_MultipleEcosystems(t *testing.T) {
	r := newTestRegistry(t,
		&ecosystem.MockModule{
			NameVal: "go",
			DetectResult: ecosystem.DetectionResult{
				Detected:        true,
				Confidence:      ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{Version: "1.22"},
			},
		},
		&ecosystem.MockModule{
			NameVal: "javascript",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
				SuggestedConfig: ecosystem.ModuleConfig{
					Version:        "22",
					PackageManager: "npm",
				},
			},
		},
		&ecosystem.MockModule{
			NameVal: "docker",
			DetectResult: ecosystem.DetectionResult{
				Detected:   true,
				Confidence: ecosystem.ConfidenceCertain,
			},
		},
		&ecosystem.MockModule{
			NameVal: "rust",
			DetectResult: ecosystem.DetectionResult{
				Detected: false,
			},
		},
	)

	summary := r.DetectAll("/tmp")
	p := summary.Project

	if !p.HasGoMod {
		t.Error("HasGoMod should be true")
	}
	if !p.HasPackageJSON {
		t.Error("HasPackageJSON should be true")
	}
	if !p.HasDockerfile {
		t.Error("HasDockerfile should be true")
	}
	if p.HasCargoToml {
		t.Error("HasCargoToml should be false (rust not detected)")
	}

	// 4 entries: go, javascript, node (alias for javascript), docker
	if len(p.Ecosystems) != 4 {
		t.Errorf("Ecosystems count = %d, want 4", len(p.Ecosystems))
	}
	for _, name := range []string{"go", "javascript", "node", "docker"} {
		if !p.Ecosystems[name] {
			t.Errorf("Ecosystems[%s] should be true", name)
		}
	}
}
