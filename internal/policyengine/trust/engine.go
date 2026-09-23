package trust

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

type McpTrustEngine struct {
	configPath string
	config     *TrustConfig
	// forceFallback pins every server to Tier3Fallback. It is set when the
	// trust config exists but cannot be loaded, so the operator's manual
	// overrides are never silently replaced by computed (possibly higher) tiers.
	forceFallback bool
}

// NewMcpTrustEngine loads the manual tier overrides from configPath. A missing
// file (or an empty path) is not an error: the engine scores servers from their
// signals alone. A file that exists but cannot be read or parsed returns an
// error together with a usable engine that assigns every server
// Tier3Fallback, the strictest hardening, so a caller that reports the error
// and continues degrades safely instead of ignoring the overrides.
func NewMcpTrustEngine(configPath string) (*McpTrustEngine, error) {
	engine := &McpTrustEngine{
		configPath: configPath,
		config:     &TrustConfig{Servers: make(map[string]TrustServerEntry)},
	}

	cfg, err := LoadTrustConfig(configPath)
	switch {
	case err == nil:
		engine.config = cfg
	case errors.Is(err, fs.ErrNotExist):
	default:
		engine.forceFallback = true
		return engine, fmt.Errorf("loading MCP trust config %s (treating every server as %s): %w", configPath, Tier3Fallback, err)
	}

	return engine, nil
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
	if e.forceFallback {
		tier = Tier3Fallback
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

	// Iterate categories in a fixed order so the result (and the float sum in
	// aggregateScore) is identical from run to run.
	categories := make([]CategoryScore, 0, len(grouped))
	for _, name := range slices.Sorted(maps.Keys(grouped)) {
		catProbes := grouped[name]
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

// LoadTrustConfig reads a trust config strictly: an unknown key (for example a
// misspelled manual_override) is an error rather than a silently ignored field.
// An empty file yields an empty config.
func LoadTrustConfig(path string) (*TrustConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading trust config: %w", err)
	}

	var cfg TrustConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
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

	if err := fileutil.WriteFileAtomic(path, data, fileutil.ModeReadWrite); err != nil {
		return fmt.Errorf("writing trust config: %w", err)
	}

	return nil
}
