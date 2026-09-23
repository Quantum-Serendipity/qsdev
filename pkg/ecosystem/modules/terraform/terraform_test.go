package terraform_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*terraform.Module)(nil)

func newModule() *terraform.Module {
	return &terraform.Module{}
}

// --- Detection tests ---

func TestDetect_TfFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .tf files")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "*.tf files found") {
		t.Errorf("expected evidence about .tf files, got %v", result.Evidence)
	}
	if result.SuggestedConfig.Extras["variant"] != "terraform" {
		t.Errorf("expected variant=terraform, got %q", result.SuggestedConfig.Extras["variant"])
	}
}

func TestDetect_TfJsonFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .tf.json files")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "*.tf.json files found") {
		t.Errorf("expected evidence about .tf.json files, got %v", result.Evidence)
	}
}

// TestDetect_OpenTofu covers W128: OpenTofu is recognised by its .tofu file
// extension (it has no marker directory), and a .tofu-only project is
// detected at all.
func TestDetect_OpenTofu(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		files       []string
		mkdirs      []string
		wantVariant string
		wantFound   bool
	}{
		{name: "tofu files only", files: []string{"main.tofu"}, wantVariant: "opentofu", wantFound: true},
		{name: "tofu json", files: []string{"main.tofu.json"}, wantVariant: "opentofu", wantFound: true},
		{name: "tofu files beside tf files", files: []string{"main.tf", "override.tofu"}, wantVariant: "opentofu", wantFound: true},
		{name: "tofu files in subdirectory", files: []string{"infra/main.tofu"}, wantVariant: "opentofu", wantFound: true},
		{name: "tf files only", files: []string{"main.tf"}, wantVariant: "terraform", wantFound: true},
		{
			name:        ".opentofu directory is not a marker",
			files:       []string{"main.tf"},
			mkdirs:      []string{".opentofu"},
			wantVariant: "terraform",
			wantFound:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, d := range tt.mkdirs {
				if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for _, f := range tt.files {
				writeFile(t, dir, f, `resource "null_resource" "x" {}`)
			}
			result := newModule().Detect(dir)
			if result.Detected != tt.wantFound {
				t.Fatalf("Detected = %v, want %v (evidence %v)", result.Detected, tt.wantFound, result.Evidence)
			}
			if got := result.SuggestedConfig.Extras["variant"]; got != tt.wantVariant {
				t.Errorf("variant = %q, want %q", got, tt.wantVariant)
			}
		})
	}
}

// TestDetect_Subdirectories covers W129/W146: configuration below the root
// (infra/, terraform/) is detected with the same bounded walk cloudcommon
// uses for provider detection, and the directories are recorded.
func TestDetect_Subdirectories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     []string
		wantFound bool
		wantDirs  string
	}{
		{name: "infra subdirectory", files: []string{"infra/main.tf"}, wantFound: true, wantDirs: "infra"},
		{name: "root plus modules validates the root only", files: []string{"main.tf", "modules/aws/aws.tf"}, wantFound: true},
		{
			name:      "root plus environment",
			files:     []string{"main.tf", "envs/prod/main.tf"},
			wantFound: true,
			wantDirs:  ".,envs/prod",
		},
		{
			name:      "environments plus shared modules",
			files:     []string{"envs/prod/main.tf", "envs/dev/main.tf", "modules/vpc/main.tf", "infra/modules/db/main.tf"},
			wantFound: true,
			wantDirs:  "envs/dev,envs/prod",
		},
		{name: "modules-only repository", files: []string{"modules/vpc/main.tf"}, wantFound: true, wantDirs: "modules/vpc"},
		{name: "root only records nothing", files: []string{"main.tf"}, wantFound: true},
		{name: "lock file in subdirectory", files: []string{"infra/.terraform.lock.hcl"}, wantFound: true},
		{name: "provider cache ignored", files: []string{".terraform/modules/x/main.tf"}},
		{name: "too deep ignored", files: []string{"a/b/c/d/main.tf"}},
		{name: "unsafe directory name not recorded", files: []string{"in fra/main.tf"}, wantFound: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				writeFile(t, dir, f, `provider "aws" {}`)
			}
			result := newModule().Detect(dir)
			if result.Detected != tt.wantFound {
				t.Fatalf("Detected = %v, want %v (evidence %v)", result.Detected, tt.wantFound, result.Evidence)
			}
			if got := result.SuggestedConfig.Extras[terraform.ExtraConfigDirs]; got != tt.wantDirs {
				t.Errorf("%s = %q, want %q", terraform.ExtraConfigDirs, got, tt.wantDirs)
			}
		})
	}
}

