package cigeneration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// maxImagePinAge bounds how far an image pin may fall behind its tag. Image
// tags are rebuilt upstream (snyk/snyk:node is rebuilt daily), so the pin
// cannot be required to match the tag the way an action SHA must match its
// release tag; that check would fail every day. Instead a pin whose tag has
// moved on fails once the pinned image is older than this, which keeps base
// image and CLI fixes flowing into generated workflows without Dependabot,
// which cannot see this catalog.
const maxImagePinAge = 90 * 24 * time.Hour

// manifestAccept lists the manifest media types a registry may serve for an
// image reference: multi-platform indexes and single-platform manifests.
const manifestAccept = "application/vnd.oci.image.index.v1+json, " +
	"application/vnd.docker.distribution.manifest.list.v2+json, " +
	"application/vnd.oci.image.manifest.v1+json, " +
	"application/vnd.docker.distribution.manifest.v2+json"

// maxRegistryBody caps how much of a registry response is read.
const maxRegistryBody = 4 << 20

// registryClient speaks just enough of the OCI distribution API to verify a
// pin: anonymous bearer-token auth, manifest and blob reads.
type registryClient struct {
	http    *http.Client
	authURL string // token endpoint; empty means the registry needs no auth
	service string
	baseURL string
}

// dockerHub is the registry every ImageRef in the catalog lives in.
func dockerHub() registryClient {
	return registryClient{
		http:    &http.Client{Timeout: 30 * time.Second},
		authURL: "https://auth.docker.io/token",
		service: "registry.docker.io",
		baseURL: "https://registry-1.docker.io",
	}
}

// imagePinStatus is what the registry says about a pin today.
type imagePinStatus struct {
	// TagDigest is the digest the pin's tag resolves to now.
	TagDigest string
	// Created is the creation time of the pinned linux/amd64 image. It is
	// fetched only when the tag has moved away from the pin.
	Created time.Time
}

func (c registryClient) token(ctx context.Context, repo string) (string, error) {
	if c.authURL == "" {
		return "", nil
	}
	q := url.Values{"service": {c.service}, "scope": {"repository:" + repo + ":pull"}}
	body, _, err := c.get(ctx, http.MethodGet, c.authURL+"?"+q.Encode(), "", "")
	if err != nil {
		return "", fmt.Errorf("fetching registry token: %w", err)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("decoding registry token: %w", err)
	}
	return tok.Token, nil
}

var errNotFound = errors.New("not found")

// get performs one registry request and returns the body and headers of a 200
// response. A 404 is reported as errNotFound so callers can word it.
func (c registryClient) get(ctx context.Context, method, rawURL, token, accept string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("building request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s: %w", method, rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRegistryBody))
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", rawURL, err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return body, resp.Header, nil
	case http.StatusNotFound:
		return nil, nil, fmt.Errorf("%s %s: %w", method, rawURL, errNotFound)
	default:
		return nil, nil, fmt.Errorf("%s %s: HTTP %d", method, rawURL, resp.StatusCode)
	}
}

// manifestByDigest fetches a manifest and checks that its bytes hash to the
// digest asked for, so the pin is verified by content rather than by a header.
func (c registryClient) manifestByDigest(ctx context.Context, repo, digest, token string) ([]byte, error) {
	body, _, err := c.get(ctx, http.MethodGet, c.baseURL+"/v2/"+repo+"/manifests/"+digest, token, manifestAccept)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != digest {
		return nil, fmt.Errorf("manifest for %s hashes to %s", digest, got)
	}
	return body, nil
}

type manifestDoc struct {
	MediaType string `json:"mediaType"`
	Manifests []struct {
		Digest   string `json:"digest"`
		Platform struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
}

// platformManifest resolves a pinned index to its linux/amd64 manifest, the
// platform of GitHub-hosted runners. A pin to a single-platform manifest is
// returned unchanged.
func (c registryClient) platformManifest(ctx context.Context, repo string, body []byte, token string) (manifestDoc, error) {
	var doc manifestDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return manifestDoc{}, fmt.Errorf("decoding manifest: %w", err)
	}
	if len(doc.Manifests) == 0 {
		return doc, nil
	}
	for _, m := range doc.Manifests {
		if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
			sub, err := c.manifestByDigest(ctx, repo, m.Digest, token)
			if err != nil {
				return manifestDoc{}, fmt.Errorf("fetching linux/amd64 manifest: %w", err)
			}
			var platform manifestDoc
			if err := json.Unmarshal(sub, &platform); err != nil {
				return manifestDoc{}, fmt.Errorf("decoding linux/amd64 manifest: %w", err)
			}
			return platform, nil
		}
	}
	return manifestDoc{}, errors.New("image index has no linux/amd64 manifest, so GitHub-hosted runners cannot run it")
}

