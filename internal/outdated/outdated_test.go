package outdated

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockLookPath returns a function that reports the given binaries as found,
// and all others as missing.
func mockLookPath(available map[string]bool) func(string) (string, error) {
	return func(name string) (string, error) {
		if available[name] {
			return "/usr/bin/" + name, nil
		}
		return "", fmt.Errorf("%s: not found", name)
	}
}

func TestRunOutdated_NoEcosystems(t *testing.T) {
	var buf bytes.Buffer
	result, err := RunOutdated(context.Background(), &buf, "/tmp", nil, OutdatedOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Ecosystems) != 0 {
		t.Errorf("expected 0 ecosystems, got %d", len(result.Ecosystems))
	}
	if !strings.Contains(buf.String(), "No ecosystems detected") {
		t.Errorf("expected 'No ecosystems detected' message, got %q", buf.String())
	}
}

func TestRunOutdated_BinaryNotFound(t *testing.T) {
	// Override lookPathFunc so no binaries are found.
	original := lookPathFunc
	lookPathFunc = mockLookPath(map[string]bool{})
	defer func() { lookPathFunc = original }()

	var buf bytes.Buffer
	ecosystems := []string{"javascript", "python"}
	result, err := RunOutdated(context.Background(), &buf, "/tmp", ecosystems, OutdatedOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Ecosystems) != 2 {
		t.Fatalf("expected 2 ecosystem checks, got %d", len(result.Ecosystems))
	}

	for _, check := range result.Ecosystems {
		if !check.Skipped {
			t.Errorf("ecosystem %q should be skipped when binary not found", check.Name)
		}
		if check.SkipReason == "" {
			t.Errorf("ecosystem %q should have a skip reason", check.Name)
		}
		if !strings.Contains(check.SkipReason, "not found on PATH") {
			t.Errorf("ecosystem %q skip reason should mention 'not found on PATH', got %q", check.Name, check.SkipReason)
		}
	}

	output := buf.String()
	if !strings.Contains(output, "skipped") {
		t.Errorf("output should contain 'skipped', got %q", output)
	}
}

func TestRunOutdated_BinaryNotFound_JavaScript(t *testing.T) {
	// Override lookPathFunc so no binaries are found.
	original := lookPathFunc
	lookPathFunc = mockLookPath(map[string]bool{})
	defer func() { lookPathFunc = original }()

	var buf bytes.Buffer
	result, err := RunOutdated(context.Background(), &buf, "/tmp", []string{"javascript"}, OutdatedOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Ecosystems) != 1 {
		t.Fatalf("expected 1 ecosystem check, got %d", len(result.Ecosystems))
	}

	check := result.Ecosystems[0]
	if !check.Skipped {
		t.Error("javascript should be skipped when npm/pnpm/yarn are not found")
	}
	// The skip reason should mention all three binaries.
	if !strings.Contains(check.SkipReason, "npm") || !strings.Contains(check.SkipReason, "pnpm") || !strings.Contains(check.SkipReason, "yarn") {
		t.Errorf("skip reason should mention npm, pnpm, and yarn, got %q", check.SkipReason)
	}
}

func TestRunOutdated_EcosystemFilter(t *testing.T) {
	// Override lookPathFunc so no binaries are found.
	// We're testing filtering, not execution.
	original := lookPathFunc
	lookPathFunc = mockLookPath(map[string]bool{})
	defer func() { lookPathFunc = original }()

	var buf bytes.Buffer
	ecosystems := []string{"javascript", "python", "go"}
	opts := OutdatedOptions{Ecosystem: "python"}

	result, err := RunOutdated(context.Background(), &buf, "/tmp", ecosystems, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only python should be checked.
	if len(result.Ecosystems) != 1 {
		t.Fatalf("expected 1 ecosystem check with filter, got %d", len(result.Ecosystems))
	}
	if result.Ecosystems[0].Name != "python" {
		t.Errorf("expected filtered ecosystem to be 'python', got %q", result.Ecosystems[0].Name)
	}
}

func TestRunOutdated_EcosystemFilter_NoMatch(t *testing.T) {
	original := lookPathFunc
	lookPathFunc = mockLookPath(map[string]bool{})
	defer func() { lookPathFunc = original }()

	var buf bytes.Buffer
	ecosystems := []string{"javascript", "python"}
	opts := OutdatedOptions{Ecosystem: "rust"}

	result, err := RunOutdated(context.Background(), &buf, "/tmp", ecosystems, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Ecosystems) != 0 {
		t.Errorf("expected 0 ecosystem checks when filter doesn't match, got %d", len(result.Ecosystems))
	}
}

func TestRunOutdated_UnknownEcosystem(t *testing.T) {
	original := lookPathFunc
	lookPathFunc = mockLookPath(map[string]bool{})
	defer func() { lookPathFunc = original }()

	var buf bytes.Buffer
	// An ecosystem that has no commands registered should be silently skipped.
	result, err := RunOutdated(context.Background(), &buf, "/tmp", []string{"haskell"}, OutdatedOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Ecosystems) != 0 {
		t.Errorf("expected 0 ecosystem checks for unknown ecosystem, got %d", len(result.Ecosystems))
	}
}

func TestHasAnyOutdated(t *testing.T) {
	tests := []struct {
		name     string
		result   OutdatedResult
		expected bool
	}{
		{
			name:     "empty result",
			result:   OutdatedResult{},
			expected: false,
		},
		{
			name: "no outdated",
			result: OutdatedResult{
				Ecosystems: []EcosystemCheck{
					{Name: "go", HasOutdated: false},
					{Name: "python", HasOutdated: false},
				},
			},
			expected: false,
		},
		{
			name: "one outdated",
			result: OutdatedResult{
				Ecosystems: []EcosystemCheck{
					{Name: "go", HasOutdated: false},
					{Name: "javascript", HasOutdated: true},
				},
			},
			expected: true,
		},
		{
			name: "all outdated",
			result: OutdatedResult{
				Ecosystems: []EcosystemCheck{
					{Name: "go", HasOutdated: true},
					{Name: "javascript", HasOutdated: true},
				},
			},
			expected: true,
		},
		{
			name: "skipped only",
			result: OutdatedResult{
				Ecosystems: []EcosystemCheck{
					{Name: "go", Skipped: true, HasOutdated: false},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HasAnyOutdated()
			if got != tt.expected {
				t.Errorf("HasAnyOutdated() = %v, want %v", got, tt.expected)
			}
		})
	}
}
