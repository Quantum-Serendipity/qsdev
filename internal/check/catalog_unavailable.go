package check

// RunCatalogUnavailable runs the checks for a project whose catalog or tool
// registry could not load: failure (CatalogLoadFailure or
// ToolRegistryFailure), followed by every check that needs nothing from the
// catalog. The rest, and config integrity itself, validate against catalog
// data and cannot run.
func RunCatalogUnavailable(ctx CheckContext, failure CheckResult) *CheckReport {
	results := []CheckResult{failure}
	results = append(results, CheckBinaryCompatibility(ctx)...)
	results = append(results, CheckOrgOverlay(ctx))
	results = append(results, CheckFileState(ctx)...)

	return BuildReport(results, ctx.BinaryVersion, projectName(ctx))
}
