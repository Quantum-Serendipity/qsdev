package projectctx

import (
	"context"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Tier is the priority band a tool occupies for ceiling pruning. Lower tiers are
// more core and are pruned last. It maps directly onto
// spi.ToolRegistration.Tier (also "lower is more core").
type Tier int

const (
	// TierCritical tools are fundamental project awareness and are never dropped;
	// when even the critical band overflows the ceiling they are consolidated
	// rather than removed.
	TierCritical Tier = 0
	// TierStandard tools are common introspection, pruned after extended.
	TierStandard Tier = 1
	// TierExtended tools are specialized; they are the first to be pruned.
	TierExtended Tier = 2
)

// ToolPruner reduces a tool catalog to fit a framework's tool-count ceiling. It
// is stateless and safe to share.
type ToolPruner struct{}

// NewToolPruner constructs a ToolPruner.
func NewToolPruner() *ToolPruner { return &ToolPruner{} }

// Prune returns a subset of tools that fits within ceiling. A ceiling of zero or
// negative, or a catalog already within the ceiling, returns a copy unchanged.
//
// Strategy:
//  1. Tier pruning — drop the lowest-priority (highest-Tier) tools first, one at
//     a time, until the catalog fits. Tools in the most-core tier present are
//     never dropped here.
//  2. Consolidation fallback — if the most-core tier alone still exceeds the
//     ceiling, collapse same-category tools into a single consolidated tool that
//     dispatches to its members, recovering slots without losing functionality.
//     Consolidation is best-effort: a ceiling below the number of distinct
//     categories cannot be honored without dropping capability, which Prune
//     refuses to do silently — it returns the consolidated set even if it still
//     exceeds the ceiling.
func (p *ToolPruner) Prune(tools []spi.ToolRegistration, ceiling int) []spi.ToolRegistration {
	out := append([]spi.ToolRegistration(nil), tools...)
	if ceiling <= 0 || len(out) <= ceiling {
		return out
	}

	// Order by tier ascending (more core first), then name for determinism. The
	// least-core tools end up at the tail and are removed first.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Name < out[j].Name
	})

	minTier := out[0].Tier
	n := len(out)
	for n > ceiling && out[n-1].Tier > minTier {
		n--
	}
	out = out[:n]
	if len(out) <= ceiling {
		return out
	}
	return p.consolidate(out, ceiling)
}

// consolidate collapses same-category groups into single dispatcher tools,
// largest group first, until the catalog fits the ceiling or no further
// consolidation is possible (every remaining category has a single tool).
func (p *ToolPruner) consolidate(tools []spi.ToolRegistration, ceiling int) []spi.ToolRegistration {
	// Preserve original order via an index; group by category.
	groups := map[string][]spi.ToolRegistration{}
	var order []string
	for _, t := range tools {
		if _, seen := groups[t.Category]; !seen {
			order = append(order, t.Category)
		}
		groups[t.Category] = append(groups[t.Category], t)
	}

	total := len(tools)
	for total > ceiling {
		// Pick the largest consolidatable group (size > 1); ties broken by
		// category name for determinism.
		best := ""
		for _, cat := range order {
			if len(groups[cat]) <= 1 {
				continue
			}
			if best == "" || len(groups[cat]) > len(groups[best]) ||
				(len(groups[cat]) == len(groups[best]) && cat < best) {
				best = cat
			}
		}
		if best == "" {
			break // nothing left to consolidate
		}
		members := groups[best]
		total -= len(members) - 1
		groups[best] = []spi.ToolRegistration{consolidatedTool(best, members)}
	}

	// Reassemble in original category order.
	out := make([]spi.ToolRegistration, 0, total)
	for _, cat := range order {
		out = append(out, groups[cat]...)
	}
	return out
}

// consolidatedTool builds a single dispatcher tool that forwards to one of its
// member tools selected by the "tool" argument, passing that member's arguments
// under "args". Its tier is the most-core tier among its members.
func consolidatedTool(category string, members []spi.ToolRegistration) spi.ToolRegistration {
	names := make([]string, 0, len(members))
	byName := make(map[string]spi.ToolHandler, len(members))
	minTier := members[0].Tier
	for _, m := range members {
		names = append(names, m.Name)
		byName[m.Name] = m.Handler
		if m.Tier < minTier {
			minTier = m.Tier
		}
	}
	sort.Strings(names)

	return spi.ToolRegistration{
		Name: category + "_ops",
		Description: "Consolidated " + category + " operations. Set \"tool\" to one of: " +
			strings.Join(names, ", ") + " and pass that tool's arguments under \"args\".",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tool": map[string]any{
					"type": "string", "enum": names,
					"description": "The consolidated member tool to invoke.",
				},
				"args": map[string]any{
					"type":        "object",
					"description": "Arguments forwarded verbatim to the selected member tool.",
				},
			},
			"required": []string{"tool"},
		},
		Category: category,
		Tier:     minTier,
		Handler:  consolidatedHandler(byName),
	}
}

// consolidatedHandler dispatches to the member handler named by the "tool"
// argument.
func consolidatedHandler(byName map[string]spi.ToolHandler) spi.ToolHandler {
	return func(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
		name, _ := req.Arguments["tool"].(string)
		h, ok := byName[name]
		if !ok {
			return &spi.ToolResult{
				Text:    "unknown consolidated tool: " + name,
				IsError: true,
			}, nil
		}
		sub := &spi.ToolRequest{Name: name, Meta: req.Meta, Arguments: map[string]any{}}
		if a, ok := req.Arguments["args"].(map[string]any); ok {
			sub.Arguments = a
		}
		return h(ctx, cc, sub)
	}
}
