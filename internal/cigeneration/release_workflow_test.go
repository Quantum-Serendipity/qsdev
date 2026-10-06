package cigeneration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Action references the release workflow tests look for, without the @pin.
const (
	actionHardenRunner     = "step-security/harden-runner@"
	actionCheckout         = "actions/checkout@"
	actionAttestProvenance = "actions/attest-build-provenance@"
)

// releaseWorkflowPath is the workflow whose OIDC identity every shipped
// binary pins for self-update verification (branding.ReleaseWorkflow).
var releaseWorkflowPath = filepath.Join(".github", "workflows", branding.ReleaseWorkflow)

// readReleaseWorkflow decodes the release workflow and fails if it has no jobs.
func readReleaseWorkflow(t *testing.T) workflowFile {
	t.Helper()

	var wf workflowFile
	readRepoYAML(t, releaseWorkflowPath, &wf)
	if len(wf.Jobs) == 0 {
		t.Fatalf("%s has no jobs", releaseWorkflowPath)
	}
	return wf
}

// goreleaserConfig is the part of .goreleaser.yaml that decides which files
// a release publishes.
type goreleaserConfig struct {
	Archives []struct {
		Formats         []string `yaml:"formats"`
		FormatOverrides []struct {
			Formats []string `yaml:"formats"`
		} `yaml:"format_overrides"`
	} `yaml:"archives"`
	NFPMs []struct {
		Formats []string `yaml:"formats"`
	} `yaml:"nfpms"`
	Checksum struct {
		NameTemplate string `yaml:"name_template"`
	} `yaml:"checksum"`
}

// nfpmExtensions maps an nfpm packager to the extension of the file its
// ConventionalFileName produces. A packager missing here fails the test
// rather than going unattested.
var nfpmExtensions = map[string]string{
	"deb":        "deb",
	"rpm":        "rpm",
	"apk":        "apk",
	"archlinux":  "pkg.tar.zst",
	"ipk":        "ipk",
	"termux.deb": "deb",
}

// publishedArtifactGlobs derives, from .goreleaser.yaml, the dist/ glob of
// every archive format, every nfpm package format and the checksum file.
func publishedArtifactGlobs(t *testing.T) []string {
	t.Helper()

	var cfg goreleaserConfig
	readRepoYAML(t, ".goreleaser.yaml", &cfg)

	var globs []string
	add := func(g string) {
		if !slices.Contains(globs, g) {
			globs = append(globs, g)
		}
	}
	for _, a := range cfg.Archives {
		for _, f := range a.Formats {
			add("dist/*." + f)
		}
		for _, o := range a.FormatOverrides {
			for _, f := range o.Formats {
				add("dist/*." + f)
			}
		}
	}
	for _, n := range cfg.NFPMs {
		for _, f := range n.Formats {
			ext, ok := nfpmExtensions[f]
			if !ok {
				t.Errorf(".goreleaser.yaml nfpm format %q has no known file extension; add it to nfpmExtensions", f)
				continue
			}
			add("dist/*." + ext)
		}
	}
	name := cfg.Checksum.NameTemplate
	if name == "" || strings.Contains(name, "{{") {
		t.Errorf(".goreleaser.yaml checksum.name_template = %q, want a literal file name the attestation can name", name)
	} else {
		add("dist/" + name)
	}
	if len(globs) == 0 {
		t.Fatal(".goreleaser.yaml publishes no archives, packages or checksum; the parser or the layout changed")
	}
	return globs
}

// splitSubjectPaths splits an attest subject-path input, which takes a comma-
// or newline-separated list of globs.
func splitSubjectPaths(v string) []string {
	return strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' || r == '\t' || r == '\r' })
}

