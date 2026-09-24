package generate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// FragmentAccumulator collects fragments from multiple producers and resolves
// them into a final set of GeneratedFile entries. Not safe for concurrent use.
type FragmentAccumulator struct {
	producers     map[string]types.FragmentProducer
	producerOrder []string
	fragments     []types.FragmentEntry
	hooks         *LifecycleHookRegistry
}

// SetHookRegistry attaches a lifecycle hook registry to the accumulator.
func (a *FragmentAccumulator) SetHookRegistry(r *LifecycleHookRegistry) {
	a.hooks = r
}

func NewFragmentAccumulator() *FragmentAccumulator {
	return &FragmentAccumulator{
		producers: make(map[string]types.FragmentProducer),
	}
}

func (a *FragmentAccumulator) RegisterProducer(name string, p types.FragmentProducer) error {
	if _, exists := a.producers[name]; exists {
		return fmt.Errorf("registering producer: duplicate name %q", name)
	}
	a.producers[name] = p
	a.producerOrder = append(a.producerOrder, name)
	return nil
}

func (a *FragmentAccumulator) Add(f types.FragmentEntry) {
	a.fragments = append(a.fragments, f)
}

func (a *FragmentAccumulator) AddBatch(fragments []types.FragmentEntry) {
	a.fragments = append(a.fragments, fragments...)
}

