package logging

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

const redacted = "[REDACTED]"

// redactedReflectVal is the redaction marker as a reflect.Value, computed once
// so the per-map-node redaction walk does not re-box the string on every call.
var redactedReflectVal = reflect.ValueOf(redacted)

// Redactor scrubs secret values from log attributes.
type Redactor struct {
	valuePatterns []*regexp.Regexp
	keyDeny       map[string]bool
	envNames      map[string]bool
	urlCredRe     *regexp.Regexp
}

// NewRedactor creates a Redactor with default secret patterns.
func NewRedactor() *Redactor {
	r := &Redactor{
		keyDeny:  make(map[string]bool, len(secrets.SensitiveKeyPatterns)),
		envNames: make(map[string]bool, len(secrets.KnownCredentialVars)),
	}

	for _, p := range secrets.SensitiveKeyPatterns {
		r.keyDeny[strings.ToLower(p)] = true
	}
	for _, v := range secrets.KnownCredentialVars {
		r.envNames[v] = true
	}

	r.valuePatterns = compileValuePatterns()
	r.urlCredRe = regexp.MustCompile(`://[^:@\s]+:[^:@\s]+@`)

	return r
}

func compileValuePatterns() []*regexp.Regexp {
	patterns := []string{
		`AKIA[A-Z0-9]{16}`,
		`gh[pousr]_[A-Za-z0-9_]{36,}`,
		`github_pat_[A-Za-z0-9_]{22,}`,
		`glpat-[A-Za-z0-9\-_]{20,}`,
		`sk_(live|test)_[A-Za-z0-9]{24,}`,
		`npm_[A-Za-z0-9]{36,}`,
		`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+`,
		`-----BEGIN\s+(?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
		`AccountKey=[A-Za-z0-9+/=]{44,}`,
		`AIza[A-Za-z0-9_-]{35}`,
		`mongodb(?:\+srv)?://[^:]+:[^@\s]+@[^\s]+`,
		`hv[sbr]\.[A-Za-z0-9_-]{24,}`,
		`xox[bpras]-[A-Za-z0-9-]{10,}`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return compiled
}

// RedactAttr scrubs secret values from a single slog.Attr.
func (r *Redactor) RedactAttr(a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		scrubbed := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			scrubbed[i] = r.RedactAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(scrubbed...)}
	}

	if r.isKeyDenied(a.Key) {
		return slog.String(a.Key, redacted)
	}

	if a.Value.Kind() == slog.KindString {
		scrubbed := r.RedactString(a.Value.String())
		if scrubbed != a.Value.String() {
			return slog.String(a.Key, scrubbed)
		}
	}

	return a
}

// RedactString scrubs secret patterns from a string value.
func (r *Redactor) RedactString(s string) string {
	for _, p := range r.valuePatterns {
		s = p.ReplaceAllString(s, redacted)
	}
	if r.urlCredRe.MatchString(s) {
		s = r.redactURLCredentials(s)
	}
	return s
}

func (r *Redactor) redactURLCredentials(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.User == nil {
		return r.urlCredRe.ReplaceAllString(s, "://"+redacted+":"+redacted+"@")
	}
	u.User = url.UserPassword(redacted, redacted)
	return u.String()
}

// isKeyDenied checks whether an attribute key indicates a secret value.
// Uses word-boundary matching: "token" matches but "tokenizer" does not.
func (r *Redactor) isKeyDenied(key string) bool {
	lower := strings.ToLower(key)

	if r.envNames[key] || r.envNames[strings.ToUpper(key)] {
		return true
	}

	// Normalize hyphens to underscores so "api-key" matches "api_key".
	normalized := strings.ReplaceAll(lower, "-", "_")

	for pattern := range r.keyDeny {
		if matchesWordBoundary(lower, pattern) || matchesWordBoundary(normalized, pattern) {
			return true
		}
	}
	return false
}

