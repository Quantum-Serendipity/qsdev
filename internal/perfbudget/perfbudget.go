// Package perfbudget holds qsdev's performance budgets and the verdict logic
// that holds measurements to them.
//
// budgets.yaml, embedded here, is the single source of every budget number:
// a test names a budget by id and never spells out a limit. Measured budgets
// are CPU time (user plus system, the child's own and that of every
// descendant it waited for), so a busy machine does not fail them. A
// calibration factor scales them from the reference machine to the one
// running the test, and the document's allowance absorbs run-to-run noise.
// Wall-time and size budgets record the plan's targets for surfaces whose
// owning workstream measures them; each carries that owner's tracked skip.
//
// A budget a row is known to miss carries a tracked skip naming the
// workstream that will fix it: the row then skips instead of failing, and
// says when the skip can go. The skip waives the budget only up to its
// ceiling, the row's measured level plus headroom, so a covered row that
// regresses past it still fails. There is no other way to waive a budget.
//
// The package is an ordinary library so its own tests run everywhere; the
// measurement suites that use it are built with the perf tag.
package perfbudget

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed budgets.yaml
var budgetsYAML []byte

var (
	// workstreamRe matches a remediation workstream id such as XD-WS3; a
	// tracked skip must name one, so every waived budget has an owner.
	workstreamRe = regexp.MustCompile(`^[A-Z0-9]+-WS[0-9]+$`)
	idRe         = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Budgets is the parsed budgets document.
type Budgets struct {
	// Allowance multiplies every CPU budget to absorb run-to-run noise.
	Allowance float64
	// Runs is how many timed runs a measurement takes, after Warmups untimed
	// ones.
	Runs, Warmups int
	// Reference is the calibration measured on the machine the budget
	// numbers hold on; see Calibrate.
	Reference Reference
	Budgets   []Budget
}

// Reference is the calibration of the reference machine: the CPU p50 of
// `python3 -c pass` and of the testdata/noop Go program there.
type Reference struct {
	Machine   string
	PythonCPU time.Duration
	NoopCPU   time.Duration
}

// Budget is one budget. Exactly one of CPUP95 (a CPU p95, scaled by the
// calibration factor and the allowance), WallMax (a deadline) and MaxBytes
// (a size) is set. Only CPU budgets are measured by this package's harness;
// the others belong to their owning workstream and carry its tracked skip.
type Budget struct {
	ID       string
	Surface  string
	CPUP95   time.Duration
	WallMax  time.Duration
	MaxBytes int64
	Skips    []Skip
}

// Skip is a tracked skip: rows of the budget it belongs to whose subject is
// in Subjects and whose payload is in Payloads (an empty list matches all)
// skip, rather than fail, while over budget. Workstream owns the fix.
//
// On a CPU budget the skip waives the budget only up to Ceiling, a CPU p95
// scaled like the budget: the rows' measured level plus headroom. A covered
// row over its ceiling fails, so a regression on a surface whose budget is
// still owed fails the gate too. An owner-only wall or size budget is not
// measured here, so its skips have no ceiling.
type Skip struct {
	Workstream string
	Subjects   []string
	Payloads   []string
	Reason     string
	Ceiling    time.Duration
}

// covers reports whether the skip applies to row.
func (s Skip) covers(row Row) bool {
	return (len(s.Subjects) == 0 || slices.Contains(s.Subjects, row.Subject)) &&
		(len(s.Payloads) == 0 || slices.Contains(s.Payloads, row.Payload))
}

// skipFor returns the first tracked skip covering row, if any.
func (b Budget) skipFor(row Row) (Skip, bool) {
	i := slices.IndexFunc(b.Skips, func(s Skip) bool { return s.covers(row) })
	if i < 0 {
		return Skip{}, false
	}
	return b.Skips[i], true
}

// ownerSkip returns the budget's unfiltered tracked skip, the one that
// hands the whole budget to its owner, if it has one.
func (b Budget) ownerSkip() (Skip, bool) {
	return b.skipFor(Row{Budget: b.ID})
}

// Lookup returns the budget with the given id.
func (b Budgets) Lookup(id string) (Budget, bool) {
	i := slices.IndexFunc(b.Budgets, func(bud Budget) bool { return bud.ID == id })
	if i < 0 {
		return Budget{}, false
	}
	return b.Budgets[i], true
}

// Load parses the embedded budgets.yaml.
func Load() (Budgets, error) {
	return Parse(budgetsYAML)
}

type document struct {
	Allowance float64           `yaml:"allowance"`
	Runs      int               `yaml:"runs"`
	Warmups   int               `yaml:"warmups"`
	Reference referenceDocument `yaml:"reference"`
	Budgets   []budgetDocument  `yaml:"budgets"`
}

type referenceDocument struct {
	Machine   string  `yaml:"machine"`
	PythonCPU float64 `yaml:"python_cpu_p50_ms"`
	NoopCPU   float64 `yaml:"noop_cpu_p50_ms"`
}

func (rd referenceDocument) reference() (Reference, error) {
	switch {
	case rd.Machine == "":
		return Reference{}, errors.New("budgets: reference.machine must name the reference machine")
	case rd.PythonCPU <= 0:
		return Reference{}, fmt.Errorf("budgets: reference.python_cpu_p50_ms %v must be positive", rd.PythonCPU)
	case rd.NoopCPU <= 0:
		return Reference{}, fmt.Errorf("budgets: reference.noop_cpu_p50_ms %v must be positive", rd.NoopCPU)
	}
	return Reference{Machine: rd.Machine, PythonCPU: msDuration(rd.PythonCPU), NoopCPU: msDuration(rd.NoopCPU)}, nil
}

// msDuration converts a millisecond count to a Duration.
func msDuration(v float64) time.Duration {
	return time.Duration(v * float64(time.Millisecond))
}

type budgetDocument struct {
	ID       string         `yaml:"id"`
	Surface  string         `yaml:"surface"`
	CPUP95   *float64       `yaml:"cpu_p95_ms"`
	WallMax  *float64       `yaml:"wall_max_ms"`
	MaxBytes *int64         `yaml:"max_bytes"`
	Skips    []skipDocument `yaml:"skips"`
}

type skipDocument struct {
	Workstream string   `yaml:"workstream"`
	Subjects   []string `yaml:"subjects"`
	Payloads   []string `yaml:"payloads"`
	Reason     string   `yaml:"reason"`
	CeilingMS  *float64 `yaml:"ceiling_ms"`
}

// Parse reads and validates a budgets document. Unknown keys, a duplicate
// or malformed id, a budget that does not set exactly one positive limit, a
// skip that does not name a workstream id and a reason, a skip payload that
// is not in Payloads(), a CPU budget's skip without a ceiling above the
// budget, a ceiling on a wall or size budget's skip, a wall or size budget
// without an unfiltered tracked skip, and non-positive settings or
// reference values are errors.
func Parse(data []byte) (Budgets, error) {
	var doc document
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return Budgets{}, fmt.Errorf("parsing budgets: %w", err)
	}
	b := Budgets{Allowance: doc.Allowance, Runs: doc.Runs, Warmups: doc.Warmups}
	if err := b.checkSettings(); err != nil {
		return Budgets{}, err
	}
	ref, err := doc.Reference.reference()
	if err != nil {
		return Budgets{}, err
	}
	b.Reference = ref
	if len(doc.Budgets) == 0 {
		return Budgets{}, errors.New("budgets: no budgets declared")
	}
	for _, bd := range doc.Budgets {
		bud, err := bd.budget()
		if err != nil {
			return Budgets{}, err
		}
		if _, dup := b.Lookup(bud.ID); dup {
			return Budgets{}, fmt.Errorf("budgets: duplicate id %q", bud.ID)
		}
		b.Budgets = append(b.Budgets, bud)
	}
	return b, nil
}

