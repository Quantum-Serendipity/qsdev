package posture

import (
	"math"
	"testing"
)

func TestWeightMultiplier(t *testing.T) {
	tests := []struct {
		weight LayerWeight
		want   float64
	}{
		{WeightCritical, 10.0},
		{WeightHigh, 7.5},
		{WeightMedium, 5.0},
		{WeightLow, 2.5},
		{LayerWeight("unknown"), 0.0},
	}
	for _, tt := range tests {
		got := WeightMultiplier(tt.weight)
		if got != tt.want {
			t.Errorf("WeightMultiplier(%s) = %f, want %f", tt.weight, got, tt.want)
		}
	}
}

func TestScoreToGrade_BoundaryValues(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		// A+ boundary
		{"97.0 -> A+", 97.0, "A+"},
		{"96.5 -> A+ (rounds to 97)", 96.5, "A+"},
		{"96.4 -> A (rounds to 96)", 96.4, "A"},
		{"100 -> A+", 100.0, "A+"},

		// A boundary
		{"93.0 -> A", 93.0, "A"},
		{"92.5 -> A (rounds to 93)", 92.5, "A"},
		{"92.4 -> A- (rounds to 92)", 92.4, "A-"},

		// A- boundary
		{"90.0 -> A-", 90.0, "A-"},
		{"89.5 -> A- (rounds to 90)", 89.5, "A-"},
		{"89.4 -> B+ (rounds to 89)", 89.4, "B+"},

		// B+ boundary
		{"87.0 -> B+", 87.0, "B+"},
		{"86.5 -> B+ (rounds to 87)", 86.5, "B+"},
		{"86.4 -> B (rounds to 86)", 86.4, "B"},

		// B boundary
		{"83.0 -> B", 83.0, "B"},
		{"82.5 -> B (rounds to 83)", 82.5, "B"},
		{"82.4 -> B- (rounds to 82)", 82.4, "B-"},

		// B- boundary
		{"80.0 -> B-", 80.0, "B-"},
		{"79.5 -> B- (rounds to 80)", 79.5, "B-"},
		{"79.4 -> C+ (rounds to 79)", 79.4, "C+"},

		// C+ boundary
		{"77.0 -> C+", 77.0, "C+"},
		{"76.5 -> C+ (rounds to 77)", 76.5, "C+"},
		{"76.4 -> C (rounds to 76)", 76.4, "C"},

		// C boundary
		{"73.0 -> C", 73.0, "C"},

		// C- boundary
		{"70.0 -> C-", 70.0, "C-"},
		{"69.5 -> C- (rounds to 70)", 69.5, "C-"},
		{"69.4 -> D+ (rounds to 69)", 69.4, "D+"},

		// D+ boundary
		{"67.0 -> D+", 67.0, "D+"},

		// D boundary
		{"63.0 -> D", 63.0, "D"},

		// D- boundary
		{"60.0 -> D-", 60.0, "D-"},
		{"59.5 -> D- (rounds to 60)", 59.5, "D-"},
		{"59.4 -> F (rounds to 59)", 59.4, "F"},

		// F
		{"0 -> F", 0.0, "F"},
		{"50 -> F", 50.0, "F"},
		{"59 -> F", 59.0, "F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreToGrade(tt.score)
			if got != tt.want {
				t.Errorf("ScoreToGrade(%f) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestComputeDefenseScore_AllEnabled(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerEnabled},
		{Name: "b", Weight: WeightHigh, Status: LayerEnabled},
		{Name: "c", Weight: WeightMedium, Status: LayerEnabled},
		{Name: "d", Weight: WeightLow, Status: LayerEnabled},
	}
	got := ComputeDefenseScore(layers)
	if got != 100.0 {
		t.Errorf("all enabled: got %f, want 100.0", got)
	}
}

func TestComputeDefenseScore_AllDisabled(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerDisabled},
		{Name: "b", Weight: WeightHigh, Status: LayerDisabled},
	}
	got := ComputeDefenseScore(layers)
	if got != 0.0 {
		t.Errorf("all disabled: got %f, want 0.0", got)
	}
}

