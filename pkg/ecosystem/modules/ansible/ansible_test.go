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
		// U11-17: vault secrets printed through ad-hoc modules, and ad-hoc
		// command execution on inventory hosts.
		"ansible localhost -m debug -a var=x -e @vault.yml --vault-password-file .vp",
		"ansible localhost -m debug -a var=x",
		"ansible all -m shell -a id",
		"ansible all -m ansible.builtin.shell -a id",
		"ansible all -m command -a id",
		"ansible all -m ansible.builtin.command -a id",
		"ansible all -m raw -a id",
		"ansible all -m ansible.builtin.raw -a id",
		"ansible all --module-name shell -a id",
		"ansible all --module-name=ansible.builtin.raw -a id",
		"ansible all --module-name debug -a var=x",
		"env ANSIBLE_CONFIG=x ansible all -m shell -a id",
		"ansible-playbook site.yml --vault-id prod@prompt",
		"ansible-playbook site.yml --vault-password-file .vp",
		"ansible-playbook site.yml --vault-pass-file .vp",
		"ansible-playbook --ask-vault-pass site.yml",
		"env X=1 ansible-playbook site.yml --vault-id prod@prompt",
		"ansible all -m ping --ask-vault-pass",
		// Ad-hoc arguments run the default command module or feed a
		// command-running module.
		"ansible all -a id",
		"ansible all -a 'rm -rf /tmp/x'",
		"ansible all --args id",
		"ansible all --args=id",
		"ansible all -aid",
		"env X=1 ansible all -a id",
		// Module spellings without -a: no space, collection-qualified, and
		// the other command-running modules.
		"ansible all -mshell -o",
		"ansible all -m ansible.legacy.shell -o",
		"ansible localhost -m ansible.legacy.debug -e @vault.yml",
		"ansible all -m script -o",
		"ansible all -m ansible.builtin.script -o",
		"ansible all -m expect -o",
		"ansible all -m win_shell -o",
		"ansible all -m ansible.windows.win_command -o",
		"ansible all --module-name=raw -o",
		"ansible all --module-name ansible.builtin.debug -o",
		"ansible all -mansible.builtin.shell -o",
		"ansible all -m community.windows.win_shell -o",
		// -J is --ask-vault-pass; --ask-vault-password is its alias.
		"ansible-playbook site.yml -J",
		"ansible all -m ping -J",
		"ansible-playbook site.yml --ask-vault-password",
		// ansible-pull clones a playbook repository by URL and runs it.
		"ansible-pull -U https://example.com/repo.git",
		"ansible-pull",
		"env X=1 ansible-pull -U https://example.com/repo.git",
	}
	allowed := []string{
		"ansible-playbook --syntax-check site.yml",
		"ansible-playbook -i inventories/dev site.yml --check",
		"ansible all -m ping",
		"ansible all -m setup",
		"ansible-galaxy collection list",
		"ansible-galaxy role init myrole",
		"ansible-vault encrypt secrets.yml",
		"ansible-vault create new.yml",
		"ansible all -m ping -o",
		"ansible all -m ansible.builtin.ping",
		"ansible all -m setup --tree out",
		"ansible-playbook -i inventories/dev site.yml --check -e app=web",
		"ansible-playbook site.yml -e target=shell -v",
		"ansible-inventory --list",
		"ansible all -m ping --limit webshell -o",
		"ansible all -m ping -e x=debug -o",
		"ansible all -m ping --limit rawhosts -o",
	}
	for _, cmd := range denied {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("no deny rule blocks %q", cmd)
		}
	}
	for _, cmd := range allowed {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("deny rule %q over-blocks %q", rule, cmd)
		}
	}
	// The generated settings carry the literal rule the U11-WS1 acceptance
	// greps for.
	if !slices.Contains(rules, "Bash(ansible * -m shell *)") {
		t.Errorf("DenyRules lacks %q", "Bash(ansible * -m shell *)")
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
