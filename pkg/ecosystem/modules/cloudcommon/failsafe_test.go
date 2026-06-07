package cloudcommon

import "testing"

func TestValidateFailSafe_AllActive(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	denyRules := BashDenyRules(AWS)
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if !report.AllLayersActive {
		t.Error("expected AllLayersActive to be true")
	}
	if len(report.Statuses) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(report.Statuses))
	}
	for _, s := range report.Statuses {
		if !s.Active {
			t.Errorf("expected layer %d to be active, got details: %s", s.Layer, s.Details)
		}
	}
}

func TestValidateFailSafe_Layer1Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{} // No AWS_PROFILE set.
	denyRules := BashDenyRules(AWS)
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when env var missing")
	}

	layer1 := report.Statuses[0]
	if layer1.Layer != LayerEnvironmentSeparation {
		t.Errorf("expected first status to be LayerEnvironmentSeparation, got %d", layer1.Layer)
	}
	if layer1.Active {
		t.Error("expected Layer1 to be inactive")
	}

	// Layers 2 and 3 should still be active.
	if !report.Statuses[1].Active {
		t.Error("expected Layer2 to be active")
	}
	if !report.Statuses[2].Active {
		t.Error("expected Layer3 to be active")
	}
}

func TestValidateFailSafe_Layer2Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	denyRules := BashDenyRules(AWS)
	// Provide only the first ReadDeny path, omitting the rest.
	readDeny := ReadDenyPaths(AWS)[:1]

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when ReadDeny paths missing")
	}

	layer2 := report.Statuses[1]
	if layer2.Layer != LayerCredentialFileMasking {
		t.Errorf("expected second status to be LayerCredentialFileMasking, got %d", layer2.Layer)
	}
	if layer2.Active {
		t.Error("expected Layer2 to be inactive")
	}

	// Layers 1 and 3 should still be active.
	if !report.Statuses[0].Active {
		t.Error("expected Layer1 to be active")
	}
	if !report.Statuses[2].Active {
		t.Error("expected Layer3 to be active")
	}
}

func TestValidateFailSafe_Layer3Missing(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
	}
	// Provide only the first deny rule, omitting the rest.
	denyRules := BashDenyRules(AWS)[:1]
	readDeny := ReadDenyPaths(AWS)

	report := ValidateFailSafe(AWS, envVars, denyRules, readDeny)

	if report.AllLayersActive {
		t.Error("expected AllLayersActive to be false when deny rules missing")
	}

	layer3 := report.Statuses[2]
	if layer3.Layer != LayerAgentDenyRules {
		t.Errorf("expected third status to be LayerAgentDenyRules, got %d", layer3.Layer)
	}
	if layer3.Active {
		t.Error("expected Layer3 to be inactive")
	}

	// Layers 1 and 2 should still be active.
	if !report.Statuses[0].Active {
		t.Error("expected Layer1 to be active")
	}
	if !report.Statuses[1].Active {
		t.Error("expected Layer2 to be active")
	}
}

func TestValidateAllProviders_MultiProviderPartialDegradation(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"AWS_PROFILE": "dev-project",
		// GCP env var intentionally missing.
	}

	// Combine all deny rules and read-deny paths for both providers.
	allDenyRules := append(BashDenyRules(AWS), BashDenyRules(GCP)...)
	allReadDeny := append(ReadDenyPaths(AWS), ReadDenyPaths(GCP)...)

	reports := ValidateAllProviders([]CloudProvider{AWS, GCP}, envVars, allDenyRules, allReadDeny)

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	// AWS should be fully active.
	awsReport := reports[0]
	if awsReport.Provider != AWS {
		t.Errorf("expected first report to be AWS, got %s", awsReport.Provider)
	}
	if !awsReport.AllLayersActive {
		t.Error("expected AWS AllLayersActive to be true")
	}

	// GCP should be degraded (missing env var).
	gcpReport := reports[1]
	if gcpReport.Provider != GCP {
		t.Errorf("expected second report to be GCP, got %s", gcpReport.Provider)
	}
	if gcpReport.AllLayersActive {
		t.Error("expected GCP AllLayersActive to be false (missing env var)")
	}

	// GCP Layer 1 should be inactive, Layers 2 and 3 should be active.
	if gcpReport.Statuses[0].Active {
		t.Error("expected GCP Layer1 to be inactive")
	}
	if !gcpReport.Statuses[1].Active {
		t.Error("expected GCP Layer2 to be active")
	}
	if !gcpReport.Statuses[2].Active {
		t.Error("expected GCP Layer3 to be active")
	}
}