// TestReleaseAttestsEveryPublishedArtifact is the U26-06 regression test for
// the release-provenance claim: the build-provenance attestation names every
// file GoReleaser publishes. The expected globs come from .goreleaser.yaml,
// so adding a package format without attesting it fails here.
func TestReleaseAttestsEveryPublishedArtifact(t *testing.T) {
	t.Parallel()

	wf := readReleaseWorkflow(t)
	var subjects []string
	for name, job := range wf.Jobs {
		for _, s := range job.Steps {
			if !strings.HasPrefix(s.Uses, actionAttestProvenance) {
				continue
			}
			v := s.With["subject-path"]
			if strings.Contains(v, "${{") {
				t.Errorf("job %s: attest subject-path %q is an expression; list the globs .goreleaser.yaml produces", name, v)
			}
			subjects = append(subjects, splitSubjectPaths(v)...)
		}
	}
	for _, want := range publishedArtifactGlobs(t) {
		if !slices.Contains(subjects, want) {
			t.Errorf("%s attests subject-path %v, missing %s, which .goreleaser.yaml publishes", releaseWorkflowPath, subjects, want)
		}
	}
}

// TestReleaseJobOutputsConsumed rejects dead plumbing in the release
// workflow: every job output must be read by another job as
// needs.<job>.outputs.<name>.
func TestReleaseJobOutputsConsumed(t *testing.T) {
	t.Parallel()

	wf := readReleaseWorkflow(t)
	b, err := os.ReadFile(filepath.Join(repoRoot, releaseWorkflowPath))
	if err != nil {
		t.Fatalf("reading %s: %v", releaseWorkflowPath, err)
	}
	for name, job := range wf.Jobs {
		for out := range job.Outputs {
			if !strings.Contains(string(b), "needs."+name+".outputs."+out) {
				t.Errorf("job %s: output %s is consumed by no job", name, out)
			}
		}
	}
}

// allowedEndpointRe is one harden-runner allowed-endpoints entry: host:port.
var allowedEndpointRe = regexp.MustCompile(`^(\*\.)?[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+:\d+$`)

// TestReleaseJobsBlockEgress is the U26-V01 regression test for the release
// lane: the jobs hold the id-token signing identity and the tap tokens, so
// each starts with harden-runner blocking all egress outside an explicit
// host:port allowlist, not merely auditing it.
func TestReleaseJobsBlockEgress(t *testing.T) {
	t.Parallel()

	for name, job := range readReleaseWorkflow(t).Jobs {
		if len(job.Steps) == 0 || !strings.HasPrefix(job.Steps[0].Uses, actionHardenRunner) {
			t.Errorf("job %s: first step is not %s", name, strings.TrimSuffix(actionHardenRunner, "@"))
			continue
		}
		with := job.Steps[0].With
		if got := with["egress-policy"]; got != "block" {
			t.Errorf("job %s: harden-runner egress-policy = %q, want block", name, got)
		}
		endpoints := strings.Fields(with["allowed-endpoints"])
		if len(endpoints) == 0 {
			t.Errorf("job %s: harden-runner allowed-endpoints is empty", name)
		}
		for _, e := range endpoints {
			if !allowedEndpointRe.MatchString(e) {
				t.Errorf("job %s: allowed endpoint %q is not host:port", name, e)
			}
		}
	}
}

// TestReleaseCheckoutDropsCredentials keeps the GITHUB_TOKEN out of the
// release checkout's .git/config, where any later step of a job holding the
// signing identity could read it.
func TestReleaseCheckoutDropsCredentials(t *testing.T) {
	t.Parallel()

	checkouts := 0
	for name, job := range readReleaseWorkflow(t).Jobs {
		for i, s := range job.Steps {
			if !strings.HasPrefix(s.Uses, actionCheckout) {
				continue
			}
			checkouts++
			if got := s.With["persist-credentials"]; got != "false" {
				t.Errorf("job %s step %d (%s): persist-credentials = %q, want false", name, i+1, s.Name, got)
			}
		}
	}
	if checkouts == 0 {
		t.Fatalf("%s has no checkout step; the parser or the layout changed", releaseWorkflowPath)
	}
}

// Release CI gate (U26-06): the release workflow only ships a tag whose
// commit passed every check branch protection requires on main.
const (
	// verifyCIJob is the release.yml job every other release job needs.
	verifyCIJob = "verify-ci"
	// verifyCIScript compares the tag commit's check runs with the list.
	verifyCIScript = "scripts/verify-ci.sh"
)

// requiredChecksPath is the single list of required check-run names, read by
// the release gate.
var requiredChecksPath = filepath.Join(".github", "required-checks.txt")

