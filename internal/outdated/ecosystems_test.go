package outdated

import (
	"testing"
)

func TestCommandsForEcosystem(t *testing.T) {
	cmds := CommandsForEcosystem("javascript")
	if len(cmds) != 3 {
		t.Fatalf("expected 3 javascript commands, got %d", len(cmds))
	}

	expectedBinaries := []string{"npm", "pnpm", "yarn"}
	for i, cmd := range cmds {
		if cmd.Binary != expectedBinaries[i] {
			t.Errorf("javascript command[%d]: expected binary %q, got %q", i, expectedBinaries[i], cmd.Binary)
		}
		if cmd.Ecosystem != "javascript" {
			t.Errorf("javascript command[%d]: expected ecosystem %q, got %q", i, "javascript", cmd.Ecosystem)
		}
		if !cmd.OutdatedOnExit1 {
			t.Errorf("javascript command[%d]: expected OutdatedOnExit1 to be true", i)
		}
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