func TestComputeDefenseScore_AllNotApplicable(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerNotApplicable},
		{Name: "b", Weight: WeightHigh, Status: LayerNotApplicable},
	}
	got := ComputeDefenseScore(layers)
	if got != 100.0 {
		t.Errorf("all not-applicable: got %f, want 100.0", got)
	}
}

func TestComputeDefenseScore_EmptyLayers(t *testing.T) {
	got := ComputeDefenseScore(nil)
	if got != 100.0 {
		t.Errorf("empty layers: got %f, want 100.0", got)
	}
}

func TestComputeDefenseScore_MixedStatus(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerEnabled},     // 10/10
		{Name: "b", Weight: WeightHigh, Status: LayerDisabled},        // 0/7.5
		{Name: "c", Weight: WeightMedium, Status: LayerNotApplicable}, // excluded
	}
	// totalWeight = 10 + 7.5 = 17.5
	// earnedWeight = 10
	// score = 10/17.5 * 100 = 57.142857...
	got := ComputeDefenseScore(layers)
	want := (10.0 / 17.5) * 100.0
	if math.Abs(got-want) > 0.001 {
		t.Errorf("mixed: got %f, want %f", got, want)
	}
}

func TestComputeDefenseScore_Partial(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightHigh, Status: LayerPartial, Score: 5},
	}
	// totalWeight = 7.5
	// earnedWeight = 7.5 * 5/10 = 3.75
	// score = 3.75 / 7.5 * 100 = 50
	got := ComputeDefenseScore(layers)
	if got != 50.0 {
		t.Errorf("partial (5/10): got %f, want 50.0", got)
	}
}

func TestComputeDefenseScore_PartialFullScore(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightMedium, Status: LayerPartial, Score: 10},
	}
	got := ComputeDefenseScore(layers)
	if got != 100.0 {
		t.Errorf("partial (10/10): got %f, want 100.0", got)
	}
}

func TestComputeDefenseScore_PartialZeroScore(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightMedium, Status: LayerPartial, Score: 0},
	}
	got := ComputeDefenseScore(layers)
	if got != 0.0 {
		t.Errorf("partial (0/10): got %f, want 0.0", got)
	}
}

func TestComputeAggregateScore_40_30_30(t *testing.T) {
	// 100 * 0.40 + 80 * 0.30 + 60 * 0.30 = 40 + 24 + 18 = 82
	agg := ComputeAggregateScore(100, 80, 60)
	if agg.Total != 82.0 {
		t.Errorf("total: got %f, want 82.0", agg.Total)
	}
	if agg.Grade != "B-" {
		t.Errorf("grade: got %q, want %q", agg.Grade, "B-")
	}
	if agg.Defense != 100.0 {
		t.Errorf("defense: got %f, want 100.0", agg.Defense)
	}
	if agg.Config != 80.0 {
		t.Errorf("config: got %f, want 80.0", agg.Config)
	}
	if agg.DepHealth != 60.0 {
		t.Errorf("deps: got %f, want 60.0", agg.DepHealth)
	}
}

func TestComputeAggregateScore_AllZeros(t *testing.T) {
	agg := ComputeAggregateScore(0, 0, 0)
	if agg.Total != 0.0 {
		t.Errorf("total: got %f, want 0.0", agg.Total)
	}
	if agg.Grade != "F" {
		t.Errorf("grade: got %q, want %q", agg.Grade, "F")
	}
}

func TestComputeAggregateScore_AllHundred(t *testing.T) {
	agg := ComputeAggregateScore(100, 100, 100)
	if agg.Total != 100.0 {
		t.Errorf("total: got %f, want 100.0", agg.Total)
	}
	if agg.Grade != "A+" {
		t.Errorf("grade: got %q, want %q", agg.Grade, "A+")
	}
}

func TestComputeAggregateScore_Rounding(t *testing.T) {
	// 89.5 * 0.40 + 89.5 * 0.30 + 89.5 * 0.30 = 89.5
	agg := ComputeAggregateScore(89.5, 89.5, 89.5)
	if agg.Total != 89.5 {
		t.Errorf("total: got %f, want 89.5", agg.Total)
	}
	if agg.Grade != "A-" {
		t.Errorf("grade at 89.5: got %q, want %q", agg.Grade, "A-")
	}
}

