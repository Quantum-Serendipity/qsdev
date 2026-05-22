package check

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// supportedSchemaVersions lists the schema versions this binary understands.
var supportedSchemaVersions = []string{"1"}

// CheckBinaryCompatibility verifies that the qsdev binary version satisfies
// the constraint in .qsdev.yaml and that the config schema version is supported.
func CheckBinaryCompatibility(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return []CheckResult{
			{
				Category: CategoryBinaryCompat,
				Name:     branding.Get().AppName + "_version_constraint",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No " + branding.Get().ConfigFile + " found",
			},
		}
	}

	var results []CheckResult

	// Check qsdev_version constraint.
	results = append(results, checkVersionConstraint(ctx)...)

	// Check schema version.
	results = append(results, checkSchemaVersion(ctx)...)

	return results
}

func checkVersionConstraint(ctx CheckContext) []CheckResult {
	constraint := ctx.QsdevConfig.QsdevVersion
	if constraint == "" {
		return []CheckResult{
			{
				Category: CategoryBinaryCompat,
				Name:     branding.Get().AppName + "_version_constraint",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "No version constraint specified",
			},
		}
	}

	err := config.CheckBinaryVersion(constraint, ctx.BinaryVersion)
	if err != nil {
		return []CheckResult{
			{
				Category:    CategoryBinaryCompat,
				Name:        branding.Get().AppName + "_version_constraint",
				Status:      StatusFail,
				Severity:    SeverityCritical,
				Message:     err.Error(),
				Remediation: "Update " + branding.Get().AppName + " to a version satisfying " + constraint,
			},
		}
	}

	return []CheckResult{
		{
			Category: CategoryBinaryCompat,
			Name:     branding.Get().AppName + "_version_constraint",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "Binary version " + ctx.BinaryVersion + " satisfies " + constraint,
		},
	}
}

func checkSchemaVersion(ctx CheckContext) []CheckResult {
	sv := ctx.QsdevConfig.Version
	if sv == 0 {
		return []CheckResult{
			{
				Category:    CategoryBinaryCompat,
				Name:        "config_schema_version",
				Status:      StatusWarn,
				Severity:    SeverityMedium,
				Message:     "No version specified in " + branding.Get().ConfigFile,
				Remediation: "Add 'version: 1' to " + branding.Get().ConfigFile,
			},
		}
	}

	svStr := fmt.Sprintf("%d", sv)
	for _, supported := range supportedSchemaVersions {
		if svStr == supported {
			return []CheckResult{
				{
					Category: CategoryBinaryCompat,
					Name:     "config_schema_version",
					Status:   StatusPass,
					Severity: SeverityInfo,
					Message:  "Schema version " + svStr + " is supported",
				},
			}
		}
	}

	return []CheckResult{
		{
			Category:    CategoryBinaryCompat,
			Name:        "config_schema_version",
			Status:      StatusFail,
			Severity:    SeverityCritical,
			Message:     fmt.Sprintf("Schema version %d is not supported by this binary", sv),
			Remediation: "Update " + branding.Get().AppName + " or change schema_version to a supported version",
		},
	}
}
