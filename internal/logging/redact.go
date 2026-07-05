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

// nameValueTrimCutset is the whitespace RE2's \s matches; redactNamedValues trims
// it from the tail of a captured value so the separator run before the next
// key-like token stays verbatim in the output rather than being redacted.
const nameValueTrimCutset = "\t\n\f\r "

// redactedReflectVal is the redaction marker as a reflect.Value, computed once
// so the per-map-node redaction walk does not re-box the string on every call.
var redactedReflectVal = reflect.ValueOf(redacted)

// Redactor scrubs secret values from log attributes.
type Redactor struct {
	valuePatterns []*regexp.Regexp
	urlCredRe     *regexp.Regexp
	nameValRe     *regexp.Regexp
	keyBoundaryRe *regexp.Regexp
}

// NewRedactor creates a Redactor with default secret patterns.
func NewRedactor() *Redactor {
	return &Redactor{
		valuePatterns: compileValuePatterns(),
		urlCredRe:     regexp.MustCompile(`://[^:@\s]+:[^:@\s]+@`),
		// Matches "NAME=value" and "NAME: value" pairs so a sensitive credential
		// NAME (e.g. DATABASE_PASSWORD) redacts its value even when the value
		// itself matches no credential-shape pattern. Group 2 anchors the value's
		// START only (its first whitespace-delimited token); the value's true END
		// — which extends across internal spaces to the next key or end-of-line —
		// is computed in redactNamedValues because RE2 (Go's regexp) has no
		// lookahead to stop the capture at the next key boundary.
		nameValRe: regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*[:=]\s*(\S+)`),
		// Marks the next "NAME=" / "NAME:" key boundary that terminates a value:
		// a NAME (optionally spaced from its separator) that is preceded by
		// whitespace. The leading \s requirement means an intra-value token such
		// as a=b (no preceding space) stays part of the value, while a genuine
		// following pair on the same line ends it.
		keyBoundaryRe: regexp.MustCompile(`\s[A-Za-z_][A-Za-z0-9_]*\s*[:=]`),
	}
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

// RedactString scrubs secret patterns from a string value. It runs the
// value-shape passes (AKIA…, ghp_…, JWT, PEM, …) and URL-userinfo stripping,
// then a NAME=value pass that redacts the value of any sensitive credential
// variable — closing the leak where a keyword-less secret (DATABASE_PASSWORD=…,
// BW_SESSION=…) has no recognizable value shape.
func (r *Redactor) RedactString(s string) string {
	for _, p := range r.valuePatterns {
		s = p.ReplaceAllString(s, redacted)
	}
	if r.urlCredRe.MatchString(s) {
		s = r.redactURLCredentials(s)
	}
	return r.redactNamedValues(s)
}

// redactNamedValues redacts the VALUE of any "NAME=value" or "NAME: value" pair
// whose NAME is a sensitive credential variable (per secrets.IsSensitiveName).
// The NAME and separator are preserved; only the value is replaced. It is
// conservative — a non-sensitive name such as PATH=/usr/bin or KEYBOARD=us is
// left untouched. This runs on every log message and MCP tool result, so it
// does a single regex pass over the submatch indexes rather than re-matching
// each hit.
func (r *Redactor) redactNamedValues(s string) string {
	if !strings.ContainsAny(s, "=:") {
		return s
	}
	matches := r.nameValRe.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	last := 0
	for _, m := range matches {
		// m holds pair offsets: match (m[0:2]), group 1 NAME (m[2:4]),
		// group 2 value-start token (m[4:6]).
		if m[0] < last {
			// This pair begins inside a value already redacted for an earlier
			// sensitive NAME (e.g. an "a=b" token nested in a redacted value);
			// skip it so we neither double-write nor leak part of that value.
			continue
		}
		if !secrets.IsSensitiveName(s[m[2]:m[3]]) {
			continue
		}
		valStart := m[4]
		valEnd := r.valueEnd(s, valStart)
		if valEnd <= valStart {
			continue
		}
		// Keep everything through "NAME<sep>" and redact the whole value —
		// internal spaces included — up to the computed end.
		b.WriteString(s[last:valStart])
		b.WriteString(redacted)
		last = valEnd
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// valueEnd returns the offset at which a sensitive NAME's value ends. The value
// runs from valStart to end-of-line, EXCEPT it stops before the next
// whitespace-preceded "NAME=" / "NAME:" key so a following pair on the same line
// is redacted independently rather than swallowed. Trailing whitespace before
// that boundary is excluded so the separator run is preserved verbatim.
func (r *Redactor) valueEnd(s string, valStart int) int {
	end := len(s)
	// A value never spans a newline (RE2's \S, like the value token, excludes it).
	if nl := strings.IndexByte(s[valStart:], '\n'); nl >= 0 {
		end = valStart + nl
	}
	// Stop before the next key-like token on the same line.
	if loc := r.keyBoundaryRe.FindStringIndex(s[valStart:end]); loc != nil {
		end = valStart + loc[0]
	}
	trimmed := strings.TrimRight(s[valStart:end], nameValueTrimCutset)
	return valStart + len(trimmed)
}

func (r *Redactor) redactURLCredentials(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.User == nil {
		return r.urlCredRe.ReplaceAllString(s, "://"+redacted+":"+redacted+"@")
	}
	u.User = url.UserPassword(redacted, redacted)
	return u.String()
}

// isKeyDenied checks whether an attribute key indicates a secret value. It
// delegates to secrets.IsSensitiveName — the single shared credential-name
// predicate — so the exact canon (KnownCredentialVars) and the token-boundary
// keyword match ("token" matches, "tokenizer" does not) stay unified across the
// slog attr path, the NAME=value message path, and the env probe.
func (r *Redactor) isKeyDenied(key string) bool {
	return secrets.IsSensitiveName(key)
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
			ensureCopy()
			if rep, ok := assignableRedaction(redactedReflectVal, elemType); ok {
				out.SetMapIndex(k, rep)
			} else {
				// The element type cannot hold the "[REDACTED]" marker (e.g.
				// map[string][]byte or map[string]<struct>); a denied key is still
				// secret, so drop the value to its zero rather than leaking it
				// through an incomplete value-pattern scrub.
				out.SetMapIndex(k, reflect.Zero(elemType))
			}
			continue
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
// copy-on-write. For each field it first applies the key deny-list to the
// field's effective name (its json tag, else the Go field name) — exactly as
// redactMap does for a map key — so a plainly-named secret (e.g. a Token field
// holding a value the pattern scrub would miss) cannot survive behind a concrete
// struct type. It then redacts the field value itself. Structs with unexported
// fields are routed through redactViaJSON before reaching here (see redactReflect).
func (r *Redactor) redactStruct(rv reflect.Value) (reflect.Value, bool) {
	t := rv.Type()
	var out reflect.Value
	changed := false
	ensureCopy := func() {
		if changed {
			return
		}
		out = reflect.New(t).Elem()
		for j := 0; j < rv.NumField(); j++ {
			if t.Field(j).PkgPath == "" {
				out.Field(j).Set(rv.Field(j))
			}
		}
		changed = true
	}

	for i := 0; i < rv.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}

		if r.isKeyDenied(structFieldKey(field)) {
			if rep, ok := assignableRedaction(redactedReflectVal, field.Type); ok {
				ensureCopy()
				out.Field(i).Set(rep)
				continue
			}
			// Field type cannot hold the redaction marker; fall through to a
			// value-pattern scrub of the (typed) value instead.
		}

		nv, ch := r.redactReflect(rv.Field(i))
		if !ch {
			continue
		}
		ensureCopy()
		out.Field(i).Set(coerce(nv, field.Type))
	}
	if !changed {
		return rv, false
	}
	return out, true
}

// structFieldKey returns the name used to match a struct field against the secret
// key deny-list: the json tag's name (its first comma-segment) when present and
// usable, otherwise the Go field name. A "-" tag (not serialized) or an empty
// name both fall back to the field name so an in-memory secret is still caught.
func structFieldKey(f reflect.StructField) string {
	tag, ok := f.Tag.Lookup("json")
	if !ok {
		return f.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return f.Name
	}
	return name
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
	// v cannot be represented as dst — e.g. a redacted map[string]any tree whose
	// concrete destination slot rejected reconstruction (redactViaJSON's
	// best-effort reencode failed). Fail closed to the zero value rather than
	// returning an unassignable value that would panic at Set: the secret is
	// dropped, never leaked.
	return reflect.Zero(dst)
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
