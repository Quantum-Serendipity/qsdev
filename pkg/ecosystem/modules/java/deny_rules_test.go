package java_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: goals that fetch an arbitrary artifact named on the
// command line are denied; ordinary build goals stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&java.Module{}).DenyRules(ecosystem.ModuleConfig{PackageManager: "both"})
	for _, cmd := range []string{
		"mvn dependency:get -Dartifact=evil:pkg:1.0",
		"mvn dependency:copy -Dartifact=evil:pkg:1.0 -DoutputDirectory=lib",
		"mvn dependency:get",
		"mvn -q dependency:get -Dartifact=evil:pkg:1.0",
		"mvn org.apache.maven.plugins:maven-dependency-plugin:3.6.1:get -Dartifact=evil:pkg:1.0",
		"./mvnw dependency:get -Dartifact=evil:pkg:1.0",
		"./mvnw -B dependency:copy -Dartifact=evil:pkg:1.0",
		"mvn dependency:resolve -U",
		"./gradlew dependencies --configuration runtimeClasspath",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"mvn verify",
		"mvn package",
		"mvn dependency:tree",
		"./mvnw verify",
		"./gradlew build",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
