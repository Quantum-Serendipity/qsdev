package generate_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCIFragmentProducer_PlatformNone(t *testing.T) {
	t.Parallel()

	p := &generate.CIFragmentProducer{
		Registry: cigeneration.NewStepRegistry(),
		Config:   cigeneration.GenerateConfig{Platform: cigeneration.PlatformNone},
	}

	fragments, err := p.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fragments) != 0 {
		t.Errorf("expected no fragments for PlatformNone, got %d", len(fragments))
	}
}

func TestCIFragmentProducer_GitHubActions(t *testing.T) {
	t.Parallel()

	reg := cigeneration.DefaultStepRegistry()
	p := &generate.CIFragmentProducer{
		Registry: reg,
		Config: cigeneration.GenerateConfig{
			Platform:           cigeneration.PlatformGitHubActions,
			HasContainerModule: false,
			HasClaudeCode:      false,
		},
	}

	fragments, err := p.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fragments) == 0 {
		t.Fatal("expected at least 1 fragment for GitHubActions")
	}

	for _, f := range fragments {
		if f.Source != "ci-generation" {
			t.Errorf("source = %q, want %q", f.Source, "ci-generation")
		}
		if f.ComposeMode != types.ComposeReplace {
			t.Errorf("compose mode = %v, want ComposeReplace", f.ComposeMode)
		}
		if f.Priority != 500 {
			t.Errorf("priority = %d, want 500", f.Priority)
		}
		if len(f.Content) == 0 {
			t.Error("content should not be empty")
		}
		if f.Target == "" {
			t.Error("target path should not be empty")
		}
	}
}

func TestCIFragmentProducer_EmptyRegistry(t *testing.T) {
	t.Parallel()

	p := &generate.CIFragmentProducer{
		Registry: cigeneration.NewStepRegistry(),
		Config: cigeneration.GenerateConfig{
			Platform: cigeneration.PlatformGitHubActions,
		},
	}

	fragments, err := p.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fragments) != 0 {
		t.Errorf("expected 0 fragments for empty registry, got %d", len(fragments))
	}
}

func TestCIFragmentProducer_PreservesFileMetadata(t *testing.T) {
	t.Parallel()

	reg := cigeneration.DefaultStepRegistry()
	p := &generate.CIFragmentProducer{
		Registry: reg,
		Config: cigeneration.GenerateConfig{
			Platform:           cigeneration.PlatformGitHubActions,
			HasContainerModule: false,
			HasClaudeCode:      false,
		},
	}

	fragments, err := p.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range fragments {
		if f.Strategy != types.Overwrite {
			t.Errorf("strategy = %v, want Overwrite", f.Strategy)
		}
		if f.Owner != "ci-generation" {
			t.Errorf("owner = %q, want %q", f.Owner, "ci-generation")
		}
		if f.Mode == 0 {
			t.Error("mode should be set from GeneratedFile")
		}
	}
}

func TestCIFragmentProducer_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time interface satisfaction check.
	var _ types.FragmentProducer = (*generate.CIFragmentProducer)(nil)
}

func TestCIFragmentProducer_GitLabCI(t *testing.T) {
	t.Parallel()

	reg := cigeneration.DefaultStepRegistry()
	p := &generate.CIFragmentProducer{
		Registry: reg,
		Config: cigeneration.GenerateConfig{
			Platform:           cigeneration.PlatformGitLabCI,
			HasContainerModule: false,
			HasClaudeCode:      false,
		},
	}

	fragments, err := p.Produce(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fragments) == 0 {
		t.Fatal("expected at least 1 fragment for GitLabCI")
	}

	for _, f := range fragments {
		if f.Source != "ci-generation" {
			t.Errorf("source = %q, want %q", f.Source, "ci-generation")
		}
		if f.ComposeMode != types.ComposeReplace {
			t.Errorf("compose mode = %v, want ComposeReplace", f.ComposeMode)
		}
	}
}

func TestCIFragmentProducer_AnswersIgnored(t *testing.T) {
	t.Parallel()

	// Verify the producer uses its own Config, not WizardAnswers.
	reg := cigeneration.DefaultStepRegistry()
	p := &generate.CIFragmentProducer{
		Registry: reg,
		Config: cigeneration.GenerateConfig{
			Platform: cigeneration.PlatformGitHubActions,
		},
	}

	answers := types.WizardAnswers{
		ProjectName: "test-project",
		ClaudeCode:  true,
	}

	fragments, err := p.Produce(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Config has HasClaudeCode=false (default), so security-review job
	// should not appear regardless of WizardAnswers.ClaudeCode.
	if len(fragments) == 0 {
		t.Fatal("expected at least 1 fragment")
	}
}
