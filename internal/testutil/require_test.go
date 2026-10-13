package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// recorder is a testing.TB that records Skipf and Fatalf instead of ending
// the test. Like the real methods, both end the calling goroutine.
type recorder struct {
	testing.TB
	skipped, failed bool
	msg             string
}

func (r *recorder) Helper() {}

func (r *recorder) Skipf(format string, args ...any) {
	r.skipped = true
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.failed = true
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// record runs fn against a recorder on its own goroutine, so a Skipf or
// Fatalf ends only that goroutine, and returns what fn recorded.
func record(fn func(testing.TB)) *recorder {
	r := &recorder{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done
	return r
}

// emptyPath points PATH at a fresh empty directory, so no tool resolves.
func emptyPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func TestRequireTool_SkipsWhenAbsent(t *testing.T) {
	emptyPath(t)
	t.Setenv(string(RequireNix), "")
	var got string
	r := record(func(tb testing.TB) { got = RequireTool(tb, "nix", RequireNix) })
	if !r.skipped || r.failed {
		t.Fatalf("skipped=%v failed=%v, want a skip", r.skipped, r.failed)
	}
	if !strings.Contains(r.msg, "nix") {
		t.Errorf("skip message %q does not name the tool", r.msg)
	}
	if strings.Contains(r.msg, string(RequireNix)) {
		t.Errorf("skip message %q names the switch, which is unset", r.msg)
	}
	if got != "" {
		t.Errorf("RequireTool returned %q after skipping", got)
	}
}

func TestRequireTool_FailsWhenRequired(t *testing.T) {
	emptyPath(t)
	t.Setenv(string(RequireNix), "1")
	r := record(func(tb testing.TB) { RequireTool(tb, "nix-instantiate", RequireNix) })
	if !r.failed || r.skipped {
		t.Fatalf("skipped=%v failed=%v, want a failure", r.skipped, r.failed)
	}
	if want := "(" + string(RequireNix) + "=1)"; !strings.HasSuffix(r.msg, want) {
		t.Errorf("failure message %q does not end with %q", r.msg, want)
	}
	if !strings.Contains(r.msg, "nix-instantiate") {
		t.Errorf("failure message %q does not name the tool", r.msg)
	}
}

func TestRequireTool_ReturnsPath(t *testing.T) {
	dir := t.TempDir()
	name := "qsdev-fake-tool"
	file := name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	want := filepath.Join(dir, file)
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // an executable test fixture
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv(string(RequireNix), "1")
	var got string
	r := record(func(tb testing.TB) { got = RequireTool(tb, name, RequireNix) })
	if r.skipped || r.failed {
		t.Fatalf("skipped=%v failed=%v (%s), want the path", r.skipped, r.failed, r.msg)
	}
	if got != want {
		t.Errorf("RequireTool = %q, want %q", got, want)
	}
}

func TestUnavailable_SkipsOrFails(t *testing.T) {
	tests := []struct {
		name, value  string
		skip, failed bool
	}{
		{"unset skips", "", true, false},
		{"zero skips", "0", true, false},
		{"other value skips", "yes", true, false},
		{"one fails", "1", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(string(RequireE3), tt.value)
			// A literal % in the arguments must not be read as a verb.
			r := record(func(tb testing.TB) { Unavailable(tb, RequireE3, "probe %s failed", "100%") })
			if r.skipped != tt.skip || r.failed != tt.failed {
				t.Fatalf("skipped=%v failed=%v, want skipped=%v failed=%v", r.skipped, r.failed, tt.skip, tt.failed)
			}
			want := "probe 100% failed"
			if tt.failed {
				want += " (" + string(RequireE3) + "=1)"
			}
			if r.msg != want {
				t.Errorf("message = %q, want %q", r.msg, want)
			}
		})
	}
}

func TestSwitchTools(t *testing.T) {
	t.Parallel()
	for _, s := range Switches() {
		if len(s.Tools()) == 0 {
			t.Errorf("switch %s governs no tool", s)
		}
	}
	if got := RequireNix.Tools(); !strings.Contains(strings.Join(got, ","), "nix-instantiate") {
		t.Errorf("RequireNix.Tools() = %v, want nix-instantiate among them", got)
	}
	if got := Switch("QSDEV_UNKNOWN").Tools(); got != nil {
		t.Errorf("unknown switch governs %v, want none", got)
	}
}

func TestUnsetEnv_RestoresValue(t *testing.T) {
	const a, b = "QSDEV_TESTUTIL_UNSET_A", "QSDEV_TESTUTIL_UNSET_B"
	t.Setenv(a, "va")
	t.Setenv(b, "")
	t.Run("unset", func(t *testing.T) {
		UnsetEnv(t, a, b)
		for _, k := range []string{a, b} {
			if v, ok := os.LookupEnv(k); ok {
				t.Errorf("%s is still set to %q", k, v)
			}
		}
	})
	if v, ok := os.LookupEnv(a); !ok || v != "va" {
		t.Errorf("%s = %q (set %v) after the subtest, want restored to %q", a, v, ok, "va")
	}
	if v, ok := os.LookupEnv(b); !ok || v != "" {
		t.Errorf("%s = %q (set %v) after the subtest, want restored to empty", b, v, ok)
	}
}