// readRequiredChecks returns the check names in required-checks.txt, one per
// non-blank line; lines starting with # are comments.
func readRequiredChecks(t *testing.T) []string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(repoRoot, requiredChecksPath))
	if err != nil {
		t.Fatalf("reading %s: %v", requiredChecksPath, err)
	}
	var checks []string
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if slices.Contains(checks, line) {
			t.Errorf("%s:%d: %q is listed twice", requiredChecksPath, i+1, line)
		}
		checks = append(checks, line)
	}
	if len(checks) == 0 {
		t.Fatalf("%s lists no checks; the release gate would pass any commit", requiredChecksPath)
	}
	return checks
}

// pushesToMain reports whether a workflow runs on a push to main, which is
// the run whose check runs land on the commit a release tag points to.
func pushesToMain(t *testing.T, file string, wf workflowFile) bool {
	t.Helper()

	push, ok := wf.On["push"]
	if !ok {
		return false
	}
	var cfg struct {
		Branches []string `yaml:"branches"`
		Tags     []string `yaml:"tags"`
	}
	if push.Kind == yaml.MappingNode {
		if err := push.Decode(&cfg); err != nil {
			t.Fatalf("%s: decoding on.push: %v", file, err)
		}
	}
	if len(cfg.Branches) == 0 {
		// No branch filter: every branch, unless only tags are filtered.
		return len(cfg.Tags) == 0
	}
	return slices.Contains(cfg.Branches, "main")
}

// matrixLegs returns the value list GitHub appends to a matrix job's check
// name, one "a, b" string per leg, or nil for a job without a matrix. It
// supports a matrix of literal axes or of include entries, not both.
func matrixLegs(m yaml.Node) ([]string, error) {
	switch m.Kind {
	case 0:
		return nil, nil
	case yaml.MappingNode:
	default:
		return nil, errors.New("matrix is not a literal mapping")
	}
	var axes, include [][]string
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, val := m.Content[i].Value, m.Content[i+1]
		switch key {
		case "exclude":
			return nil, errors.New("matrix exclude is not supported")
		case "include":
			for _, entry := range val.Content {
				if entry.Kind != yaml.MappingNode {
					return nil, errors.New("matrix include entry is not a mapping")
				}
				var vals []string
				for j := 1; j < len(entry.Content); j += 2 {
					vals = append(vals, entry.Content[j].Value)
				}
				include = append(include, vals)
			}
		default:
			if val.Kind != yaml.SequenceNode {
				return nil, fmt.Errorf("matrix axis %s is not a literal list", key)
			}
			var vals []string
			for _, v := range val.Content {
				vals = append(vals, v.Value)
			}
			axes = append(axes, vals)
		}
	}
	if len(axes) > 0 && len(include) > 0 {
		return nil, errors.New("matrix mixes axes and include entries")
	}
	if len(include) > 0 {
		legs := make([]string, 0, len(include))
		for _, vals := range include {
			legs = append(legs, strings.Join(vals, ", "))
		}
		return legs, nil
	}
	combos := [][]string{nil}
	for _, axis := range axes {
		var next [][]string
		for _, c := range combos {
			for _, v := range axis {
				next = append(next, append(slices.Clone(c), v))
			}
		}
		combos = next
	}
	legs := make([]string, 0, len(combos))
	for _, c := range combos {
		legs = append(legs, strings.Join(c, ", "))
	}
	return legs, nil
}

// checkRunNames returns the check-run names a job produces: its name (or id),
// with a "(…)" suffix per matrix leg.
func checkRunNames(id string, job workflowJob) ([]string, error) {
	base := id
	if job.Name != "" {
		if strings.Contains(job.Name, "${{") {
			return nil, fmt.Errorf("job name %q is an expression", job.Name)
		}
		base = job.Name
	}
	legs, err := matrixLegs(job.Strategy.Matrix)
	if err != nil {
		return nil, err
	}
	if len(legs) == 0 {
		return []string{base}, nil
	}
	names := make([]string, 0, len(legs))
	for _, leg := range legs {
		names = append(names, base+" ("+leg+")")
	}
	return names, nil
}