func (b Budgets) checkSettings() error {
	switch {
	case b.Allowance <= 0:
		return fmt.Errorf("budgets: allowance %v must be positive", b.Allowance)
	case b.Runs < 1:
		return fmt.Errorf("budgets: runs %d must be positive", b.Runs)
	case b.Warmups < 0:
		return fmt.Errorf("budgets: warmups %d must not be negative", b.Warmups)
	}
	return nil
}

func (bd budgetDocument) budget() (Budget, error) {
	if !idRe.MatchString(bd.ID) {
		return Budget{}, fmt.Errorf("budgets: id %q must be lowercase words joined by hyphens", bd.ID)
	}
	bud := Budget{ID: bd.ID, Surface: bd.Surface}
	set := 0
	for _, lim := range []struct {
		v   *float64
		dst *time.Duration
	}{{bd.CPUP95, &bud.CPUP95}, {bd.WallMax, &bud.WallMax}} {
		if lim.v == nil {
			continue
		}
		set++
		if *lim.v <= 0 {
			return Budget{}, fmt.Errorf("budgets: %s: limit %v must be positive", bd.ID, *lim.v)
		}
		*lim.dst = msDuration(*lim.v)
	}
	if bd.MaxBytes != nil {
		set++
		if *bd.MaxBytes <= 0 {
			return Budget{}, fmt.Errorf("budgets: %s: max_bytes %d must be positive", bd.ID, *bd.MaxBytes)
		}
		bud.MaxBytes = *bd.MaxBytes
	}
	if set != 1 {
		return Budget{}, fmt.Errorf("budgets: %s must set exactly one of cpu_p95_ms, wall_max_ms and max_bytes", bd.ID)
	}
	for _, sd := range bd.Skips {
		s, err := sd.skip(bud)
		if err != nil {
			return Budget{}, err
		}
		bud.Skips = append(bud.Skips, s)
	}
	if bud.CPUP95 == 0 {
		if _, ok := bud.ownerSkip(); !ok {
			return Budget{}, fmt.Errorf("budgets: %s: a wall_max_ms or max_bytes budget is owner-only and needs an unfiltered tracked skip naming its owner", bd.ID)
		}
	}
	return bud, nil
}

