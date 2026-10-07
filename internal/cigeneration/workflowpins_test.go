package cigeneration

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWorkflowPins(t *testing.T) {
	t.Parallel()

	const (
		shaA = "de0fac2e4500dabe0009e67214ff5f5447ce83dd"
		shaB = "6e4298ebc4db23e847df9b2e2de2939d6f066c67"
	)

	tests := []struct {
		name    string
		files   map[string]string
		want    map[string]WorkflowPin
		wantErr string
	}{
		{
			name: "subpath action keeps its full path",
			files: map[string]string{
				"scan.yml": "steps:\n  - uses: google/osv-scanner-action/osv-scanner-action@" + shaB + " # v2.5.1\n",
			},
			want: map[string]WorkflowPin{
				"google/osv-scanner-action/osv-scanner-action": {SHA: shaB, Tag: "v2.5.1", File: "scan.yml"},
			},
		},
		{
			name: "missing tag comment is an error",
			files: map[string]string{
				"ci.yml": "steps:\n  - uses: actions/checkout@" + shaA + "\n",
			},
			wantErr: "ci.yml:2",
		},
		{
			name: "non-.yml files are ignored",
			files: map[string]string{
				"ci.yml":     "  - uses: actions/checkout@" + shaA + " # v6.0.2\n",
				"notes.md":   "  - uses: actions/checkout@" + shaB + "\n",
				"ci.yaml~":   "  - uses: actions/checkout@" + shaB + " # v1\n",
				"README.txt": "uses: other/action@" + shaB + " # v9\n",
			},
			want: map[string]WorkflowPin{
				"actions/checkout": {SHA: shaA, Tag: "v6.0.2", File: "ci.yml"},
			},
		},
		{
			name: ".yaml workflows are parsed too",
			files: map[string]string{
				"ci.yaml": "  - uses: actions/checkout@" + shaA + " # v6.0.2\n",
			},
			want: map[string]WorkflowPin{
				"actions/checkout": {SHA: shaA, Tag: "v6.0.2", File: "ci.yaml"},
			},
		},
		{
			name: "CRLF line endings do not leak into the tag",
			files: map[string]string{
				"ci.yml": "steps:\r\n  - uses: actions/checkout@" + shaA + " # v6.0.2\r\n  - run: true\r\n",
			},
			want: map[string]WorkflowPin{
				"actions/checkout": {SHA: shaA, Tag: "v6.0.2", File: "ci.yml"},
			},
		},
		{
			name: "same pin in two files agrees",
			files: map[string]string{
				"a.yml": "  - uses: actions/checkout@" + shaA + " # v6.0.2\n",
				"b.yml": "  - uses: actions/checkout@" + shaA + " # v6.0.2\n",
			},
			want: map[string]WorkflowPin{
				"actions/checkout": {SHA: shaA, Tag: "v6.0.2", File: "a.yml"},
			},
		},
		{
			name: "conflicting pins for one action are an error",
			files: map[string]string{
				"a.yml": "  - uses: actions/checkout@" + shaA + " # v6.0.2\n",
				"b.yml": "  - uses: actions/checkout@" + shaB + " # v6.1.0\n",
			},
			wantErr: "actions/checkout",
		},
		{
			name: "unpinned and local actions are not pins",
			files: map[string]string{
				"ci.yml": "  - uses: actions/checkout@v4\n  - uses: ./local\n  - uses: docker://alpine@sha256:" + strings.Repeat("a", 64) + "\n",
			},
			want: map[string]WorkflowPin{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for name, body := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := ParseWorkflowPins(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseWorkflowPins() error = %v, want one mentioning %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWorkflowPins() error = %v", err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("ParseWorkflowPins() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWorkflowPinsMissingDir(t *testing.T) {
	t.Parallel()

	if _, err := ParseWorkflowPins(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("ParseWorkflowPins() on a missing directory returned no error")
	}
}
