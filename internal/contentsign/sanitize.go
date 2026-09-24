package contentsign

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// HTML scrubbing regexps, compiled once. These are deliberately conservative:
// the input is documentation HTML, so only constructs that hide content from a
// human reader while remaining in the byte stream are removed. Legitimate
// visible markup (e.g. <code>, <p>) is left untouched.
var (
	// htmlCommentRe matches HTML comments, including multi-line ones.
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	// htmlScriptRe matches a complete <script>...</script> block.
	htmlScriptRe = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`)
	// htmlStyleRe matches a complete <style>...</style> block.
	htmlStyleRe = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style\s*>`)
	// htmlTagRe matches a single HTML start or end tag. Capture groups:
	//   1: "/" for an end tag (empty for a start tag)
	//   2: the tag name
	//   3: the raw attribute text (quoted values may contain '>')
	// Go's RE2 has no backreferences, so balanced/nested elements cannot be
	// matched by one expression; stripHiddenElements walks this tag stream and
	// tracks the open-element stack instead.
	htmlTagRe = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9:-]*)((?:[^>"']|"[^"]*"|'[^']*')*)>`)
	// htmlAttrRe matches one attribute in a tag's attribute text. Capture
	// groups: 1 the name, then the value when present — 2 double-quoted,
	// 3 single-quoted, 4 unquoted (all valid HTML).
	htmlAttrRe = regexp.MustCompile(`([^\s"'>/=]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+)))?`)
	// cssCommentRe matches CSS comments, which can split a declaration
	// (display:/**/none) without changing its meaning.
	cssCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// cssImportantRe matches a trailing !important priority on a CSS value.
	cssImportantRe = regexp.MustCompile(`\s*!\s*important\s*$`)
	// cssZeroRe matches a CSS number or length that is exactly zero (0, 0.0,
	// .0, 0px, 0em, 0%), but not a small non-zero one such as 0.875rem.
	cssZeroRe = regexp.MustCompile(`^[+-]?(?:0+(?:\.0*)?|\.0+)(?:[a-z]+|%)?$`)
)

// Sanitize category names recorded in SanitizeReport.Categories.
const (
	catZeroWidth = "zero-width"         // ZWSP/ZWNJ/ZWJ, LRM/RLM, word-joiner, isolates, BOM, soft hyphen
	catBiDi      = "bidi"               // BiDi embedding/override controls
	catLineSep   = "line-separator"     // U+2028/U+2029 line & paragraph separators
	catVarSel    = "variation-selector" // variation selectors (base + supplement)
	catTag       = "tag"                // Unicode TAG characters (instruction smuggling)
	catSpecials  = "specials"           // U+FFF0..U+FFFF specials block
	catControl   = "control"            // C0/C1 control characters and DEL
	catFormat    = "format"             // other Cf/Zl/Zp runes caught by the category net
)

// stripRange is an inclusive [lo, hi] rune range assigned to a report category.
type stripRange struct {
	lo, hi rune
	cat    string
}

// invisibleRanges are the dangerous invisible/format ranges removed when
// StripInvisible is set. They are kept sorted by lo so the lookup can stop
// early. Ranges do not overlap, so each rune maps to at most one category.
var invisibleRanges = []stripRange{
	{0x00AD, 0x00AD, catZeroWidth}, // soft hyphen (invisible conditional hyphen)
	{0x034F, 0x034F, catZeroWidth}, // combining grapheme joiner (Mn, not Cf)
	{0x115F, 0x1160, catZeroWidth}, // Hangul choseong/jungseong fillers (Lo, not Cf)
	{0x180B, 0x180D, catVarSel},    // Mongolian free variation selectors (Mn, not Cf)
	{0x200B, 0x200F, catZeroWidth}, // zero-width space/non-joiner/joiner, LRM, RLM
	{0x2028, 0x2029, catLineSep},   // line separator, paragraph separator
	{0x202A, 0x202E, catBiDi},      // LRE, RLE, PDF, LRO, RLO
	{0x2060, 0x2069, catZeroWidth}, // word joiner, invisible operators, isolates
	{0x3164, 0x3164, catZeroWidth}, // Hangul filler (Lo, not Cf)
	{0xFE00, 0xFE0F, catVarSel},    // variation selectors 1..16
	{0xFEFF, 0xFEFF, catZeroWidth}, // BOM / zero-width no-break space
	{0xFFA0, 0xFFA0, catZeroWidth}, // halfwidth Hangul filler (Lo, not Cf)
	{0xFFF0, 0xFFFF, catSpecials},  // specials block
	{0xE0000, 0xE007F, catTag},     // tag characters (most dangerous)
	{0xE0100, 0xE01EF, catVarSel},  // variation selectors supplement
}

// SanitizeText runs the Unicode sanitization pipeline over s according to opts
// and returns the cleaned string plus a report of what changed.
//
// The stages follow the canonicalize-before-match principle: ALL removal and
// normalization happens before the structural HTML scrub, so a stripped
// character can never "repair" a hidden construct that the HTML matcher already
// cleared. The order is: (1) strip invisible/control runes (de-obfuscate), (2)
// NFKC normalization (fold compatibility forms such as fullwidth '＜' to '<'),
// (3) re-strip iff NFKC changed something (in case a decomposition exposed a
// strippable rune), then (4) the HTML scrub on fully canonical text.
//
// Were the HTML scrub to run first, an interleaved zero-width/control character
// (<scr␣ipt>) or a compatibility-equivalent bracket (＜script＞) would slip past
// the literal-letter regexps and then be normalized away — reconstituting the
// dangerous construct in the output. The order is deterministic (identical input
// and opts always yield identical output) and idempotent (each strip pass and
// NFKC are individually idempotent, and step 4 only ever removes content).
func SanitizeText(s string, opts SanitizeOptions) (string, SanitizeReport) {
	report := SanitizeReport{}

	// 1. De-obfuscate: remove invisible/control runes so they cannot survive into
	//    the structural matcher and be repaired by a later strip.
	if opts.StripInvisible || opts.StripControl {
		s = stripRunes(s, opts, &report)
	}

	// 2. Canonicalize compatibility forms (fullwidth, ligatures, ...).
	if opts.NormalizeNFKC {
		normalized := norm.NFKC.String(s)
		if normalized != s {
			report.NFKCChanged = true
			// 3. Re-strip only when NFKC actually changed the string, in case a
			//    compatibility decomposition exposed a newly strippable rune. This
			//    is skipped in the default (NFKC-off) profile, so the common-path
			//    strip counts are unaffected.
			if opts.StripInvisible || opts.StripControl {
				normalized = stripRunes(normalized, opts, &report)
			}
		}
		s = normalized
	}

	// 4. Structural HTML scrub, now operating on fully canonical text.
	if opts.StripHTML {
		s = stripHTML(s)
	}

	return s, report
}

// stripHTML removes HTML comments, <script>/<style> blocks, and inline-hidden
// elements, leaving visible markup intact.
func stripHTML(s string) string {
	s = htmlCommentRe.ReplaceAllString(s, "")
	s = htmlScriptRe.ReplaceAllString(s, "")
	s = htmlStyleRe.ReplaceAllString(s, "")
	return stripHiddenElements(s)
}

// voidElements are the HTML elements that never have content or an end tag
// (per the HTML standard), so they must not be counted as open elements.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// htmlTag is one tag located by htmlTagRe in the input.
type htmlTag struct {
	start, end int    // byte offsets of the whole tag
	isEnd      bool   // </name>
	name       string // lower-cased element name
	attrs      string // raw attribute text of a start tag
}

// nextTag returns the first tag at or after offset i, or false if none.
func nextTag(s string, i int) (htmlTag, bool) {
	loc := htmlTagRe.FindStringSubmatchIndex(s[i:])
	if loc == nil {
		return htmlTag{}, false
	}
	return htmlTag{
		start: i + loc[0],
		end:   i + loc[1],
		isEnd: loc[3] > loc[2],
		name:  strings.ToLower(s[i+loc[4] : i+loc[5]]),
		attrs: s[i+loc[6] : i+loc[7]],
	}, true
}

// stripHiddenElements removes every element that is hidden from a human
// reader (see isHiddenTag), together with ALL of its content. It follows the
// browser's view of where that element ends: at its own matching end tag
// (nested same-name elements and void elements such as <br> are accounted
// for), at the end tag of an enclosing element (which implicitly closes it),
// or — when never closed — at the end of the input, so an unterminated hidden
// element fails closed instead of leaking its content. Non-hidden tags, and
// the text between them, are emitted verbatim, so legitimate visible markup is
// preserved.
func stripHiddenElements(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var open []string // stack of open (visible) element names
	for i := 0; i < len(s); {
		tag, ok := nextTag(s, i)
		if !ok {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i:tag.start]) // text before this tag, verbatim

		switch {
		case tag.isEnd:
			if idx := lastIndex(open, tag.name); idx >= 0 {
				open = open[:idx]
			}
		case isHiddenTag(tag.attrs):
			if voidElements[tag.name] {
				i = tag.end // a hidden void element has no content: drop just the tag
			} else {
				i = skipHiddenElement(s, tag.end, tag.name, open)
			}
			continue
		case !voidElements[tag.name]:
			open = append(open, tag.name)
		}
		b.WriteString(s[tag.start:tag.end])
		i = tag.end
	}
	return b.String()
}

// skipHiddenElement returns the offset where the hidden element name, whose
// opening tag ended at pos, ends. Inside it, it tracks its own open-element
// stack: an end tag closes the innermost open element of that name (and any
// left open inside it); the hidden element's own end tag is consumed. An end
// tag for an element in ancestors (the enclosing open elements) implicitly
// closes the hidden element and is kept. A stray end tag matching nothing is
// ignored, as browsers do. An element that is never closed extends to the end
// of the input.
func skipHiddenElement(s string, pos int, name string, ancestors []string) int {
	stack := []string{name}
	for i := pos; i < len(s); {
		tag, ok := nextTag(s, i)
		if !ok {
			break
		}
		i = tag.end
		if !tag.isEnd {
			if !voidElements[tag.name] {
				stack = append(stack, tag.name)
			}
			continue
		}
		switch idx := lastIndex(stack, tag.name); {
		case idx == 0:
			return tag.end // the hidden element's own end tag
		case idx > 0:
			stack = stack[:idx]
		case lastIndex(ancestors, tag.name) >= 0:
			return tag.start // an enclosing element closes, ending this one too
		}
	}
	return len(s) // never closed: hidden through the end of the input
}

// lastIndex returns the index of the last occurrence of name in stack, or -1.
func lastIndex(stack []string, name string) int {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == name {
			return i
		}
	}
	return -1
}

