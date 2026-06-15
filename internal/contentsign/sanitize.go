package contentsign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	// htmlHiddenElementRe matches a single element that carries an inline style
	// attribute, together with its text content up to the next closing tag.
	// Go's RE2 has no backreferences, so the closing tag is matched by shape
	// (</tag>) rather than by name; this is conservative because hidden
	// wrappers in documentation are single inline elements with no nesting.
	// The match is only removed when hiddenStyleRe confirms a hiding rule.
	htmlHiddenElementRe = regexp.MustCompile(
		`(?is)<[a-z][a-z0-9]*\b[^>]*\bstyle\s*=\s*("[^"]*"|'[^']*')[^>]*>.*?</[a-z][a-z0-9]*\s*>`)
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
	catZeroWidth = "zero-width"         // ZWSP/ZWNJ/ZWJ, LRM/RLM, word-joiner, isolates, BOM
	catBiDi      = "bidi"               // BiDi embedding/override controls
	catVarSel    = "variation-selector" // variation selectors (base + supplement)
	catTag       = "tag"                // Unicode TAG characters (instruction smuggling)
	catSpecials  = "specials"           // U+FFF0..U+FFFF specials block
	catControl   = "control"            // C0/C1 control characters
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
	{0x200B, 0x200F, catZeroWidth}, // zero-width space/non-joiner/joiner, LRM, RLM
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
	s = htmlHiddenElementRe.ReplaceAllStringFunc(s, func(m string) string {
		if styleAttr := styleAttrRe.FindStringSubmatch(m); styleAttr != nil {
			// Group 2 (double-quoted) or 3 (single-quoted) holds the value.
			value := styleAttr[2] + styleAttr[3]
			if hiddenStyleRe.MatchString(value) {
				return ""
			}
		}
		return m
	})
	return s
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

// isStrippedControl reports whether r is a C0 or C1 control character that must
// be removed. The whitespace controls tab, newline, and carriage return are
// preserved.
func isStrippedControl(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return false
	}
	if r <= 0x1F { // C0 controls
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

	report := SanitizeReport{}
	sanitized := sanitizeJSONValue(root, opts, &report)

	// Encode with HTML escaping disabled so that <, >, and & in legitimate
	// documentation HTML are preserved verbatim; the output then differs from the
	// input only by the dangerous characters we deliberately stripped.
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
