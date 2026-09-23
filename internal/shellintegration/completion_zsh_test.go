package shellintegration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCompletionInstaller_ZshCompinitRescansFpath guards against `compinit -C`:
// the block is appended after the user's own compinit, whose dump lacks
// _qsdev, and -C would reuse that stale dump so completion never registers.
func TestCompletionInstaller_ZshCompinitRescansFpath(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".zshrc")
	if err := os.WriteFile(rcFile, []byte("autoload -Uz compinit && compinit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A name no system-wide completion can already provide, so the live check
	// below cannot pass because of a pre-existing dump entry.
	const name = "qsdevcomptest"
	root := testRootCmd()
	root.Use = name
	installer := &CompletionInstaller{BinaryName: name, HomeDir: tmpDir}
	if err := installer.Install(root, "zsh", rcFile); err != nil {
		t.Fatalf("Install zsh failed: %v", err)
	}

	data, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}
	rc := string(data)
	if strings.Contains(rc, "compinit -C") {
		t.Errorf("zsh block must not use compinit -C:\n%s", rc)
	}
	if !strings.Contains(rc, zshCompinitLine) {
		t.Errorf("zsh block missing %q:\n%s", zshCompinitLine, rc)
	}

	// When zsh is available, prove the completion actually registers for a
	// user whose own compinit already wrote a dump without _qsdev.
	zsh, err := exec.LookPath("zsh")
	if err != nil || runtime.GOOS == "windows" {
		return
	}
	env := append(os.Environ(), "ZDOTDIR="+tmpDir, "HOME="+tmpDir)
	// First shell writes the stale dump the way the user's compinit would.
	warm := exec.Command(zsh, "-f", "-c", "autoload -Uz compinit && compinit -d "+filepath.Join(tmpDir, ".zcompdump"))
	warm.Env = env
	if out, err := warm.CombinedOutput(); err != nil {
		t.Skipf("zsh compinit unavailable: %v\n%s", err, out)
	}
	// -f keeps system rc files (which may run their own compinit) out of it.
	check := exec.Command(zsh, "-f", "-c", "source "+rcFile+" >/dev/null 2>&1; print -r -- ${_comps["+name+"]}")
	check.Env = env
	out, err := check.CombinedOutput()
	if err != nil {
		t.Skipf("zsh check failed: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "_"+name {
		t.Errorf("completion not registered after sourcing rc; _comps[%s] = %q", name, got)
	}
}
