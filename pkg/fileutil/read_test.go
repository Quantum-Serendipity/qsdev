package fileutil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestReadRegularFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	small := write("small", "abc")
	atLimit := write("at-limit", "0123456789")
	overLimit := write("over-limit", "0123456789x")

	tests := []struct {
		name    string
		path    string
		limit   int64
		want    string
		wantErr error
	}{
		{name: "regular file", path: small, limit: 10, want: "abc"},
		{name: "exactly at the limit", path: atLimit, limit: 10, want: "0123456789"},
		{name: "over the limit", path: overLimit, limit: 10, wantErr: fileutil.ErrTooLarge},
		{name: "missing file", path: filepath.Join(dir, "missing"), limit: 10, wantErr: fs.ErrNotExist},
		{name: "directory", path: dir, limit: 10, wantErr: fileutil.ErrNotRegular},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := fileutil.ReadRegularFile(tt.path, tt.limit)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ReadRegularFile error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("ReadRegularFile returned %q with an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadRegularFile: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("ReadRegularFile = %q, want %q", got, tt.want)
			}
		})
	}
}
