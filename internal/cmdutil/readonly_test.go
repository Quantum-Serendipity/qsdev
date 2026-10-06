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

func TestReadOnlyInvocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mark    bool
		viaFlag string
		lostBy  []string
		args    []string
		want    bool
	}{
		{name: "unmarked", args: nil, want: false},
		{name: "always read-only", mark: true, want: true},
		{name: "via flag set", mark: true, viaFlag: "dry-run", args: []string{"--dry-run"}, want: true},
		{name: "via flag unset", mark: true, viaFlag: "dry-run", want: false},
		{name: "via flag set false", mark: true, viaFlag: "dry-run", args: []string{"--dry-run=false"}, want: false},
		{name: "lost by a flag", mark: true, lostBy: []string{"probe"}, args: []string{"--probe"}, want: false},
		{name: "lost-by flag absent", mark: true, lostBy: []string{"probe"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{Use: "status"}
			cmd.Flags().Bool("dry-run", false, "")
			cmd.Flags().Bool("probe", false, "")
			if tt.mark {
				MarkReadOnly(cmd, tt.viaFlag, tt.lostBy...)
			}
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatal(err)
			}
			if got := ReadOnlyInvocation(cmd); got != tt.want {
				t.Errorf("ReadOnlyInvocation(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
