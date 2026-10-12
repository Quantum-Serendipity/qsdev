package rules

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestComposerHomeVerdicts pins that the tracked Composer home keeps the
// self-protection its policy had under the state directory: a shell command
// cannot delete, move or roll back the hardened config.json (or the nested
// .gitignore) through the directory holding it, a glob or a find, while
// composer's own working space (cache, vendored plugins, the auth.json link)
// stays freely usable.
func TestComposerHomeVerdicts(t *testing.T) {
	t.Parallel()
	project := filepath.Join(homeDir(t), "project")
	dir := branding.Get().ComposerHomeDir()
	abs := filepath.ToSlash(filepath.Join(project, dir))
	bash := func(cmd string) EvalContext {
		return EvalContext{ToolName: "Bash", Command: cmd, CWD: project}
	}

	tests := []struct {
		name string
		cmd  string
		want Verdict
	}{
		{"remove home", "rm -rf " + dir, Deny},
		{"remove home trailing slash", "rm -rf " + dir + "/", Deny},
		{"remove home absolute", "rm -rf " + abs, Deny},
		{"remove home dot spelling", "rm -rf ./" + dir + "/.", Deny},
		{"move home away", "mv " + dir + " /tmp/x", Deny},
		{"glob config.json", "rm " + dir + "/conf*.json", Deny},
		{"glob gitignore", "rm " + dir + "/.git*", Deny},
		{"glob home", "rm -rf " + dir[:4] + "*", Deny},
		{"find delete config.json", "find " + dir + " -name config.json -delete", Deny},
		{"git rm home", "git rm -r " + dir, Deny},
		{"git checkout home", "git checkout HEAD~1 -- " + dir, Deny},
		{"git restore home", "git restore --source HEAD~1 " + dir, Deny},
		{"cd home remove glob", "cd " + dir + " && rm -f conf*.json", Deny},
		{"cd home remove everything", "cd " + dir + " && rm -rf ./*", Deny},
		{"cd home remove itself", "cd " + dir + " && rm -rf .", Deny},
		{"literal config.json", "rm " + dir + "/config.json", Deny},
		// Spellings the bash scanner cannot resolve: the raw-command scan
		// must see the directory itself (regression: these were allowed
		// once the policy left the protected state directory).
		{"python rmtree home", `python3 -c "import shutil; shutil.rmtree('` + dir + `')"`, Deny},
		{"python remove concatenated config", `python3 -c "import os; os.remove('` + dir + `/'+'config.json')"`, Deny},
		{"node rmSync home", `node -e "require('fs').rmSync('` + dir + `',{recursive:true})"`, Deny},
		{"variable remove home", "D=" + dir + "; rm -rf $D", Deny},
		{"variable remove config", "D=" + dir + "; rm $D/config.json", Deny},
		{"python glob config", `python3 -c "import os,glob; [os.remove(f) for f in glob.glob('` + dir + `/conf*')]"`, Deny},
		{"python remove cache", `python3 -c "import shutil; shutil.rmtree('` + dir + `/cache')"`, Allow},
		{"variable remove cache", "D=" + dir + "/cache; rm -rf $D", Allow},

		{"list home", "ls " + dir, Allow},
		{"list cache", "ls " + dir + "/cache", Allow},
		{"cat config.json", "cat " + dir + "/config.json", Allow},
		{"remove cache", "rm -rf " + dir + "/cache", Allow},
		{"remove cache json glob", "rm -f " + dir + "/cache/*.json", Allow},
		{"cd home remove cache glob", "cd " + dir + " && rm -rf cach?", Allow},
		{"remove json glob elsewhere", "rm -f build/conf*.json", Allow},
		{"remove vendored plugin", "rm -rf " + dir + "/vendor/x", Allow},
		{"remove auth link", "rm " + dir + "/auth.json", Allow},
		{"cd home remove cache", "cd " + dir + " && rm -rf cache", Allow},
		{"remove lookalike", "rm -rf " + dir + "-old", Allow},
		{"git diff home", "git diff " + dir, Allow},

		{"powershell remove home", "ps:Remove-Item -Recurse -Force " + dir, Deny},
		{"powershell move home", "ps:Move-Item " + dir + " C:/tmp/x", Deny},
		{"powershell remove cache", "ps:Remove-Item -Recurse -Force " + dir + "/cache", Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := bash(tt.cmd)
			if cmd, ok := strings.CutPrefix(tt.cmd, "ps:"); ok {
				ctx = EvalContext{ToolName: "PowerShell", Command: cmd, CWD: project}
			}
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tt.want {
				t.Errorf("%q: verdict = %v (first: %s), want %v", tt.cmd, got, firstRuleID(matches), tt.want)
			}
		})
	}
}
