package logging

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// RedactionMarker is the text that replaces every redacted secret.
const RedactionMarker = "[REDACTED]"

const redacted = RedactionMarker

// nameValueTrimCutset is the whitespace RE2's \s matches; redactNamedValues trims
// it from the tail of a captured value so the separator run before the next
// key-like token stays verbatim in the output rather than being redacted.
const nameValueTrimCutset = "\t\n\f\r "

// redactedReflectVal is the redaction marker as a reflect.Value, computed once
// so the per-map-node redaction walk does not re-box the string on every call.
var redactedReflectVal = reflect.ValueOf(redacted)

// privateKeyBlockRe extends the canon private-key header
// (secrets.PrivateKeyHeaderPattern) through the key body to its END line, so
// the key material is redacted, not just the BEGIN line. The canon owns the header
// shape; extending it to END is this package's redaction mechanic. (?s) lets the
// body span real newlines, and the lazy body equally spans the literal "\n"
// escapes of a JSON-encoded key. A block with no END marker (a truncated
// excerpt) is redacted through the end of the input: fail closed.
var privateKeyBlockRe = regexp.MustCompile(`(?s)` + secrets.PrivateKeyHeaderPattern +
	`.*?(?:` + privateKeyFooterPattern + `|\z)`)

// privateKeyFooterPattern is the END line matching the canon header's shape;
// the block extension and the excerpt END scan share this one copy.
const privateKeyFooterPattern = `-----END [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?-----`

var (
	privateKeyBeginRe = regexp.MustCompile(secrets.PrivateKeyHeaderPattern)
	privateKeyEndRe   = regexp.MustCompile(privateKeyFooterPattern)
)

// Redactor scrubs secret values from log attributes.
type Redactor struct {
	// redactCanonMatch is the redactValueMatch method value, bound once so the
	// per-pattern replace pass does not allocate a closure on every call.
	redactCanonMatch func(string) string
	urlCredRe        *regexp.Regexp
	urlTokenRe       *regexp.Regexp
	nameValRe        *regexp.Regexp
	keyBoundaryRe    *regexp.Regexp
	flagPairRe       *regexp.Regexp
}

// NewRedactor creates a Redactor with default secret patterns.
func NewRedactor() *Redactor {
	r := &Redactor{
		// user:password userinfo. The user may be empty (redis://:pw@host, the
		// form Redis and some DSNs use for a password-only credential) and the
		// password may contain ':', '@', and — as unencoded base64 or generated
		// passwords often do in logged DSNs — '/', '?' or '#'. What separates
		// userinfo from host:port followed by a path holding an '@'
		// ("http://h:8080/a:b@c", "http://localhost:4873/@scope/pkg") is the
		// port rule: a part after ':' that is all digits up to a '/', '?' or
		// '#' is a port, so only a password with a non-digit before its first
		// such byte may continue past it. The user part may not open an IPv6
		// literal ('[' / ']'), so "http://[::1]:4873/@types%2fnode" is a host.
		urlCredRe: regexp.MustCompile(`://[^:@\s/?#\[\]]*:(?:[^@\s/?#]+|[0-9]*[^0-9@\s/?#][^@\s]*)@`),
		// Token-only userinfo (https://TOKEN@host/...), the form git hosts use
		// for PATs and CI job tokens. Requiring a scheme:// keeps the scp-style
		// git@host:path form (which has no scheme) untouched.
		urlTokenRe: regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^/@\s:]+@`),
		// Matches "NAME=value" and "NAME: value" pairs so a sensitive credential
		// NAME (e.g. DATABASE_PASSWORD) redacts its value even when the value
		// itself matches no credential-shape pattern. The NAME may be wrapped in
		// plain or backslash-escaped quotes, so JSON ("NAME":"value"), JSON
		// embedded in a JSON string (\"NAME\":\"value\") and single-quoted
		// dict reprs ('NAME': 'value', as Python prints headers) are covered.
		// Groups: 1 opening quote, 2 NAME, 3 closing quote, 4 separator. The
		// match ends where the value starts and consumes none of it, because
		// the value may itself begin a pair ("error: token=abc"); the value's
		// end is computed in redactNamedValues (RE2 has no lookahead).
		nameValRe: regexp.MustCompile(`(\\?["'])?([A-Za-z_][A-Za-z0-9_]*)(\\?["'])?\s*([:=])\s*`),
		// Marks the next "NAME=" / "NAME:" key boundary that terminates a value:
		// a NAME (optionally spaced from its separator) that is preceded by
		// whitespace. The leading \s requirement means an intra-value token such
		// as a=b (no preceding space) stays part of the value, while a genuine
		// following pair on the same line ends it.
		keyBoundaryRe: regexp.MustCompile(`\s[A-Za-z_][A-Za-z0-9_]*\s*[:=]`),
		// A whitespace- or start-anchored command-line flag followed by its
		// separate value ("--password hunter2"). Groups: 1 anchor, 2 flag,
		// 3 spacing, 4 value. The value may not start with '-', so a following
		// flag ("--api-key --verbose") is never taken for a value.
		flagPairRe: regexp.MustCompile(`(^|\s)(-{1,2}[A-Za-z][A-Za-z0-9_-]*)(\s+)([^\s-]\S*)`),
	}
	r.redactCanonMatch = r.redactValueMatch
	return r
}

// redactValueMatch is the replacement for one secrets canon match. A match that
// carries URL userinfo (the canon's database connection-string shape) has only
// its credentials redacted, the way the URL pass treats every scheme, so the
// host stays visible for diagnosability; any other match is redacted whole.
func (r *Redactor) redactValueMatch(m string) string {
	if r.urlCredRe.MatchString(m) {
		return r.redactURLCredentials(m)
	}
	return redacted
}

// RedactAttr scrubs secret values from a single slog.Attr. LogValuers are
// resolved first so a lazily-computed value is scrubbed too. Besides strings,
// it scrubs KindAny values — errors (the ubiquitous "error", err attribute),
// argv slices, maps and structs — which a JSON or text handler would otherwise
// serialize verbatim. A sensitive key is checked before group recursion, so a
// group named "password" or "credentials" is redacted whole rather than having
// its benignly-named members passed through (see denyAttr).
func (r *Redactor) RedactAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	if r.isKeyDenied(a.Key) {
		return denyAttr(a)
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		scrubbed := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			scrubbed[i] = r.RedactAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(scrubbed...)}
	}

	switch a.Value.Kind() {
	case slog.KindString:
		scrubbed := r.RedactString(a.Value.String())
		if scrubbed != a.Value.String() {
			return slog.String(a.Key, scrubbed)
		}
	case slog.KindAny:
		return slog.Attr{Key: a.Key, Value: r.redactAnyValue(a.Value.Any())}
	}

	return a
}

