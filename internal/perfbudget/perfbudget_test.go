package perfbudget

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// validHeader is the top of every test document: settings that parse.
const validHeader = "allowance: 1.25\nruns: 20\nwarmups: 2\n" + validReference

// validReference is a reference calibration that parses.
const validReference = "reference: {machine: test, python_cpu_p50_ms: 30, noop_cpu_p50_ms: 10}\n"

// settings is the top of a document with every setting but the reference.
const settings = "allowance: 1.25\nruns: 20\nwarmups: 2\n"

func TestParseRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "unknown top-level key",
			doc:  validHeader + "extra: 1\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "extra",
		},
		{
			name: "unknown budget key",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, cpu_p99_ms: 20}\n",
			want: "cpu_p99_ms",
		},
		{
			name: "unknown skip key",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3, reason: r, ceiling_ms: 20, owner: me}]}\n",
			want: "owner",
		},
		{
			name: "skip payload not in the corpus",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3, payloads: [bash_ls, bash_9000seg], reason: r, ceiling_ms: 20}]}\n",
			want: `"bash_9000seg"`,
		},
		{
			name: "cpu skip without a ceiling",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3, reason: r}]}\n",
			want: "needs a ceiling_ms",
		},
		{
			name: "cpu skip ceiling at the budget",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3, reason: r, ceiling_ms: 10}]}\n",
			want: "must be above the budget",
		},
		{
			name: "cpu skip ceiling below the budget",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3, reason: r, ceiling_ms: -5}]}\n",
			want: "must be above the budget",
		},
		{
			name: "ceiling on a wall budget's skip",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, wall_max_ms: 3000, skips: [{workstream: U17-WS1, reason: r, ceiling_ms: 4000}]}\n",
			want: "not measured here",
		},
		{
			name: "ceiling on a size budget's skip",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, max_bytes: 10, skips: [{workstream: XD-WS11, reason: r, ceiling_ms: 20}]}\n",
			want: "not measured here",
		},
		{
			name: "duplicate id",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n  - {id: a, surface: b, cpu_p95_ms: 20}\n",
			want: `duplicate id "a"`,
		},
		{
			name: "malformed id",
			doc:  validHeader + "budgets:\n  - {id: Hook_Small, surface: a, cpu_p95_ms: 10}\n",
			want: `id "Hook_Small"`,
		},
		{
			name: "empty id",
			doc:  validHeader + "budgets:\n  - {surface: a, cpu_p95_ms: 10}\n",
			want: `id ""`,
		},
		{
			name: "no limit",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a}\n",
			want: "exactly one of",
		},
		{
			name: "two limits",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, wall_max_ms: 20, skips: [{workstream: XD-WS7, reason: r}]}\n",
			want: "exactly one of",
		},
		{
			name: "zero cpu limit",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 0}\n",
			want: "must be positive",
		},
		{
			name: "negative wall limit",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, wall_max_ms: -1, skips: [{workstream: XD-WS7, reason: r}]}\n",
			want: "must be positive",
		},
		{
			name: "zero size limit",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, max_bytes: 0, skips: [{workstream: XD-WS11, reason: r}]}\n",
			want: "must be positive",
		},
		{
			name: "zero allowance",
			doc:  "allowance: 0\nruns: 20\nwarmups: 2\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "allowance",
		},
		{
			name: "negative allowance",
			doc:  "allowance: -1.25\nruns: 20\nwarmups: 2\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "allowance",
		},
		{
			name: "zero runs",
			doc:  "allowance: 1.25\nruns: 0\nwarmups: 2\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "runs",
		},
		{
			name: "negative warmups",
			doc:  "allowance: 1.25\nruns: 20\nwarmups: -1\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "warmups",
		},
		{
			name: "no reference",
			doc:  settings + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "reference.machine",
		},
		{
			name: "unknown reference key",
			doc:  settings + "reference: {machine: m, python_cpu_p50_ms: 30, noop_cpu_p50_ms: 10, qsdev_version_cpu_p50_ms: 50}\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "qsdev_version_cpu_p50_ms",
		},
		{
			name: "zero reference python",
			doc:  settings + "reference: {machine: m, python_cpu_p50_ms: 0, noop_cpu_p50_ms: 10}\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "python_cpu_p50_ms",
		},
		{
			name: "negative reference noop",
			doc:  settings + "reference: {machine: m, python_cpu_p50_ms: 30, noop_cpu_p50_ms: -1}\nbudgets:\n  - {id: a, surface: a, cpu_p95_ms: 10}\n",
			want: "noop_cpu_p50_ms",
		},
		{
			name: "no budgets",
			doc:  validHeader + "budgets: []\n",
			want: "no budgets",
		},
		{
			name: "skip owner is not a workstream id",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: perf-team, reason: r}]}\n",
			want: `"perf-team" is not a workstream id`,
		},
		{
			name: "skip owner missing",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{reason: r}]}\n",
			want: `"" is not a workstream id`,
		},
		{
			name: "skip without a reason",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, cpu_p95_ms: 10, skips: [{workstream: XD-WS3}]}\n",
			want: "needs a reason",
		},
		{
			name: "wall budget without a skip",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, wall_max_ms: 3000}\n",
			want: "owner-only",
		},
		{
			name: "size budget without a skip",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, max_bytes: 32768}\n",
			want: "owner-only",
		},
		{
			name: "wall budget whose only skip is filtered",
			doc:  validHeader + "budgets:\n  - {id: a, surface: a, wall_max_ms: 3000, skips: [{workstream: U17-WS1, subjects: [package-guard], reason: r}]}\n",
			want: "owner-only",
		},
		{
			name: "not yaml",
			doc:  "budgets: [",
			want: "parsing budgets",
		},
		{
			name: "empty document",
			doc:  "",
			want: "parsing budgets",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tt.doc))
			if err == nil {
				t.Fatalf("Parse accepted:\n%s", tt.doc)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse error = %q, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestParseAccepts(t *testing.T) {
	t.Parallel()
	doc := validHeader + `budgets:
  - id: hook-small
    surface: a hook
    cpu_p95_ms: 25.5
    skips:
      - workstream: XD-WS3
        subjects: [selfprotect]
        payloads: [bash_ls, edit_2k]
        reason: start-up
        ceiling_ms: 40
  - id: deadline
    surface: a deadline
    wall_max_ms: 3000
    skips:
      - {workstream: U17-WS1, reason: no seam yet}
  - id: footprint
    surface: a size
    max_bytes: 32768
    skips:
      - {workstream: XD-WS11, reason: too big}
`
	b, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if b.Allowance != 1.25 || b.Runs != 20 || b.Warmups != 2 {
		t.Errorf("settings = %v/%d/%d, want 1.25/20/2", b.Allowance, b.Runs, b.Warmups)
	}
	if want := (Reference{Machine: "test", PythonCPU: 30 * time.Millisecond, NoopCPU: 10 * time.Millisecond}); b.Reference != want {
		t.Errorf("Reference = %+v, want %+v", b.Reference, want)
	}
	small, ok := b.Lookup("hook-small")
	if !ok {
		t.Fatal(`Lookup("hook-small") found nothing`)
	}
	if small.CPUP95 != 25500*time.Microsecond {
		t.Errorf("CPUP95 = %v, want 25.5ms", small.CPUP95)
	}
	if len(small.Skips) != 1 || small.Skips[0].Workstream != "XD-WS3" ||
		!slices.Equal(small.Skips[0].Subjects, []string{"selfprotect"}) ||
		!slices.Equal(small.Skips[0].Payloads, []string{"bash_ls", "edit_2k"}) ||
		small.Skips[0].Ceiling != 40*time.Millisecond {
		t.Errorf("skips = %+v", small.Skips)
	}
	if d, _ := b.Lookup("deadline"); len(d.Skips) != 1 || d.Skips[0].Ceiling != 0 {
		t.Errorf("an owner-only skip has a ceiling: %+v", d.Skips)
	}
	if d, _ := b.Lookup("deadline"); d.WallMax != 3*time.Second {
		t.Errorf("WallMax = %v, want 3s", d.WallMax)
	}
	if f, _ := b.Lookup("footprint"); f.MaxBytes != 32768 {
		t.Errorf("MaxBytes = %d, want 32768", f.MaxBytes)
	}
	if _, ok := b.Lookup("missing"); ok {
		t.Error(`Lookup("missing") found a budget`)
	}
}

func TestLoadEmbedded(t *testing.T) {
	t.Parallel()
	b, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// The surfaces of the XD plan's budget table (crosscut/dx-performance.md,
	// "Budgets"), by the ids the suites name them with.
	cpu := []string{
		"hook-inbinary-small", "hook-inbinary-large", "hook-script-miss", "hook-script-hit",
		"toolcall-critical-bash-standard", "toolcall-critical-bash-full",
		"toolcall-critical-edit-full", "toolcall-critical-read-full", "toolcall-total-bash-full",
		"sandbox-wrap", "cli-startup", "cli-readonly", "cli-mcp-status", "cli-init",
	}
	ownerOnly := map[string]string{
		"package-guard-network-verdict": "U17-WS1",
		"sandbox-wrap":                  "XD-WS4",
		"zero-to-shell":                 "XD-WS7",
		"enter-shell":                   "XD-WS10",
		"state-footprint":               "XD-WS11",
	}
	for _, id := range cpu {
		bud, ok := b.Lookup(id)
		if !ok {
			t.Errorf("budgets.yaml has no budget %q", id)
			continue
		}
		if bud.CPUP95 <= 0 {
			t.Errorf("%s: want a cpu_p95_ms budget, got %+v", id, bud)
		}
	}
	for id, owner := range ownerOnly {
		bud, ok := b.Lookup(id)
		if !ok {
			t.Errorf("budgets.yaml has no budget %q", id)
			continue
		}
		if s, ok := bud.ownerSkip(); !ok || s.Workstream != owner {
			t.Errorf("%s: want an unfiltered tracked skip owned by %s, got %+v", id, owner, bud.Skips)
		}
	}
	if len(b.Budgets) != len(cpu)+len(ownerOnly)-1 { // sandbox-wrap is in both
		t.Errorf("budgets.yaml has %d budgets, want the plan's %d", len(b.Budgets), len(cpu)+len(ownerOnly)-1)
	}
}

func TestNewDist(t *testing.T) {
	t.Parallel()
	ms := func(v ...int) []time.Duration {
		out := make([]time.Duration, len(v))
		for i, x := range v {
			out[i] = time.Duration(x) * time.Millisecond
		}
		return out
	}
	tests := []struct {
		name          string
		samples       []time.Duration
		p50, p95, max int
	}{
		{name: "empty", samples: nil},
		{name: "one", samples: ms(7), p50: 7, p95: 7, max: 7},
		{name: "two", samples: ms(9, 3), p50: 3, p95: 9, max: 9},
		{name: "five unsorted", samples: ms(5, 1, 4, 2, 3), p50: 3, p95: 5, max: 5},
		// Twenty samples 1..20 in reverse: rank ceil(0.5*20)=10 and
		// ceil(0.95*20)=19.
		{name: "twenty", samples: ms(20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1), p50: 10, p95: 19, max: 20},
		// Twenty-one samples: ranks ceil(10.5)=11 and ceil(19.95)=20.
		{name: "twenty-one", samples: ms(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21), p50: 11, p95: 20, max: 21},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before := slices.Clone(tt.samples)
			d := newDist(tt.samples)
			want := Dist{
				P50: time.Duration(tt.p50) * time.Millisecond,
				P95: time.Duration(tt.p95) * time.Millisecond,
				Max: time.Duration(tt.max) * time.Millisecond,
			}
			if d != want {
				t.Errorf("newDist = %+v, want %+v", d, want)
			}
			if !slices.Equal(tt.samples, before) {
				t.Errorf("newDist reordered its input: %v, was %v", tt.samples, before)
			}
		})
	}
}
