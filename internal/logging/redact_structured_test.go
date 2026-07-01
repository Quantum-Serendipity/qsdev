package logging

import (
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

// Secret-shaped fixtures are built by concatenation so the contiguous literal
// never appears in source (the repo convention for not tripping ripsecrets).
const (
	awsKey  = "AKIA" + "IOSFODNN7EXAMPLE"
	ghToken = "ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef1234"
)

// row is a small all-exported struct, exercising the reflection struct walk
// (redactStruct) directly rather than the JSON-normalization fallback.
type row struct {
	Name string
	Note string
}

// auditEntry carries an unexported field, so it is routed through the
// JSON-normalization path (redactViaJSON). It guards the fix that restores the
// original concrete type after redaction (see TestRedactStructured_UnexportedFieldStruct).
type auditEntry struct {
	Detail string
	seq    int //nolint:unused // present to force the unexported-field code path
}

// secretRow is an all-exported struct whose field names hit the key deny-list,
// exercising redactStruct's field-name deny-list check: a value the pattern
// scrub alone would miss (Password/APIKey hold plain text) must still be redacted
// because the field NAME is sensitive. APIKey uses a json tag to prove the tag —
// not the Go field name — is what the deny-list matches.
type secretRow struct {
	Password string
	APIKey   string `json:"api_key"`
	Host     string
}

// rejectingDetail has an unexported field (routing it through redactViaJSON) and
// a custom UnmarshalJSON that fails once the value carries the redaction marker,
// so redactViaJSON's best-effort reencode cannot restore the concrete type.
// Nested in a concretely-typed slice, the redacted generic tree is unassignable
// to the slot — coerce must fail closed (zero the slot) instead of panicking.
type rejectingDetail struct {
	Detail string
	seq    int //nolint:unused // forces the unexported-field (redactViaJSON) path
}

func (d *rejectingDetail) UnmarshalJSON(b []byte) error {
	var aux struct{ Detail string }
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	if strings.Contains(aux.Detail, redacted) {
		return errors.New("rejectingDetail: refusing redaction marker")
	}
	d.Detail = aux.Detail
	return nil
}

func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal redacted value: %v", err)
	}
	return string(b)
}

// TestRedactStructured walks heterogeneous payloads and asserts (via the
// JSON wire form) that secrets are gone, the redaction marker is present, and
// non-secret siblings are preserved. It covers nested generic containers
// (regression), concrete containers reached only through reflection, struct
// fields, and embedded URL credentials.
func TestRedactStructured(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	tests := []struct {
		name    string
		input   any
		absent  []string // secret fragments that must NOT survive
		present []string // substrings that MUST survive
	}{
		{
			name: "nested map[string]any and []any (regression)",
			input: map[string]any{
				"msg":   "hello world",
				"creds": map[string]any{"aws": awsKey},
				"items": []any{"safe", ghToken},
			},
			absent:  []string{awsKey, ghToken},
			present: []string{redacted, "hello world", "safe"},
		},
		{
			name:    "concrete map[string]string value is redacted",
			input:   map[string]string{"blob": "leaked " + awsKey, "name": "alice"},
			absent:  []string{awsKey},
			present: []string{redacted, "alice"},
		},
		{
			name:    "concrete []string element is redacted",
			input:   []string{"clean", "embedded " + ghToken},
			absent:  []string{ghToken},
			present: []string{redacted, "clean"},
		},
		{
			name: "concrete []map[string]any nested token is redacted",
			input: []map[string]any{
				{"x": "v"},
				{"nested": ghToken},
			},
			absent:  []string{ghToken},
			present: []string{redacted, "v"},
		},
		{
			name: "slice of exported struct has string fields redacted",
			input: []row{
				{Name: "build", Note: "see " + awsKey},
				{Name: "deploy", Note: "ok"},
			},
			absent:  []string{awsKey},
			present: []string{redacted, "build", "deploy", "ok"},
		},
		{
			name: "deny-listed keys redact non-secret-shaped values",
			input: map[string]any{
				"password": "pw-cleartext",
				"api_key":  "apikey-cleartext",
				"token":    "token-cleartext",
				"host":     "keepme",
			},
			absent:  []string{"pw-cleartext", "apikey-cleartext", "token-cleartext"},
			present: []string{redacted, "keepme"},
		},
		{
			// redactStruct must apply the key deny-list to FIELD NAMES (incl. the
			// json tag), not just map keys: a plain value behind a secret-named
			// field would otherwise survive the pattern scrub.
			name:    "struct field names hit the deny-list (incl. json tag)",
			input:   secretRow{Password: "pw-cleartext", APIKey: "apikey-cleartext", Host: "keepme"},
			absent:  []string{"pw-cleartext", "apikey-cleartext"},
			present: []string{redacted, "keepme"},
		},
		{
			// isKeyDenied lowercases, so a mixed-case key must still match.
			name:    "mixed-case deny-listed key is normalized",
			input:   map[string]any{"PassWord": "mc-cleartext", "Secret": "mc-secret", "ok": "fine"},
			absent:  []string{"mc-cleartext", "mc-secret"},
			present: []string{redacted, "fine"},
		},
		{
			// A denied key whose value is a nested container is wholesale-redacted
			// to the marker (the whole subtree is dropped), not walked into.
			name:    "denied key with nested-map value is wholesale-redacted",
			input:   map[string]any{"secret": map[string]any{"user": "uval", "pass": "p-cleartext"}},
			absent:  []string{"uval", "p-cleartext"},
			present: []string{redacted},
		},
		{
			name: "embedded URL credentials are redacted, host preserved",
			// Split so the user:pass@ credential is never a contiguous literal
			// (keeps the ripsecrets pre-commit hook happy); the runtime DSN still
			// exercises the URL-credential redaction path.
			input:   map[string]any{"dsn": "postgres://appuser:" + "s3cretpw" + "@pg.example.com:5432/maindb"},
			absent:  []string{"appuser", "s3cretpw"},
			present: []string{"pg.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := r.RedactStructured(tt.input)
			out := marshalJSON(t, got)
			for _, s := range tt.absent {
				if strings.Contains(out, s) {
					t.Errorf("secret fragment %q still present in output: %s", s, out)
				}
			}
			for _, s := range tt.present {
				if !strings.Contains(out, s) {
					t.Errorf("expected %q to be present in output: %s", s, out)
				}
			}
		})
	}
}