// TestDetect_ImpliedByCloudProvider covers W146: whenever cloudcommon sees a
// provider in .tf files, the terraform module is detected too.
func TestDetect_ImpliedByCloudProvider(t *testing.T) {
	t.Parallel()
	for _, rel := range []string{"main.tf", "infra/main.tf", "infra/modules/aws/main.tf", "terraform/main.tofu"} {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, dir, rel, `provider "aws" {}`)
			if len(cloudcommon.DetectTerraformProviders(dir)) == 0 {
				t.Fatal("precondition: cloudcommon found no provider")
			}
			if !newModule().Detect(dir).Detected {
				t.Error("cloud provider detected from Terraform files but terraform module not detected")
			}
		})
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetect_LockfileOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".terraform.lock.hcl"), []byte("# lockfile"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when only lockfile is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable for lockfile-only, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, ".terraform.lock.hcl found") {
		t.Errorf("expected evidence about lockfile, got %v", result.Evidence)
	}
}

func TestDetect_Empty(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("expected ConfidenceAbsent, got %v", result.Confidence)
	}
	// Extras map should still be initialized.
	if result.SuggestedConfig.Extras == nil {
		t.Error("expected Extras map to be initialized, got nil")
	}
}

// --- DevenvNix tests ---

func TestDevenvNix_Terraform(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected fragment to contain languages.terraform, got:\n%s", fragment)
	}
	if strings.Contains(fragment, "languages.opentofu") {
		t.Errorf("expected fragment not to contain languages.opentofu, got:\n%s", fragment)
	}
}

func TestDevenvNix_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.opentofu") {
		t.Errorf("expected fragment to contain languages.opentofu, got:\n%s", fragment)
	}
	if strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected fragment not to contain languages.terraform, got:\n%s", fragment)
	}
}

func TestDevenvNix_WithVersion(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Version: "1.8.0",
		Extras:  map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, `version = "1.8.0"`) {
		t.Errorf("expected fragment to contain version = \"1.8.0\", got:\n%s", fragment)
	}
}

func TestDevenvNix_DefaultVariant(t *testing.T) {
	m := newModule()
	// No variant set, should default to terraform.
	config := ecosystem.ModuleConfig{}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected default variant to produce languages.terraform, got:\n%s", fragment)
	}
}

