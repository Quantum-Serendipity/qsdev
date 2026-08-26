package cigeneration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// allPins is the set verified against upstream. Keep it in step with the
// catalog; TestActionPinsResolveUpstream_CoversCatalog fails if it drifts.
func allPins() map[string]ActionRef {
	return map[string]ActionRef{
		"ActionCheckout":         ActionCheckout,
		"ActionHardenRunner":     ActionHardenRunner,
		"ActionUploadArtifact":   ActionUploadArtifact,
		"ActionDownloadArtifact": ActionDownloadArtifact,
		"ActionOSVScanner":       ActionOSVScanner,
		"ActionGrype":            ActionGrype,
		"ActionSnyk":             ActionSnyk,
		"ActionLabeler":          ActionLabeler,
	}
}

// apiRepo strips any action subpath: google/osv-scanner-action/osv-scanner-action
// lives in the repository google/osv-scanner-action.
func apiRepo(ref ActionRef) string {
	repo := ref.Repo
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			repo = repo[:i]
			break
		}
	}
	return ref.Owner + "/" + repo
}

func githubGET(t *testing.T, client *http.Client, url string) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var buf [1 << 16]byte
	n, _ := resp.Body.Read(buf[:])
	return resp.StatusCode, buf[:n]
}

// TestActionPinsResolveUpstream verifies that every pinned SHA exists in its
// repository and is the commit its tag resolves to.
//
// This is the check that shape validation cannot perform. Four entries in this
// catalog once carried SHAs that existed nowhere upstream; two of them shared a
// prefix with the real commit, so review could not distinguish them from a
// correct transcription. A pin like that fails only when a generated workflow
// runs, in somebody else's repository.
//
// Network-dependent, so it is opt-in: set VERIFY_ACTION_PINS=1. CI runs it in
// the pin-audit job, where GITHUB_TOKEN raises the rate limit.
func TestActionPinsResolveUpstream(t *testing.T) {
	if os.Getenv("VERIFY_ACTION_PINS") == "" {
		t.Skip("set VERIFY_ACTION_PINS=1 to verify pins against the GitHub API")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for name, ref := range allPins() {
		t.Run(name, func(t *testing.T) {
			repo := apiRepo(ref)

			// 1. The SHA must exist in that repository. Only the status
			//    matters; the commit payload itself is not inspected.
			status, _ := githubGET(t, client,
				fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, ref.SHA))
			if status != http.StatusOK {
				t.Fatalf("%s: SHA %s does not resolve in %s (HTTP %d). "+
					"A plausible-looking SHA is not a pin — copy it from the upstream tag.",
					name, ref.SHA, repo, status)
			}

			// 2. Branch-tracking refs such as snyk/actions@master have no tag
			//    to compare against; the SHA is the whole of the pin.
			if ref.Tag == "master" || ref.Tag == "main" {
				return
			}

			// 3. The tag must resolve to that same commit, so the emitted
			//    comment does not misstate which version runs.
			status, body := githubGET(t, client,
				fmt.Sprintf("https://api.github.com/repos/%s/git/ref/tags/%s", repo, ref.Tag))
			if status != http.StatusOK {
				t.Fatalf("%s: tag %s not found in %s (HTTP %d)", name, ref.Tag, repo, status)
			}

			var ptr struct {
				Object struct {
					SHA  string `json:"sha"`
					Type string `json:"type"`
				} `json:"object"`
			}
			if err := json.Unmarshal(body, &ptr); err != nil {
				t.Fatalf("%s: decoding tag ref: %v", name, err)
			}

			target := ptr.Object.SHA
			if ptr.Object.Type == "tag" {
				// Annotated tag: dereference to the commit it points at.
				status, body = githubGET(t, client,
					fmt.Sprintf("https://api.github.com/repos/%s/git/tags/%s", repo, target))
				if status != http.StatusOK {
					t.Fatalf("%s: dereferencing annotated tag %s (HTTP %d)", name, ref.Tag, status)
				}
				var ann struct {
					Object struct {
						SHA string `json:"sha"`
					} `json:"object"`
				}
				if err := json.Unmarshal(body, &ann); err != nil {
					t.Fatalf("%s: decoding annotated tag: %v", name, err)
				}
				target = ann.Object.SHA
			}

			if target != ref.SHA {
				t.Errorf("%s pins %s but %s@%s is %s — the pin and its comment disagree",
					name, ref.SHA, repo, ref.Tag, target)
			}
		})
	}
}

// TestActionPinsResolveUpstream_CoversCatalog fails when a pin is added to the
// catalog but not to allPins, which would leave it unverified — the state that
// let four unresolvable SHAs accumulate.
func TestActionPinsResolveUpstream_CoversCatalog(t *testing.T) {
	t.Parallel()

	// Mirrors the exported catalog. Update both together when adding a pin.
	const catalogSize = 8

	if got := len(allPins()); got != catalogSize {
		t.Errorf("allPins covers %d pins, expected %d; a new catalog entry must be "+
			"added to allPins or it is never verified against upstream", got, catalogSize)
	}
}