func TestComputeAggregateScore_DefenseHeavy(t *testing.T) {
	// defense=50, config=100, deps=100 => 50*0.4 + 100*0.3 + 100*0.3 = 20 + 30 + 30 = 80
	agg := ComputeAggregateScore(50, 100, 100)
	if agg.Total != 80.0 {
		t.Errorf("total: got %f, want 80.0", agg.Total)
	}
	if agg.Grade != "B-" {
		t.Errorf("grade: got %q, want %q", agg.Grade, "B-")
	}
}

func TestScoreToGrade_Determinism(t *testing.T) {
	// Run the same score 100 times, ensure same grade every time.
	for i := 0; i < 100; i++ {
		got := ScoreToGrade(89.5)
		if got != "A-" {
			t.Fatalf("iteration %d: ScoreToGrade(89.5) = %q, want A-", i, got)
		}
	}
}

func TestComputeDefenseScore_Determinism(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerEnabled},
		{Name: "b", Weight: WeightHigh, Status: LayerPartial, Score: 7},
		{Name: "c", Weight: WeightMedium, Status: LayerDisabled},
		{Name: "d", Weight: WeightLow, Status: LayerNotApplicable},
	}
	first := ComputeDefenseScore(layers)
	for i := 0; i < 100; i++ {
		got := ComputeDefenseScore(layers)
		if got != first {
			t.Fatalf("iteration %d: non-deterministic result %f != %f", i, got, first)
		}
	}
}

func TestComputeAggregateScore_RoundingEdge(t *testing.T) {
	// Test that Total is rounded to 1 decimal place.
	// defense=89.33, config=91.67, deps=85.11
	// 89.33*0.40 + 91.67*0.30 + 85.11*0.30
	// = 35.732 + 27.501 + 25.533 = 88.766
	// Round to 1 decimal = 88.8
	agg := ComputeAggregateScore(89.33, 91.67, 85.11)
	if agg.Total != 88.8 {
		t.Errorf("total: got %f, want 88.8", agg.Total)
	}
}

func TestScoreToGrade_NegativeScore(t *testing.T) {
	got := ScoreToGrade(-5.0)
	if got != "F" {
		t.Errorf("ScoreToGrade(-5) = %q, want F", got)
	}
}

func TestScoreToGrade_OverHundred(t *testing.T) {
	got := ScoreToGrade(105.0)
	if got != "A+" {
		t.Errorf("ScoreToGrade(105) = %q, want A+", got)
	}
}

func TestComputeDefenseScore_MixedPartialAndNA(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerPartial, Score: 8},
		{Name: "b", Weight: WeightHigh, Status: LayerNotApplicable},
		{Name: "c", Weight: WeightMedium, Status: LayerEnabled},
		{Name: "d", Weight: WeightLow, Status: LayerDisabled},
	}
	// totalWeight: 10 + 5 + 2.5 = 17.5 (high excluded)
	// earned: 10 * 8/10 + 5 + 0 = 8 + 5 = 13
	// score: 13 / 17.5 * 100 = 74.285...
	got := ComputeDefenseScore(layers)
	want := (13.0 / 17.5) * 100.0
	if math.Abs(got-want) > 0.001 {
		t.Errorf("mixed partial+NA: got %f, want %f", got, want)
	}
}

func TestComputeDefenseScore_AllLayerWeights(t *testing.T) {
	// One of each weight, all enabled
	layers := []DefenseLayer{
		{Name: "critical", Weight: WeightCritical, Status: LayerEnabled},
		{Name: "high", Weight: WeightHigh, Status: LayerEnabled},
		{Name: "medium", Weight: WeightMedium, Status: LayerEnabled},
		{Name: "low", Weight: WeightLow, Status: LayerEnabled},
	}
	got := ComputeDefenseScore(layers)
	if got != 100.0 {
		t.Errorf("all weights enabled: got %f, want 100.0", got)
	}
}