func TestDevenvNix_NoVersion(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(fragment, "version") {
		t.Errorf("expected no version line when Version is empty, got:\n%s", fragment)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Basic(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{}
	files := m.SecurityConfigs(config)

	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}

	f := files[0]
	if f.Path != ".terraformrc" {
		t.Errorf("expected path .terraformrc, got %q", f.Path)
	}

	content := string(f.Content)
	if !strings.Contains(content, "disable_checkpoint = true") {
		t.Error("expected disable_checkpoint = true in content")
	}
	if !strings.Contains(content, "Terraform >= 0.13") {
		t.Error("expected version requirement comment for Terraform >= 0.13")
	}
	if strings.Contains(content, "provider_installation {") {
		t.Error("expected no provider_installation block without registry_mirror")
	}
}

func TestSecurityConfigs_WithMirror(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"registry_mirror": "https://mirror.example.com/providers/",
		},
	}
	files := m.SecurityConfigs(config)

	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "disable_checkpoint = true") {
		t.Error("expected disable_checkpoint = true in content")
	}
	if !strings.Contains(content, "provider_installation") {
		t.Error("expected provider_installation block with registry_mirror set")
	}
	if !strings.Contains(content, "network_mirror") {
		t.Error("expected network_mirror block")
	}
	if !strings.Contains(content, "https://mirror.example.com/providers/") {
		t.Error("expected mirror URL in content")
	}
	if !strings.Contains(content, `exclude = ["registry.terraform.io/*/*"]`) {
		t.Error("expected direct exclude for registry.terraform.io")
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	hooks := m.PreCommitHooks(config)

	if len(hooks) != 4 {
		t.Fatalf("expected 4 hooks, got %d", len(hooks))
	}

	// Real git-hooks.nix hook names are hyphenated (terraform-format /
	// terraform-validate); underscored names do not exist as built-ins.
	expectedIDs := []string{"terraform-format", "terraform-validate", "tflint", "tfsec"}
	for i, id := range expectedIDs {
		if hooks[i].ID != id {
			t.Errorf("hook[%d]: expected ID %q, got %q", i, id, hooks[i].ID)
		}
	}

	// All four are custom hooks (BuiltIn:false): the built-in terraform-format
	// would discard the tofu/terraform binary selection and -check flags, so the
	// hooks are rendered with a NixPackage that puts the binary on PATH.
	// terraform-validate chains init and validate through `sh -c`, so its
	// NixPackage provides `sh` rather than the Terraform binary.
	for i, h := range hooks {
		if h.BuiltIn {
			t.Errorf("%s should not be BuiltIn (custom hook preserving entry)", h.ID)
		}
		if h.NixPackage == "" {
			t.Errorf("%s should set a NixPackage so its binary resolves", hooks[i].ID)
		}
	}
	// Default variant resolves to the terraform package.
	if hooks[0].NixPackage != "terraform" {
		t.Errorf("terraform-format NixPackage = %q, want terraform", hooks[0].NixPackage)
	}

	// Default variant should use "terraform" in entry.
	if !strings.Contains(hooks[0].Entry, "terraform") {
		t.Errorf("expected terraform in fmt entry, got %q", hooks[0].Entry)
	}
	if !strings.Contains(hooks[1].Entry, "terraform") {
		t.Errorf("expected terraform in validate entry, got %q", hooks[1].Entry)
	}
}

func TestPreCommitHooks_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	hooks := m.PreCommitHooks(config)

	if len(hooks) != 4 {
		t.Fatalf("expected 4 hooks, got %d", len(hooks))
	}

	// OpenTofu variant should use "tofu" in entry for fmt and validate.
	if !strings.Contains(hooks[0].Entry, "tofu") {
		t.Errorf("expected tofu in fmt entry for opentofu variant, got %q", hooks[0].Entry)
	}
	if !strings.Contains(hooks[1].Entry, "tofu") {
		t.Errorf("expected tofu in validate entry for opentofu variant, got %q", hooks[1].Entry)
	}
}

// --- DenyRules tests ---