// matchesWordBoundary checks if pattern appears in s at a word boundary.
// A word boundary is: start/end of string, underscore, hyphen, or transition
// between non-letter and letter.
func matchesWordBoundary(s, pattern string) bool {
	idx := strings.Index(s, pattern)
	if idx < 0 {
		return false
	}

	if idx > 0 {
		prev := s[idx-1]
		if isWordChar(prev) && prev != '_' && prev != '-' {
			return false
		}
	}

	end := idx + len(pattern)
	if end < len(s) {
		next := s[end]
		if isWordChar(next) && next != '_' && next != '-' {
			return false
		}
	}

	return true
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// RedactStructured returns a redacted copy of a JSON-serializable value tree,
// applying BOTH the value-pattern scrub (RedactString) to every nested string
// and the key deny-list (the same one RedactAttr enforces on the log surface) to
// every string-keyed map. Unlike a type-switch walk, it descends through ANY
// concrete container — map[string]string, []string, []map[string]any, slices of
// structs — via reflection, so a payload cannot hide secrets behind a concrete
// type the walker did not anticipate.
//
// It is copy-on-write: a container is reallocated only when one of its children
// actually changes, so a payload with no secrets is returned untouched (the same
// value, not a deep copy). The returned value JSON-marshals identically to the
// input apart from the redactions.
func (r *Redactor) RedactStructured(v any) any {
	if v == nil {
		return nil
	}
	nv, changed := r.redactReflect(reflect.ValueOf(v))
	if !changed || !nv.IsValid() {
		return v
	}
	return nv.Interface()
}

// redactReflect is the reflection core of RedactStructured. It returns the
// (possibly new) value and whether anything changed; an unchanged subtree is
// reported with changed=false so callers can skip allocating a copy.
func (r *Redactor) redactReflect(rv reflect.Value) (reflect.Value, bool) {
	if !rv.IsValid() {
		return rv, false
	}
	switch rv.Kind() {
	case reflect.Interface:
		if rv.IsNil() {
			return rv, false
		}
		// Unwrap to the concrete value; the caller assigns back into the
		// interface-typed slot, so returning the concrete value is correct.
		return r.redactReflect(rv.Elem())
	case reflect.Pointer:
		if rv.IsNil() {
			return rv, false
		}
		nv, changed := r.redactReflect(rv.Elem())
		if !changed {
			return rv, false
		}
		p := reflect.New(rv.Elem().Type())
		p.Elem().Set(nv)
		return p, true
	case reflect.String:
		s := rv.String()
		rs := r.RedactString(s)
		if rs == s {
			return rv, false
		}
		out := reflect.ValueOf(rs)
		if out.Type() != rv.Type() {
			out = out.Convert(rv.Type())
		}
		return out, true
	case reflect.Map:
		return r.redactMap(rv)
	case reflect.Slice, reflect.Array:
		return r.redactSeq(rv)
	case reflect.Struct:
		// A struct carrying unexported state (e.g. time.Time) cannot be safely
		// field-copied via reflection and may rely on a custom MarshalJSON, so
		// normalize it through JSON to a generic tree before walking.
		if hasUnexportedField(rv.Type()) {
			return r.redactViaJSON(rv)
		}
		return r.redactStruct(rv)
	default:
		return rv, false
	}
}

// redactMap walks a map value. For string-keyed maps it first applies the key
// deny-list (a sensitive key name redacts the whole value, matching RedactAttr),
// then redacts the value itself. Copy-on-write: the map is rebuilt only once a
// child changes, backfilling the already-seen entries.
func (r *Redactor) redactMap(rv reflect.Value) (reflect.Value, bool) {
	if rv.IsNil() {
		return rv, false
	}
	stringKey := rv.Type().Key().Kind() == reflect.String
	elemType := rv.Type().Elem()

	var out reflect.Value
	changed := false
	ensureCopy := func() {
		if changed {
			return
		}
		out = reflect.MakeMapWithSize(rv.Type(), rv.Len())
		it := rv.MapRange()
		for it.Next() {
			out.SetMapIndex(it.Key(), it.Value())
		}
		changed = true
	}

	iter := rv.MapRange()
	for iter.Next() {
		k := iter.Key()
		val := iter.Value()

		if stringKey && r.isKeyDenied(k.String()) {
			if rep, ok := assignableRedaction(redactedReflectVal, elemType); ok {
				ensureCopy()
				out.SetMapIndex(k, rep)
				continue
			}
			// Elem type cannot hold the redaction marker; fall through to a
			// value-pattern scrub of the (typed) value instead.
		}

		nv, ch := r.redactReflect(val)
		if !ch {
			continue
		}
		ensureCopy()
		out.SetMapIndex(k, coerce(nv, elemType))
	}
	if !changed {
		return rv, false
	}
	return out, true
}

// redactSeq walks a slice or array, redacting each element copy-on-write.
func (r *Redactor) redactSeq(rv reflect.Value) (reflect.Value, bool) {
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return rv, false
	}
	n := rv.Len()
	var out reflect.Value
	changed := false
	for i := 0; i < n; i++ {
		nv, ch := r.redactReflect(rv.Index(i))
		if !ch {
			continue
		}
		if !changed {
			if rv.Kind() == reflect.Slice {
				out = reflect.MakeSlice(rv.Type(), n, n)
			} else {
				out = reflect.New(rv.Type()).Elem()
			}
			reflect.Copy(out, rv)
			changed = true
		}
		out.Index(i).Set(coerce(nv, rv.Type().Elem()))
	}
	if !changed {
		return rv, false
	}
	return out, true
}

