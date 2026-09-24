package claudecode

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

// newLifecycleRoot builds a root command with a docs subtree and recording
// stand-ins for the top-level enable/disable lifecycle commands.
func newLifecycleRoot(calls map[string][]string, failTool string) *cobra.Command {
	root := &cobra.Command{Use: "qsdev"}
	for _, verb := range []string{"enable", "disable"} {
		root.AddCommand(&cobra.Command{
			Use: verb + " <tool>",
			RunE: func(_ *cobra.Command, args []string) error {
				calls[verb] = append(calls[verb], args[0])
				if args[0] == failTool {
					return errors.New("boom")
				}
				return nil
			},
		})
	}
	root.AddCommand(docsCmd())
	return root
}

// TestDocsEnableDisable_DelegateToToolLifecycle pins F099: docs enable/disable
// must change the project through the tool lifecycle instead of only printing
// the list of servers.
func TestDocsEnableDisable_DelegateToToolLifecycle(t *testing.T) {
	t.Parallel()
	allTools, err := docServerTools(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(allTools, "local-docs-devdocs") || !slices.Contains(allTools, "local-docs-zim") {
		t.Fatalf("documentation tools = %v, want the catalog's local-docs tools", allTools)
	}

	tests := []struct {
		name      string
		args      []string
		failTool  string
		wantVerb  string
		wantTools []string
		wantErr   bool
	}{
		{name: "enable all", args: []string{"docs", "enable"}, wantVerb: "enable", wantTools: allTools},
		{name: "disable all", args: []string{"docs", "disable"}, wantVerb: "disable", wantTools: allTools},
		{name: "enable one server", args: []string{"docs", "enable", "local-docs-zim"}, wantVerb: "enable", wantTools: []string{"local-docs-zim"}},
		{name: "unknown server", args: []string{"docs", "enable", "nope"}, wantVerb: "enable", wantErr: true},
		{name: "failure is reported after trying every tool", args: []string{"docs", "enable"}, failTool: "local-docs-devdocs", wantVerb: "enable", wantTools: allTools, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := map[string][]string{}
			root := newLifecycleRoot(calls, tt.failTool)
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(tt.args)

			err := root.Execute()
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v\n%s", err, tt.wantErr, out.String())
			}
			if !slices.Equal(calls[tt.wantVerb], tt.wantTools) {
				t.Errorf("%s called for %v, want %v", tt.wantVerb, calls[tt.wantVerb], tt.wantTools)
			}
		})
	}
}

func TestDocsEnable_WithoutLifecycleCommandFails(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "qsdev"}
	root.AddCommand(docsCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"docs", "enable"})
	if err := root.Execute(); err == nil {
		t.Fatal("docs enable succeeded without a tool lifecycle to delegate to")
	}
}