// CollectAll iterates producers in registration order and collects fragments.
// If a producer fails, its error is recorded and collection continues so every
// failure is reported, but any failure makes CollectAll return the joined
// errors: a partial collection would silently omit an addon's files while the
// command reports success.
func (a *FragmentAccumulator) CollectAll(answers types.WizardAnswers) error {
	if len(a.producerOrder) == 0 {
		return nil
	}

	var errs []error

	for _, name := range a.producerOrder {
		p := a.producers[name]
		fragments, err := p.Produce(answers)
		if err != nil {
			// The error text is returned below for the caller to report; it
			// is kept out of the log because a producer's error can quote the
			// content it generated, which carries service credentials.
			slog.Warn("fragment producer failed", "producer", name)
			errs = append(errs, fmt.Errorf("producer %q: %w", name, err))
			continue
		}
		a.fragments = append(a.fragments, fragments...)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if a.hooks != nil {
		ctx := LifecycleContext{
			Phase:     PostCollect,
			Fragments: &a.fragments,
			Answers:   answers,
		}
		if err := a.hooks.Execute(PostCollect, ctx); err != nil {
			return fmt.Errorf("executing PostCollect hooks: %w", err)
		}
	}

	return nil
}

// Resolve combines all collected fragments into resolved GeneratedFile entries.
func (a *FragmentAccumulator) Resolve() ([]types.GeneratedFile, error) {
	if len(a.fragments) == 0 {
		return nil, nil
	}

	sorted := make([]types.FragmentEntry, len(a.fragments))
	copy(sorted, a.fragments)
	slices.SortStableFunc(sorted, types.CompareFragments)

	groups := groupByTarget(sorted)

	var files []types.GeneratedFile
	for _, target := range orderedTargets(sorted) {
		group := groups[target]
		sortByPriority(group)

		if err := validateComposeMode(target, group); err != nil {
			return nil, err
		}

		resolved, err := resolveGroup(target, group)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", target, err)
		}
		files = append(files, resolved)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	if a.hooks != nil {
		ctx := LifecycleContext{
			Phase: PostResolve,
			Files: &files,
		}
		if err := a.hooks.Execute(PostResolve, ctx); err != nil {
			return nil, fmt.Errorf("executing PostResolve hooks: %w", err)
		}
	}

	return files, nil
}

// FragmentSet returns a copy of the collected fragments.
func (a *FragmentAccumulator) FragmentSet() []types.FragmentEntry {
	cp := make([]types.FragmentEntry, len(a.fragments))
	copy(cp, a.fragments)
	return cp
}

func groupByTarget(sorted []types.FragmentEntry) map[string][]types.FragmentEntry {
	groups := make(map[string][]types.FragmentEntry)
	for _, f := range sorted {
		groups[f.Target] = append(groups[f.Target], f)
	}
	return groups
}

// sortByPriority orders one target's fragments highest priority first, with
// source then tag as deterministic tie-breakers. The global SortKey cannot be
// reused here: it orders by source first, so across sources the "highest"
// fragment would be the alphabetically-first source, not the top priority.
func sortByPriority(group []types.FragmentEntry) {
	sort.SliceStable(group, func(i, j int) bool {
		a, b := group[i], group[j]
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Tag < b.Tag
	})
}

// orderedTargets returns unique target paths preserving first-seen order
// from the already-sorted slice.
func orderedTargets(sorted []types.FragmentEntry) []string {
	seen := make(map[string]bool)
	var targets []string
	for _, f := range sorted {
		if !seen[f.Target] {
			seen[f.Target] = true
			targets = append(targets, f.Target)
		}
	}
	return targets
}

func validateComposeMode(target string, group []types.FragmentEntry) error {
	if len(group) <= 1 {
		return nil
	}
	mode := group[0].ComposeMode
	for _, f := range group[1:] {
		if f.ComposeMode != mode {
			return fmt.Errorf(
				"mixed compose modes for %q: %s from %s vs %s from %s",
				target, mode, group[0].Source, f.ComposeMode, f.Source,
			)
		}
	}
	return nil
}

func resolveGroup(target string, group []types.FragmentEntry) (types.GeneratedFile, error) {
	// The group is sorted by sortByPriority (highest priority first).
	highest := group[0]

	mode := highest.Mode
	if mode == 0 {
		mode = fileutil.ModeReadWrite
	}

	base := types.GeneratedFile{
		Path:     target,
		Mode:     mode,
		Strategy: highest.Strategy,
		Owner:    highest.Owner,
	}

	switch highest.ComposeMode {
	case types.ComposeReplace:
		warnDiscardedReplacements(target, group)
		base.Content = highest.Content
		return base, nil

	case types.ComposeAppend:
		var buf bytes.Buffer
		for i, f := range group {
			if i > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(f.Content)
		}
		base.Content = buf.Bytes()
		return base, nil

	case types.ComposeSection:
		content, err := resolveSection(group)
		if err != nil {
			return types.GeneratedFile{}, err
		}
		base.Content = content
		return base, nil

	case types.ComposeMergeJSON:
		content, err := resolveJSON(group)
		if err != nil {
			return types.GeneratedFile{}, err
		}
		base.Content = content
		return base, nil

	case types.ComposeMergeYAML:
		content, err := resolveYAML(group)
		if err != nil {
			return types.GeneratedFile{}, err
		}
		base.Content = content
		return base, nil

	default:
		return types.GeneratedFile{}, fmt.Errorf("unsupported compose mode: %s", highest.ComposeMode)
	}
}

// warnDiscardedReplacements logs when ComposeReplace drops content contributed
// by a different source than the winning (highest-priority) fragment, so a
// priority conflict between producers is visible instead of silent.
func warnDiscardedReplacements(target string, group []types.FragmentEntry) {
	winner := group[0]
	for _, f := range group[1:] {
		if f.Source != winner.Source && !bytes.Equal(f.Content, winner.Content) {
			slog.Warn("replace fragment discarded by higher-priority source",
				"target", target,
				"winner", winner.Source, "winner_priority", winner.Priority,
				"discarded", f.Source, "discarded_priority", f.Priority)
		}
	}
}

// resolveSection merges fragments using section markers. A fragment with
// Tag=="" is the base document; if multiple exist the highest-priority one
// wins. Each tagged fragment replaces the section carrying its own tag, or
// is appended when the document has no section for that tag yet.
func resolveSection(group []types.FragmentEntry) ([]byte, error) {
	// Find the base fragment (empty tag). The group is sorted highest
	// priority first, so the first base fragment wins.
	var baseContent []byte
	var tagged []types.FragmentEntry

	for _, f := range group {
		if f.Tag == "" {
			if baseContent == nil {
				baseContent = f.Content
			}
		} else {
			tagged = append(tagged, f)
		}
	}

	if baseContent == nil {
		// No base fragment: wrap each tagged fragment with section markers
		// and concatenate.
		var buf bytes.Buffer
		for i, f := range tagged {
			if i > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(buildSectionBlock(f.Tag, f.Content))
		}
		return buf.Bytes(), nil
	}

	// Start with base content, then splice each tagged fragment into the
	// section carrying its tag.
	result := baseContent
	for _, f := range tagged {
		sectionBlock := buildSectionBlock(f.Tag, f.Content)

		merged, err := merge.ReplaceTaggedSection(result, f.Tag, sectionBlock)
		if err != nil {
			if errors.Is(err, merge.ErrMarkersNotFound) {
				// Base doesn't have markers for this tag yet -- append the section.
				var buf bytes.Buffer
				buf.Write(result)
				if len(result) > 0 && result[len(result)-1] != '\n' {
					buf.WriteByte('\n')
				}
				buf.Write(sectionBlock)
				result = buf.Bytes()
				continue
			}
			return nil, fmt.Errorf("merging section %q: %w", f.Tag, err)
		}
		result = merged
	}

	return result, nil
}

func buildSectionBlock(tag string, content []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(merge.SectionBeginLine(tag) + "\n")
	buf.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString(merge.EndMarker + "\n")
	return buf.Bytes()
}

func resolveJSON(group []types.FragmentEntry) ([]byte, error) {
	merged := make(map[string]any)

	// Iterate in reverse so higher-priority fragments (sorted first) overwrite last.
	for i := len(group) - 1; i >= 0; i-- {
		var m map[string]any
		if err := json.Unmarshal(group[i].Content, &m); err != nil {
			return nil, fmt.Errorf("parsing JSON from %s: %w", group[i].Source, err)
		}
		merged = merge.DeepMergeJSON(merged, m)
	}

	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling merged JSON: %w", err)
	}
	return append(data, '\n'), nil
}

func resolveYAML(group []types.FragmentEntry) ([]byte, error) {
	merged := make(map[string]any)

	for i := len(group) - 1; i >= 0; i-- {
		var m map[string]any
		if err := yaml.Unmarshal(group[i].Content, &m); err != nil {
			return nil, fmt.Errorf("parsing YAML from %s: %w", group[i].Source, err)
		}
		merged = merge.DeepMergeJSON(merged, m)
	}

	return DeterministicYAML(merged)
}
