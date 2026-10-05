package ecosystem

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestFileDeclares(t *testing.T) {
	t.Parallel()
	key := regexp.MustCompile(`(?:^|[\s,{\[(])` + regexp.QuoteMeta(":tool") + `(?:[\s,{}\[\]()]|$)`)
	tests := []struct {
		name    string
		src     string
		comment byte
		want    bool
	}{
		{name: "declared", src: "{:aliases {:tool {:main-opts []}}}\n", comment: ';', want: true},
		{name: "at end of file", src: ":tool", comment: ';', want: true},
		{name: "absent", src: "{:deps {}}\n", comment: ';'},
		{name: "longer keyword", src: "{:aliases {:tool-old {}}}\n", comment: ';'},
		{name: "line comment", src: "{:aliases\n ;; {:tool {}}\n {}}\n", comment: ';'},
		{name: "trailing comment", src: "{:deps {} ; :tool\n}\n", comment: ';'},
		{name: "in a string", src: "{:doc \"use :tool \"}\n", comment: ';'},
		{name: "in a multi-line string", src: "{:doc \"line one\n :tool \"}\n", comment: ';'},
		{name: "after a string holding the comment character", src: "{:doc \"a;b\" :tool {}}\n", comment: ';', want: true},
		{name: "after an escaped quote", src: "{:doc \"a\\\"b\" :tool {}}\n", comment: ';', want: true},
		{name: "hash comment", src: "[\n  # {:tool, \"1\"}\n]\n", comment: '#'},
		{name: "interpolation is not a comment", src: "[{:x, \"#{v}\"}, {:tool, \"1\"}]\n", comment: '#', want: true},
		{name: "CRLF", src: "{:aliases\r\n {:tool {}}}\r\n", comment: ';', want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "src")
			if err := os.WriteFile(path, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := FileDeclares(path, tt.comment, key); got != tt.want {
				t.Errorf("FileDeclares(%q) = %v, want %v", tt.src, got, tt.want)
			}
		})
	}
	if FileDeclares(filepath.Join(t.TempDir(), "missing"), ';', key) {
		t.Error("FileDeclares(missing file) = true, want false")
	}
}
