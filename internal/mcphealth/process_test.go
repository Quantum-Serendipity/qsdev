package mcphealth

import (
	"os"
	"slices"
	"testing"
)

func TestBuildProcessEnv_ResolvesHomeVar(t *testing.T) {
	t.Parallel()

	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	env := map[string]string{
		"TEST_DIR": "${HOME}/subdir",
	}

	result := buildProcessEnv(env)

	expected := "TEST_DIR=" + home + "/subdir"
	if !slices.Contains(result, expected) {
		t.Errorf("expected %q in env, not found", expected)
	}
}

func TestBuildProcessEnv_MissingVarStaysLiteral(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"TOKEN": "${QSDEV_TEST_NONEXISTENT_VAR_ABCDEF}",
	}

	result := buildProcessEnv(env)

	expected := "TOKEN=${QSDEV_TEST_NONEXISTENT_VAR_ABCDEF}"
	if !slices.Contains(result, expected) {
		t.Errorf("expected %q in env (literal unresolved), not found", expected)
	}
}

func TestBuildProcessEnv_InheritsOSEnv(t *testing.T) {
	t.Parallel()

	result := buildProcessEnv(nil)

	if len(result) == 0 {
		t.Error("expected inherited OS environment entries, got empty slice")
	}
}
