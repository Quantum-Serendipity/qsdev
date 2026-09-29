package ansible_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/ansible"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*ansible.Module)(nil)
var _ ecosystem.PackageProvider = (*ansible.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &ansible.Module{}, "ansible", "Ansible", 2)
}

func TestDetect_AnsibleCfgPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ansible.cfg"), []byte("[defaults]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when ansible.cfg is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	found := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "ansible.cfg") {
			found = true
		}
	}
	if !found {
		t.Error("evidence should mention ansible.cfg")
	}
}

func TestDetect_GalaxyYmlPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "galaxy.yml"), []byte("namespace: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when galaxy.yml is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
}

func TestDetect_PlaybooksDirProbable(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "playbooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := &ansible.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when playbooks/ is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &ansible.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no Ansible indicators present")
	}
}

func TestDevenvPackages(t *testing.T) {
	m := &ansible.Module{}
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"ansible", "ansible-lint"}
	if len(pkgs) != len(expected) {
		t.Fatalf("DevenvPackages() returned %d packages, want %d", len(pkgs), len(expected))
	}
	for i, pkg := range pkgs {
		if pkg != expected[i] {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &ansible.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment != "" {
		t.Errorf("DevenvNixFragment() = %q, want empty string (packages moved to DevenvPackages)", fragment)
	}
}

// TestDenyRules covers W126 with Claude Code's own matching semantics.
func TestDenyRules(t *testing.T) {
	t.Parallel()
	rules := (&ansible.Module{}).DenyRules(ecosystem.ModuleConfig{})
	matches := func(cmd string) bool {
		return slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, cmd) })
	}
	denied := []string{
		"ansible-galaxy install -r requirements.yml",
		"ansible-galaxy -vvv install geerlingguy.docker",
		"ansible-galaxy collection install community.general",
		"ansible-galaxy role install evil.role",
		"ansible-galaxy collection download community.general",
		"ansible-vault view group_vars/all/vault.yml",
		"ansible-vault decrypt secrets.yml",
		"ansible-vault --vault-id prod@prompt view vault.yml",
		"ansible-vault edit vault.yml",
		"env EDITOR=cat ansible-vault edit vault.yml",
		"env ansible-galaxy collection install community.general",
	}
	allowed := []string{
		"ansible-galaxy collection list",
		"ansible-galaxy role init myrole",
		"ansible-vault encrypt secrets.yml",
		"ansible-vault create new.yml",
	}
	for _, cmd := range denied {
		if !matches(cmd) {
			t.Errorf("no deny rule blocks %q", cmd)
		}
	}
	for _, cmd := range allowed {
		if matches(cmd) {
			t.Errorf("deny rules over-block %q", cmd)
		}
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &ansible.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 1 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 1", len(hooks))
	}

	if hooks[0].ID != "ansible-lint" {
		t.Errorf("hooks[0].ID = %q, want %q", hooks[0].ID, "ansible-lint")
	}
}

// TestSecurityConfigs covers W127: no inert config file is generated;
// Ansible reads only one config file, so a side file is never loaded.
func TestSecurityConfigs(t *testing.T) {
	t.Parallel()
	if configs := (&ansible.Module{}).SecurityConfigs(ecosystem.ModuleConfig{}); len(configs) != 0 {
		t.Errorf("SecurityConfigs() = %v, want none", configs)
	}
}