// isHiddenTag reports whether an opening tag's attributes hide the element
// from a human reader: the boolean hidden attribute, or a style attribute
// (quoted or not) that declares a hiding rule (see isHidingStyle). Only the
// first style attribute counts, as in browsers. Attribute values are
// entity-decoded first, since browsers decode them (display&colon;none).
func isHiddenTag(attrs string) bool {
	sawStyle := false
	for _, m := range htmlAttrRe.FindAllStringSubmatch(attrs, -1) {
		switch strings.ToLower(m[1]) {
		case "hidden":
			return true
		case "style":
			if sawStyle {
				continue
			}
			sawStyle = true
			if isHidingStyle(html.UnescapeString(m[2] + m[3] + m[4])) {
				return true
			}
		}
	}
	return false
}

// isHidingStyle reports whether a CSS declaration list hides its element:
// display:none, visibility:hidden|collapse, opacity:0, font-size:0, or
// color:transparent. Values are matched exactly per declaration, so a small
// but visible font-size such as 0.875rem is not treated as hidden. Comments
// and CSS escapes (display:n\6f ne) are resolved first, as browsers do.
func isHidingStyle(style string) bool {
	style = strings.ToLower(cssUnescape(cssCommentRe.ReplaceAllString(style, "")))
	for decl := range strings.SplitSeq(style, ";") {
		prop, val, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(cssImportantRe.ReplaceAllString(val, ""))
		switch strings.TrimSpace(prop) {
		case "display":
			if val == "none" {
				return true
			}
		case "visibility":
			if val == "hidden" || val == "collapse" {
				return true
			}
		case "opacity", "font-size":
			if cssZeroRe.MatchString(val) {
				return true
			}
		case "color":
			if val == "transparent" {
				return true
			}
		}
	}
	return false
}

