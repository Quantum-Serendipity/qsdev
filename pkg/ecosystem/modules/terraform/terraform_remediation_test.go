package terraform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDevenvNix_VersionPinning covers the devenv contract: only
// languages.terraform has a version option, it needs the nixpkgs-terraform
// input, and the value must be a valid, safely quoted release.
func TestDevenvNix_VersionPinning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		variant   string
		version   string
		wantLine  string
		wantInput bool
		wantErr   bool
	}{
		{name: "terraform pinned", variant: "terraform", version: "1.8.5", wantLine: `version = "1.8.5";`, wantInput: true},
		{name: "terraform v prefix", variant: "terraform", version: "v1.9", wantLine: `version = "1.9";`, wantInput: true},
		{name: "terraform unpinned", variant: "terraform"},
		{name: "opentofu has no version option", variant: "opentofu", version: "1.8.0"},
		{name: "invalid version rejected", variant: "terraform", version: `1.8"; x = "`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Version: tt.version, Extras: map[string]string{"variant": tt.variant}}
			m := &terraform.Module{}
			frag, err := m.DevenvNixFragment(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got fragment:\n%s", frag)
				}
				if in := m.DevenvYamlInputs(cfg); len(in) != 0 {
					t.Errorf("DevenvYamlInputs = %v for invalid version, want none", in)
				}
				return
			}
			if err != nil {
				t.Fatalf("DevenvNixFragment: %v", err)
			}
			if tt.wantLine == "" && strings.Contains(frag, "version") {
				t.Errorf("fragment must not set a version:\n%s", frag)
			}
			if tt.wantLine != "" && !strings.Contains(frag, tt.wantLine) {
				t.Errorf("fragment missing %q:\n%s", tt.wantLine, frag)
			}
			in := m.DevenvYamlInputs(cfg)
			gotInput := len(in) == 1 && in[0].URL == "github:stackbuilders/nixpkgs-terraform" && in[0].Follows == "nixpkgs"
			if gotInput != tt.wantInput {
				t.Errorf("DevenvYamlInputs = %v, want nixpkgs-terraform input: %v", in, tt.wantInput)
			}
		})
	}
}

// TestPreCommitHooks_ValidateInitializesFirst guards against running
// `validate` on an uninitialized working tree (fresh clone), which fails
// every commit touching .tf files.
func TestPreCommitHooks_ValidateInitializesFirst(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"terraform", "opentofu"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			bin := map[string]string{"terraform": "terraform", "opentofu": "tofu"}[variant]
			var validate *ecosystem.HookConfig
			hooks := (&terraform.Module{}).PreCommitHooks(ecosystem.ModuleConfig{Extras: map[string]string{"variant": variant}})
			for i := range hooks {
				if hooks[i].ID == "terraform-validate" {
					validate = &hooks[i]
				}
			}
			if validate == nil {
				t.Fatal("terraform-validate hook missing")
			}
			want := "sh -c '" + bin + " init -backend=false -input=false >/dev/null && " + bin + " validate'"
			if validate.Entry != want {
				t.Errorf("Entry = %q, want %q", validate.Entry, want)
			}
			// Entry rewriting prefixes the first word with
			// ${pkgs.<NixPackage>}/bin/, so the package must provide `sh`.
			if validate.NixPackage != "bash" {
				t.Errorf("NixPackage = %q, want bash (provides the `sh` the entry starts with)", validate.NixPackage)
			}
		})
	}
}

func TestSecurityConfigs_TerraformrcIsSkip(t *testing.T) {
	t.Parallel()
	for _, f := range (&terraform.Module{}).SecurityConfigs(ecosystem.ModuleConfig{}) {
		if f.Strategy != types.Skip {
			t.Errorf("%s Strategy = %v, want Skip (never replace a user CLI config)", f.Path, f.Strategy)
		}
	}
}

// TestSecretDeclarations_FollowProviders verifies AWS static keys are only
// declared for AWS-provider projects, and never as required.
func TestSecretDeclarations_FollowProviders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		tf        string
		wantNames []string
	}{
		{"gcp only", `provider "google" {}`, nil},
		{"no provider", `terraform {}`, nil},
		{"aws", `provider "aws" { region = "us-east-1" }`, []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tt.tf+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			m := &terraform.Module{}
			decls := m.SecretDeclarations(m.Detect(dir).SuggestedConfig)
			var names []string
			for _, d := range decls {
				names = append(names, d.Name)
				if d.Required {
					t.Errorf("%s is Required; static keys must be optional (profile/SSO is the isolation model)", d.Name)
				}
			}
			if strings.Join(names, ",") != strings.Join(tt.wantNames, ",") {
				t.Errorf("declared %v, want %v", names, tt.wantNames)
			}
		})
	}
}