// TestDenyRules covers W125 with Claude Code's own matching semantics:
// destructive, lock-bypassing and secret-printing subcommands are denied for
// both binaries, including after global options such as -chdir, while
// read-only plan/validate runs stay allowed.
func TestDenyRules(t *testing.T) {
	t.Parallel()
	denied := []string{
		"terraform init",
		"terraform init -upgrade",
		"terraform -chdir=infra init -upgrade",
		"terraform apply -auto-approve",
		"terraform -chdir=infra apply -auto-approve",
		"terraform -chdir=infra -input=false apply -destroy",
		"terraform destroy -auto-approve",
		"terraform -chdir=infra destroy",
		"terraform get -update",
		"terraform import aws_s3_bucket.b bucket",
		"terraform force-unlock 1234",
		"terraform providers lock",
		"terraform state pull",
		"terraform -chdir=infra state pull",
		"terraform state rm aws_s3_bucket.b",
		"terraform output -json",
		"terraform -chdir=infra output -raw db_password",
		"terraform show -json",
		"terraform show -no-color -json plan.out",
		"tofu apply -auto-approve",
		"tofu -chdir=infra destroy",
		"tofu state pull",
	}
	allowed := []string{
		"terraform plan",
		"terraform -chdir=infra plan -out=tfplan",
		"terraform plan -var initial_count=1",
		"terraform plan -target=module.getter",
		"terraform validate",
		"terraform fmt -check -recursive",
		"terraform state list",
		"terraform show",
		"tofu -chdir=infra validate",
	}
	for _, variant := range []string{"terraform", "opentofu"} {
		rules := newModule().DenyRules(ecosystem.ModuleConfig{Extras: map[string]string{"variant": variant}})
		matches := func(cmd string) bool {
			return slices.ContainsFunc(rules, func(r string) bool { return denyutil.MatchesBashRule(r, cmd) })
		}
		for _, cmd := range denied {
			if !matches(cmd) {
				t.Errorf("variant %s: no deny rule blocks %q", variant, cmd)
			}
		}
		for _, cmd := range allowed {
			if matches(cmd) {
				t.Errorf("variant %s: deny rules over-block %q", variant, cmd)
			}
		}
	}
}

// TestReadDenyRules covers W132: state, variable files and the registry
// token are read-denied.
func TestReadDenyRules(t *testing.T) {
	t.Parallel()
	rules := newModule().ReadDenyRules(ecosystem.ModuleConfig{})
	for _, want := range []string{"**/*.tfstate*", "**/*.tfvars", "**/*.tfvars.json", "~/.terraform.d/credentials.tfrc.json"} {
		if !slices.Contains(rules, want) {
			t.Errorf("ReadDenyRules missing %q: %v", want, rules)
		}
	}
}

// TestSemgrepRuleSets covers W137: only rulesets the Semgrep registry
// serves are returned (p/terraform-aws is a 404).
func TestSemgrepRuleSets(t *testing.T) {
	t.Parallel()
	if got := newModule().SemgrepRuleSets(); !slices.Equal(got, []string{"p/terraform"}) {
		t.Errorf("SemgrepRuleSets = %v, want [p/terraform]", got)
	}
}

