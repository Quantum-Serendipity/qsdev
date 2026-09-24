package sectools_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateScancodeYml_Structure(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	if f.Path != ".scancode.yml" {
		t.Errorf("Path = %q, want %q", f.Path, ".scancode.yml")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "license-compliance" {
		t.Errorf("Owner = %q, want %q", f.Owner, "license-compliance")
	}
}

// scancodePolicy mirrors the license policy file `scancode --license-policy`
// loads: a license_policies list whose entries ScanCode matches by
// license_key and copies onto each scanned file.
type scancodePolicy struct {
	LicensePolicies []struct {
		LicenseKey      string `yaml:"license_key"`
		SPDXLicenseKey  string `yaml:"spdx_license_key"`
		Label           string `yaml:"label"`
		ComplianceAlert string `yaml:"compliance_alert"`
	} `yaml:"license_policies"`
}

// loadScancodePolicy parses the generated policy and indexes it by ScanCode
// license key, failing on the duplicate keys ScanCode rejects.
func loadScancodePolicy(t *testing.T) map[string]struct{ spdx, label, alert string } {
	t.Helper()
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	var p scancodePolicy
	if err := yaml.Unmarshal(f.Content, &p); err != nil {
		t.Fatalf("policy is not valid YAML: %v\n%s", err, f.Content)
	}
	if len(p.LicensePolicies) == 0 {
		t.Fatalf("policy has no license_policies:\n%s", f.Content)
	}
	byKey := make(map[string]struct{ spdx, label, alert string }, len(p.LicensePolicies))
	for _, e := range p.LicensePolicies {
		if e.LicenseKey == "" || e.SPDXLicenseKey == "" || e.Label == "" {
			t.Errorf("incomplete policy entry %+v", e)
		}
		if _, dup := byKey[e.LicenseKey]; dup {
			t.Errorf("duplicate license_key %q (ScanCode rejects the whole policy)", e.LicenseKey)
		}
		byKey[e.LicenseKey] = struct{ spdx, label, alert string }{e.SPDXLicenseKey, e.Label, e.ComplianceAlert}
	}
	return byKey
}

// TestGenerateScancodeYml_Policy is the F218 regression: the file is a real
// ScanCode --license-policy policy, and the prohibited set covers the
// -or-later variants (GPL-2.0+, AGPL-3.0-or-later, ...) that are the most
// common declarations, not only the -only ones.
func TestGenerateScancodeYml_Policy(t *testing.T) {
	t.Parallel()
	policy := loadScancodePolicy(t)

	tests := []struct {
		key, spdx, alert string
	}{
		{"mit", "MIT", ""},
		{"apache-2.0", "Apache-2.0", ""},
		{"bsd-simplified", "BSD-2-Clause", ""},
		{"bsd-new", "BSD-3-Clause", ""},
		{"isc", "ISC", ""},
		{"lgpl-2.1", "LGPL-2.1-only", "warning"},
		{"lgpl-2.1-plus", "LGPL-2.1-or-later", "warning"},
		{"lgpl-3.0-plus", "LGPL-3.0-or-later", "warning"},
		{"mpl-2.0", "MPL-2.0", "warning"},
		{"epl-2.0", "EPL-2.0", "warning"},
		{"gpl-2.0", "GPL-2.0-only", "error"},
		{"gpl-2.0-plus", "GPL-2.0-or-later", "error"},
		{"gpl-3.0", "GPL-3.0-only", "error"},
		{"gpl-3.0-plus", "GPL-3.0-or-later", "error"},
		{"agpl-3.0", "AGPL-3.0-only", "error"},
		{"agpl-3.0-plus", "AGPL-3.0-or-later", "error"},
		{"mongodb-sspl-1.0", "SSPL-1.0", "error"},
		{"bsl-1.1", "BUSL-1.1", "error"},
	}
	for _, tt := range tests {
		t.Run(tt.spdx, func(t *testing.T) {
			t.Parallel()
			got, ok := policy[tt.key]
			if !ok {
				t.Fatalf("policy has no entry for ScanCode key %q (%s)", tt.key, tt.spdx)
			}
			if got.spdx != tt.spdx {
				t.Errorf("%s: spdx_license_key = %q, want %q", tt.key, got.spdx, tt.spdx)
			}
			if got.alert != tt.alert {
				t.Errorf("%s: compliance_alert = %q, want %q", tt.key, got.alert, tt.alert)
			}
		})
	}
}

// TestGenerateScancodeYml_AlertsAndLabels checks that every entry uses one of
// the three policy outcomes and that label and alert agree.
func TestGenerateScancodeYml_AlertsAndLabels(t *testing.T) {
	t.Parallel()
	wantLabel := map[string]string{
		"":        "Approved License",
		"warning": "Restricted License",
		"error":   "Prohibited License",
	}
	for key, e := range loadScancodePolicy(t) {
		label, ok := wantLabel[e.alert]
		if !ok {
			t.Errorf("%s: unexpected compliance_alert %q", key, e.alert)
			continue
		}
		if e.label != label {
			t.Errorf("%s: label = %q, want %q for alert %q", key, e.label, label, e.alert)
		}
	}
}

// TestGenerateScancodeYml_NoInventedKeys guards against the old invented
// format (allowed/blocked/review/paths), which ScanCode does not read.
func TestGenerateScancodeYml_NoInventedKeys(t *testing.T) {
	t.Parallel()
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(f.Content, &top); err != nil {
		t.Fatalf("policy is not valid YAML: %v", err)
	}
	if len(top) != 1 || top["license_policies"] == nil {
		keys := make([]string, 0, len(top))
		for k := range top {
			keys = append(keys, k)
		}
		t.Errorf("policy top-level keys = %v, want only license_policies", keys)
	}
}

func TestGenerateLicenseExceptionsYml_Structure(t *testing.T) {
	f, err := sectools.GenerateLicenseExceptionsYml()
	if err != nil {
		t.Fatalf("GenerateLicenseExceptionsYml() error: %v", err)
	}
	if f.Path != ".license-exceptions.yml" {
		t.Errorf("Path = %q, want %q", f.Path, ".license-exceptions.yml")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Skip {
		t.Errorf("Strategy = %v, want Skip", f.Strategy)
	}
	if f.Owner != "license-compliance" {
		t.Errorf("Owner = %q, want %q", f.Owner, "license-compliance")
	}
}

func TestGenerateLicenseExceptionsYml_Content(t *testing.T) {
	f, err := sectools.GenerateLicenseExceptionsYml()
	if err != nil {
		t.Fatalf("GenerateLicenseExceptionsYml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "exceptions: []") {
		t.Error("content should contain empty exceptions list")
	}
	if !strings.Contains(content, "justification") {
		t.Error("content should contain example with justification field")
	}
}

// TestGenerateScancodeYml_PassesRipsecrets guards against the generated
// policy tripping the ripsecrets pre-commit hook that qsdev installs: bare
// SPDX ids such as GPL-3.0-or-later under a *_key field look like secrets to
// it, which would block users from committing the file.
func TestGenerateScancodeYml_PassesRipsecrets(t *testing.T) {
	t.Parallel()
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	for i, line := range strings.Split(string(f.Content), "\n") {
		if strings.Contains(line, "spdx_license_key:") && !strings.HasSuffix(line, "# pragma: allowlist secret") {
			t.Errorf("line %d lacks the ripsecrets allowlist pragma: %q", i+1, line)
		}
	}

	bin, err := exec.LookPath("ripsecrets")
	if err != nil {
		t.Skip("ripsecrets not available")
	}
	path := filepath.Join(t.TempDir(), ".scancode.yml")
	if err := os.WriteFile(path, f.Content, 0o644); err != nil {
		t.Fatalf("writing policy: %v", err)
	}
	if out, err := exec.Command(bin, path).CombinedOutput(); err != nil {
		t.Errorf("ripsecrets flagged the generated policy: %v\n%s", err, out)
	}
}
