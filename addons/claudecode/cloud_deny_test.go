package claudecode

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

func TestCloudDenyRulesNoSkillConflicts(t *testing.T) {
	t.Parallel()

	allCloudRules := cloudcommon.AllBashDenyRules([]cloudcommon.CloudProvider{
		cloudcommon.AWS, cloudcommon.GCP, cloudcommon.Azure,
	})
	skills := BuiltinSkillDefinitions()

	conflicts := ValidateDenyRuleConflicts(allCloudRules, skills)
	unexpected := FilterExpectedConflicts(conflicts)

	if len(unexpected) > 0 {
		for _, c := range unexpected {
			t.Errorf("unexpected conflict: %s", c.Message)
		}
	}
}

func TestCloudDenyRulesSorted(t *testing.T) {
	t.Parallel()

	rules := cloudcommon.AllBashDenyRules([]cloudcommon.CloudProvider{
		cloudcommon.AWS, cloudcommon.GCP, cloudcommon.Azure,
	})
	for i := 1; i < len(rules); i++ {
		if rules[i] < rules[i-1] {
			t.Errorf("rules not sorted: %q before %q", rules[i-1], rules[i])
		}
	}
}
