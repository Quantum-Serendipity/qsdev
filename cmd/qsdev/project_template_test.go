package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
)

// Preparing a project runs qsdev (init, disable, ...) in fresh processes,
// which dominates this package's run time under -race when hundreds of
// subtests each prepare the same project. A prepared project holds no path
// of the directory, HOME or TMPDIR it was made in, so cachedProject makes it
// once per kind and hands every later caller a fresh copy of it, run with
// the caller's own isolated environment.

// projectTemplate is the cached result of one preparation.
type projectTemplate struct {
	once      sync.Once
	dir       string // a copy outside every test's temp dir; "" if preparing failed
	committed bool
}

var (
	projectTemplates sync.Map // key -> *projectTemplate

	templateRootOnce sync.Once
	templateRoot     string
	templateRootErr  error
)

// cachedProject returns what prepare(t, env, optedOut) returns: the first
// caller for key and optedOut prepares the project itself, every later one
// gets a copy of that project in its own temp dir.
func cachedProject(t *testing.T, key string, env []string, optedOut bool,
	prepare func(*testing.T, []string, bool) (string, bool),
) (string, bool) {
	t.Helper()
	key += "/opted-out=" + strconv.FormatBool(optedOut)
	v, _ := projectTemplates.LoadOrStore(key, &projectTemplate{})
	tmpl := v.(*projectTemplate)

	var dir string
	var committed, prepared bool
	tmpl.once.Do(func() {
		prepared = true
		dir, committed = prepare(t, env, optedOut)
		saved, err := saveProjectTemplate(dir)
		if err != nil {
			t.Fatalf("saving the %s project template: %v", key, err)
		}
		tmpl.dir, tmpl.committed = saved, committed
	})
	if prepared {
		return dir, committed
	}
	if tmpl.dir == "" {
		t.Fatalf("preparing the %s project failed in another test", key)
	}
	dst := t.TempDir()
	if err := os.CopyFS(dst, os.DirFS(tmpl.dir)); err != nil {
		t.Fatalf("copying the %s project template: %v", key, err)
	}
	return dst, tmpl.committed
}

// saveProjectTemplate copies dir into a new directory under the package's
// template root, which outlives the test that prepared it.
func saveProjectTemplate(dir string) (string, error) {
	templateRootOnce.Do(func() {
		templateRoot, templateRootErr = os.MkdirTemp("", "qsdev-project-templates-")
	})
	if templateRootErr != nil {
		return "", fmt.Errorf("creating the template root: %w", templateRootErr)
	}
	saved, err := os.MkdirTemp(templateRoot, "project-")
	if err != nil {
		return "", fmt.Errorf("creating a template dir: %w", err)
	}
	if err := os.CopyFS(saved, os.DirFS(dir)); err != nil {
		return "", fmt.Errorf("copying %s: %w", dir, err)
	}
	return saved, nil
}

// removeProjectTemplates deletes every saved template; TestMain calls it once
// the tests are done.
func removeProjectTemplates() {
	if templateRoot != "" {
		_ = os.RemoveAll(templateRoot)
	}
}

// templateCleanupRunner runs the tests and then removes the saved templates
// and the guardrail PATH's bin dir, so the cleanup happens before gdev's
// TestMain exits.
type templateCleanupRunner struct{ m *testing.M }

func (r templateCleanupRunner) Run() int {
	code := r.m.Run()
	removeProjectTemplates()
	removeGuardrailBinDir()
	return code
}