// cssUnescape resolves CSS backslash escapes: a backslash followed by 1-6 hex
// digits (and one optional whitespace) is that code point, and a backslash
// followed by any other character is that character, so an escaped keyword
// such as n\6f ne or n\one reads as "none". Invalid code points become
// U+FFFD, per the CSS Syntax spec.
func cssUnescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i + 1
		for j < len(s) && j-i <= 6 && isHexDigit(s[j]) {
			j++
		}
		if j == i+1 { // not a hex escape: the next character stands for itself
			b.WriteByte(s[j])
			i = j + 1
			continue
		}
		// At most six hex digits fit in 32 bits, so ParseUint cannot fail;
		// the range check comes before the conversion to rune.
		cp, _ := strconv.ParseUint(s[i+1:j], 16, 32)
		r := unicode.ReplacementChar
		if cp != 0 && cp <= unicode.MaxRune && (cp < 0xD800 || cp > 0xDFFF) {
			r = rune(cp)
		}
		b.WriteRune(r)
		if j < len(s) && strings.IndexByte(" \t\n\r\f", s[j]) >= 0 {
			j++ // a single whitespace terminates a hex escape and is consumed
		}
		i = j
	}
	return b.String()
}

// isHexDigit reports whether c is an ASCII hex digit.
func isHexDigit(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// stripRunes performs the single-pass rune filter for invisible and control
// characters, accumulating stripped counts into report.
func stripRunes(s string, opts SanitizeOptions, report *SanitizeReport) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if cat, ok := classifyStrip(r, opts); ok {
			report.RunesStripped++
			if report.Categories == nil {
				report.Categories = make(map[string]int)
			}
			report.Categories[cat]++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// classifyStrip reports whether r should be stripped and, if so, its category.
//
// The explicit invisibleRanges are checked first: they assign precise report
// categories and cover the dangerous runes that are NOT general-category Format
// (variation selectors and Hangul fillers, which are Mn/Lo). Anything not
// enumerated then falls through to a Unicode category net — Cf (format), Zl
// (line separator), and Zp (paragraph separator) — so newly assigned or
// less-common format runes (e.g. U+180E, U+17B4, the Arabic/Syriac Cf block) are
// stripped without hand-maintaining the list. A blanket Mn check is deliberately
// avoided: it would strip legitimate combining accents (U+0300..U+036F).
func classifyStrip(r rune, opts SanitizeOptions) (string, bool) {
	// Every invisible/format rune lives at or above the lowest explicit range
	// (U+00AD), so ASCII can skip both the range scan and the category net — the
	// common case in documentation, which is overwhelmingly ASCII.
	if opts.StripInvisible && r >= invisibleRanges[0].lo {
		for _, rg := range invisibleRanges {
			if r < rg.lo {
				break // ranges are sorted; no later range can match
			}
			if r <= rg.hi {
				return rg.cat, true
			}
		}
		if unicode.In(r, unicode.Cf, unicode.Zl, unicode.Zp) {
			return catFormat, true
		}
	}
	if opts.StripControl && isStrippedControl(r) {
		return catControl, true
	}
	return "", false
}

// isStrippedControl reports whether r is a C0 or C1 control character (or DEL)
// that must be removed. The whitespace controls tab, newline, and carriage
// return are preserved.
func isStrippedControl(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return false
	}
	if r <= 0x1F || r == 0x7F { // C0 controls and DEL
		return true
	}
	if r >= 0x80 && r <= 0x9F { // C1 controls
		return true
	}
	return false
}

// SanitizeJSONStrings sanitizes only the string VALUES of a JSON document,
// leaving object keys, structure, and non-string scalars untouched, so that an
// external indexer (e.g. the DevDocs SQLite indexer) still sees the same shape.
//
// It decodes into a generic tree using a json.Decoder with UseNumber so numeric
// literals round-trip without lossy float reformatting, walks the tree applying
// SanitizeText to every string value, then re-marshals. The returned report
// aggregates the per-value reports. ctx is honored once before the CPU-bound
// walk.
//
// Because it buffers both the decoded tree and the re-encoded output in memory
// (~2-3x the input), callers that read raw from disk should bound the input
// size before calling it (as IngestDevDocs does).
func SanitizeJSONStrings(ctx context.Context, raw []byte, opts SanitizeOptions) ([]byte, SanitizeReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, SanitizeReport{}, fmt.Errorf("sanitizing JSON: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, SanitizeReport{}, fmt.Errorf("decoding JSON: %w", err)
	}
	// Reject trailing data after the top-level value: a single Decode silently
	// ignores anything after the first JSON value, so the re-encode below would
	// otherwise truncate a malformed/concatenated document without error.
	if _, err := dec.Token(); err != io.EOF {
		return nil, SanitizeReport{}, fmt.Errorf("decoding JSON: unexpected trailing data after top-level value")
	}

	report := SanitizeReport{}
	sanitized, err := sanitizeJSONValue(root, opts, &report, 0)
	if err != nil {
		return nil, SanitizeReport{}, err
	}

	// Encode with HTML escaping disabled so that <, >, and & in legitimate
	// documentation HTML are preserved verbatim. Re-encoding the decoded tree may
	// reorder object keys and normalize insignificant whitespace; only string
	// VALUES change in content (by the dangerous characters stripped), which is
	// what the downstream indexer consumes. The manifest hash is recomputed over
	// the rewritten file, so byte-layout changes are harmless.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(sanitized); err != nil {
		return nil, SanitizeReport{}, fmt.Errorf("encoding JSON: %w", err)
	}
	// json.Encoder appends a trailing newline; trim it to match json.Marshal.
	out := bytes.TrimRight(buf.Bytes(), "\n")
	return out, report, nil
}

