package policy

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

func LoadPolicyFile(path string) (*SecurityPolicy, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loading policy %s: %w", path, err)
	}
	defer f.Close()

	var sp SecurityPolicy
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&sp); err != nil {
		return nil, fmt.Errorf("loading policy %s: %w", path, err)
	}

	if sp.APIVersion != "qsdev/v1" {
		return nil, fmt.Errorf("loading policy %s: unsupported apiVersion %q (expected \"qsdev/v1\")", path, sp.APIVersion)
	}
	if sp.Kind != "SecurityPolicy" {
		return nil, fmt.Errorf("loading policy %s: unsupported kind %q (expected \"SecurityPolicy\")", path, sp.Kind)
	}
	switch sp.Settings.FailMode {
	case "", FailClosed, FailOpen:
	default:
		return nil, fmt.Errorf("loading policy %s: invalid settings.fail_mode %q (expected %q or %q)", path, sp.Settings.FailMode, FailClosed, FailOpen)
	}

	seen := make(map[string]struct{}, len(sp.Rules))
	for i := range sp.Rules {
		if err := validateRule(&sp.Rules[i]); err != nil {
			return nil, fmt.Errorf("loading policy %s: rule index %d: %w", path, i, err)
		}
		if _, dup := seen[sp.Rules[i].ID]; dup {
			return nil, fmt.Errorf("loading policy %s: duplicate rule id %q", path, sp.Rules[i].ID)
		}
		seen[sp.Rules[i].ID] = struct{}{}
	}

	return &sp, nil
}

func LoadPolicyFiles(paths ...string) (*SecurityPolicy, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("loading policies: no paths provided")
	}

	base, err := LoadPolicyFile(paths[0])
	if err != nil {
		return nil, err
	}

	for _, path := range paths[1:] {
		overlay, err := LoadPolicyFile(path)
		if err != nil {
			return nil, err
		}
		if err := mergePolicy(base, overlay); err != nil {
			return nil, fmt.Errorf("merging policy %s: %w", path, err)
		}
	}

	return base, nil
}

// ResolveFailMode reports how a caller must treat a policy set that failed to
// load or compile. Enforcement fails closed by default: the result is FailOpen
// only when the base policy (paths[0], the most trusted file) loads cleanly on
// its own and explicitly sets settings.fail_mode: fail_open. An overlay can
// never opt the policy set into failing open, and a base file that is missing,
// malformed or invalid yields FailClosed.
func ResolveFailMode(paths []string) FailMode {
	if len(paths) == 0 {
		return FailClosed
	}
	base, err := LoadPolicyFile(paths[0])
	if err != nil || base.Settings.FailMode != FailOpen {
		return FailClosed
	}
	return FailOpen
}

func mergePolicy(base, overlay *SecurityPolicy) error {
	baseIndex := make(map[string]int, len(base.Rules))
	for i := range base.Rules {
		baseIndex[base.Rules[i].ID] = i
	}

	for _, rule := range overlay.Rules {
		idx, exists := baseIndex[rule.ID]
		if !exists {
			baseIndex[rule.ID] = len(base.Rules)
			base.Rules = append(base.Rules, rule)
			continue
		}

		existing := &base.Rules[idx]

		if err := enforceSecurityFloor(existing, &rule); err != nil {
			return err
		}

		base.Rules[idx] = rule
	}

	return nil
}

// enforceSecurityFloor rejects an overlay rule that would weaken the rule it
// replaces. Severity may only be raised. An enforce_always rule is the floor
// itself: an overlay may reword it (name, description, category, message) or
// raise its severity, but may not lower its bypass tier, disable it, switch it
// to monitor mode, change its action, or change its conditions, because each of
// those neutralizes the rule while it still reports as enforce_always.
func enforceSecurityFloor(existing, incoming *PolicyRule) error {
	if incoming.Severity > existing.Severity {
		return fmt.Errorf("security floor violation: rule %q cannot lower severity from %s to %s", existing.ID, existing.Severity, incoming.Severity)
	}

	if existing.BypassTier != EnforceAlways {
		return nil
	}

	if incoming.BypassTier != EnforceAlways {
		return fmt.Errorf("security floor violation: rule %q cannot weaken bypass_tier from enforce_always to %s", existing.ID, incoming.BypassTier)
	}

	if !incoming.IsEnabled() && existing.IsEnabled() {
		return fmt.Errorf("security floor violation: rule %q with enforce_always bypass_tier cannot be disabled", existing.ID)
	}

	if incoming.MonitorMode && !existing.MonitorMode {
		return fmt.Errorf("security floor violation: rule %q with enforce_always bypass_tier cannot be switched to monitor_mode", existing.ID)
	}

	if enforcementAction(existing.Action) != enforcementAction(incoming.Action) {
		return fmt.Errorf("security floor violation: rule %q with enforce_always bypass_tier cannot change its action (only its message may be overridden)", existing.ID)
	}

	if !reflect.DeepEqual(existing.Conditions, incoming.Conditions) {
		return fmt.Errorf("security floor violation: rule %q with enforce_always bypass_tier cannot change its conditions", existing.ID)
	}

	return nil
}

// enforcementAction returns the parts of an action that decide the verdict,
// clearing the purely presentational message an overlay may reword.
func enforcementAction(a Action) Action {
	a.Message = ""
	return a
}

func validateRule(rule *PolicyRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule id is required")
	}

	if rule.Severity < Critical || rule.Severity > Low {
		return fmt.Errorf("rule %q: invalid severity %d", rule.ID, rule.Severity)
	}

	if rule.BypassTier < EnforceAlways || rule.BypassTier > Command {
		return fmt.Errorf("rule %q: invalid bypass_tier %d", rule.ID, rule.BypassTier)
	}

	switch rule.Action.Type {
	case Block, Warn, Audit, Prompt:
	default:
		return fmt.Errorf("rule %q: invalid action type %q", rule.ID, rule.Action.Type)
	}

	return nil
}
