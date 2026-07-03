package devenv

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestIsSensitiveEnvWithholdsConcatenatedCredentials proves the env_info probe
// withholds credential variables whose keyword is concatenated to another word
// with no separator: AUTHORIZATION / PROXY_AUTHORIZATION ("auth" with no
// right-hand token boundary) and bare PRIVATE / PRIVATE_FOO. Token-boundary
// matching alone missed these, so their values (e.g. a "Basic <base64>"
// Authorization header, which value-shape redaction does not catch) leaked.
func TestIsSensitiveEnvWithholdsConcatenatedCredentials(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"AUTHORIZATION", "PROXY_AUTHORIZATION", "PRIVATE", "PRIVATE_FOO"} {
		if !isSensitiveEnv(name) {
			t.Errorf("isSensitiveEnv(%q) = false, want true (its value would be emitted)", name)
		}
	}
}

// TestEnvInfoWithholdsAuthorizationValues proves the env probe never emits the
// value of these previously-leaking variables: each name is recorded in
// filtered_names (name only) and its value is absent from the whole serialized
// result. The sentinel values are low-entropy, hyphenated markers that the
// value-level redactor does not strip, so only name-based withholding can hide
// them — making this a true regression guard for the leak.
func TestEnvInfoWithholdsAuthorizationValues(t *testing.T) {
	// Build values at runtime (never contiguous secret-shaped literals) that
	// mirror the real leak vector (a "Basic <token>" header) yet stay
	// low-entropy so value-shape redaction leaves them intact.
	cases := map[string]string{
		"AUTHORIZATION":       "Basic " + "sentinel-auth-leak-42",
		"PROXY_AUTHORIZATION": "Basic " + "sentinel-proxy-auth-leak-42",
		"PRIVATE":             "sentinel-" + "private-leak-42",
		"PRIVATE_FOO":         "sentinel-" + "private-foo-leak-42",
	}
	for name, val := range cases {
		t.Setenv(name, val)
	}

	env := newEnvInfo(t.TempDir())
	res := call(t, env.handle, map[string]any{"probe": "env"})

	blob, err := json.Marshal(res.Structured)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(blob)
	for name, val := range cases {
		if strings.Contains(out, val) {
			t.Errorf("env_info leaked the value of %s: %s", name, out)
		}
	}

	structured := res.Structured.(map[string]any)
	envProbe := structured["env"].(map[string]any)
	filtered, _ := envProbe["filtered_names"].([]string)
	for name := range cases {
		found := false
		for _, n := range filtered {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not recorded as filtered; got %v", name, filtered)
		}
	}
}
