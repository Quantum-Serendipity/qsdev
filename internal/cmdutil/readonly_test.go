package cmdutil

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestMarkReadOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		mark     bool
		viaFlag  string
		lostBy   []string
		wantArgs []string
	}{
		{name: "always read-only", mark: true, wantArgs: []string{"mcp", "status"}},
		{name: "read-only via flag", mark: true, viaFlag: "dry-run", wantArgs: []string{"mcp", "status", "--dry-run"}},
		{name: "read-only unless probing", mark: true, lostBy: []string{"probe", "probe-untrusted"}, wantArgs: []string{"mcp", "status"}},
		{name: "unmarked"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := &cobra.Command{Use: "qsdev"}
			group := &cobra.Command{Use: "mcp"}
			leaf := &cobra.Command{Use: "status [server]", Annotations: map[string]string{"other": "kept"}}
			root.AddCommand(group)
			group.AddCommand(leaf)
			if tt.mark {
				if got := MarkReadOnly(leaf, tt.viaFlag, tt.lostBy...); got != leaf {
					t.Fatal("MarkReadOnly did not return its command")
				}
			}

			args, ok := ReadOnlyArgs(leaf)
			if ok != tt.mark {
				t.Fatalf("ReadOnlyArgs ok = %v, want %v", ok, tt.mark)
			}
			if !slices.Equal(args, tt.wantArgs) {
				t.Errorf("ReadOnlyArgs = %q, want %q", args, tt.wantArgs)
			}
			if got := ReadOnlyLostBy(leaf); !slices.Equal(got, tt.lostBy) {
				t.Errorf("ReadOnlyLostBy = %q, want %q", got, tt.lostBy)
			}
			if leaf.Annotations["other"] != "kept" {
				t.Error("MarkReadOnly dropped an existing annotation")
			}
		})
	}
}

func TestMarkReadOnly_NilAnnotations(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "qsdev"}
	leaf := &cobra.Command{Use: "status"}
	root.AddCommand(leaf)
	MarkReadOnly(leaf, "")
	if args, ok := ReadOnlyArgs(leaf); !ok || !slices.Equal(args, []string{"status"}) {
		t.Errorf("ReadOnlyArgs = %q, %v; want [status], true", args, ok)
	}
}