// TestCheckRunNames pins the check-name derivation TestRequiredChecksRunOnPushToMain
// relies on to GitHub's "<job> (<matrix values>)" form.
func TestCheckRunNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		job     string
		want    []string
		wantErr bool
	}{
		{name: "plain", job: "runs-on: x\n", want: []string{"j"}},
		{name: "named", job: "name: Lint\n", want: []string{"Lint"}},
		{name: "axis", job: "strategy:\n  matrix:\n    os: [a, b]\n", want: []string{"j (a)", "j (b)"}},
		{
			name: "two axes",
			job:  "strategy:\n  matrix:\n    os: [a, b]\n    go: ['1', '2']\n",
			want: []string{"j (a, 1)", "j (a, 2)", "j (b, 1)", "j (b, 2)"},
		},
		{
			name: "include",
			job:  "strategy:\n  matrix:\n    include:\n      - os: a\n        script: x y\n      - os: b\n        script: z\n",
			want: []string{"j (a, x y)", "j (b, z)"},
		},
		{name: "expression matrix", job: "strategy:\n  matrix: ${{ fromJSON(x) }}\n", wantErr: true},
		{name: "exclude", job: "strategy:\n  matrix:\n    os: [a]\n    exclude: [{os: a}]\n", wantErr: true},
		{name: "expression name", job: "name: ${{ matrix.os }}\n", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var job workflowJob
			if err := yaml.Unmarshal([]byte(tc.job), &job); err != nil {
				t.Fatalf("parsing job: %v", err)
			}
			got, err := checkRunNames("j", job)
			if (err != nil) != tc.wantErr {
				t.Fatalf("checkRunNames() error = %v, wantErr %t", err, tc.wantErr)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("checkRunNames() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRequiredChecksRunOnPushToMain keeps required-checks.txt satisfiable by
// the release gate: the check runs on a release tag's commit are the ones
// its push to main produced, so every listed name must be a check run (with
// the "(…)" suffix of a matrix leg) of a job in a workflow that runs on push
// to main. A pull_request-only check would never appear and would block
// every release.
func TestRequiredChecksRunOnPushToMain(t *testing.T) {
	t.Parallel()

	produced := map[string]string{}
	jobErrs := map[string]error{}
	for file, wf := range readRepoWorkflows(t) {
		if !pushesToMain(t, file, wf) {
			continue
		}
		for id, job := range wf.Jobs {
			names, err := checkRunNames(id, job)
			if err != nil {
				jobErrs[id] = fmt.Errorf("%s job %s: %w", file, id, err)
				continue
			}
			for _, n := range names {
				produced[n] = file + " job " + id
			}
		}
	}
	for _, check := range readRequiredChecks(t) {
		if _, ok := produced[check]; ok {
			continue
		}
		job, _, _ := strings.Cut(check, " (")
		if err, ok := jobErrs[job]; ok {
			t.Errorf("%s: %q: cannot derive the check names of %v", requiredChecksPath, check, err)
			continue
		}
		t.Errorf("%s: %q is not a check run of any job in a workflow that runs on push to main", requiredChecksPath, check)
	}
}

// TestReleaseGatedOnCI is the U26-06 regression test for the release gate:
// a read-only verify-ci job runs scripts/verify-ci.sh against
// required-checks.txt for the tag's commit, and every other release job
// needs it, so a tag on a commit that did not pass CI publishes nothing.
func TestReleaseGatedOnCI(t *testing.T) {
	t.Parallel()

	wf := readReleaseWorkflow(t)
	gate, ok := wf.Jobs[verifyCIJob]
	if !ok {
		t.Fatalf("%s has no %s job", releaseWorkflowPath, verifyCIJob)
	}
	if got := gate.Permissions["checks"]; got != "read" {
		t.Errorf("job %s: permissions checks = %q, want read", verifyCIJob, got)
	}
	for perm, level := range gate.Permissions {
		if level != "read" && level != "none" {
			t.Errorf("job %s: permissions %s: %s; the gate only reads", verifyCIJob, perm, level)
		}
	}
	if len(gate.Needs) != 0 {
		t.Errorf("job %s: needs %v; the gate runs first", verifyCIJob, gate.Needs)
	}
	runsScript := false
	for _, s := range gate.Steps {
		text := stepText(s)
		if strings.Contains(s.Run, verifyCIScript) &&
			strings.Contains(s.Run, filepath.ToSlash(requiredChecksPath)) &&
			(strings.Contains(text, "GITHUB_SHA") || strings.Contains(text, "github.sha")) {
			runsScript = true
		}
	}
	if !runsScript {
		t.Errorf("job %s: no step runs %s against %s for the tag commit", verifyCIJob, verifyCIScript, filepath.ToSlash(requiredChecksPath))
	}
	for name, job := range wf.Jobs {
		if name != verifyCIJob && !slices.Contains(job.Needs, verifyCIJob) {
			t.Errorf("job %s: needs %v, want it to include %s", name, job.Needs, verifyCIJob)
		}
	}
}

// Gateway image publishing (U26-04): every generated docker-compose.gateway.yaml
// references container.ImageRepository, so the release publishes, signs,
// attests and verifies it.
const (
	// gatewayImageJob is the release.yml job that publishes the gateway image.
	gatewayImageJob = "gateway-image"
	// gatewayDockerfile is the image's Dockerfile, built from the repo root.
	gatewayDockerfile = "build/docker/Dockerfile"
	// imageEnv names the workflow env var holding the image repository.
	imageEnv = "IMAGE"
	// digestEnv names the step env var holding the pushed index digest.
	digestEnv = "DIGEST"
)

// gatewayPlatforms are the platforms the gateway image is published for.
var gatewayPlatforms = []string{"linux/amd64", "linux/arm64"}

// readGatewayImageJob returns the release workflow and its gateway-image job.
func readGatewayImageJob(t *testing.T) (workflowFile, workflowJob) {
	t.Helper()

	wf := readReleaseWorkflow(t)
	job, ok := wf.Jobs[gatewayImageJob]
	if !ok {
		t.Fatalf("%s has no %s job; nothing publishes %s", releaseWorkflowPath, gatewayImageJob, container.ImageRepository)
	}
	return wf, job
}

// stepIndex returns the index of the first step of job whose run script
// contains every one of subs, or -1.
func stepIndex(job workflowJob, subs ...string) int {
	for i, s := range job.Steps {
		if containsAll(s.Run, subs...) {
			return i
		}
	}
	return -1
}

// containsAll reports whether s contains every one of subs.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// digestOutputRe is how a step reads the index digest the manifest step
// publishes as an output.
var digestOutputRe = regexp.MustCompile(`^\$\{\{ steps\.[a-z0-9-]+\.outputs\.digest \}\}$`)

// TestGatewayImagePublishedSignedAttested is the U26-04 regression test: the
// release publishes the image the generator references, for amd64 and
// arm64, under the version tag (no "v") and :latest, then signs and attests
// the multi-arch index by digest.
func TestGatewayImagePublishedSignedAttested(t *testing.T) {
	t.Parallel()

	wf, job := readGatewayImageJob(t)

	cfg := branding.Get()
	wantRepo := "ghcr.io/" + strings.ToLower(cfg.GitHubOwner+"/"+cfg.GitHubRepo)
	if container.ImageRepository != wantRepo {
		t.Errorf("container.ImageRepository = %q, want %q (the GHCR package of %s)", container.ImageRepository, wantRepo, branding.RepoURL())
	}
	if got := wf.Env[imageEnv]; got != container.ImageRepository {
		t.Errorf("%s env %s = %q, want container.ImageRepository %q", releaseWorkflowPath, imageEnv, got, container.ImageRepository)
	}

	if !slices.Contains(job.Needs, verifyCIJob) {
		t.Errorf("job %s: needs %v, want %s", gatewayImageJob, job.Needs, verifyCIJob)
	}
	for _, perm := range []string{"packages", "id-token", "attestations"} {
		if got := job.Permissions[perm]; got != "write" {
			t.Errorf("job %s: permissions %s = %q, want write", gatewayImageJob, perm, got)
		}
	}
	for perm, level := range job.Permissions {
		if level == "write" && !slices.Contains([]string{"packages", "id-token", "attestations"}, perm) {
			t.Errorf("job %s: permissions %s: write; the image job writes only packages, id-token and attestations", gatewayImageJob, perm)
		}
	}
	if !digestOutputRe.MatchString(job.Outputs["digest"]) {
		t.Errorf("job %s: outputs.digest = %q, want the manifest step's digest output", gatewayImageJob, job.Outputs["digest"])
	}

	build := stepIndex(job, "docker build", "-f "+gatewayDockerfile, "--platform", "docker push")
	if build < 0 {
		t.Fatalf("job %s: no step builds and pushes %s per platform", gatewayImageJob, gatewayDockerfile)
	}
	for _, p := range gatewayPlatforms {
		if !strings.Contains(job.Steps[build].Run, p) {
			t.Errorf("job %s: step %q does not build %s", gatewayImageJob, job.Steps[build].Name, p)
		}
	}
	if v := stepIndex(job, `VERSION=${GITHUB_REF_NAME#v}`, "GITHUB_ENV"); v < 0 || v > build {
		t.Errorf("job %s: no step before the build sets VERSION to the tag without its v, the tag ImageForVersion pins", gatewayImageJob)
	}

	manifest := stepIndex(job, "docker manifest create", "docker manifest push", `"${IMAGE}:${VERSION}"`, `"${IMAGE}:latest"`, "GITHUB_OUTPUT")
	if manifest < 0 {
		t.Fatalf("job %s: no step pushes the %s:${VERSION} and :latest index and outputs its digest", gatewayImageJob, imageEnv)
	}
	if manifest < build {
		t.Errorf("job %s: the index is pushed before its per-platform images", gatewayImageJob)
	}

	sign := stepIndex(job, "cosign sign", "--yes", "--recursive", `"${IMAGE}@${DIGEST}"`)
	if sign < 0 {
		t.Fatalf("job %s: no step runs cosign sign --yes --recursive on ${IMAGE}@${DIGEST}", gatewayImageJob)
	}
	if !digestOutputRe.MatchString(job.Steps[sign].Env[digestEnv]) {
		t.Errorf("job %s: cosign sign env %s = %q, want the manifest step's digest output", gatewayImageJob, digestEnv, job.Steps[sign].Env[digestEnv])
	}

	attested := false
	for i, s := range job.Steps {
		if !strings.HasPrefix(s.Uses, actionAttestProvenance) {
			continue
		}
		attested = true
		if i < manifest {
			t.Errorf("job %s: attestation runs before the index is pushed", gatewayImageJob)
		}
		if got := s.With["subject-name"]; got != "${{ env."+imageEnv+" }}" {
			t.Errorf("job %s: attest subject-name = %q, want ${{ env.%s }}", gatewayImageJob, got, imageEnv)
		}
		if got := s.With["subject-digest"]; !digestOutputRe.MatchString(got) {
			t.Errorf("job %s: attest subject-digest = %q, want the manifest step's digest output", gatewayImageJob, got)
		}
		if got := s.With["push-to-registry"]; got != "true" {
			t.Errorf("job %s: attest push-to-registry = %q, want true", gatewayImageJob, got)
		}
	}
	if !attested {
		t.Errorf("job %s: no %s step", gatewayImageJob, strings.TrimSuffix(actionAttestProvenance, "@"))
	}
}

// flagValue returns the value of a --flag "value" or --flag=value argument in
// a shell script, unquoted, or "" when the flag is absent.
func flagValue(script, flag string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(flag) + `[= ]+("([^"]*)"|'([^']*)'|(\S+))`)
	m := re.FindStringSubmatch(script)
	if m == nil {
		return ""
	}
	return m[2] + m[3] + m[4]
}

// TestGatewayImageIdentityMatchesReleaseWorkflow pins the gateway image's
// signature to the identity self-update already trusts: the job verifies
// its own signature with the exact certificate identity
// branding.ReleaseWorkflowIdentity produces for the pushed tag (not a
// regexp) and the GitHub Actions OIDC issuer, so a signature from any other
// workflow, ref or repository fails the release.
func TestGatewayImageIdentityMatchesReleaseWorkflow(t *testing.T) {
	t.Parallel()

	if filepath.Base(releaseWorkflowPath) != branding.ReleaseWorkflow {
		t.Fatalf("release workflow %s is not branding.ReleaseWorkflow %s", releaseWorkflowPath, branding.ReleaseWorkflow)
	}
	_, job := readGatewayImageJob(t)

	sign := stepIndex(job, "cosign sign")
	verify := stepIndex(job, "cosign verify")
	if verify < 0 {
		t.Fatalf("job %s: no step runs cosign verify on the signed image", gatewayImageJob)
	}
	if verify < sign {
		t.Errorf("job %s: cosign verify runs before cosign sign", gatewayImageJob)
	}
	s := job.Steps[verify]
	if strings.Contains(s.Run, "--certificate-identity-regexp") || strings.Contains(s.Run, "--certificate-oidc-issuer-regexp") {
		t.Errorf("job %s: cosign verify matches the identity by regexp; pin the exact identity", gatewayImageJob)
	}
	issuer, identity := branding.ReleaseWorkflowIdentity("${GITHUB_REF_NAME}")
	if got := flagValue(s.Run, "--certificate-identity"); got != identity {
		t.Errorf("job %s: cosign verify --certificate-identity = %q, want %q (branding.ReleaseWorkflowIdentity)", gatewayImageJob, got, identity)
	}
	if got := flagValue(s.Run, "--certificate-oidc-issuer"); got != issuer {
		t.Errorf("job %s: cosign verify --certificate-oidc-issuer = %q, want %q", gatewayImageJob, got, issuer)
	}
	if !strings.Contains(s.Run, `"${IMAGE}@${DIGEST}"`) || !digestOutputRe.MatchString(s.Env[digestEnv]) {
		t.Errorf("job %s: cosign verify does not check ${IMAGE}@${DIGEST} of the pushed index", gatewayImageJob)
	}
}

const (
	// actionGoreleaser is the step that builds and publishes the binaries.
	actionGoreleaser = "goreleaser/goreleaser-action@"
	// gatewayDigestEnv carries the gateway image's index digest into
	// GoReleaser's ldflags.
	gatewayDigestEnv = "GATEWAY_IMAGE_DIGEST"
	// gatewayDigestVar is the internal/version var the digest is stamped into.
	gatewayDigestVar = "github.com/Quantum-Serendipity/qsdev/internal/version.gatewayImageDigest"
)

// TestReleaseStampsGatewayDigest pins the digest plumbing that makes the
// generated compose digest-pinned: the gateway-image job outputs the index
// digest, every GoReleaser run in the release workflow waits for it and
// passes it as GATEWAY_IMAGE_DIGEST, and .goreleaser.yaml stamps that env
// into internal/version with envOrDefault so a snapshot build without it
// still renders.
func TestReleaseStampsGatewayDigest(t *testing.T) {
	t.Parallel()

	wf, gateway := readGatewayImageJob(t)
	if !digestOutputRe.MatchString(gateway.Outputs["digest"]) {
		t.Errorf("job %s: outputs.digest = %q, want the manifest step's digest output", gatewayImageJob, gateway.Outputs["digest"])
	}

	wantEnv := "${{ needs." + gatewayImageJob + ".outputs.digest }}"
	runs := 0
	for name, job := range wf.Jobs {
		for _, s := range job.Steps {
			if !strings.HasPrefix(s.Uses, actionGoreleaser) {
				continue
			}
			runs++
			if !slices.Contains(job.Needs, gatewayImageJob) {
				t.Errorf("job %s: runs GoReleaser but needs %v, not %s", name, job.Needs, gatewayImageJob)
			}
			if got := s.Env[gatewayDigestEnv]; got != wantEnv {
				t.Errorf("job %s: GoReleaser env %s = %q, want %q", name, gatewayDigestEnv, got, wantEnv)
			}
		}
	}
	if runs == 0 {
		t.Fatalf("%s runs no %s step; the parser or the layout changed", releaseWorkflowPath, strings.TrimSuffix(actionGoreleaser, "@"))
	}

	var cfg struct {
		Builds []struct {
			ID      string   `yaml:"id"`
			Ldflags []string `yaml:"ldflags"`
		} `yaml:"builds"`
	}
	readRepoYAML(t, ".goreleaser.yaml", &cfg)
	if len(cfg.Builds) == 0 {
		t.Fatal(".goreleaser.yaml has no builds; the parser or the layout changed")
	}
	want := "-X " + gatewayDigestVar + `={{ envOrDefault "` + gatewayDigestEnv + `" "" }}`
	for _, b := range cfg.Builds {
		if !slices.Contains(b.Ldflags, want) {
			t.Errorf(".goreleaser.yaml build %s: ldflags %q do not contain %q", b.ID, b.Ldflags, want)
		}
	}
}
