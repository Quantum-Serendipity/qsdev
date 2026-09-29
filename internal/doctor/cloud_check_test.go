package doctor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

func TestNewCloudSection(t *testing.T) {
	t.Parallel()

	if got := NewCloudSection(nil, nil); got != nil {
		t.Errorf("NewCloudSection(nil, nil) = %+v, want nil", got)
	}

	reports := []cloudcommon.FailSafeReport{
		cloudcommon.ValidateFailSafe(cloudcommon.AWS,
			map[string]string{"AWS_PROFILE": "dev"},
			cloudcommon.BashDenyRules(cloudcommon.AWS),
			cloudcommon.ReadDenyPaths(cloudcommon.AWS)),
		cloudcommon.ValidateFailSafe(cloudcommon.GCP,
			map[string]string{"CLOUDSDK_ACTIVE_CONFIG_NAME": "<gcloud configuration name>"},
			cloudcommon.BashDenyRules(cloudcommon.GCP),
			cloudcommon.ReadDenyPaths(cloudcommon.GCP)),
		cloudcommon.ValidateFailSafe(cloudcommon.Azure, nil, nil, nil),
	}
	cs := NewCloudSection(reports, []string{"parsing devenv.local.nix: bad"})
	if cs == nil || !cs.Detected {
		t.Fatalf("NewCloudSection() = %+v, want a detected section", cs)
	}

	tests := []struct {
		name       string
		wantStatus string
		wantLayers [3]bool
	}{
		{"aws", "isolated", [3]bool{true, true, true}},
		{"gcp", "degraded", [3]bool{false, true, true}},
		{"azure", "misconfigured", [3]bool{false, false, false}},
	}
	if len(cs.Providers) != len(tests) {
		t.Fatalf("got %d providers, want %d", len(cs.Providers), len(tests))
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := cs.Providers[i]
			if p.Name != tt.name || p.Status != tt.wantStatus {
				t.Errorf("provider = %s/%s, want %s/%s", p.Name, p.Status, tt.name, tt.wantStatus)
			}
			if len(p.Layers) != 3 {
				t.Fatalf("got %d layers, want 3", len(p.Layers))
			}
			for j, l := range p.Layers {
				if l.Active != tt.wantLayers[j] {
					t.Errorf("layer %s active = %v, want %v", l.Name, l.Active, tt.wantLayers[j])
				}
				if wantEnforced := j != 0; l.Enforced != wantEnforced {
					t.Errorf("layer %s enforced = %v, want %v", l.Name, l.Enforced, wantEnforced)
				}
			}
		})
	}
}

func TestFormatReport_CloudSection(t *testing.T) {
	t.Parallel()

	cs := NewCloudSection([]cloudcommon.FailSafeReport{
		cloudcommon.ValidateFailSafe(cloudcommon.AWS, map[string]string{"AWS_PROFILE": "PLACEHOLDER"}, nil,
			cloudcommon.ReadDenyPaths(cloudcommon.AWS)),
	}, []string{"parsing devenv.local.nix: bad"})
	r := &Report{QsdevVersion: "0.1.0"}
	r.SetCloudSection(cs)

	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	out := buf.String()
	for _, want := range []string{
		"Cloud Credential Isolation",
		"AWS",
		"[FAIL] misconfigured",
		"[WARN] environment separation: AWS_PROFILE holds a placeholder value",
		"[OK] credential file masking",
		"[FAIL] agent deny rules: missing Deny for:",
		"[WARN] parsing devenv.local.nix: bad",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}
