package denyutil

import "testing"

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
			matches := func(cmd string) bool {
				for _, r := range tt.rules {
					if MatchesBashRule(r, cmd) {
						return true
					}
				}
				return false
			}
			for _, cmd := range tt.denied {
				if !matches(cmd) {
					t.Errorf("no rule denies %q; rules: %v", cmd, tt.rules)
				}
			}
			for _, cmd := range tt.allowed {
				if matches(cmd) {
					t.Errorf("rules over-match %q", cmd)
				}
			}
		})
	}
}
