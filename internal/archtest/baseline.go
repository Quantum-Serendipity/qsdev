package archtest

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Rule is one architecture check over a parsed repository.
type Rule struct {
	ID    string
	Check func(*Repo) []Violation
}

// Violation is Count occurrences of a rule breach in one subject: an import
// edge "from -> to" or a package directory.
type Violation struct {
	Rule    string
	Subject string
	Count   int
}

// Key identifies a baseline entry.
type Key struct {
	Rule    string
	Subject string
}

func (k Key) String() string { return "[" + k.Rule + "] " + k.Subject }

// Set maps each violated key to its count.
type Set map[Key]int

// ErrMalformedBaseline reports a baseline line that does not parse.
var ErrMalformedBaseline = errors.New("malformed baseline entry")

// Collect runs every rule and sums the counts per key.
func Collect(repo *Repo, rules []Rule) Set {
	s := Set{}
	for _, r := range rules {
		for _, v := range r.Check(repo) {
			s[Key{v.Rule, v.Subject}] += v.Count
		}
	}
	return s
}

// Compare lists every difference between the computed violations and the
// baseline. Any difference fails: new or raised entries are regressions,
// and stale or lowered entries must be removed so the baseline shrinks.
func Compare(got, baseline Set) []string {
	var problems []string
	for _, k := range sortedKeys(got, baseline) {
		n, b := got[k], baseline[k]
		switch {
		case b == 0:
			problems = append(problems, fmt.Sprintf("new violation %s (count %d): fix it; the baseline may only shrink", k, n))
		case n == 0:
			problems = append(problems, fmt.Sprintf("baseline entry %s no longer occurs: delete it from baseline.txt", k))
		case n > b:
			problems = append(problems, fmt.Sprintf("%s: count rose from %d to %d: fix the new sites; the baseline may only shrink", k, b, n))
		case n < b:
			problems = append(problems, fmt.Sprintf("baseline entry %s now occurs %d times (baseline %d): lower it in baseline.txt", k, n, b))
		}
	}
	return problems
}

// CompareMonotone reports entries of current that are absent from, or
// higher than, base.
func CompareMonotone(current, base Set) []string {
	var problems []string
	for _, k := range sortedKeys(current) {
		n, b := current[k], base[k]
		switch {
		case b == 0:
			problems = append(problems, fmt.Sprintf("baseline.txt gained entry %s (count %d); it may only shrink", k, n))
		case n > b:
			problems = append(problems, fmt.Sprintf("baseline.txt raised %s from %d to %d; it may only shrink", k, b, n))
		}
	}
	return problems
}

// ParseBaseline reads "rule<TAB>subject<TAB>count" lines. Blank lines and
// lines starting with # are ignored, and CRLF line endings are accepted.
func ParseBaseline(r io.Reader) (Set, error) {
	s := Set{}
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(text) == "" || strings.HasPrefix(text, "#") {
			continue
		}
		k, n, err := parseEntry(text)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if _, dup := s[k]; dup {
			return nil, fmt.Errorf("line %d: duplicate %s: %w", line, k, ErrMalformedBaseline)
		}
		s[k] = n
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}
	return s, nil
}

func parseEntry(text string) (Key, int, error) {
	fields := strings.Split(text, "\t")
	if len(fields) != 3 {
		return Key{}, 0, fmt.Errorf("%q: want rule<TAB>subject<TAB>count: %w", text, ErrMalformedBaseline)
	}
	n, err := strconv.Atoi(fields[2])
	if err != nil || n < 1 {
		return Key{}, 0, fmt.Errorf("%q: count must be a positive integer: %w", text, ErrMalformedBaseline)
	}
	return Key{fields[0], fields[1]}, n, nil
}

const baselineHeader = `# Architecture baseline: known violations, which may only shrink.
# Format: rule<TAB>subject<TAB>count. TestArchitecture requires an exact
# match; CI's TestBaselineMonotone rejects new entries and raised counts.
# After fixing sites, regenerate with:
#   GOWORK=off go test ./internal/archtest -run TestArchitecture -update
`

// FormatBaseline renders s sorted by rule then subject, after a header.
func FormatBaseline(s Set) []byte {
	var buf bytes.Buffer
	buf.WriteString(baselineHeader)
	for _, k := range sortedKeys(s) {
		fmt.Fprintf(&buf, "%s\t%s\t%d\n", k.Rule, k.Subject, s[k])
	}
	return buf.Bytes()
}

func sortedKeys(sets ...Set) []Key {
	var keys []Key
	for _, s := range sets {
		for k := range s {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b Key) int {
		return cmp.Or(cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Subject, b.Subject))
	})
	return slices.Compact(keys)
}