// TestPreCommitHooks_FilesAndDirs covers W128/W129: hooks trigger on .tofu
// and JSON configuration, and validate/tflint reach configuration below the
// root.
func TestPreCommitHooks_FilesAndDirs(t *testing.T) {
	t.Parallel()
	hooks := newModule().PreCommitHooks(ecosystem.ModuleConfig{Extras: map[string]string{
		"variant": "opentofu", terraform.ExtraConfigDirs: "infra,modules/aws",
	}})
	pattern := hooks[0].Files
	for _, h := range hooks {
		if len(h.Types) != 0 || h.Files != pattern {
			t.Errorf("%s: Types=%v Files=%q, want no types and the shared files pattern %q", h.ID, h.Types, h.Files, pattern)
		}
	}
	re := regexp.MustCompile(pattern)
	for _, f := range []string{"main.tf", "main.tofu", "main.tf.json", "main.tofu.json", "prod.tfvars", "infra/x.tfvars.json"} {
		if !re.MatchString(f) {
			t.Errorf("files pattern %q does not match %q", pattern, f)
		}
	}
	for _, f := range []string{"main.go", "tf.md", "README.tofu.md"} {
		if re.MatchString(f) {
			t.Errorf("files pattern %q matches %q", pattern, f)
		}
	}
	validate := hooks[1].Entry
	for _, want := range []string{"for d in infra modules/aws;", "tofu -chdir=$d init -backend=false", "tofu -chdir=$d validate || exit 1"} {
		if !strings.Contains(validate, want) {
			t.Errorf("validate entry %q missing %q", validate, want)
		}
	}
	// The entry is embedded verbatim in a Nix double-quoted string.
	if strings.ContainsAny(validate, `"\`) || strings.Contains(validate, "${") {
		t.Errorf("validate entry %q is not safe inside a Nix string", validate)
	}
	if hooks[2].Entry != "tflint --recursive" {
		t.Errorf("tflint entry = %q, want tflint --recursive", hooks[2].Entry)
	}
}

// --- CICommands tests ---

func TestCICommands_Terraform(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	cmds := m.CICommands(config)

	if len(cmds) != 5 {
		t.Fatalf("expected 5 CI commands, got %d", len(cmds))
	}

	// First three commands should use the terraform binary.
	for i := 0; i < 3; i++ {
		if !strings.Contains(cmds[i].Command, "terraform") {
			t.Errorf("cmd[%d]: expected terraform in command, got %q", i, cmds[i].Command)
		}
	}

	// Verify phases.
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("init command should be Install phase, got %v", cmds[0].Phase)
	}
	if cmds[1].Phase != ecosystem.CIPhaseTest {
		t.Errorf("validate command should be Test phase, got %v", cmds[1].Phase)
	}
	if cmds[2].Phase != ecosystem.CIPhaseTest {
		t.Errorf("plan command should be Test phase, got %v", cmds[2].Phase)
	}
	if cmds[3].Phase != ecosystem.CIPhaseScan {
		t.Errorf("tflint command should be Scan phase, got %v", cmds[3].Phase)
	}
	if cmds[4].Phase != ecosystem.CIPhaseScan {
		t.Errorf("tfsec command should be Scan phase, got %v", cmds[4].Phase)
	}

	// init should use -backend=false.
	if !strings.Contains(cmds[0].Command, "-backend=false") {
		t.Errorf("init command should contain -backend=false, got %q", cmds[0].Command)
	}
}

func TestCICommands_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	cmds := m.CICommands(config)

	if len(cmds) != 5 {
		t.Fatalf("expected 5 CI commands, got %d", len(cmds))
	}

	// First three commands should use the tofu binary.
	for i := 0; i < 3; i++ {
		if !strings.Contains(cmds[i].Command, "tofu") {
			t.Errorf("cmd[%d]: expected tofu in command for opentofu variant, got %q", i, cmds[i].Command)
		}
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("expected 1 package manager, got %d", len(pms))
	}

	pm := pms[0]
	if pm.Name != "terraform-registry" {
		t.Errorf("expected name terraform-registry, got %q", pm.Name)
	}
	if pm.LockFile != ".terraform.lock.hcl" {
		t.Errorf("expected lockfile .terraform.lock.hcl, got %q", pm.LockFile)
	}
	if pm.FrozenInstallCommand != "terraform init -lockfile=readonly" {
		t.Errorf("expected frozen install command terraform init -lockfile=readonly, got %q", pm.FrozenInstallCommand)
	}
	if pm.AgeGatingSupport {
		t.Error("expected AgeGatingSupport=false")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("expected 2 wizard fields, got %d", len(fields))
	}

	// First field should be variant select.
	if fields[0].Type != ecosystem.FieldTypeSelect {
		t.Errorf("expected first field to be Select, got %v", fields[0].Type)
	}
	if len(fields[0].Options) != 2 {
		t.Errorf("expected 2 options for variant field, got %d", len(fields[0].Options))
	}

	// Second field should be version input.
	if fields[1].Type != ecosystem.FieldTypeInput {
		t.Errorf("expected second field to be Input, got %v", fields[1].Type)
	}
}

// --- Metadata tests ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "terraform", "Terraform/OpenTofu", 1)
}

// --- helpers ---

func containsEvidence(evidence []string, substr string) bool {
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
