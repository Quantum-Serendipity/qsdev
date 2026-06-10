package policy

import (
	"fmt"
	"os"

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

func enforceSecurityFloor(existing, incoming *PolicyRule) error {
	if existing.BypassTier == EnforceAlways && incoming.BypassTier != EnforceAlways {
		return fmt.Errorf("security floor violation: rule %q cannot weaken bypass_tier from enforce_always to %s", existing.ID, incoming.BypassTier)
	}

	if existing.BypassTier == EnforceAlways && incoming.Enabled != nil && !*incoming.Enabled {
		return fmt.Errorf("security floor violation: rule %q with enforce_always bypass_tier cannot be disabled", existing.ID)
	}

	if incoming.Severity > existing.Severity {
		return fmt.Errorf("security floor violation: rule %q cannot lower severity from %s to %s", existing.ID, existing.Severity, incoming.Severity)
	}

	return nil
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