// TestRedactStructured_DenyListMatchesRedactAttr proves the structured walk
// applies the SAME key deny-list as the log surface (RedactAttr): a sensitive
// key name redacts its value even when the value is not secret-shaped.
func TestRedactStructured_DenyListMatchesRedactAttr(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	const plain = "hunter2" // not secret-shaped; only the key triggers redaction
	keys := []string{"password", "api_key", "token", "access_token", "secret"}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			got := r.RedactStructured(map[string]any{key: plain})
			m, ok := got.(map[string]any)
			if !ok {
				t.Fatalf("expected map[string]any, got %T", got)
			}
			if m[key] != redacted {
				t.Errorf("RedactStructured key %q = %v, want %q", key, m[key], redacted)
			}

			// Parity: RedactAttr must agree for the same key/value.
			attr := r.RedactAttr(slog.String(key, plain))
			if attr.Value.String() != redacted {
				t.Errorf("RedactAttr key %q = %q, want %q (parity with RedactStructured)",
					key, attr.Value.String(), redacted)
			}
		})
	}
}

// TestRedactStructured_CopyOnWrite proves a secret-free payload is returned
// untouched (same underlying value, no deep copy) and that scalar leaves are
// preserved exactly when a sibling does change.
func TestRedactStructured_CopyOnWrite(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	t.Run("no secrets returns same value", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{
			"name":  "alice",
			"count": 42,
			"ok":    true,
			"ratio": 3.14,
			"tags":  []any{"a", "b"},
			"inner": map[string]any{"k": "v"},
		}
		got := r.RedactStructured(input)
		if !reflect.DeepEqual(got, input) {
			t.Fatalf("expected deep-equal output, got %#v", got)
		}
		// Copy-on-write: the identical map header must be returned, not a copy.
		if reflect.ValueOf(got).Pointer() != reflect.ValueOf(input).Pointer() {
			t.Error("expected the same map to be returned for a secret-free payload (no deep copy)")
		}
	})

	t.Run("scalar leaves untouched when a sibling changes", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{
			"k":     awsKey,
			"count": 42,
			"ok":    true,
			"ratio": 3.14,
		}
		out, ok := r.RedactStructured(input).(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any result")
		}
		if out["k"] != redacted {
			t.Errorf("secret value = %v, want %q", out["k"], redacted)
		}
		if out["count"] != 42 {
			t.Errorf("int leaf changed: got %#v (%T), want int 42", out["count"], out["count"])
		}
		if out["ok"] != true {
			t.Errorf("bool leaf changed: got %#v", out["ok"])
		}
		if out["ratio"] != 3.14 {
			t.Errorf("float leaf changed: got %#v (%T), want float64 3.14", out["ratio"], out["ratio"])
		}
		// The original payload must not be mutated (copy-on-write).
		if input["k"] != awsKey {
			t.Errorf("input was mutated: input[\"k\"] = %v", input["k"])
		}
	})
}

// TestRedactStructured_Nil verifies a nil input returns nil.
func TestRedactStructured_Nil(t *testing.T) {
	t.Parallel()
	r := NewRedactor()
	if got := r.RedactStructured(nil); got != nil {
		t.Errorf("RedactStructured(nil) = %v, want nil", got)
	}
}

