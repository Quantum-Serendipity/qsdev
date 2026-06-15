package contentsign

import (
	"os"
	"path/filepath"
	"testing"
)

// writeIndex writes raw bytes as a named index.json under dir and returns its path.
func writeIndex(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("writing index %q: %v", name, err)
	}
	return p
}

func TestContentDiffDevDocs(t *testing.T) {
	t.Parallel()

	const base = `{"entries":[{"name":"Foreword","path":"book/foreword","type":"Guide"},{"name":"Intro","path":"book/intro","type":"Guide"}],"types":[{"name":"Guide"}]}`

	tests := []struct {
		name     string
		oldBody  string
		newBody  string
		wantAdd  int
		wantRem  int
		wantMod  int
		sizeSign int // -1 negative, 0 zero, 1 positive
	}{
		{
			name:    "identical",
			oldBody: base,
			newBody: base,
		},
		{
			name:     "additions",
			oldBody:  base,
			newBody:  `{"entries":[{"name":"Foreword","path":"book/foreword","type":"Guide"},{"name":"Intro","path":"book/intro","type":"Guide"},{"name":"New","path":"book/new","type":"Guide"}]}`,
			wantAdd:  1,
			sizeSign: 1,
		},
		{
			name:     "removals",
			oldBody:  base,
			newBody:  `{"entries":[{"name":"Foreword","path":"book/foreword","type":"Guide"}]}`,
			wantRem:  1,
			sizeSign: -1,
		},
		{
			name:     "modified name",
			oldBody:  base,
			newBody:  `{"entries":[{"name":"Foreword","path":"book/foreword","type":"Guide"},{"name":"Introduction","path":"book/intro","type":"Guide"}],"types":[{"name":"Guide"}]}`,
			wantMod:  1,
			sizeSign: 1, // "Intro" -> "Introduction" lengthens the index
		},
		{
			name:     "modified type",
			oldBody:  base,
			newBody:  `{"entries":[{"name":"Foreword","path":"book/foreword","type":"Guide"},{"name":"Intro","path":"book/intro","type":"Reference"}],"types":[{"name":"Guide"}]}`,
			wantMod:  1,
			sizeSign: 1, // "Guide" -> "Reference" lengthens the index
		},
		{
			name:     "mixed add remove modify",
			oldBody:  base,
			newBody:  `{"entries":[{"name":"Foreword v2","path":"book/foreword","type":"Guide"},{"name":"Added","path":"book/added","type":"Guide"}]}`,
			wantAdd:  1, // book/added
			wantRem:  1, // book/intro
			wantMod:  1, // book/foreword name changed
			sizeSign: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			oldPath := writeIndex(t, dir, "old.json", tc.oldBody)
			newPath := writeIndex(t, dir, "new.json", tc.newBody)

			diff, err := ContentDiffDevDocs(oldPath, newPath)
			if err != nil {
				t.Fatalf("ContentDiffDevDocs: %v", err)
			}
			if diff.AddedEntries != tc.wantAdd {
				t.Errorf("AddedEntries = %d, want %d", diff.AddedEntries, tc.wantAdd)
			}
			if diff.RemovedEntries != tc.wantRem {
				t.Errorf("RemovedEntries = %d, want %d", diff.RemovedEntries, tc.wantRem)
			}
			if diff.ModifiedEntries != tc.wantMod {
				t.Errorf("ModifiedEntries = %d, want %d", diff.ModifiedEntries, tc.wantMod)
			}
			if got := sign(diff.SizeChangeBytes); got != tc.sizeSign {
				t.Errorf("sign(SizeChangeBytes=%d) = %d, want %d", diff.SizeChangeBytes, got, tc.sizeSign)
			}
		})
	}
}

func TestContentDiffDevDocsErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	good := writeIndex(t, dir, "good.json", `{"entries":[]}`)
	malformed := writeIndex(t, dir, "bad.json", `{"entries":[`)
	missing := filepath.Join(dir, "does-not-exist.json")

	tests := []struct {
		name    string
		oldPath string
		newPath string
	}{
		{name: "old missing", oldPath: missing, newPath: good},
		{name: "new missing", oldPath: good, newPath: missing},
		{name: "old malformed", oldPath: malformed, newPath: good},
		{name: "new malformed", oldPath: good, newPath: malformed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ContentDiffDevDocs(tc.oldPath, tc.newPath); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

// sign reports the sign of n as -1, 0, or 1.
func sign(n int64) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}
