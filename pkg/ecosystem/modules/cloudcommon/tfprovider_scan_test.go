package cloudcommon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTerraformProviders_Layouts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  []string
		skip  []string
	}{
		{
			name: "root backend plus aws module",
			files: map[string]string{
				"main.tf":            `terraform { backend "s3" {} }`,
				"modules/aws/aws.tf": `provider "aws" {}`,
			},
			want: []string{"aws"},
		},
		{
			name: "nested module three levels down",
			files: map[string]string{
				"main.tf":                        `terraform {}`,
				"infra/modules/gcp/providers.tf": `provider "google" {}`,
			},
			want: []string{"google"},
		},
		{
			name: "fully qualified registry source",
			files: map[string]string{
				"versions.tf": "terraform {\n  required_providers {\n    azurerm = {\n      source = \"registry.terraform.io/hashicorp/azurerm\"\n    }\n  }\n}\n",
			},
			want: []string{"azurerm"},
		},
		{
			name: "opentofu registry source",
			files: map[string]string{
				"versions.tf": "terraform {\n  required_providers {\n    aws = { source = \"registry.opentofu.org/hashicorp/aws\" }\n  }\n}\n",
			},
			want: []string{"aws"},
		},
		{
			name: "provider cache and vendored trees ignored",
			files: map[string]string{
				"main.tf":                      `terraform {}`,
				".terraform/modules/x/main.tf": `provider "aws" {}`,
				"vendor/lib/main.tf":           `provider "google" {}`,
			},
			skip: []string{"aws", "google"},
		},
		{
			name: "too deep is ignored",
			files: map[string]string{
				"a/b/c/d/main.tf": `provider "aws" {}`,
			},
			skip: []string{"aws"},
		},
		{
			name: "google-beta provider block (W136)",
			files: map[string]string{
				"main.tf": "provider \"google-beta\" {\n  project = \"p\"\n}\n",
			},
			want: []string{"google-beta"},
			skip: []string{"google"},
		},
		{
			name: "google-beta required provider source (W136)",
			files: map[string]string{
				"versions.tf": "terraform {\n  required_providers {\n    google-beta = { source = \"hashicorp/google-beta\" }\n  }\n}\n",
			},
			want: []string{"google-beta"},
		},
		{
			name: "opentofu .tofu files are scanned (W128)",
			files: map[string]string{
				"infra/main.tofu": `provider "azurerm" {}`,
			},
			want: []string{"azurerm"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for rel, content := range tt.files {
				full := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				writeTFFile(t, filepath.Dir(full), filepath.Base(full), content)
			}
			got := DetectTerraformProviders(dir)
			for _, p := range tt.want {
				if !got[p] {
					t.Errorf("provider %q not detected; got %v", p, got)
				}
			}
			for _, p := range tt.skip {
				if got[p] {
					t.Errorf("provider %q detected from an excluded location; got %v", p, got)
				}
			}
		})
	}
}
