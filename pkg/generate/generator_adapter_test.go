package generate

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type mockGenerator struct {
	files []types.GeneratedFile
	err   error
}

func (m *mockGenerator) Generate(types.WizardAnswers) ([]types.GeneratedFile, error) {
	return m.files, m.err
}

func TestGeneratorAdapter_Identity(t *testing.T) {
	t.Parallel()

	gen := &mockGenerator{
		files: []types.GeneratedFile{
			{Path: "a.txt", Content: []byte("alpha"), Mode: 0o644, Strategy: types.Overwrite},
			{Path: "b.sh", Content: []byte("#!/bin/sh"), Mode: 0o755, Strategy: types.Skip},
		},
	}

	adapter := NewGeneratorAdapter("testgen", gen)
	fragments, err := adapter.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("expected 2 fragments, got %d", len(fragments))
	}

	// Resolve through accumulator to verify identity.
	a := NewFragmentAccumulator()
	a.AddBatch(fragments)

	files, err := a.Resolve()
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	for i, f := range files {
		orig := gen.files[i]
		if f.Path != orig.Path {
			t.Errorf("file %d: expected path %q, got %q", i, orig.Path, f.Path)
		}
		if string(f.Content) != string(orig.Content) {
			t.Errorf("file %d: expected content %q, got %q", i, string(orig.Content), string(f.Content))
		}
		expectedMode := orig.Mode
		if expectedMode == 0 {
			expectedMode = 0o644
		}
		if f.Mode != expectedMode {
			t.Errorf("file %d: expected mode %o, got %o", i, expectedMode, f.Mode)
		}
		if f.Strategy != orig.Strategy {
			t.Errorf("file %d: expected strategy %v, got %v", i, orig.Strategy, f.Strategy)
		}
	}
}

func TestGeneratorAdapter_PreservesOwner(t *testing.T) {
	t.Parallel()

	gen := &mockGenerator{
		files: []types.GeneratedFile{
			{Path: "owned.txt", Content: []byte("data"), Owner: "devenv"},
		},
	}

	adapter := NewGeneratorAdapter("testgen", gen)
	fragments, err := adapter.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	if fragments[0].Owner != "devenv" {
		t.Fatalf("expected owner 'devenv', got %q", fragments[0].Owner)
	}
}

func TestGeneratorAdapter_PropagatesError(t *testing.T) {
	t.Parallel()

	gen := &mockGenerator{err: errTest}
	adapter := NewGeneratorAdapter("testgen", gen)

	_, err := adapter.Produce(types.WizardAnswers{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGeneratorAdapter_DefaultMode(t *testing.T) {
	t.Parallel()

	gen := &mockGenerator{
		files: []types.GeneratedFile{
			{Path: "nomode.txt", Content: []byte("data"), Mode: 0},
		},
	}

	adapter := NewGeneratorAdapter("testgen", gen)
	fragments, err := adapter.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	if fragments[0].Mode != os.FileMode(0o644) {
		t.Fatalf("expected default mode 0644, got %o", fragments[0].Mode)
	}
}