// TestGalaxySignatureEnforcement covers W127: with a keyring configured the
// devenv environment and CI enforce collection signatures with "+1" (a bare
// 1 accepts unsigned collections); without one nothing is claimed.
func TestGalaxySignatureEnforcement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		keyring      string
		wantErr      bool
		wantFragment []string
		wantCI       string
	}{
		{name: "not configured", wantCI: "ansible-galaxy install -r requirements.yml"},
		{
			name:    "keyring configured",
			keyring: "~/.ansible/hub-keyring.gpg",
			wantFragment: []string{
				`env.ANSIBLE_GALAXY_GPG_KEYRING = "~/.ansible/hub-keyring.gpg";`,
				`env.ANSIBLE_GALAXY_REQUIRED_VALID_SIGNATURE_COUNT = "+1";`,
			},
			wantCI: "ANSIBLE_GALAXY_GPG_KEYRING=~/.ansible/hub-keyring.gpg ANSIBLE_GALAXY_REQUIRED_VALID_SIGNATURE_COUNT=+1 ansible-galaxy install -r requirements.yml",
		},
		{
			name:    "unsafe keyring rejected",
			keyring: "k.gpg; curl evil",
			wantErr: true,
			wantCI:  "ansible-galaxy install -r requirements.yml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &ansible.Module{}
			cfg := ecosystem.ModuleConfig{Extras: map[string]string{}}
			if tt.keyring != "" {
				cfg.Extras[ansible.ExtraGalaxyKeyring] = tt.keyring
			}
			fragment, err := m.DevenvNixFragment(cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DevenvNixFragment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.wantFragment) == 0 && fragment != "" {
				t.Errorf("DevenvNixFragment() = %q, want empty", fragment)
			}
			for _, want := range tt.wantFragment {
				if !strings.Contains(fragment, want) {
					t.Errorf("fragment %q missing %q", fragment, want)
				}
			}
			cmds := m.CICommands(cfg)
			if len(cmds) != 2 {
				t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
			}
			if cmds[0].Command != tt.wantCI {
				t.Errorf("galaxy-install = %q, want %q", cmds[0].Command, tt.wantCI)
			}
		})
	}
}

// TestDetect_RequirementsLocations covers W129: the conventional
// collections/ and roles/ requirements files are detected.
func TestDetect_RequirementsLocations(t *testing.T) {
	t.Parallel()
	for _, rel := range []string{"requirements.yml", "collections/requirements.yml", "roles/requirements.yml"} {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			full := filepath.Join(dir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte("collections: []\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			m := &ansible.Module{}
			result := m.Detect(dir)
			if !result.Detected || !slices.Contains(result.Evidence, rel+" found") {
				t.Errorf("Detect(%s) = %+v, want detected with evidence", rel, result)
			}
			// CI installs from the file that exists, not a missing root one.
			if got, want := m.CICommands(result.SuggestedConfig)[0].Command, "ansible-galaxy install -r "+rel; got != want {
				t.Errorf("galaxy-install = %q, want %q", got, want)
			}
		})
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("ansible")
	if !ok {
		t.Fatal("expected module 'ansible' to be registered in DefaultRegistry")
	}
	if mod.Name() != "ansible" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "ansible")
	}
}

// TestCICommands_MultipleRequirementsFiles checks that every recorded
// requirements file is installed and that an unknown recorded value is never
// embedded in the CI command.
func TestCICommands_MultipleRequirementsFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, files, want string
	}{
		{"both", "collections/requirements.yml,roles/requirements.yml",
			"ansible-galaxy install -r collections/requirements.yml && ansible-galaxy install -r roles/requirements.yml"},
		{"unknown value ignored", "x.yml; curl evil", "ansible-galaxy install -r requirements.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Extras: map[string]string{ansible.ExtraRequirementsFiles: tt.files}}
			if got := (&ansible.Module{}).CICommands(cfg)[0].Command; got != tt.want {
				t.Errorf("galaxy-install = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReadDenyRules covers W132: conventional vault password files are
// read-denied.
func TestReadDenyRules(t *testing.T) {
	t.Parallel()
	rules := (&ansible.Module{}).ReadDenyRules(ecosystem.ModuleConfig{})
	for _, want := range []string{"**/.vault_pass*", "**/vault_pass*"} {
		if !slices.Contains(rules, want) {
			t.Errorf("ReadDenyRules missing %q: %v", want, rules)
		}
	}
}