func TestComputeDefenseScore_SinglePartialLayer(t *testing.T) {
	layers := []DefenseLayer{
		{Name: "a", Weight: WeightCritical, Status: LayerPartial, Score: 3},
	}
	// totalWeight = 10, earned = 10 * 3/10 = 3
	// score = 3/10 * 100 = 30
	got := ComputeDefenseScore(layers)
	if got != 30.0 {
		t.Errorf("single partial: got %f, want 30.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_T1NotPenalizedForT3(t *testing.T) {
	t.Parallel()
	// T1 user has all T1 layers enabled but T2/T3 layers disabled.
	// The score should be 100 because T2/T3 layers are excluded.
	layers := []DefenseLayer{
		{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled, MinTier: 1},
		{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled, MinTier: 1},
		{Name: "lock-file-enforcement", Weight: WeightHigh, Status: LayerEnabled, MinTier: 1},
		{Name: "vulnerability-scanning", Weight: WeightHigh, Status: LayerEnabled, MinTier: 1},
		{Name: "age-gating", Weight: WeightHigh, Status: LayerDisabled, MinTier: 2},
		{Name: "secrets-scanning", Weight: WeightMedium, Status: LayerDisabled, MinTier: 2},
		{Name: "sast", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
		{Name: "nix-hardening", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
		{Name: "container-security", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
		{Name: "license-compliance", Weight: WeightLow, Status: LayerDisabled, MinTier: 3},
	}

	got := ComputeTierRelativeDefenseScore(layers, 1)
	if got != 100.0 {
		t.Errorf("T1 with all T1 layers enabled: got %f, want 100.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_T2IncludesT1AndT2(t *testing.T) {
	t.Parallel()
	layers := []DefenseLayer{
		{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled, MinTier: 1},
		{Name: "install-script-blocking", Weight: WeightHigh, Status: LayerEnabled, MinTier: 1},
		{Name: "age-gating", Weight: WeightHigh, Status: LayerDisabled, MinTier: 2},
		{Name: "sast", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
	}

	got := ComputeTierRelativeDefenseScore(layers, 2)
	// T2 includes T1 (pretooluse + install-script) and T2 (age-gating).
	// T3 (sast) excluded.
	// totalWeight = 10 + 7.5 + 7.5 = 25
	// earned = 10 + 7.5 + 0 = 17.5
	// score = 17.5 / 25 * 100 = 70
	if got != 70.0 {
		t.Errorf("T2 mixed: got %f, want 70.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_T3IncludesAll(t *testing.T) {
	t.Parallel()
	layers := []DefenseLayer{
		{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled, MinTier: 1},
		{Name: "sast", Weight: WeightMedium, Status: LayerEnabled, MinTier: 3},
	}

	got := ComputeTierRelativeDefenseScore(layers, 3)
	if got != 100.0 {
		t.Errorf("T3 all enabled: got %f, want 100.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_EmptyLayers(t *testing.T) {
	t.Parallel()
	got := ComputeTierRelativeDefenseScore(nil, 1)
	if got != 100.0 {
		t.Errorf("empty layers: got %f, want 100.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_AllAboveTier(t *testing.T) {
	t.Parallel()
	// All layers are T3, but user is T1 — nothing counts.
	layers := []DefenseLayer{
		{Name: "sast", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
		{Name: "nix-hardening", Weight: WeightMedium, Status: LayerDisabled, MinTier: 3},
	}
	got := ComputeTierRelativeDefenseScore(layers, 1)
	if got != 100.0 {
		t.Errorf("all above tier: got %f, want 100.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_NAExcluded(t *testing.T) {
	t.Parallel()
	layers := []DefenseLayer{
		{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerEnabled, MinTier: 1},
		{Name: "container-security", Weight: WeightMedium, Status: LayerNotApplicable, MinTier: 3},
	}
	// At T3, container-security is in scope but NA, so excluded.
	got := ComputeTierRelativeDefenseScore(layers, 3)
	if got != 100.0 {
		t.Errorf("NA excluded at T3: got %f, want 100.0", got)
	}
}

func TestComputeTierRelativeDefenseScore_PartialScoring(t *testing.T) {
	t.Parallel()
	layers := []DefenseLayer{
		{Name: "pretooluse-hooks", Weight: WeightCritical, Status: LayerPartial, Score: 5, MinTier: 1},
	}
	// totalWeight = 10, earned = 10 * 5/10 = 5
	// score = 5/10 * 100 = 50
	got := ComputeTierRelativeDefenseScore(layers, 1)
	if got != 50.0 {
		t.Errorf("partial scoring: got %f, want 50.0", got)
	}
}
