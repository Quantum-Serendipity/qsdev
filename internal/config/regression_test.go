package config

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestResolveConfig_ClientNotAliased(t *testing.T) {
	t.Parallel()

	project := &types.QsdevConfig{
		Client: &types.ClientConfig{
			Name:       "acme",
			Compliance: []string{"soc2"},
			AllowedMCP: []string{"context7"},
			BlockedMCP: []string{"github"},
		},
	}
	result, err := ResolveConfig(nil, project, nil)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	resolved := result.Config.Client
	if resolved == project.Client {
		t.Fatal("resolved Client aliases the project's *ClientConfig")
	}

	resolved.Name = "changed"
	resolved.BlockedMCP[0] = "changed"
	resolved.AllowedMCP[0] = "changed"
	resolved.Compliance[0] = "changed"
	if project.Client.Name != "acme" || project.Client.BlockedMCP[0] != "github" ||
		project.Client.AllowedMCP[0] != "context7" || project.Client.Compliance[0] != "soc2" {
		t.Errorf("mutating the resolved Client changed the project config: %+v", project.Client)
	}
}

func TestParseQsdevConfigBytes_RejectsExtraDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "second document", input: "version: 1\n---\nversion: 99\nbogus: 1\n", wantErr: "multiple YAML documents"},
		{name: "second document with security", input: "version: 1\n---\nsecurity:\n  level: strict\n", wantErr: "multiple YAML documents"},
		{name: "single document", input: "version: 1\n"},
		{name: "leading document marker", input: "---\nversion: 1\n"},
		{name: "comment before document marker", input: "# header\n---\nversion: 1\n"},
		{name: "empty trailing document", input: "version: 1\n---\n"},
		{name: "comment-only trailing document", input: "version: 1\n---\n# nothing here\n"},
		{name: "third document after empty one", input: "version: 1\n---\n---\nsecurity:\n  level: strict\n", wantErr: "multiple YAML documents"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseQsdevConfigBytes([]byte(tt.input))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ParseQsdevConfigBytes: unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseQsdevConfigBytes error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateQsdevConfig_Tier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tier    string
		wantErr bool
	}{
		{name: "unset", tier: ""},
		{name: "valid", tier: "full"},
		{name: "typo", tier: "ful", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateQsdevConfig(&types.QsdevConfig{Version: 1, Tier: tt.tier}, ValidateOptions{})
			var found bool
			for _, e := range errs {
				if e.Field == "tier" {
					found = true
				}
			}
			if found != tt.wantErr {
				t.Errorf("tier %q: validation errors = %v, want tier error: %v", tt.tier, errs, tt.wantErr)
			}
		})
	}
}

func TestCheckBinaryVersion_Prerelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		constraint string
		version    string
		wantOK     bool
	}{
		{name: "rc after floor", constraint: ">= 0.8.0", version: "0.8.1-rc1", wantOK: true},
		{name: "pseudo-version after floor", constraint: ">= 0.8.0", version: "v0.8.1-0.20260101000000-abcdef123456", wantOK: true},
		{name: "rc of the floor itself", constraint: ">= 0.8.0", version: "0.8.0-rc1", wantOK: false},
		{name: "pessimistic range", constraint: "~> 0.8", version: "0.8.2-rc1", wantOK: true},
		{name: "above range", constraint: "~> 0.8", version: "0.9.0", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckBinaryVersion(tt.constraint, tt.version)
			if (err == nil) != tt.wantOK {
				t.Errorf("CheckBinaryVersion(%q, %q) = %v, want ok=%v", tt.constraint, tt.version, err, tt.wantOK)
			}
		})
	}
}

func TestUpgradeCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		builtBy string
		want    string
	}{
		{builtBy: "nix", want: "nix flake update"},
		{builtBy: "manual", want: "go install github.com/Quantum-Serendipity/qsdev/cmd/qsdev@latest"},
		{builtBy: "goreleaser", want: ""},
		{builtBy: "make", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.builtBy, func(t *testing.T) {
			t.Parallel()
			if got := upgradeCommand(tt.builtBy); got != tt.want {
				t.Errorf("upgradeCommand(%q) = %q, want %q", tt.builtBy, got, tt.want)
			}
		})
	}

	msg := (&VersionMismatchError{BinaryVersion: "1.0.0", Constraint: ">= 2.0.0"}).Error()
	if !strings.Contains(msg, "package manager") {
		t.Errorf("Error() without an upgrade command = %q, want a generic update hint", msg)
	}
}