// redactAnyValue scrubs a KindAny attribute value. An error is flattened to its
// redacted message (which is how the handlers render it anyway); a Stringer
// whose rendering carries a secret (e.g. a *url.URL with userinfo, printed via
// String() by the text handler) is replaced by its redacted rendering; any
// other value is walked by RedactStructured so nested map keys, struct fields
// and argv slices are scrubbed.
func (r *Redactor) redactAnyValue(v any) slog.Value {
	if isNilPointer(v) {
		return slog.AnyValue(v)
	}
	if err, ok := v.(error); ok {
		return slog.StringValue(r.RedactString(err.Error()))
	}
	if st, ok := v.(fmt.Stringer); ok {
		str := st.String()
		if red := r.RedactString(str); red != str {
			return slog.StringValue(red)
		}
	}
	return slog.AnyValue(r.RedactStructured(v))
}

// isNilPointer reports whether v is a typed nil pointer, whose Error/String
// methods may panic on a nil receiver.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// RedactString scrubs secret patterns from a string value. It runs the
// value-shape passes (whole PEM blocks, then the secrets.ValuePatterns canon)
// and URL-userinfo stripping (including an empty user, redis://:pw@host), then a NAME=value pass
// that redacts the value of any sensitive credential variable — closing the
// leak where a keyword-less secret (DATABASE_PASSWORD=…, BW_SESSION=…,
// {"api_token":"…"}) has no recognizable value shape — and finally a flag pass
// that redacts the separate value of a sensitive flag ("--password hunter2").
//
// In the NAME pass an unquoted sensitive NAME followed by ':' (a header, YAML
// or "error:"-prefix form such as "Authorization: Basic dXNl…==") redacts the
// rest of the line, so scheme words, base64 padding and Digest sub-pairs are
// all covered. A NAME followed by '=' redacts up to the next "NAME=" pair on
// the line, where an '=' that is base64 padding does not count as a pair.
func (r *Redactor) RedactString(s string) string {
	// Whole key blocks go first, so the canon's header-only private-key entry
	// never consumes a header and leaves the key body behind. Each replace is
	// guarded by a non-allocating MatchString: a regexp replace copies its input
	// even when nothing matches, and most log lines match no canon pattern.
	if privateKeyBlockRe.MatchString(s) {
		s = privateKeyBlockRe.ReplaceAllString(s, redacted)
	}
	for _, p := range secrets.CompiledValuePatterns() {
		if p.MatchString(s) {
			s = p.ReplaceAllStringFunc(s, r.redactCanonMatch)
		}
	}
	if r.urlCredRe.MatchString(s) {
		s = r.redactURLCredentials(s)
	}
	s = r.urlTokenRe.ReplaceAllString(s, "${1}"+redacted+"@")
	s = r.redactNamedValues(s)
	return r.redactFlagPairs(s)
}

