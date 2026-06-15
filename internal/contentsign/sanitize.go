package contentsign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

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
	//   3: "/" for a self-closing start tag (empty otherwise)
	// Go's RE2 has no backreferences, so balanced/nested elements cannot be
	// matched by one expression; stripHiddenElements walks this tag stream and
	// tracks depth instead. As with the block regexps above, '>' inside a
	// quoted attribute value is not handled — documentation markup does not
	// rely on it.
	htmlTagRe = regexp.MustCompile(`(?is)<(/?)([a-z][a-z0-9]*)\b[^>]*?(/?)>`)
	// styleAttrRe extracts the first style="..." (or '...') attribute value from
	// an element's opening tag, so hiding rules are tested against the style
	// declaration only, not the element's visible text content.
	styleAttrRe = regexp.MustCompile(`(?is)\bstyle\s*=\s*("([^"]*)"|'([^']*)')`)
	// hiddenStyleRe detects the dangerous hiding declarations inside a style
	// attribute value (whitespace-insensitive around the colon).
	hiddenStyleRe = regexp.MustCompile(
		`(?is)(display\s*:\s*none|font-size\s*:\s*0|visibility\s*:\s*hidden|color\s*:\s*transparent)`)
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
	{0x200B, 0x200F, catZeroWidth}, // zero-width space/non-joiner/joiner, LRM, RLM
	{0x2028, 0x2029, catLineSep},   // line separator, paragraph separator
	{0x202A, 0x202E, catBiDi},      // LRE, RLE, PDF, LRO, RLO
	{0x2060, 0x2069, catZeroWidth}, // word joiner, invisible operators, isolates
	{0xFE00, 0xFE0F, catVarSel},    // variation selectors 1..16
	{0xFEFF, 0xFEFF, catZeroWidth}, // BOM / zero-width no-break space
	{0xFFF0, 0xFFFF, catSpecials},  // specials block
	{0xE0000, 0xE007F, catTag},     // tag characters (most dangerous)
	{0xE0100, 0xE01EF, catVarSel},  // variation selectors supplement
}

// SanitizeText runs the Unicode sanitization pipeline over s according to opts
// and returns the cleaned string plus a report of what changed.
//
// The stages are applied in a fixed order — HTML scrubbing, optional NFKC
// normalization, then a single rune-filter pass that removes invisible/format
// characters and control characters. This order is deterministic (identical
// input and opts always yield identical output) and idempotent for the strip
// stages (applying it twice equals applying it once); NFKC is idempotent too.
func SanitizeText(s string, opts SanitizeOptions) (string, SanitizeReport) {
	report := SanitizeReport{}

	if opts.StripHTML {
		s = stripHTML(s)
	}

	if opts.NormalizeNFKC {
		normalized := norm.NFKC.String(s)
		if normalized != s {
			report.NFKCChanged = true
		}
		s = normalized
	}

	if opts.StripInvisible || opts.StripControl {
		s = stripRunes(s, opts, &report)
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

// stripHiddenElements removes every element whose opening tag carries a hiding
// inline style, together with ALL of its content up to the matching closing
// tag. It scans the tag stream once, tracking nesting depth, so a nested child
// element does not prematurely terminate the hidden span — a single bounded
// regex stops at the first inner </tag> and leaks the trailing (hidden)
// content. Non-hidden tags, and the text between them, are emitted verbatim, so
// legitimate visible markup is preserved.
func stripHiddenElements(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		loc := htmlTagRe.FindStringSubmatchIndex(s[i:])
		if loc == nil {
			b.WriteString(s[i:])
			break
		}
		start, end := i+loc[0], i+loc[1]
		b.WriteString(s[i:start]) // text before this tag, verbatim

		tag := s[start:end]
		isEndTag := loc[3] > loc[2]    // capture group 1 ("/") matched a leading slash
		isSelfClose := loc[7] > loc[6] // capture group 3 ("/") matched a trailing slash
		if isEndTag || isSelfClose || !isHiddenTag(tag) {
			b.WriteString(tag)
			i = end
			continue
		}
		// Hidden opening tag: drop it and everything through its matching close.
		i = skipHiddenElement(s, end)
	}
	return b.String()
}

// isHiddenTag reports whether an opening tag's style attribute declares one of
// the hiding rules (display:none, font-size:0, visibility:hidden, transparent).
func isHiddenTag(tag string) bool {
	m := styleAttrRe.FindStringSubmatch(tag)
	if m == nil {
		return false
	}
	// Group 2 (double-quoted) or 3 (single-quoted) holds the value.
	return hiddenStyleRe.MatchString(m[2] + m[3])
}

// skipHiddenElement returns the offset just past the closing tag that matches a
// hidden element whose opening tag ended at pos, tracking nesting depth across
// intervening start/end tags. If the element is never closed it drops only the
// opening tag (returns pos) so that legitimate trailing content is not lost.
func skipHiddenElement(s string, pos int) int {
	depth := 1
	for i := pos; i < len(s); {
		loc := htmlTagRe.FindStringSubmatchIndex(s[i:])
		if loc == nil {
			return pos // unterminated element: drop only the opening tag
		}
		tagEnd := i + loc[1]
		switch {
		case loc[3] > loc[2]: // end tag
			depth--
			if depth == 0 {
				return tagEnd // matching close found
			}
		case loc[7] <= loc[6]: // start tag that is not self-closing
			depth++
		}
		i = tagEnd
	}
	return pos // unterminated element: drop only the opening tag
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
func classifyStrip(r rune, opts SanitizeOptions) (string, bool) {
	if opts.StripInvisible {
		for _, rg := range invisibleRanges {
			if r < rg.lo {
				break // ranges are sorted; no later range can match
			}
			if r <= rg.hi {
				return rg.cat, true
			}
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
	sanitized := sanitizeJSONValue(root, opts, &report)

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

// sanitizeJSONValue recursively sanitizes string values within a decoded JSON
// tree, merging each string's report into agg. Object keys and non-string
// scalars (numbers, booleans, null) are returned unchanged.
func sanitizeJSONValue(v any, opts SanitizeOptions, agg *SanitizeReport) any {
	switch val := v.(type) {
	case string:
		clean, r := SanitizeText(val, opts)
		mergeReport(agg, r)
		return clean
	case map[string]any:
		for k, child := range val {
			val[k] = sanitizeJSONValue(child, opts, agg)
		}
		return val
	case []any:
		for i, child := range val {
			val[i] = sanitizeJSONValue(child, opts, agg)
		}
		return val
	default:
		return v
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