// ErrJSONTooDeep is returned by SanitizeJSONStrings for a document nested more
// than maxJSONDepth levels deep.
var ErrJSONTooDeep = errors.New("contentsign: JSON nesting exceeds sanitization depth limit")

// maxJSONDepth bounds the recursion in sanitizeJSONValue so a pathologically
// nested db.json (deeply nested arrays/objects, well within the byte-size bound)
// cannot exhaust the goroutine stack. Legitimate documentation JSON nests only a
// handful of levels; 1000 is far beyond any real structure while staying clear
// of the stack limit.
const maxJSONDepth = 1000

// sanitizeJSONValue recursively sanitizes string values within a decoded JSON
// tree, merging each string's report into agg. Object keys and non-string
// scalars (numbers, booleans, null) are returned unchanged. A subtree deeper
// than maxJSONDepth fails with ErrJSONTooDeep: it is rejected rather than
// passed through unsanitized.
func sanitizeJSONValue(v any, opts SanitizeOptions, agg *SanitizeReport, depth int) (any, error) {
	if depth >= maxJSONDepth {
		return nil, fmt.Errorf("sanitizing JSON: %w (%d levels)", ErrJSONTooDeep, maxJSONDepth)
	}
	switch val := v.(type) {
	case string:
		clean, r := SanitizeText(val, opts)
		mergeReport(agg, r)
		return clean, nil
	case map[string]any:
		for k, child := range val {
			clean, err := sanitizeJSONValue(child, opts, agg, depth+1)
			if err != nil {
				return nil, err
			}
			val[k] = clean
		}
		return val, nil
	case []any:
		for i, child := range val {
			clean, err := sanitizeJSONValue(child, opts, agg, depth+1)
			if err != nil {
				return nil, err
			}
			val[i] = clean
		}
		return val, nil
	default:
		return v, nil
	}
}

// mergeReport folds src into dst: summing RunesStripped, merging per-category
// counts, and OR-ing NFKCChanged.
func mergeReport(dst *SanitizeReport, src SanitizeReport) {
	dst.RunesStripped += src.RunesStripped
	if src.NFKCChanged {
		dst.NFKCChanged = true
	}
	for cat, n := range src.Categories {
		if dst.Categories == nil {
			dst.Categories = make(map[string]int)
		}
		dst.Categories[cat] += n
	}
}
