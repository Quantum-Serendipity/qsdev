package outdated

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandsForEcosystem(t *testing.T) {
	cmds := CommandsForEcosystem("javascript")
	if len(cmds) != 4 {
		t.Fatalf("expected 4 javascript commands, got %d", len(cmds))
	}

	expected := []struct {
		binary          string
		outdatedOnExit1 bool
	}{{"npm", true}, {"pnpm", true}, {"yarn", true}, {"bun", false}}
	for i, cmd := range cmds {
		if cmd.Binary != expected[i].binary {
			t.Errorf("javascript command[%d]: expected binary %q, got %q", i, expected[i].binary, cmd.Binary)
		}
		if cmd.Ecosystem != "javascript" {
			t.Errorf("javascript command[%d]: expected ecosystem %q, got %q", i, "javascript", cmd.Ecosystem)
		}
		if cmd.OutdatedOnExit1 != expected[i].outdatedOnExit1 {
			t.Errorf("javascript command[%d]: OutdatedOnExit1 = %t, want %t", i, cmd.OutdatedOnExit1, expected[i].outdatedOnExit1)
		}
	}
}

// TestCommandsExitCodeContract pins the flags that make exit status 1 mean
// "outdated packages found" for tools that otherwise exit 0 when packages are
// outdated and use 1 for their own failures.
func TestCommandsExitCodeContract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		eco      string
		binary   string
		wantArgs []string
	}{
		{"rust", "cargo", []string{"outdated", "--exit-code", "1"}},
		{"php", "composer", []string{"outdated", "--direct", "--strict"}},
	}
	for _, tt := range tests {
		t.Run(tt.binary, func(t *testing.T) {
			t.Parallel()
			cmds := CommandsForEcosystem(tt.eco)
			if len(cmds) != 1 || cmds[0].Binary != tt.binary {
				t.Fatalf("commands for %s = %+v, want one %s command", tt.eco, cmds, tt.binary)
			}
			if !cmds[0].OutdatedOnExit1 {
				t.Error("OutdatedOnExit1 = false, want true")
			}
			if strings.Join(cmds[0].Args, " ") != strings.Join(tt.wantArgs, " ") {
				t.Errorf("args = %v, want %v", cmds[0].Args, tt.wantArgs)
			}
		})
	}
}

func TestYarnBerryUnsupported(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{"yarnrc.yml", map[string]string{".yarnrc.yml": "nodeLinker: pnp\n"}, true},
		{"packageManager yarn 4", map[string]string{"package.json": `{"packageManager":"yarn@4.1.0"}`}, true},
		{"packageManager yarn 1", map[string]string{"package.json": `{"packageManager":"yarn@1.22.19"}`}, false},
		{"classic without markers", map[string]string{"package.json": `{"name":"x"}`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, body := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := yarnBerryUnsupported(dir) != ""; got != tt.want {
				t.Errorf("unsupported = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestCommandsForEcosystem_Go(t *testing.T) {
	cmds := CommandsForEcosystem("go")
	if len(cmds) != 1 {
		t.Fatalf("expected 1 go command, got %d", len(cmds))
	}
	if cmds[0].Binary != "go" {
		t.Errorf("expected binary %q, got %q", "go", cmds[0].Binary)
	}
	if cmds[0].OutdatedOnExit1 {
		t.Error("expected OutdatedOnExit1 to be false for go")
	}
}

func TestCommandsForEcosystem_Java(t *testing.T) {
	cmds := CommandsForEcosystem("java")
	if len(cmds) != 2 {
		t.Fatalf("expected 2 java commands, got %d", len(cmds))
	}
	if cmds[0].Binary != "mvn" {
		t.Errorf("expected first java binary %q, got %q", "mvn", cmds[0].Binary)
	}
	if cmds[1].Binary != "gradle" {
		t.Errorf("expected second java binary %q, got %q", "gradle", cmds[1].Binary)
	}
}

func TestCommandsForEcosystem_Unknown(t *testing.T) {
	cmds := CommandsForEcosystem("haskell")
	if cmds != nil {
		t.Errorf("expected nil for unknown ecosystem, got %v", cmds)
	}
}

func TestSupportedEcosystems(t *testing.T) {
	ecosystems := SupportedEcosystems()

	// Should be deduplicated — javascript and java each have multiple commands
	// but should appear only once.
	seen := make(map[string]int)
	for _, eco := range ecosystems {
		seen[eco]++
	}
	for eco, count := range seen {
		if count != 1 {
			t.Errorf("ecosystem %q appeared %d times, expected 1", eco, count)
		}
	}

	// Verify expected ecosystems are present.
	expected := []string{"javascript", "python", "go", "rust", "dotnet", "ruby", "php", "elixir", "java"}
	for _, exp := range expected {
		if seen[exp] == 0 {
			t.Errorf("expected ecosystem %q not found in SupportedEcosystems()", exp)
		}
	}
}