// skip validates a skip of bud: it names an owning workstream and a reason,
// only payloads of the corpus and, on a CPU budget, a ceiling above the
// budget; a wall or size budget's skip has no ceiling.
func (sd skipDocument) skip(bud Budget) (Skip, error) {
	id := bud.ID
	if !workstreamRe.MatchString(sd.Workstream) {
		return Skip{}, fmt.Errorf("budgets: %s: skip owner %q is not a workstream id like XD-WS3", id, sd.Workstream)
	}
	if sd.Reason == "" {
		return Skip{}, fmt.Errorf("budgets: %s: the %s skip needs a reason", id, sd.Workstream)
	}
	for _, p := range sd.Payloads {
		if _, ok := PayloadNamed(p); !ok {
			return Skip{}, fmt.Errorf("budgets: %s: the %s skip names payload %q, which is not in the corpus", id, sd.Workstream, p)
		}
	}
	s := Skip{Workstream: sd.Workstream, Subjects: sd.Subjects, Payloads: sd.Payloads, Reason: sd.Reason}
	switch {
	case bud.CPUP95 == 0 && sd.CeilingMS != nil:
		return Skip{}, fmt.Errorf("budgets: %s: the %s skip sets ceiling_ms, but a wall_max_ms or max_bytes budget is not measured here", id, sd.Workstream)
	case bud.CPUP95 == 0:
		return s, nil
	case sd.CeilingMS == nil:
		return Skip{}, fmt.Errorf("budgets: %s: the %s skip needs a ceiling_ms, the CPU p95 up to which it waives the budget", id, sd.Workstream)
	}
	s.Ceiling = msDuration(*sd.CeilingMS)
	if s.Ceiling <= bud.CPUP95 {
		return Skip{}, fmt.Errorf("budgets: %s: the %s skip's ceiling_ms %v must be above the budget's cpu_p95_ms %s", id, sd.Workstream, *sd.CeilingMS, ms(bud.CPUP95))
	}
	return s, nil
}
