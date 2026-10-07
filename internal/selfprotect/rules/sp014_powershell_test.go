package rules

import (
	"slices"
	"testing"
)

// TestSP014_BashPowerShellParity pins that SP-014 judges a guardrail-weakening
// command the same in both shell dialects: each row is one invocation as
// Bash writes it and as PowerShell does, where array arguments
// (`'teardown','--force'`, `teardown,--force`), array literals and splats
// (`@(...)`, `@a`), Start-Process (saps) with -ArgumentList and
// [Diagnostics.Process]::Start pass the same words to the program.
func TestSP014_BashPowerShellParity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bash, powershell string
		verdict          Verdict
	}{
		{"qsdev teardown --force", `qsdev 'teardown','--force'`, Deny},
		{"qsdev teardown --force", `& qsdev teardown,--force`, Deny},
		{"qsdev self-update --no-strict", `qsdev self-update,--no-strict`, Deny},
		{`qsdev $(echo teardown --force)`, `& qsdev @('teardown','--force')`, Deny},
		{`a=(teardown --force); qsdev "${a[@]}"`, `$a='teardown','--force'; & qsdev @a`, Deny},
		{`qsdev $(echo defaults reset)`, `& qsdev @("defaults","reset")`, Deny},
		{"nohup qsdev teardown --force", `Start-Process qsdev -ArgumentList 'teardown','--force'`, Deny},
		{"nohup qsdev self-update --no-strict", `Start-Process qsdev -ArgumentList 'self-update','--no-strict'`, Deny},
		{"nohup qsdev session allow", `Start-Process qsdev -ArgumentList 'session','allow'`, Deny},
		{"nohup qsdev defaults reset", `Start-Process qsdev -ArgumentList 'defaults','reset'`, Deny},
		{"nohup qsdev teardown --force", `Start-Process -FilePath qsdev -ArgumentList teardown,--force`, Deny},
		{"qsdev teardown --force", `saps qsdev 'teardown','--force' -Wait`, Deny},
		{"qsdev teardown --force", `[Diagnostics.Process]::Start('qsdev','teardown --force')`, Deny},
		{`sh -c "qsdev teardown --force"`, `Start-Process cmd -ArgumentList '/c','qsdev','teardown','--force'`, Deny},
		{"unset CLAUDECODE; qsdev teardown --force", `$env:CLAUDECODE=$null; Start-Process qsdev -ArgumentList 'teardown','--force' -Wait`, Deny},
		// Controls, already judged before.
		{"qsdev teardown --force", "qsdev teardown --force", Deny},
		{"'qsdev' teardown", `& 'qsdev' teardown`, Deny},
		{"nohup qsdev teardown --force", `Start-Process qsdev -ArgumentList 'teardown --force'`, Deny},
		{"qsdev teardown --dry-run", `qsdev 'teardown','--dry-run'`, Allow},
		{"qsdev status", `qsdev 'status','--format','json'`, Allow},
		{"qsdev repair", `Start-Process qsdev -ArgumentList 'repair'`, Allow},
	}
	for _, tt := range tests {
		for _, tc := range []struct{ tool, command string }{{"Bash", tt.bash}, {"PowerShell", tt.powershell}} {
			t.Run(tc.tool+"/"+tc.command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tc.tool, Command: tc.command, SensitiveCommands: sensitiveFixture}
				got, matches := Tier1Rules.EvaluateAll(&ctx)
				if got != tt.verdict {
					t.Fatalf("verdict = %v, want %v (matches %v)", got, tt.verdict, matchIDs(matches))
				}
				if tt.verdict == Deny && !slices.Contains(matchIDs(matches), "SP-014") {
					t.Errorf("matches = %v, want SP-014 among them", matchIDs(matches))
				}
			})
		}
	}
}

// TestSP008_PowerShellSessionEnv pins that a PowerShell line may not set or
// clear an agent marker or a Claude Code settings variable, which the
// session keeps for the programs it starts later (a CLI run in a new
// console would then pass for a human's), while reading one stays allowed.
func TestSP008_PowerShellSessionEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		verdict Verdict
	}{
		{`$env:CLAUDECODE=$null`, Deny},
		{`$env:CLAUDECODE = ''`, Deny},
		{`${env:ClaudeCode}=0`, Deny},
		{`Remove-Item Env:CLAUDECODE`, Deny},
		{`Remove-Item -Path Env:\CLAUDECODE`, Deny},
		{`rm env:/claudecode`, Deny},
		{`Set-Item env:CLAUDECODE ''`, Deny},
		{`Get-Item Env:CLAUDECODE | Remove-Item`, Deny},
		{`[Environment]::SetEnvironmentVariable('CLAUDECODE', $null)`, Deny},
		{`[System.Environment]::SetEnvironmentVariable("CLAUDECODE", "", "User")`, Deny},
		{`$env:CLAUDE_CONFIG_DIR = 'C:\tmp\cfg'`, Deny},
		{`$env:CLAUDE_CODE_SIMPLE='1'; claude`, Deny},

		{`echo $env:CLAUDECODE`, Allow},
		{`if ($env:CLAUDECODE -eq '1') { 'agent' }`, Allow},
		{`if ($env:CLAUDECODE == '1') { 'agent' }`, Allow},
		{`Get-ChildItem Env:CLAUDECODE`, Allow},
		{`$env:CLAUDECODE_NOTES = 'x'`, Allow},
		{`$env:PATH = "C:\tools;$env:PATH"`, Allow},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "PowerShell", Command: tt.command, SensitiveCommands: sensitiveFixture}
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.verdict {
				t.Fatalf("verdict = %v, want %v (matches %v)", got, tt.verdict, matchIDs(matches))
			}
			if tt.verdict == Deny && !slices.Contains(matchIDs(matches), "SP-008") {
				t.Errorf("matches = %v, want SP-008 among them", matchIDs(matches))
			}
		})
	}
}