// redactFlagPairs redacts the value that follows a sensitive command-line flag
// written with a separate value in free text ("run --password hunter2 now").
// The flag decision is isSensitiveFlag, the same predicate the argv walk in
// redactSeq uses, so a benign flag ("-p 8080:80") is left untouched. A quoted
// value ('--password "hunter 2"') is redacted through its closing quote.
func (r *Redactor) redactFlagPairs(s string) string {
	if strings.IndexByte(s, '-') < 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range r.flagPairRe.FindAllStringSubmatchIndex(s, -1) {
		// m[4:6] is the flag, m[8:10] its value.
		if m[0] < last || !isSensitiveFlag(s[m[4]:m[5]]) {
			continue
		}
		if last == 0 {
			b.Grow(len(s))
		}
		b.WriteString(s[last:m[8]])
		b.WriteString(redacted)
		last = quotedValueEnd(s, m[8], m[9])
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// quotedValueEnd returns the end of a flag value token s[from:to]. When the
// token opens with a quote, the value runs through the matching closing quote
// on the same line, or to the end of the line when it is unterminated (fail
// closed), so a multi-word quoted secret is redacted whole.
func quotedValueEnd(s string, from, to int) int {
	q := s[from]
	if q != '"' && q != '\'' {
		return to
	}
	end := lineEnd(s, from)
	if i := strings.IndexByte(s[from+1:end], q); i >= 0 {
		return from + 1 + i + 1
	}
	return end
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
		// m holds pair offsets: match (m[0:2]), group 1 opening quote (m[2:4]),
		// group 2 NAME (m[4:6]), group 3 closing quote (m[6:8]), group 4
		// separator (m[8:10]); the value starts where the match ends.
		if m[0] < last {
			// This pair begins inside a value already redacted for an earlier
			// sensitive NAME (e.g. an "a=b" token nested in a redacted value);
			// skip it so we neither double-write nor leak part of that value.
			continue
		}
		if !secrets.IsSensitiveName(s[m[4]:m[5]]) {
			continue
		}
		valStart := m[1]
		if valStart == len(s) {
			continue
		}
		var from, to int
		replacement := redacted
		switch {
		case s[m[8]] == '=':
			from, to = valStart, max(r.valueEnd(s, valStart), authCredentialEnd(s, valStart))
		case m[6] >= 0 && s[m[7]-1] == '\'':
			// A single-quoted key followed by ':' is a dict-repr member
			// ({'Authorization': 'Basic dXNl…==', 'X': 1}), not JSON.
			from, to = singleQuotedMemberSpan(s, valStart)
		case m[6] >= 0:
			// A quoted key followed by ':' is a JSON member: redact its value
			// JSON-aware so the rest of the document (and its later members)
			// keeps its shape. The closing quote's length gives the escape depth.
			from, to, replacement = jsonValueSpan(s, valStart, m[7]-m[6]-1)
		default:
			// A NAME without a closing quote followed by ':' is a header, YAML
			// or prefix form ("Authorization: Basic dXNl…=="): the value is the
			// rest of the line, so no scheme word, padding or sub-pair splits
			// it. When a quote opens right before the NAME (logfmt
			// msg="auth: Basic …" user=bob) the value stops at its closing quote.
			from, to = valStart, colonValueEnd(s, valStart, m[2], m[3])
		}
		if to <= from {
			continue
		}
		// Keep everything through "NAME<sep>" and redact the whole value —
		// internal spaces included — up to the computed end.
		b.WriteString(s[last:from])
		b.WriteString(replacement)
		last = to
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// singleQuotedMemberSpan returns the span to redact for the value of a
// single-quoted dict-repr member ('Authorization': 'Basic …'). A quoted value
// is redacted between its quotes, which stay in place so later members keep
// their shape; an unterminated or unquoted value runs to the end of the line
// (fail closed), like any other colon-form value.
func singleQuotedMemberSpan(s string, valStart int) (from, to int) {
	q := s[valStart]
	if q != '\'' && q != '"' {
		return valStart, colonValueEnd(s, valStart, -1, -1)
	}
	end := quotedValueEnd(s, valStart, valStart+1)
	if end-1 > valStart && s[end-1] == q {
		return valStart + 1, end - 1
	}
	return valStart + 1, end
}

// valueEnd returns the offset at which a sensitive "NAME=" pair's value ends.
// The value runs from valStart to end-of-line, EXCEPT it stops before the next
// whitespace-preceded "NAME=" / "NAME:" key so a following pair on the same line
// is redacted independently rather than swallowed. A candidate key whose '='
// is base64 padding (see isPaddingBoundary) is part of the value, not a key:
// "Basic dXNlcjpwYXNzd29yZA==" must not end at " dXNlcjpwYXNzd29yZA=".
// Trailing whitespace before the boundary is excluded so the separator run is
// preserved verbatim. The scan loops over FindStringIndex rather than
// FindAllStringIndex so the common path allocates no slice.
func (r *Redactor) valueEnd(s string, valStart int) int {
	// A value never spans a newline.
	end := lineEnd(s, valStart)
	// Stop before the next key-like token on the same line.
	for off := valStart; off < end; {
		loc := r.keyBoundaryRe.FindStringIndex(s[off:end])
		if loc == nil {
			break
		}
		if !isPaddingBoundary(s, off+loc[1]-1, end) {
			end = off + loc[0]
			break
		}
		off += loc[1]
	}
	trimmed := strings.TrimRight(s[valStart:end], nameValueTrimCutset)
	return valStart + len(trimmed)
}

// isPaddingBoundary reports whether the key-boundary separator at sepIdx is
// really base64 padding rather than a "NAME=" separator: the separator is '='
// and it is followed by the end of the line (lineEnd) or a byte that cannot
// start a value — another '=', whitespace, a quote, or a list/map delimiter
// (',', ';', ')', ']', '}', '>'), as in a Map.toString header dump
// "{Authorization=Basic dXNl…=, Accept=…}". A genuine pair ("next=1") always
// has a value after its '='.
func isPaddingBoundary(s string, sepIdx, lineEnd int) bool {
	if s[sepIdx] != '=' {
		return false
	}
	if sepIdx+1 >= lineEnd {
		return true
	}
	switch s[sepIdx+1] {
	case '=', ' ', '\t', '\r', '\f', '\v', '\'', '"', ',', ';', ')', ']', '}', '>':
		return true
	}
	return false
}

// httpAuthSchemes are the HTTP authentication scheme words (RFC 9110 section
// 11 and the IANA registry entries seen in logs) that prefix a credential.
var httpAuthSchemes = []string{"Basic", "Bearer", "Token", "Digest", "Negotiate"}

// authCredentialEnd returns the end of the credential token that follows an
// HTTP auth scheme word at the start of the value at valStart
// ("Bearer abc=123"), or valStart when the value has no scheme prefix. The
// '=' form of a NAME=value pair otherwise ends the value at a key-like token,
// and a credential such as "abc=123" looks like one; the scheme word marks
// the next token as the credential. It scans bytes and does not allocate.
func authCredentialEnd(s string, valStart int) int {
	end := lineEnd(s, valStart)
	word := strings.IndexAny(s[valStart:end], " \t")
	if word <= 0 || !isAuthScheme(s[valStart:valStart+word]) {
		return valStart
	}
	i := valStart + word
	for i < end && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for i < end && !strings.ContainsRune(nameValueTrimCutset, rune(s[i])) {
		i++
	}
	return i
}

// isAuthScheme reports whether word is an HTTP auth scheme, case-insensitively.
func isAuthScheme(word string) bool {
	for _, scheme := range httpAuthSchemes {
		if strings.EqualFold(word, scheme) {
			return true
		}
	}
	return false
}

// colonValueEnd returns where the value of an unquoted "NAME: value" pair
// ends: the end of the line, or — when a quote (quoteStart:quoteEnd, -1 when
// absent) opens immediately before NAME — the matching closing quote, so a
// logfmt field msg="auth: Basic …" keeps its closing quote and later pairs.
// Ending at end of line is deliberate fail-closed behaviour: a header, YAML or
// "error:" prefix value may hold spaces, a scheme word, base64 padding or
// Digest sub-pairs, so benign pairs after a sensitive colon NAME on the same
// line ("password: x status: ok", Go's "map[password:x user:bob]") are
// redacted too. Do not split it at key boundaries again: "token: prefix
// abcdefXYZ=123" would leak its credential.
func colonValueEnd(s string, valStart, quoteStart, quoteEnd int) int {
	end := lineEnd(s, valStart)
	if quoteStart >= 0 {
		if q := strings.Index(s[valStart:end], s[quoteStart:quoteEnd]); q >= 0 {
			end = valStart + q
		}
	}
	return valStart + len(strings.TrimRight(s[valStart:end], nameValueTrimCutset))
}

// denyAttr redacts a whole attribute whose key (or enclosing group) is
// sensitive: every leaf value becomes RedactionMarker while keys and group
// structure stay visible for diagnosability, so an inline
// slog.Group("password", "value", x) and a logger.WithGroup("password") scope
// render alike. An empty Attr is returned as is: slog handlers drop it, and
// giving it a value would print a spurious empty-key member.
func denyAttr(a slog.Attr) slog.Attr {
	if a.Equal(slog.Attr{}) {
		return a
	}
	v := a.Value.Resolve()
	if v.Kind() != slog.KindGroup {
		return slog.String(a.Key, RedactionMarker)
	}
	members := v.Group()
	out := make([]slog.Attr, len(members))
	for i, m := range members {
		out[i] = denyAttr(m)
	}
	return slog.Attr{Key: a.Key, Value: slog.GroupValue(out...)}
}

// lineEnd returns the offset of the first newline at or after from, or len(s).
func lineEnd(s string, from int) int {
	if nl := strings.IndexByte(s[from:], '\n'); nl >= 0 {
		return from + nl
	}
	return len(s)
}

func (r *Redactor) redactURLCredentials(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.User == nil {
		return r.urlCredRe.ReplaceAllString(s, "://"+redacted+":"+redacted+"@")
	}
	// Re-insert the marker after serialising: url.URL.String would
	// percent-encode the brackets of an in-place "[REDACTED]" userinfo.
	u.User = nil
	out := u.String()
	if i := strings.Index(out, "://"); i >= 0 {
		return out[:i+3] + redacted + ":" + redacted + "@" + out[i+3:]
	}
	return out
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

// redactSeq walks a slice or array, redacting each element copy-on-write. A
// string element that follows a sensitive flag (argv such as
// ["--password", "hunter2"]) is redacted whole, since the flag names the
// secret but the value carries no NAME=value shape of its own.
func (r *Redactor) redactSeq(rv reflect.Value) (reflect.Value, bool) {
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return rv, false
	}
	n := rv.Len()
	var out reflect.Value
	changed := false
	afterSensitiveFlag := false
	for i := 0; i < n; i++ {
		elem := rv.Index(i)
		str, isString := stringElem(elem)
		var nv reflect.Value
		var ch bool
		if afterSensitiveFlag && isString && !strings.HasPrefix(str, "-") {
			nv, ch = redactedReflectVal, true
		} else {
			nv, ch = r.redactReflect(elem)
		}
		afterSensitiveFlag = isString && isSensitiveFlag(str)
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

// stringElem returns the string held by a sequence element, unwrapping an
// interface slot, and whether the element is a string at all.
func stringElem(v reflect.Value) (string, bool) {
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return "", false
	}
	return v.String(), true
}

// isSensitiveFlag reports whether arg is a command-line flag ("-x", "--name")
// without an inline value whose name denotes a credential, e.g. --password or
// --api-key, so the argument after it is a secret. A "--name=value" argument
// is handled by the NAME=value pass. A flag ending in one of
// nonSecretValueFlagSuffixes is never sensitive: the argument after it is not
// the secret itself.
func isSensitiveFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") || strings.Contains(arg, "=") {
		return false
	}
	for _, suffix := range nonSecretValueFlagSuffixes {
		if hasSuffixFold(arg, suffix) {
			return false
		}
	}
	return secrets.IsSensitiveName(strings.TrimLeft(arg, "-"))
}

// nonSecretValueFlagSuffixes mark credential-named flags whose following
// argument is not a secret: a "-stdin" flag (docker login --password-stdin)
// reads the secret from stdin and takes no value, and a "-file", "-dir" or
// "-path" flag (--token-file, --password-file) takes a path that only refers
// to where the secret is stored, which logs keep for diagnosability.
var nonSecretValueFlagSuffixes = []string{"-stdin", "-file", "-dir", "-path"}

// hasSuffixFold is strings.HasSuffix with ASCII case folding, without
// allocating a lowered copy.
func hasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
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