// checkImagePin verifies that ref.Digest exists with that exact content and
// runs on linux/amd64, and reports where ref.Tag points now.
func (c registryClient) checkImagePin(ctx context.Context, ref ImageRef) (imagePinStatus, error) {
	token, err := c.token(ctx, ref.Repository)
	if err != nil {
		return imagePinStatus{}, err
	}

	pinned, err := c.manifestByDigest(ctx, ref.Repository, ref.Digest, token)
	if err != nil {
		return imagePinStatus{}, fmt.Errorf("pinned digest %s does not resolve in %s: %w", ref.Digest, ref.Repository, err)
	}
	platform, err := c.platformManifest(ctx, ref.Repository, pinned, token)
	if err != nil {
		return imagePinStatus{}, err
	}

	_, hdr, err := c.get(ctx, http.MethodHead, c.baseURL+"/v2/"+ref.Repository+"/manifests/"+ref.Tag, token, manifestAccept)
	if err != nil {
		return imagePinStatus{}, fmt.Errorf("resolving tag %s: %w", ref.Tag, err)
	}
	st := imagePinStatus{TagDigest: hdr.Get("Docker-Content-Digest")}
	if st.TagDigest == "" {
		return imagePinStatus{}, fmt.Errorf("registry returned no digest for tag %s", ref.Tag)
	}
	if st.TagDigest == ref.Digest {
		return st, nil
	}

	blob, _, err := c.get(ctx, http.MethodGet, c.baseURL+"/v2/"+ref.Repository+"/blobs/"+platform.Config.Digest, token, "")
	if err != nil {
		return imagePinStatus{}, fmt.Errorf("fetching image config: %w", err)
	}
	var cfg struct {
		Created time.Time `json:"created"`
	}
	if err := json.Unmarshal(blob, &cfg); err != nil {
		return imagePinStatus{}, fmt.Errorf("decoding image config: %w", err)
	}
	st.Created = cfg.Created
	return st, nil
}

// imagePinDrift reports whether a pin has fallen too far behind its tag.
func imagePinDrift(ref ImageRef, st imagePinStatus, now time.Time) error {
	if st.TagDigest == ref.Digest {
		return nil
	}
	if age := now.Sub(st.Created); age > maxImagePinAge {
		return fmt.Errorf("pinned image is %d days old and %s:%s has moved on; "+
			"bump the pin to %s after reviewing the new image",
			int(age.Hours()/24), ref.Repository, ref.Tag, st.TagDigest)
	}
	return nil
}

// TestImagePinsResolveUpstream is the image counterpart of
// TestActionPinsResolveUpstream: every pinned digest must exist in the
// registry with that exact content and run on GitHub-hosted runners, and must
// not have fallen more than maxImagePinAge behind its tag.
//
// Network-dependent, so it is opt-in: set VERIFY_ACTION_PINS=1. CI runs it in
// the pin-audit job alongside the action pin check.
func TestImagePinsResolveUpstream(t *testing.T) {
	if os.Getenv("VERIFY_ACTION_PINS") == "" {
		t.Skip("set VERIFY_ACTION_PINS=1 to verify image pins against the registry")
	}

	reg := dockerHub()
	for name, ref := range allImagePins() {
		t.Run(name, func(t *testing.T) {
			st, err := reg.checkImagePin(t.Context(), ref)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if err := imagePinDrift(ref, st, time.Now()); err != nil {
				t.Errorf("%s: %v", name, err)
			}
			if st.TagDigest != ref.Digest {
				t.Logf("%s: %s:%s now resolves to %s (pin is from %s)",
					name, ref.Repository, ref.Tag, st.TagDigest, st.Created.Format(time.DateOnly))
			}
		})
	}
}
