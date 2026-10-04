package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// pinWorld is an isolated layout for the org overlay pin tests: a project, a
// temporary directory (TMPDIR, TMP and TEMP point at it) and two overlay
// files outside both.
type pinWorld struct {
	project, tmp, home string
	homeOverlay, other string
}

// newPinWorld builds a pinWorld and makes the account's home directory its
// home. Not parallel-safe: it sets the environment and package variables.
func newPinWorld(t *testing.T) pinWorld {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := pinWorld{
		project: filepath.Join(base, "project"),
		tmp:     filepath.Join(base, "tmp"),
		home:    filepath.Join(base, "home"),
		other:   filepath.Join(base, "elsewhere", "defaults.yaml"),
	}
	w.homeOverlay = HomeOrgConfigPath(w.home)
	for _, d := range []string{w.project, w.tmp, filepath.Dir(w.homeOverlay), filepath.Dir(w.other)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{w.homeOverlay, w.other} {
		if err := os.WriteFile(f, []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, w.tmp)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	origAccount, origPins, origDefault := accountHome, pinsHome, defaultOrgConfigPath
	accountHome = func() (string, error) { return w.home, nil }
	pinsHome = accountHome
	defaultOrgConfigPath = homeOrgConfigPath
	t.Cleanup(func() { accountHome, pinsHome, defaultOrgConfigPath = origAccount, origPins, origDefault })
	t.Cleanup(func() {
		pinMu.Lock()
		pinActive, pinRoot, pinned = false, "", OrgConfigPin{}
		pinMu.Unlock()
	})
	return w
}

// writeAt writes an overlay at p and returns p.
func writeAt(t *testing.T, p string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPolicyOrgConfigFile pins the structural org overlay control (U18-WS1
// rounds 3 and 4): every run reads the overlay a human pinned, whatever
// <EnvPrefix>ORG_CONFIG says (the hook cannot see every way the variable is
// set: a devenv import, a sourced fragment, an encoded name, and whether a
// human runs the CLI cannot be told reliably), without a pin only the
// account's home overlay, and never one below the project or the temporary
// directory.
func TestPolicyOrgConfigFile(t *testing.T) {
	env := branding.Get().EnvPrefix + "ORG_CONFIG"
	tests := []struct {
		name      string
		orgConfig func(w pinWorld) string // the variable's value
		pin       func(w pinWorld) OrgConfigPin
		want      func(w pinWorld) string
		drifts    bool
	}{
		{
			name:      "variable points away from the recorded overlay",
			orgConfig: func(w pinWorld) string { return w.other },
			pin:       func(w pinWorld) OrgConfigPin { return OrgConfigPin{Path: w.homeOverlay, Recorded: true} },
			want:      func(w pinWorld) string { return w.homeOverlay },
			drifts:    true,
		},
		{
			name:      "recorded overlay",
			orgConfig: func(w pinWorld) string { return w.other },
			pin:       func(w pinWorld) OrgConfigPin { return OrgConfigPin{Path: w.other, Recorded: true} },
			want:      func(w pinWorld) string { return w.other },
		},
		{
			name:      "recorded no overlay",
			orgConfig: func(w pinWorld) string { return w.other },
			pin:       func(pinWorld) OrgConfigPin { return OrgConfigPin{Recorded: true} },
			want:      func(pinWorld) string { return "" },
			drifts:    true,
		},
		{
			name:      "overlay below the project without a pin",
			orgConfig: func(w pinWorld) string { return writeAt(t, filepath.Join(w.project, "x.yaml")) },
			pin:       func(pinWorld) OrgConfigPin { return OrgConfigPin{} },
			want:      func(w pinWorld) string { return w.homeOverlay },
			drifts:    true,
		},
		{
			name:      "overlay below the temporary directory without a pin",
			orgConfig: func(w pinWorld) string { return writeAt(t, filepath.Join(w.tmp, "x", "defaults.yaml")) },
			pin:       func(pinWorld) OrgConfigPin { return OrgConfigPin{} },
			want:      func(w pinWorld) string { return w.homeOverlay },
			drifts:    true,
		},
		{
			// Round 4: an overlay outside the project and the temporary
			// directory ($HOME/evil.yaml, /tmp/claude-<uid>/x when TMPDIR is
			// elsewhere) is not approved by lying there.
			name:      "overlay elsewhere without a pin",
			orgConfig: func(w pinWorld) string { return w.other },
			pin:       func(pinWorld) OrgConfigPin { return OrgConfigPin{} },
			want:      func(w pinWorld) string { return w.homeOverlay },
			drifts:    true,
		},
		{
			name:      "home overlay without a pin",
			orgConfig: func(pinWorld) string { return "" },
			pin:       func(pinWorld) OrgConfigPin { return OrgConfigPin{} },
			want:      func(w pinWorld) string { return w.homeOverlay },
		},
		{
			name:      "variable points away from the account-wide pin",
			orgConfig: func(w pinWorld) string { return w.homeOverlay },
			pin:       func(w pinWorld) OrgConfigPin { return OrgConfigPin{Path: w.other, Recorded: true, Global: true} },
			want:      func(w pinWorld) string { return w.other },
			drifts:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newPinWorld(t)
			t.Setenv(env, tt.orgConfig(w))
			pin := tt.pin(w)
			if drift := OrgConfigDrift(w.project, pin); (drift != "") != tt.drifts {
				t.Errorf("OrgConfigDrift = %q, want drift %v", drift, tt.drifts)
			}
			if got, want := PolicyOrgConfigFile(), OrgConfigFile(); got != want {
				t.Errorf("PolicyOrgConfigFile() without a pin in force = %q, want %q", got, want)
			}
			UseOrgConfigPin(w.project, pin)
			if got, want := PolicyOrgConfigFile(), tt.want(w); got != want {
				t.Errorf("PolicyOrgConfigFile() = %q, want %q", got, want)
			}
		})
	}
}

// TestRecordOrgConfigPin pins what 'defaults pin' records: the resolved
// overlay path, kept in the account's home configuration directory (so no
// command in the checkout, such as `git clean -fdX`, removes it) and read
// back by LoadOrgConfigPin, a project pin over the account-wide one, and
// nothing for an overlay below the project.
func TestRecordOrgConfigPin(t *testing.T) {
	env := branding.Get().EnvPrefix + "ORG_CONFIG"
	w := newPinWorld(t)

	if pin, err := LoadOrgConfigPin(w.project); err != nil || pin.Recorded {
		t.Fatalf("LoadOrgConfigPin before recording = %+v, %v; want none recorded", pin, err)
	}
	t.Setenv(env, w.other)
	if got, err := RecordOrgConfigPin(w.project); err != nil || got != w.other {
		t.Fatalf("RecordOrgConfigPin = %q, %v; want %q", got, err, w.other)
	}
	file, err := OrgPinsFile()
	if err != nil || filepath.Dir(file) != filepath.Dir(w.homeOverlay) {
		t.Fatalf("OrgPinsFile = %q, %v; want it next to the home overlay %s", file, err, w.homeOverlay)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("the pin was not recorded in %s: %v", file, err)
	}
	pin, err := LoadOrgConfigPin(w.project)
	if err != nil || !pin.Recorded || pin.Global || pin.Path != w.other {
		t.Fatalf("LoadOrgConfigPin = %+v, %v; want %q recorded for the project", pin, err, w.other)
	}
	if drift := ProjectOrgConfigDrift(w.project); drift != "" {
		t.Errorf("ProjectOrgConfigDrift after recording = %q, want none", drift)
	}
	if src := ProjectOrgConfigSource(w.project); !strings.Contains(src, w.other) || !strings.Contains(src, "pinned for the project") {
		t.Errorf("ProjectOrgConfigSource = %q, want the project pin", src)
	}

	t.Setenv(env, "")
	if got, err := RecordOrgConfigPin(""); err != nil || got != w.homeOverlay {
		t.Fatalf("RecordOrgConfigPin(\"\") = %q, %v; want %q", got, err, w.homeOverlay)
	}
	if pin, err := LoadOrgConfigPin(w.project); err != nil || pin.Path != w.other || pin.Global {
		t.Errorf("project pin after an account-wide pin = %+v, %v; want %q kept", pin, err, w.other)
	}
	if pin, err := LoadOrgConfigPin(w.tmp); err != nil || !pin.Global || pin.Path != w.homeOverlay {
		t.Errorf("pin of an unpinned directory = %+v, %v; want the account-wide %q", pin, err, w.homeOverlay)
	}

	t.Setenv(env, writeAt(t, filepath.Join(w.project, "evil.yaml")))
	if _, err := RecordOrgConfigPin(w.project); err == nil || !strings.Contains(err.Error(), "below the project") {
		t.Errorf("RecordOrgConfigPin for an overlay below the project = %v, want a refusal", err)
	}
	if pin, err := LoadOrgConfigPin(w.project); err != nil || pin.Path != w.other {
		t.Errorf("LoadOrgConfigPin after a refusal = %+v, %v; want %q kept", pin, err, w.other)
	}
	if drift := ProjectOrgConfigDrift(w.project); !strings.Contains(drift, "below the project") {
		t.Errorf("ProjectOrgConfigDrift = %q, want the overlay below the project reported", drift)
	}

	if err := os.WriteFile(file, []byte("projects: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrgConfigPin(w.project); err == nil {
		t.Error("LoadOrgConfigPin of a malformed pins file succeeded, want an error")
	}
}

// TestUnanchoredAccountReadsOrgConfig pins the account the user database does
// not know (an arbitrary container or CI uid; regression of U18-WS1 against
// main): with no home directory to keep a pin in, the overlay
// <EnvPrefix>ORG_CONFIG names is read as before pins existed, except one
// below the project or the temporary directory, and 'defaults pin' explains
// that no pin is needed. A lookup that fails another way (a hung directory
// service) still reads no overlay. Not parallel: it sets the environment and
// package variables.
func TestUnanchoredAccountReadsOrgConfig(t *testing.T) {
	env := branding.Get().EnvPrefix + "ORG_CONFIG"
	w := newPinWorld(t)
	accountHome = func() (string, error) {
		return "", fmt.Errorf("looking up the current account: %w", userhome.ErrNoAccount)
	}
	pinsHome = accountHome
	t.Setenv(env, w.other)

	pin, err := LoadOrgConfigPin(w.project)
	if err != nil || !pin.Unanchored || pin.Recorded {
		t.Fatalf("LoadOrgConfigPin without an account = %+v, %v; want an unanchored pin", pin, err)
	}
	if drift := ProjectOrgConfigDrift(w.project); drift != "" {
		t.Errorf("ProjectOrgConfigDrift without an account = %q, want none", drift)
	}
	if src := ProjectOrgConfigSource(w.project); !strings.Contains(src, w.other) || !strings.Contains(src, env) {
		t.Errorf("ProjectOrgConfigSource = %q, want the overlay %s names", src, env)
	}
	UseOrgConfigPin(w.project, pin)
	if got := PolicyOrgConfigFile(); got != w.other {
		t.Errorf("PolicyOrgConfigFile() without an account = %q, want %q", got, w.other)
	}
	if _, err := RecordOrgConfigPin(w.project); err == nil || !strings.Contains(err.Error(), env) {
		t.Errorf("RecordOrgConfigPin without an account = %v, want an error naming %s", err, env)
	}

	t.Setenv(env, writeAt(t, filepath.Join(w.project, "evil.yaml")))
	if got := PolicyOrgConfigFile(); got != "" {
		t.Errorf("PolicyOrgConfigFile() for an overlay below the project = %q, want none", got)
	}

	pinsHome = func() (string, error) { return "", errors.New("getent timed out") }
	if pin, err := LoadOrgConfigPin(w.project); err == nil || pin.Unanchored {
		t.Errorf("LoadOrgConfigPin after a failed lookup = %+v, %v; want an error, not an unanchored pin", pin, err)
	}
}
