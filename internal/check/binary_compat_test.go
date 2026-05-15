package check

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckBinaryCompatibility_VersionSatisfied(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "1.5.0",
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			QsdevVersion:   ">=1.0.0",
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var versionResult *CheckResult
	for i := range results {
		if results[i].Name == "qsdev_version_constraint" {
			versionResult = &results[i]
			break
		}
	}

	if versionResult == nil {
		t.Fatal("expected qsdev_version_constraint result")
	}
	if versionResult.Status != StatusPass {
		t.Errorf("Status = %s, want %s; Message: %s", versionResult.Status, StatusPass, versionResult.Message)
	}
}

func TestCheckBinaryCompatibility_VersionNotSatisfied(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "0.9.0",
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			QsdevVersion:   ">=1.0.0",
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var versionResult *CheckResult
	for i := range results {
		if results[i].Name == "qsdev_version_constraint" {
			versionResult = &results[i]
			break
		}
	}

	if versionResult == nil {
		t.Fatal("expected qsdev_version_constraint result")
	}
	if versionResult.Status != StatusFail {
		t.Errorf("Status = %s, want %s", versionResult.Status, StatusFail)
	}
	if versionResult.Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", versionResult.Severity, SeverityCritical)
	}
}

func TestCheckBinaryCompatibility_NoConstraint(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "1.0.0",
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var versionResult *CheckResult
	for i := range results {
		if results[i].Name == "qsdev_version_constraint" {
			versionResult = &results[i]
			break
		}
	}

	if versionResult == nil {
		t.Fatal("expected qsdev_version_constraint result")
	}
	if versionResult.Status != StatusPass {
		t.Errorf("Status = %s, want %s", versionResult.Status, StatusPass)
	}
	if versionResult.Severity != SeverityInfo {
		t.Errorf("Severity = %s, want %s", versionResult.Severity, SeverityInfo)
	}
}

func TestCheckBinaryCompatibility_NoConfig(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "1.0.0",
	}

	results := CheckBinaryCompatibility(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckBinaryCompatibility_UnsupportedSchemaVersion(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "1.0.0",
		QsdevConfig: &types.QsdevConfig{
			Version: 99,
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var schemaResult *CheckResult
	for i := range results {
		if results[i].Name == "config_schema_version" {
			schemaResult = &results[i]
			break
		}
	}

	if schemaResult == nil {
		t.Fatal("expected config_schema_version result")
	}
	if schemaResult.Status != StatusFail {
		t.Errorf("Status = %s, want %s", schemaResult.Status, StatusFail)
	}
	if schemaResult.Severity != SeverityCritical {
		t.Errorf("Severity = %s, want %s", schemaResult.Severity, SeverityCritical)
	}
}

func TestCheckBinaryCompatibility_SupportedSchemaVersion(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "1.0.0",
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var schemaResult *CheckResult
	for i := range results {
		if results[i].Name == "config_schema_version" {
			schemaResult = &results[i]
			break
		}
	}

	if schemaResult == nil {
		t.Fatal("expected config_schema_version result")
	}
	if schemaResult.Status != StatusPass {
		t.Errorf("Status = %s, want %s", schemaResult.Status, StatusPass)
	}
}

func TestCheckBinaryCompatibility_DevVersion(t *testing.T) {
	ctx := CheckContext{
		BinaryVersion: "dev",
		QsdevConfig: &types.QsdevConfig{
			Version: 1,
			QsdevVersion:   ">=99.0.0",
		},
	}

	results := CheckBinaryCompatibility(ctx)

	var versionResult *CheckResult
	for i := range results {
		if results[i].Name == "qsdev_version_constraint" {
			versionResult = &results[i]
			break
		}
	}

	if versionResult == nil {
		t.Fatal("expected qsdev_version_constraint result")
	}
	// "dev" version always passes constraints.
	if versionResult.Status != StatusPass {
		t.Errorf("Status = %s, want %s; dev version should pass all constraints", versionResult.Status, StatusPass)
	}
}
