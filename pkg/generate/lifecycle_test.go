package generate_test

import (
	"errors"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// trackingHook records its execution and optionally returns an error.
type trackingHook struct {
	id       string
	executed *[]string
	err      error
}

func (h *trackingHook) Execute(_ generate.LifecycleContext) error {
	*h.executed = append(*h.executed, h.id)
	return h.err
}

func TestLifecyclePhase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phase generate.LifecyclePhase
		want  string
	}{
		{generate.PostCollect, "post-collect"},
		{generate.PostResolve, "post-resolve"},
		{generate.LifecyclePhase(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.phase.String(); got != tt.want {
				t.Errorf("LifecyclePhase(%d).String() = %q, want %q", int(tt.phase), got, tt.want)
			}
		})
	}
}

func TestLifecycleHookRegistry_ExecuteOrder(t *testing.T) {
	t.Parallel()

	var executed []string
	reg := generate.NewLifecycleHookRegistry()

	reg.Register(generate.HookRegistration{
		Owner:    "c",
		Phase:    generate.PostCollect,
		Priority: 30,
		Hook:     &trackingHook{id: "c", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "a",
		Phase:    generate.PostCollect,
		Priority: 10,
		Hook:     &trackingHook{id: "a", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "b",
		Phase:    generate.PostCollect,
		Priority: 20,
		Hook:     &trackingHook{id: "b", executed: &executed},
	})

	fragments := []types.FragmentEntry{{Owner: "test"}}
	ctx := generate.LifecycleContext{
		Phase:     generate.PostCollect,
		Fragments: &fragments,
	}

	if err := reg.Execute(generate.PostCollect, ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(executed) != 3 {
		t.Fatalf("len(executed) = %d, want 3", len(executed))
	}
	want := []string{"a", "b", "c"}
	for i, id := range want {
		if executed[i] != id {
			t.Errorf("executed[%d] = %q, want %q", i, executed[i], id)
		}
	}
}

func TestLifecycleHookRegistry_FiltersByPhase(t *testing.T) {
	t.Parallel()

	var executed []string
	reg := generate.NewLifecycleHookRegistry()

	reg.Register(generate.HookRegistration{
		Owner:    "collect-hook",
		Phase:    generate.PostCollect,
		Priority: 10,
		Hook:     &trackingHook{id: "collect", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "resolve-hook",
		Phase:    generate.PostResolve,
		Priority: 10,
		Hook:     &trackingHook{id: "resolve", executed: &executed},
	})

	fragments := []types.FragmentEntry{{Owner: "test"}}
	ctx := generate.LifecycleContext{
		Phase:     generate.PostCollect,
		Fragments: &fragments,
	}

	if err := reg.Execute(generate.PostCollect, ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(executed) != 1 {
		t.Fatalf("len(executed) = %d, want 1", len(executed))
	}
	if executed[0] != "collect" {
		t.Errorf("executed[0] = %q, want %q", executed[0], "collect")
	}
}

func TestLifecycleHookRegistry_ErrorStopsExecution(t *testing.T) {
	t.Parallel()

	var executed []string
	errBoom := errors.New("boom")
	reg := generate.NewLifecycleHookRegistry()

	reg.Register(generate.HookRegistration{
		Owner:    "first",
		Phase:    generate.PostCollect,
		Priority: 10,
		Hook:     &trackingHook{id: "first", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "second",
		Phase:    generate.PostCollect,
		Priority: 20,
		Hook:     &trackingHook{id: "second", executed: &executed, err: errBoom},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "third",
		Phase:    generate.PostCollect,
		Priority: 30,
		Hook:     &trackingHook{id: "third", executed: &executed},
	})

	fragments := []types.FragmentEntry{{Owner: "test"}}
	ctx := generate.LifecycleContext{
		Phase:     generate.PostCollect,
		Fragments: &fragments,
	}

	err := reg.Execute(generate.PostCollect, ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errBoom) {
		t.Errorf("error = %v, want wrapping of %v", err, errBoom)
	}

	if len(executed) != 2 {
		t.Fatalf("len(executed) = %d, want 2 (third should not run)", len(executed))
	}
	if executed[0] != "first" || executed[1] != "second" {
		t.Errorf("executed = %v, want [first second]", executed)
	}
}

func TestLifecycleHookRegistry_RemoveByOwner(t *testing.T) {
	t.Parallel()

	var executed []string
	reg := generate.NewLifecycleHookRegistry()

	reg.Register(generate.HookRegistration{
		Owner:    "alice",
		Phase:    generate.PostCollect,
		Priority: 10,
		Hook:     &trackingHook{id: "alice-1", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "bob",
		Phase:    generate.PostCollect,
		Priority: 20,
		Hook:     &trackingHook{id: "bob-1", executed: &executed},
	})
	reg.Register(generate.HookRegistration{
		Owner:    "alice",
		Phase:    generate.PostResolve,
		Priority: 10,
		Hook:     &trackingHook{id: "alice-2", executed: &executed},
	})

	if reg.HookCount() != 3 {
		t.Fatalf("HookCount() = %d, want 3", reg.HookCount())
	}

	reg.RemoveByOwner("alice")

	if reg.HookCount() != 1 {
		t.Fatalf("HookCount() after removal = %d, want 1", reg.HookCount())
	}

	// Execute PostCollect to verify only bob's hook remains.
	fragments := []types.FragmentEntry{{Owner: "test"}}
	ctx := generate.LifecycleContext{
		Phase:     generate.PostCollect,
		Fragments: &fragments,
	}

	if err := reg.Execute(generate.PostCollect, ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(executed) != 1 {
		t.Fatalf("len(executed) = %d, want 1", len(executed))
	}
	if executed[0] != "bob-1" {
		t.Errorf("executed[0] = %q, want %q", executed[0], "bob-1")
	}
}

func TestLifecycleHookRegistry_NilSafe(t *testing.T) {
	t.Parallel()

	var reg *generate.LifecycleHookRegistry

	if err := reg.Execute(generate.PostCollect, generate.LifecycleContext{}); err != nil {
		t.Errorf("Execute on nil registry = %v, want nil", err)
	}
	if got := reg.HookCount(); got != 0 {
		t.Errorf("HookCount on nil registry = %d, want 0", got)
	}
}

func TestLifecycleHookRegistry_PostCollectMutatesFragments(t *testing.T) {
	t.Parallel()

	reg := generate.NewLifecycleHookRegistry()

	// Register a hook that appends a fragment to the slice.
	reg.Register(generate.HookRegistration{
		Owner:    "injector",
		Phase:    generate.PostCollect,
		Priority: 10,
		Hook: generate.LifecycleHookFunc(func(ctx generate.LifecycleContext) error {
			*ctx.Fragments = append(*ctx.Fragments, types.FragmentEntry{
				Owner:   "injector",
				Target:  "injected.nix",
				Content: []byte("# injected"),
			})
			return nil
		}),
	})

	fragments := []types.FragmentEntry{
		{Owner: "original", Target: "main.nix", Content: []byte("# original")},
	}
	ctx := generate.LifecycleContext{
		Phase:     generate.PostCollect,
		Fragments: &fragments,
	}

	if err := reg.Execute(generate.PostCollect, ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(fragments) != 2 {
		t.Fatalf("len(fragments) = %d, want 2", len(fragments))
	}
	if fragments[1].Owner != "injector" {
		t.Errorf("fragments[1].Owner = %q, want %q", fragments[1].Owner, "injector")
	}
	if fragments[1].Target != "injected.nix" {
		t.Errorf("fragments[1].Target = %q, want %q", fragments[1].Target, "injected.nix")
	}
}

func TestLifecycleHookRegistry_PostResolveMutatesFiles(t *testing.T) {
	t.Parallel()

	reg := generate.NewLifecycleHookRegistry()

	// Register a hook that appends a generated file.
	reg.Register(generate.HookRegistration{
		Owner:    "appender",
		Phase:    generate.PostResolve,
		Priority: 10,
		Hook: generate.LifecycleHookFunc(func(ctx generate.LifecycleContext) error {
			*ctx.Files = append(*ctx.Files, types.GeneratedFile{
				Path:    "extra.yaml",
				Content: []byte("extra: true\n"),
				Owner:   "appender",
			})
			return nil
		}),
	})

	files := []types.GeneratedFile{
		{Path: "main.yaml", Content: []byte("main: true\n"), Owner: "core"},
	}
	ctx := generate.LifecycleContext{
		Phase: generate.PostResolve,
		Files: &files,
	}

	if err := reg.Execute(generate.PostResolve, ctx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[1].Path != "extra.yaml" {
		t.Errorf("files[1].Path = %q, want %q", files[1].Path, "extra.yaml")
	}
	if files[1].Owner != "appender" {
		t.Errorf("files[1].Owner = %q, want %q", files[1].Owner, "appender")
	}
}