// TestRedactStructured_PointerToStruct verifies a *struct input is redacted and
// returns a (redacted) pointer of the same type, while a secret-free pointer is
// returned unchanged (copy-on-write).
func TestRedactStructured_PointerToStruct(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	t.Run("redacts through pointer", func(t *testing.T) {
		t.Parallel()
		in := &row{Name: "svc", Note: "embedded " + ghToken}
		got := r.RedactStructured(in)
		p, ok := got.(*row)
		if !ok {
			t.Fatalf("expected *row, got %T", got)
		}
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if strings.Contains(p.Note, ghToken) {
			t.Errorf("token not redacted: %q", p.Note)
		}
		if !strings.Contains(p.Note, redacted) {
			t.Errorf("expected redaction marker in Note: %q", p.Note)
		}
		if p.Name != "svc" {
			t.Errorf("Name should be preserved, got %q", p.Name)
		}
		// Original must not be mutated.
		if !strings.Contains(in.Note, ghToken) {
			t.Error("input struct was mutated (copy-on-write violated)")
		}
	})

	t.Run("secret-free pointer returned unchanged", func(t *testing.T) {
		t.Parallel()
		clean := &row{Name: "svc", Note: "all good"}
		got := r.RedactStructured(clean)
		if got.(*row) != clean {
			t.Error("expected the same pointer for a secret-free struct (copy-on-write)")
		}
	})
}

// TestRedactStructured_UnexportedFieldStruct exercises the JSON-normalization
// path (a struct with an unexported field) nested inside a concretely-typed
// slice. This is a regression guard: returning a generic map[string]any here
// would panic when assigned back into the []auditEntry slot.
func TestRedactStructured_UnexportedFieldStruct(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	in := []auditEntry{{Detail: "tok " + ghToken, seq: 1}}
	got := r.RedactStructured(in)

	out, ok := got.([]auditEntry)
	if !ok {
		t.Fatalf("expected []auditEntry (original type restored), got %T", got)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 element, got %d", len(out))
	}
	if strings.Contains(out[0].Detail, ghToken) {
		t.Errorf("token not redacted: %q", out[0].Detail)
	}
	if !strings.Contains(out[0].Detail, redacted) {
		t.Errorf("expected redaction marker in Detail: %q", out[0].Detail)
	}
	// Original must not be mutated.
	if !strings.Contains(in[0].Detail, ghToken) {
		t.Error("input slice was mutated (copy-on-write violated)")
	}
}

// TestRedactStructured_DenyKeyNonStringElem covers a denied key in a concretely
// typed map whose element cannot hold the "[REDACTED]" string marker
// (map[string][]byte). The value must be dropped to its zero, not left intact by
// an incomplete value-pattern scrub.
func TestRedactStructured_DenyKeyNonStringElem(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	in := map[string][]byte{
		"password": []byte("binary-secret"),
		"name":     []byte("alice"),
	}
	out, ok := r.RedactStructured(in).(map[string][]byte)
	if !ok {
		t.Fatalf("expected map[string][]byte, got %T", r.RedactStructured(in))
	}
	if out["password"] != nil {
		t.Errorf("denied-key []byte value not dropped: %q", out["password"])
	}
	if string(out["name"]) != "alice" {
		t.Errorf("non-secret value altered: %q", out["name"])
	}
	// Copy-on-write: the original map must not be mutated.
	if string(in["password"]) != "binary-secret" {
		t.Errorf("input map mutated: %q", in["password"])
	}
}

// TestRedactStructured_FailsClosedOnUnreconstructableType guards the coerce
// fail-closed path: when a redacted, unexported-field struct nested in a concrete
// slice cannot be reconstructed (its UnmarshalJSON rejects the marker), the slot
// is zeroed rather than panicking on an unassignable map[string]any, and the
// secret never survives.
func TestRedactStructured_FailsClosedOnUnreconstructableType(t *testing.T) {
	t.Parallel()
	r := NewRedactor()

	in := []rejectingDetail{{Detail: "tok " + ghToken, seq: 1}}

	var got any
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("RedactStructured panicked instead of failing closed: %v", p)
			}
		}()
		got = r.RedactStructured(in)
	}()

	out, ok := got.([]rejectingDetail)
	if !ok {
		t.Fatalf("expected []rejectingDetail, got %T", got)
	}
	if strings.Contains(out[0].Detail, ghToken) {
		t.Errorf("secret survived the fail-closed path: %q", out[0].Detail)
	}
	if out[0].Detail != "" {
		t.Errorf("expected slot zeroed on unreconstructable type, got %q", out[0].Detail)
	}
}