// redactStruct walks the exported fields of an all-exported struct, redacting
// copy-on-write. Structs with unexported fields are routed through redactViaJSON
// before reaching here (see redactReflect).
func (r *Redactor) redactStruct(rv reflect.Value) (reflect.Value, bool) {
	t := rv.Type()
	var out reflect.Value
	changed := false
	for i := 0; i < rv.NumField(); i++ {
		if t.Field(i).PkgPath != "" { // unexported
			continue
		}
		nv, ch := r.redactReflect(rv.Field(i))
		if !ch {
			continue
		}
		if !changed {
			out = reflect.New(t).Elem()
			for j := 0; j < rv.NumField(); j++ {
				if t.Field(j).PkgPath == "" {
					out.Field(j).Set(rv.Field(j))
				}
			}
			changed = true
		}
		out.Field(i).Set(coerce(nv, out.Field(i).Type()))
	}
	if !changed {
		return rv, false
	}
	return out, true
}

// redactViaJSON normalizes a value (typically a struct with unexported state)
// to a generic JSON tree, walks it, and returns the redacted value. It preserves
// the wire representation because the redacted tree marshals to the same JSON as
// the input, minus the secrets.
//
// When the redacted tree can be re-decoded into the original type, the value is
// reconstructed so it stays assignable to its slot — a struct-with-unexported
// field nested inside a concretely-typed container (e.g. []T, map[K]T, or a
// field of another struct) would otherwise hand the caller an unassignable
// map[string]any and panic at the Set. The original type is restored only on a
// best-effort basis; if reconstruction fails the generic tree is returned, which
// is still assignable wherever the destination is an interface (the common case).
func (r *Redactor) redactViaJSON(rv reflect.Value) (reflect.Value, bool) {
	if !rv.CanInterface() {
		return rv, false
	}
	raw, err := json.Marshal(rv.Interface())
	if err != nil {
		return rv, false
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return rv, false
	}
	nv, changed := r.redactReflect(reflect.ValueOf(generic))
	if !changed || !nv.IsValid() {
		return rv, false
	}
	if restored, ok := reencodeAs(nv, rv.Type()); ok {
		return restored, true
	}
	return nv, true
}

// reencodeAs marshals the redacted generic tree and decodes it back into dst's
// type, restoring the original concrete type so the result stays assignable to
// its slot. It reports false when the round-trip cannot reproduce dst's type
// (e.g. a custom UnmarshalJSON that rejects the redaction marker).
func reencodeAs(v reflect.Value, dst reflect.Type) (reflect.Value, bool) {
	if !v.CanInterface() {
		return reflect.Value{}, false
	}
	raw, err := json.Marshal(v.Interface())
	if err != nil {
		return reflect.Value{}, false
	}
	ptr := reflect.New(dst)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return reflect.Value{}, false
	}
	return ptr.Elem(), true
}

// coerce makes v assignable to dst, converting compatible types (e.g. a plain
// string redaction into a named string field) and falling back to v unchanged.
func coerce(v reflect.Value, dst reflect.Type) reflect.Value {
	if !v.IsValid() {
		return reflect.Zero(dst)
	}
	if v.Type().AssignableTo(dst) {
		return v
	}
	if v.Type().ConvertibleTo(dst) {
		return v.Convert(dst)
	}
	return v
}

// assignableRedaction returns the "[REDACTED]" marker coerced to dst when dst can
// hold a string (directly, as a named string type, or as an interface), so a
// deny-listed key in a concretely-typed map still redacts cleanly. It reports
// false when dst cannot represent the marker (e.g. map[string]int).
func assignableRedaction(marker reflect.Value, dst reflect.Type) (reflect.Value, bool) {
	switch {
	case marker.Type().AssignableTo(dst):
		return marker, true
	case dst.Kind() == reflect.String:
		return marker.Convert(dst), true
	case dst.Kind() == reflect.Interface && marker.Type().Implements(dst):
		return marker, true
	default:
		return reflect.Value{}, false
	}
}

// hasUnexportedField reports whether a struct type has any unexported field.
func hasUnexportedField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).PkgPath != "" {
			return true
		}
	}
	return false
}
