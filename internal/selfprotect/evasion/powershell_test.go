package evasion

import "testing"

// TestPowerShellObfuscation verifies the PowerShell forms that run code no
// check can see (U18-08) are blocked as obfuscation: a command line passed
// encoded, a string run as code, and an elevated process.
func TestPowerShellObfuscation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		command string
		blocked bool
	}{
		{name: "pwsh -EncodedCommand", tool: "PowerShell", command: `pwsh -EncodedCommand AAAA`, blocked: true},
		{name: "pwsh -enc", tool: "PowerShell", command: `pwsh -enc AAAA`, blocked: true},
		{name: "powershell -e", tool: "PowerShell", command: `powershell -e AAAA`, blocked: true},
		{name: "powershell.exe -ec", tool: "PowerShell", command: `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -NoProfile -ec AAAA`, blocked: true},
		{name: "slash encoded", tool: "PowerShell", command: `powershell /EncodedCommand AAAA`, blocked: true},
		{name: "encoded with colon", tool: "PowerShell", command: `pwsh -EncodedCommand:AAAA`, blocked: true},
		{name: "encoded through cmd", tool: "PowerShell", command: `cmd /c pwsh -enc AAAA`, blocked: true},
		{name: "encoded through Start-Process", tool: "PowerShell", command: `Start-Process pwsh -ArgumentList '-enc AAAA'`, blocked: true},
		{name: "iex", tool: "PowerShell", command: `iex (gc x.ps1)`, blocked: true},
		{name: "Invoke-Expression", tool: "PowerShell", command: `Invoke-Expression $x`, blocked: true},
		{name: "INVOKE-EXPRESSION upper case", tool: "PowerShell", command: `Get-Content x.ps1 | INVOKE-EXPRESSION`, blocked: true},
		{name: "call operator iex", tool: "PowerShell", command: `& iex $s`, blocked: true},
		{name: "scriptblock Create", tool: "PowerShell", command: `& ([scriptblock]::Create($s))`, blocked: true},
		{name: "scriptblock Create spaced", tool: "PowerShell", command: `. ( [ScriptBlock] :: Create( $s ) )`, blocked: true},
		{name: "Start-Process -Verb RunAs", tool: "PowerShell", command: `Start-Process pwsh -Verb RunAs`, blocked: true},
		{name: "saps -Verb:RunAs", tool: "PowerShell", command: `saps pwsh -Verb:RunAs`, blocked: true},
		{name: "start -v", tool: "PowerShell", command: `start notepad -v runas`, blocked: true},
		{name: "pwsh -c iex", tool: "PowerShell", command: `pwsh -c 'iex $s'`, blocked: true},
		{name: "pwsh -Command Invoke-Expression", tool: "PowerShell", command: `pwsh -Command 'Invoke-Expression $s'`, blocked: true},
		{name: "pwsh -Command:iex", tool: "PowerShell", command: `pwsh -Command:iex $s`, blocked: true},
		{name: "powershell positional iex", tool: "PowerShell", command: `powershell 'iex $s'`, blocked: true},
		{name: "nested hosts", tool: "PowerShell", command: "pwsh -c \"pwsh -c 'iex `$s'\"", blocked: true},
		{name: "Start-Process host -c iex", tool: "PowerShell", command: `Start-Process pwsh -ArgumentList '-c iex $s'`, blocked: true},
		{name: "computed call target", tool: "PowerShell", command: `&('i'+'ex') $s`, blocked: true},
		{name: "computed call target spaced", tool: "PowerShell", command: `& ('Invoke-'+'Expression') $s`, blocked: true},
		{name: "sal alias", tool: "PowerShell", command: `sal x iex; x $s`, blocked: true},
		{name: "Set-Alias alias", tool: "PowerShell", command: `Set-Alias x Invoke-Expression; x $s`, blocked: true},
		{name: "New-Alias alias", tool: "PowerShell", command: `New-Alias x iex; x $s`, blocked: true},
		{name: "Set-Alias -Value:", tool: "PowerShell", command: `Set-Alias -Name x -Value:iex`, blocked: true},
		{name: "ScriptBlock full type name", tool: "PowerShell", command: `& ([System.Management.Automation.ScriptBlock]::Create($s))`, blocked: true},
		{name: "ScriptBlock short namespace", tool: "PowerShell", command: `[Management.Automation.ScriptBlock]::Create($s)`, blocked: true},
		{name: "NewScriptBlock", tool: "PowerShell", command: `& $ExecutionContext.InvokeCommand.NewScriptBlock($s)`, blocked: true},
		{name: "InvokeScript", tool: "PowerShell", command: `$ExecutionContext.InvokeCommand.InvokeScript($s)`, blocked: true},
		{name: "en dash enc", tool: "PowerShell", command: "pwsh \u2013enc AAAA", blocked: true},
		{name: "em dash EncodedCommand", tool: "PowerShell", command: "pwsh \u2014EncodedCommand AAAA", blocked: true},
		{name: "horizontal bar ec", tool: "PowerShell", command: "powershell \u2015ec AAAA", blocked: true},
		{name: "en dash Verb", tool: "PowerShell", command: "Start-Process pwsh \u2013Verb RunAs", blocked: true},

		{name: "pwsh -File", tool: "PowerShell", command: `pwsh -File build.ps1`, blocked: false},
		{name: "pwsh -ExecutionPolicy", tool: "PowerShell", command: `pwsh -ExecutionPolicy Bypass -File build.ps1`, blocked: false},
		{name: "commit message mentioning iex", tool: "PowerShell", command: `git commit -m 'mention iex'`, blocked: false},
		{name: "pwsh -c read", tool: "PowerShell", command: `pwsh -NoProfile -c 'Get-Content x.txt'`, blocked: false},
		{name: "Set-Alias of a read", tool: "PowerShell", command: `Set-Alias g Get-Content`, blocked: false},
		{name: "Start-Process without Verb", tool: "PowerShell", command: `Start-Process notepad -Verbose`, blocked: false},
		{name: "Bash iex is not PowerShell", tool: "Bash", command: `iex foo`, blocked: false},
		{name: "Bash scriptblock text", tool: "Bash", command: `grep '[scriptblock]::Create' notes.md`, blocked: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, "")
			if blocked != tt.blocked {
				t.Fatalf("Check(%q, %q) blocked = %v, want %v (category %q, reason %q)",
					tt.tool, tt.command, blocked, tt.blocked, category, reason)
			}
			if blocked && category != "obfuscation" {
				t.Errorf("category = %q, want obfuscation (reason %q)", category, reason)
			}
		})
	}
}
