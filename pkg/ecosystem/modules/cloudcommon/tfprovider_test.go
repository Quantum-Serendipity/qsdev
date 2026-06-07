package cloudcommon

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTFFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing tf file: %v", err)
	}
}

func TestDetectTerraformProviders_AWS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTFFile(t, dir, "main.tf", `
provider "aws" {
  region = "us-east-1"
}
`)

	result := DetectTerraformProviders(dir)
	if !result["aws"] {
		t.Error("expected aws provider to be detected")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 provider, got %d", len(result))
	}
}

func TestDetectTerraformProviders_RequiredProviders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTFFile(t, dir, "versions.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`)

	result := DetectTerraformProviders(dir)
	if !result["aws"] {
		t.Error("expected aws provider to be detected via required_providers")
	}
}

func TestDetectTerraformProviders_MultiCloud(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTFFile(t, dir, "providers.tf", `
provider "aws" {
  region = "us-east-1"
}

provider "google" {
  project = "my-project"
  region  = "us-central1"
}

provider "azurerm" {
  features {}
}
`)

	result := DetectTerraformProviders(dir)
	if !result["aws"] {
		t.Error("expected aws provider to be detected")
	}
	if !result["google"] {
		t.Error("expected google provider to be detected")
	}
	if !result["azurerm"] {
		t.Error("expected azurerm provider to be detected")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 providers, got %d", len(result))
	}
}

func TestDetectTerraformProviders_NoProviders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTFFile(t, dir, "main.tf", `
resource "null_resource" "example" {
  triggers = {
    always_run = timestamp()
  }
}
`)

	result := DetectTerraformProviders(dir)
	if len(result) != 0 {
		t.Errorf("expected 0 providers, got %d", len(result))
	}
}

func TestDetectTerraformProviders_NoTFFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result := DetectTerraformProviders(dir)
	if len(result) != 0 {
		t.Errorf("expected 0 providers, got %d", len(result))
	}
}
