package generate

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestDeterministicJSON_SortedKeys(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"zebra":    1,
		"alpha":    2,
		"mango":    3,
		"broccoli": 4,
	}

	data, err := DeterministicJSON(input)
	if err != nil {
		t.Fatalf("DeterministicJSON failed: %v", err)
	}

	got := string(data)
	alphaIdx := strings.Index(got, "alpha")
	brocolliIdx := strings.Index(got, "broccoli")
	mangoIdx := strings.Index(got, "mango")
	zebraIdx := strings.Index(got, "zebra")

	if alphaIdx > brocolliIdx || brocolliIdx > mangoIdx || mangoIdx > zebraIdx {
		t.Fatalf("keys not sorted in output:\n%s", got)
	}
}

func TestDeterministicJSON_TrailingNewline(t *testing.T) {
	t.Parallel()

	data, err := DeterministicJSON(map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("DeterministicJSON failed: %v", err)
	}

	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Fatalf("expected trailing newline, got %q", string(data))
	}
}

func TestDeterministicJSON_NestedMaps(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"outer_z": map[string]any{
			"inner_b": 2,
			"inner_a": 1,
		},
		"outer_a": "value",
	}

	data, err := DeterministicJSON(input)
	if err != nil {
		t.Fatalf("DeterministicJSON failed: %v", err)
	}

	got := string(data)
	outerAIdx := strings.Index(got, "outer_a")
	outerZIdx := strings.Index(got, "outer_z")
	if outerAIdx > outerZIdx {
		t.Fatalf("outer keys not sorted:\n%s", got)
	}

	innerAIdx := strings.Index(got, "inner_a")
	innerBIdx := strings.Index(got, "inner_b")
	if innerAIdx > innerBIdx {
		t.Fatalf("inner keys not sorted:\n%s", got)
	}
}

func TestDeterministicYAML_SortedKeys(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"zebra": 1,
		"alpha": 2,
		"mango": 3,
	}

	data, err := DeterministicYAML(input)
	if err != nil {
		t.Fatalf("DeterministicYAML failed: %v", err)
	}

	got := string(data)
	alphaIdx := strings.Index(got, "alpha")
	mangoIdx := strings.Index(got, "mango")
	zebraIdx := strings.Index(got, "zebra")

	if alphaIdx > mangoIdx || mangoIdx > zebraIdx {
		t.Fatalf("YAML keys not sorted:\n%s", got)
	}

	// Verify it's valid YAML.
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
}

func TestDeterministicYAML_NestedMaps(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"z_outer": map[string]any{
			"b_inner": 2,
			"a_inner": 1,
		},
		"a_outer": "value",
	}

	data, err := DeterministicYAML(input)
	if err != nil {
		t.Fatalf("DeterministicYAML failed: %v", err)
	}

	got := string(data)
	aOuterIdx := strings.Index(got, "a_outer")
	zOuterIdx := strings.Index(got, "z_outer")
	if aOuterIdx > zOuterIdx {
		t.Fatalf("outer YAML keys not sorted:\n%s", got)
	}

	aInnerIdx := strings.Index(got, "a_inner")
	bInnerIdx := strings.Index(got, "b_inner")
	if aInnerIdx > bInnerIdx {
		t.Fatalf("inner YAML keys not sorted:\n%s", got)
	}
}

func TestDeterministic_100FragmentStability(t *testing.T) {
	t.Parallel()

	producers := []string{"devenv", "claudecode", "hooks", "security", "tools"}
	var allFragments []types.FragmentEntry

	for i := 0; i < 100; i++ {
		source := producers[i%len(producers)]
		target := ""
		var mode types.ComposeMode

		switch i % 5 {
		case 0:
			target = "config.json"
			mode = types.ComposeMergeJSON
		case 1:
			target = "settings.yaml"
			mode = types.ComposeMergeYAML
		case 2:
			target = "output.txt"
			mode = types.ComposeAppend
		case 3:
			target = "main.conf"
			mode = types.ComposeReplace
		case 4:
			target = "readme.md"
			mode = types.ComposeAppend
		}

		var content []byte
		switch mode {
		case types.ComposeMergeJSON:
			content = []byte(`{"key_` + source + `_` + string(rune('a'+i%26)) + `": ` + string(rune('0'+i%10)) + `}`)
		case types.ComposeMergeYAML:
			content = []byte("key_" + source + "_" + string(rune('a'+i%26)) + ": " + string(rune('0'+i%10)) + "\n")
		default:
			content = []byte("line from " + source + " " + string(rune('A'+i%26)))
		}

		allFragments = append(allFragments, types.FragmentEntry{
			Source:      source,
			Target:      target,
			Content:     content,
			Priority:    i * 10,
			ComposeMode: mode,
		})
	}

	// Resolve 10 times with different shuffle orders, assert byte-identical output.
	var reference []types.GeneratedFile

	for iter := 0; iter < 10; iter++ {
		shuffled := make([]types.FragmentEntry, len(allFragments))
		copy(shuffled, allFragments)

		rng := rand.New(rand.NewSource(int64(iter * 42)))
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		a := NewFragmentAccumulator()
		a.AddBatch(shuffled)

		files, err := a.Resolve()
		if err != nil {
			t.Fatalf("iteration %d: Resolve failed: %v", iter, err)
		}

		if reference == nil {
			reference = files
			continue
		}

		if len(files) != len(reference) {
			t.Fatalf("iteration %d: expected %d files, got %d", iter, len(reference), len(files))
		}

		for i := range files {
			if files[i].Path != reference[i].Path {
				t.Fatalf("iteration %d, file %d: path mismatch: %q vs %q",
					iter, i, files[i].Path, reference[i].Path)
			}
			if !bytes.Equal(files[i].Content, reference[i].Content) {
				t.Fatalf("iteration %d, file %d (%s): content mismatch:\n--- reference ---\n%s\n--- got ---\n%s",
					iter, i, files[i].Path, string(reference[i].Content), string(files[i].Content))
			}
		}
	}
}
