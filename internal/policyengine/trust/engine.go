package trust

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

type McpTrustEngine struct {
	configPath string
	config     *TrustConfig
}

func NewMcpTrustEngine(configPath string) *McpTrustEngine {
	cfg, err := LoadTrustConfig(configPath)
	if err != nil {
		cfg = &TrustConfig{Servers: make(map[string]TrustServerEntry)}
	}
	return &McpTrustEngine{
		configPath: configPath,
		config:     cfg,
	}
}

func (e *McpTrustEngine) ScoreServer(info *McpServerInfo) TrustScore {
	probes := runAllProbes(info)

	categories := buildCategoryScores(probes)

	rawScore := aggregateScore(categories)

	score, ceiling := applyCeilings(rawScore, probes)

	tier := assignTier(score)

	if entry, ok := e.config.Servers[info.Name]; ok && entry.ManualOverride {
		tier = entry.Tier
	}

	return TrustScore{
		ServerName:     info.Name,
		Score:          score,
		Tier:           tier,
		Categories:     categories,
		CeilingApplied: ceiling,
		Probes:         probes,
	}
}

func (e *McpTrustEngine) ScoreAll(servers []McpServerInfo) map[string]TrustScore {
	results := make(map[string]TrustScore, len(servers))
	for i := range servers {
		results[servers[i].Name] = e.ScoreServer(&servers[i])
	}
	return results
}

func runAllProbes(info *McpServerInfo) []ProbeResult {
	results := make([]ProbeResult, 0, len(allTrustProbes))
	for _, reg := range allTrustProbes {
		results = append(results, reg.Fn(info))
	}
	return results
}

func buildCategoryScores(probes []ProbeResult) []CategoryScore {
	grouped := make(map[string][]ProbeResult)
	for _, p := range probes {
		grouped[p.Category] = append(grouped[p.Category], p)
	}

	categories := make([]CategoryScore, 0, len(grouped))
	for name, catProbes := range grouped {
		catWeight, ok := categoryWeights[name]
		if !ok {
			continue
		}

		var totalWeight float64
		var weightedSum float64
		for _, p := range catProbes {
			totalWeight += p.Weight
			if p.Pass {
				weightedSum += p.Weight * 100
			}
		}

		var catScore float64
		if totalWeight > 0 {
			catScore = weightedSum / totalWeight
		}

		categories = append(categories, CategoryScore{
			Name:   name,
			Weight: catWeight,
			Score:  catScore,
			Probes: catProbes,
		})
	}

	return categories
}

func aggregateScore(categories []CategoryScore) int {
	var total float64
	for _, cat := range categories {
		total += cat.Weight * cat.Score
	}

	score := int(total)
	return max(0, min(score, 100))
}

func applyCeilings(rawScore int, probes []ProbeResult) (int, string) {
	for _, p := range probes {
		if p.ProbeID == "no-known-vulnerabilities" && !p.Pass {
			if rawScore > 33 {
				return 33, "known-vulnerability"
			}
			return rawScore, "known-vulnerability"
		}
	}

	for _, p := range probes {
		if p.ProbeID == "no-community-content" && !p.Pass {
			if rawScore > 45 {
				return 45, "community-content"
			}
			return rawScore, ""
		}
	}

	return rawScore, ""
}

func assignTier(score int) TrustTier {
	switch {
	case score >= 75:
		return Tier1Local
	case score >= 45:
		return Tier2Enterprise
	default:
		return Tier3Fallback
	}
}

func LoadTrustConfig(path string) (*TrustConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading trust config: %w", err)
	}

	var cfg TrustConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing trust config: %w", err)
	}

	if cfg.Servers == nil {
		cfg.Servers = make(map[string]TrustServerEntry)
	}

	return &cfg, nil
}

func SaveTrustConfig(path string, config *TrustConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling trust config: %w", err)
	}

	if err := os.WriteFile(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing trust config: %w", err)
	}

	return nil
}
