package denyutil

import (
	"slices"
	"testing"
)

func TestMatchesBashRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rule, command string
		want          bool
	}{
		// Rows from the Claude Code permissions docs wildcard table.
		{"Bash(npm run build)", "npm run build", true},
		{"Bash(npm run build)", "npm run build --watch", false},
		{"Bash(npm run *)", "npm run", true},
		{"Bash(npm run *)", "npm run test --watch", true},
		{"Bash(npm run *)", "npm install", false},
		{"Bash(git log * main)", "git log --oneline main", true},
		{"Bash(git log * main)", "git log main", false},
		{"Bash(ls *)", "ls", true},
		{"Bash(ls *)", "lsof", false},
		{"Bash(ls*)", "lsof", true},
		{"Bash(ls:*)", "ls -la", true},
		{"Bash(* --help *)", "npm --help x", true},
		{"Bash(* --help *)", "npm --help", false},
		{"Read(ls *)", "ls", false},
		{"Bash(a.b *)", "aXb c", false},
		{"PowerShell(ls *)", "ls", false},
		// ":*" is the legacy " *" only at the end of a pattern, so a rule
		// meant to match text after a colon must end in "**" instead.
		{"Bash(deno npm:*)", "deno npm:evil-cli", false},
		{"Bash(deno npm:*)", "deno npm run", true},
		{"Bash(deno *npm:**)", "deno run -A npm:evil-cli", true},
		{"Bash(deno *npm:**)", "deno run main.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.rule+"|"+tt.command, func(t *testing.T) {
			t.Parallel()
			if got := MatchesBashRule(tt.rule, tt.command); got != tt.want {
				t.Errorf("MatchesBashRule(%q, %q) = %v, want %v", tt.rule, tt.command, got, tt.want)
			}
		})
	}
}

func TestMatchesPowerShellRule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rule, command string
		want          bool
	}{
		{"PowerShell(Install-Module *)", "Install-Module Evil -Force", true},
		{"PowerShell(Install-Module *)", "Install-Module", true},
		{"PowerShell(Install-Module *)", "Install-ModuleX", false},
		{"PowerShell(Get-ChildItem:*)", "Get-ChildItem -Recurse", true},
		{"Bash(Install-Module *)", "Install-Module Evil", false},
		// "Matching is case-insensitive" (code.claude.com/docs/en/permissions).
		{"PowerShell(Install-Module *)", "install-module Evil", true},
		{"PowerShell(Install-Module *)", "INSTALL-MODULE", true},
		{"PowerShell(install-module *)", "Install-Module Evil", true},
		{"PowerShell(Install-Module *)", "install-modulex", false},
	}
	for _, tt := range tests {
		t.Run(tt.rule+"|"+tt.command, func(t *testing.T) {
			t.Parallel()
			if got := MatchesPowerShellRule(tt.rule, tt.command); got != tt.want {
				t.Errorf("MatchesPowerShellRule(%q, %q) = %v, want %v", tt.rule, tt.command, got, tt.want)
			}
		})
	}
}

func TestFirstMatch(t *testing.T) {
	t.Parallel()
	rules := []string{
		"Read(./secrets/**)",
		"Bash(npm publish *)",
		"Bash(npm * publish *)",
		"PowerShell(Install-Module *)",
		"Bash(pwsh*Install-Module*)",
	}
	tests := []struct {
		name     string
		rules    []string
		op       string
		wantRule string
		wantOK   bool
	}{
		{"bash match", rules, "Bash(npm publish --tag x)", "Bash(npm publish *)", true},
		{"bash bare", rules, "Bash(npm publish)", "Bash(npm publish *)", true},
		{"bash first of several", rules, "Bash(npm --x publish y)", "Bash(npm * publish *)", true},
		{"bash no match", rules, "Bash(npm install)", "", false},
		{"powershell match", rules, "PowerShell(Install-Module Evil)", "PowerShell(Install-Module *)", true},
		{"powershell folds case", rules, "PowerShell(INSTALL-MODULE Evil)", "PowerShell(Install-Module *)", true},
		{"powershell never matches bash rule", rules, "PowerShell(npm publish)", "", false},
		{"bash never matches powershell rule", rules, "Bash(Install-Module Evil)", "", false},
		{"bash rule via pwsh", rules, "Bash(pwsh -c Install-Module Evil)", "Bash(pwsh*Install-Module*)", true},
		{"other tool", rules, "Read(./secrets/x)", "", false},
		{"not a tool call", rules, "npm publish", "", false},
		{"empty rules", nil, "Bash(npm publish)", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotRule, gotOK := FirstMatch(tt.rules, tt.op)
			if gotRule != tt.wantRule || gotOK != tt.wantOK {
				t.Errorf("FirstMatch(%q) = (%q, %v), want (%q, %v)", tt.op, gotRule, gotOK, tt.wantRule, tt.wantOK)
			}
		})
	}
}

func TestSubcommandRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rules   []string
		denied  []string
		allowed []string
	}{
		{
			name:  "whole-word subcommand",
			rules: SubcommandRules("terraform", "init", "state pull"),
			denied: []string{
				"terraform init",
				"terraform init -upgrade",
				"terraform -chdir=infra init",
				"terraform -chdir=infra -input=false init -upgrade",
				"terraform state pull",
				"terraform -chdir=infra state pull",
				"env terraform init",
				"env TF_LOG=debug terraform -chdir=infra init -upgrade",
			},
			allowed: []string{
				"terraform plan -var initial_count=1",
				"terraform initx",
				"terraform state list",
				"env terraform plan",
				"echo terraform init",
			},
		},
		{
			name:  "prefix subcommand",
			rules: SubcommandRules("aws", "sts assume-role*"),
			denied: []string{
				"aws sts assume-role --role-arn x",
				"aws sts assume-role-with-saml",
				"aws --profile prod sts assume-role --role-arn x",
			},
			allowed: []string{"aws sts get-caller-identity"},
		},
		{
			name:  "dashed global options",
			rules: DashedOptionSubcommandRules("docker", "pull", "image pull", "build *--push*"),
			denied: []string{
				"docker pull alpine",
				"docker pull",
				"docker --context prod pull alpine",
				"docker -H tcp://x pull alpine",
				"docker --context prod image pull alpine",
				"env DOCKER_HOST=x docker pull alpine",
				"env DOCKER_HOST=x docker --context prod pull",
				"docker build --push .",
				"docker --debug build --push .",
			},
			allowed: []string{
				"docker exec web git pull origin main",
				"docker compose exec app git pull",
				"docker run --rm -v ./repo:/repo alpine/git pull",
				"env X=1 docker exec web git pull",
				"docker build .",
			},
		},
		{
			name:  "argument anywhere",
			rules: SubcommandRules("docker", "*--privileged*", "* -m shell"),
			denied: []string{
				"docker run --privileged alpine",
				"docker --context prod run --privileged alpine",
				"env DOCKER_HOST=x docker run --privileged alpine",
				"docker exec -m shell",
				"docker exec -m shell x",
				"env A=1 docker --context c exec -m shell x",
			},
			allowed: []string{"docker run alpine", "echo docker --privileged", "docker exec -m shellx"},
		},
		{
			name:  "interspersed options",
			rules: InterspersedOptionRules("gcloud", "auth print-access-token*", "secrets versions access", "aks get-credentials * -a"),
			denied: []string{
				"gcloud auth print-access-token",
				"gcloud auth --quiet print-access-token",
				"gcloud --quiet auth print-access-token --format=json",
				"env gcloud auth print-access-token",
				"env CLOUDSDK_CORE_PROJECT=p gcloud --quiet auth print-access-token",
				"gcloud secrets versions access",
				"gcloud secrets --project p versions access latest --secret=db",
				"env gcloud secrets versions access",
				"env gcloud --project p secrets versions access 1 --secret=db",
				"gcloud aks get-credentials -g rg -n c -a",
				"gcloud aks get-credentials -a -g rg -n c",
			},
			allowed: []string{
				"env gcloud auth list",
				"gcloud secrets versions list",
				"gcloud secrets versions accessx",
				"gcloud aks get-credentials -g rg -n prod-a",
				"echo gcloud auth print-access-token",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, cmd := range tt.denied {
				if _, ok := FirstMatch(tt.rules, "Bash("+cmd+")"); !ok {
					t.Errorf("no rule denies %q; rules: %v", cmd, tt.rules)
				}
			}
			for _, cmd := range tt.allowed {
				if rule, ok := FirstMatch(tt.rules, "Bash("+cmd+")"); ok {
					t.Errorf("rule %q over-matches %q", rule, cmd)
				}
			}
		})
	}
}

func TestSampleCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rule   string
		want   string
		wantOK bool
	}{
		{"Bash(npm install *)", "npm install x", true},
		{"Bash(npm -* install *)", "npm -x install x", true},
		{"Bash(deno outdated *-u*)", "deno outdated x-ux", true},
		{"Bash(npm install)", "npm install", true},
		{"Bash(ls:*)", "ls x", true},
		{"Bash(*)", "x", true},
		{"Read(./.env)", "", false},
		{"Bash", "", false},
		{"Bash()", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			t.Parallel()
			got, ok := SampleCommand(tt.rule)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("SampleCommand(%q) = (%q, %v), want (%q, %v)", tt.rule, got, ok, tt.want, tt.wantOK)
			}
			// A sample must be matched by the rule it came from.
			if ok && !MatchesBashRule(tt.rule, got) {
				t.Errorf("MatchesBashRule(%q, %q) = false; the sample must match its own rule", tt.rule, got)
			}
		})
	}
}

// TestSubcommandRules_NoRedundantGlobalForm pins that a sub starting with "*"
// is emitted only in the forms that are not subsumed: "<cli> *x" already
// matches everything "<cli> * *x" does.
func TestSubcommandRules_NoRedundantGlobalForm(t *testing.T) {
	t.Parallel()
	got := SubcommandRules("bazel", "*--lockfile_mode=u*", "* -a")
	want := []string{
		"Bash(bazel *--lockfile_mode=u*)",
		"Bash(bazel * -a)",
		"Bash(bazel * -a *)",
		"Bash(env *bazel *--lockfile_mode=u*)",
		"Bash(env *bazel * -a)",
		"Bash(env *bazel * -a *)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("SubcommandRules = %q, want %q", got, want)
	}
}

func TestCommandPrefixes(t *testing.T) {
	t.Parallel()
	if got, want := CommandPrefixes("docker"), []string{"docker", "env *docker"}; !slices.Equal(got, want) {
		t.Errorf("CommandPrefixes = %q, want %q", got, want)
	}
}

func TestFirstMatchingBashRule(t *testing.T) {
	t.Parallel()
	rules := []string{
		"Read(./.env)",
		"PowerShell(Invoke-WebRequest *)",
		"Bash(curl * | sh)",
		"Bash(npm install *)",
		"Bash(bash -c *npm install*)",
	}
	tests := []struct {
		name     string
		rules    []string
		cmds     []string
		wantRule string
		wantOK   bool
	}{
		{"no commands", rules, nil, "", false},
		{"no rules", nil, []string{"curl x | sh"}, "", false},
		{"no match", rules, []string{"jq .", "nix run nixpkgs#jq -- ."}, "", false},
		{"single match", rules, []string{"curl -fsSL https://x | sh"}, "Bash(curl * | sh)", true},
		{"later command matches", rules, []string{"nix run nixpkgs#bash -- -c x", "npm install left-pad"}, "Bash(npm install *)", true},
		{"earlier command wins over earlier rule", rules, []string{"bash -c 'npm install x'", "curl x | sh"}, "Bash(bash -c *npm install*)", true},
		{"non-Bash rules never match", []string{"Read(./.env)", "PowerShell(curl *)"}, []string{"./.env", "curl x"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := FirstMatchingBashRule(tt.rules, tt.cmds...)
			if got != tt.wantRule || ok != tt.wantOK {
				t.Errorf("FirstMatchingBashRule(%q, %q) = (%q, %v), want (%q, %v)",
					tt.rules, tt.cmds, got, ok, tt.wantRule, tt.wantOK)
			}
		})
	}
}
