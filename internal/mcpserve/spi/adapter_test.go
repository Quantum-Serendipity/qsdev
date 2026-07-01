package spi

import (
	"context"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// stubAdapter is a minimal FrameworkAdapter used to exercise the registry and
// the client-matching dispatch without pulling in a concrete adapter package.
type stubAdapter struct {
	id    aiframework.FrameworkID
	match func(ClientInfo) bool // when non-nil the adapter implements ClientMatcher
}

func (a stubAdapter) ID() aiframework.FrameworkID          { return a.id }
func (a stubAdapter) Applies(context.Context, string) bool { return true }
func (a stubAdapter) Tools() []ToolRegistration            { return nil }
func (a stubAdapter) Resources() []ResourceRegistration    { return nil }
func (a stubAdapter) Prompts() []PromptRegistration        { return nil }

// matchingAdapter embeds stubAdapter and implements ClientMatcher.
type matchingAdapter struct {
	stubAdapter
}

func (a matchingAdapter) MatchesClient(c ClientInfo) bool { return a.match(c) }

var (
	_ FrameworkAdapter = stubAdapter{}
	_ FrameworkAdapter = matchingAdapter{}
	_ ClientMatcher    = matchingAdapter{}
)

func TestRegisterAndAll(t *testing.T) {
	t.Parallel()
	r := NewAdapterRegistry()
	// Register out of order; All() must come back sorted by FrameworkID.
	if err := r.Register(stubAdapter{id: aiframework.GeminiCLI}); err != nil {
		t.Fatalf("register gemini: %v", err)
	}
	if err := r.Register(stubAdapter{id: aiframework.ClaudeCode}); err != nil {
		t.Fatalf("register claudecode: %v", err)
	}
	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d adapters, want 2", len(all))
	}
	if all[0].ID() != aiframework.ClaudeCode || all[1].ID() != aiframework.GeminiCLI {
		t.Errorf("All() order = [%s %s], want [claudecode gemini]", all[0].ID(), all[1].ID())
	}
}

func TestRegisterDuplicate(t *testing.T) {
	t.Parallel()
	r := NewAdapterRegistry()
	if err := r.Register(stubAdapter{id: aiframework.ClaudeCode}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(stubAdapter{id: aiframework.ClaudeCode}); err == nil {
		t.Fatal("expected duplicate registration to error, got nil")
	}
}

func TestDefaultClientMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		id     aiframework.FrameworkID
		client string
		want   bool
	}{
		{"claudecode hyphenated", aiframework.ClaudeCode, "claude-code", true},
		{"claudecode spaced titled", aiframework.ClaudeCode, "Claude Code", true},
		{"claudecode unrelated", aiframework.ClaudeCode, "someide", false},
		{"gemini cli suffix", aiframework.GeminiCLI, "Gemini CLI", true},
		{"cursor exact", aiframework.Cursor, "cursor", true},
		{"empty client name", aiframework.ClaudeCode, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DefaultClientMatch(tt.id, ClientInfo{Name: tt.client})
			if got != tt.want {
				t.Errorf("DefaultClientMatch(%q, %q) = %v, want %v", tt.id, tt.client, got, tt.want)
			}
		})
	}
}

func TestDetectFrameworks(t *testing.T) {
	t.Parallel()
	r := NewAdapterRegistry()
	// A ClientMatcher adapter standing in for Claude Code: matches any client
	// whose name contains "claude" (case-insensitively), mirroring the real
	// adapter's override.
	cc := matchingAdapter{
		stubAdapter: stubAdapter{
			id: aiframework.ClaudeCode,
			match: func(c ClientInfo) bool {
				return strings.Contains(strings.ToLower(c.Name), "claude")
			},
		},
	}
	// A default-matched adapter that has no ClientMatcher.
	gemini := stubAdapter{id: aiframework.GeminiCLI}
	if err := r.Register(cc); err != nil {
		t.Fatalf("register cc: %v", err)
	}
	if err := r.Register(gemini); err != nil {
		t.Fatalf("register gemini: %v", err)
	}

	tests := []struct {
		name   string
		client string
		want   []aiframework.FrameworkID
	}{
		{"claude-code matches cc only", "claude-code", []aiframework.FrameworkID{aiframework.ClaudeCode}},
		{"Claude Code matches cc only", "Claude Code", []aiframework.FrameworkID{aiframework.ClaudeCode}},
		{"gemini matches gemini only", "gemini-cli", []aiframework.FrameworkID{aiframework.GeminiCLI}},
		{"unrelated matches nothing", "someide", nil},
		{"empty matches nothing", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.DetectFrameworks(ClientInfo{Name: tt.client})
			gotIDs := make([]aiframework.FrameworkID, 0, len(got))
			for _, a := range got {
				gotIDs = append(gotIDs, a.ID())
			}
			if len(gotIDs) != len(tt.want) {
				t.Fatalf("DetectFrameworks(%q) = %v, want %v", tt.client, gotIDs, tt.want)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.want[i] {
					t.Errorf("DetectFrameworks(%q)[%d] = %q, want %q", tt.client, i, gotIDs[i], tt.want[i])
				}
			}
		})
	}
}

// TestNormalizeIdentToken pins the normalization that DefaultClientMatch relies
// on: lowercase and strip every non-alphanumeric rune.
func TestNormalizeIdentToken(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"Claude Code", "claudecode"},
		{"claude-code", "claudecode"},
		{"Gemini CLI", "geminicli"},
		{"", ""},
		{"---", ""},
		{"v1.2_x", "v12x"},
	}
	for _, tt := range tests {
		if got := normalizeIdentToken(tt.in); got != tt.want {
			t.Errorf("normalizeIdentToken(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
