package cigeneration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeRegistry serves a single repository over the subset of the OCI
// distribution API that checkImagePin uses.
type fakeRegistry struct {
	repo      string
	manifests map[string][]byte // by digest
	tags      map[string]string // tag -> digest
	blobs     map[string][]byte // by digest
}

func (f *fakeRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/token" {
		_, _ = w.Write([]byte(`{"token":"tok"}`))
		return
	}
	if r.Header.Get("Authorization") != "Bearer tok" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	prefix := "/v2/" + f.repo + "/"
	rest, ok := strings.CutPrefix(r.URL.Path, prefix)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch {
	case strings.HasPrefix(rest, "manifests/"):
		ref := strings.TrimPrefix(rest, "manifests/")
		if d, isTag := f.tags[ref]; isTag {
			ref = d
		}
		body, ok := f.manifests[ref]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Docker-Content-Digest", ref)
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
	case strings.HasPrefix(rest, "blobs/"):
		body, ok := f.blobs[strings.TrimPrefix(rest, "blobs/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func digestOf(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// publishImage adds a linux image created at the given time to the registry
// and returns the digests of its index and of its platform manifest.
func publishImage(t *testing.T, f *fakeRegistry, created time.Time, arch string) (index, platform string) {
	t.Helper()

	cfg := mustJSON(t, map[string]any{"created": created.Format(time.RFC3339Nano)})
	f.blobs[digestOf(cfg)] = cfg

	man := mustJSON(t, map[string]any{
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"config":    map[string]any{"digest": digestOf(cfg)},
	})
	platform = digestOf(man)
	f.manifests[platform] = man

	idx := mustJSON(t, map[string]any{
		"mediaType": "application/vnd.oci.image.index.v1+json",
		"manifests": []map[string]any{{
			"digest":   platform,
			"platform": map[string]string{"os": "linux", "architecture": arch},
		}},
	})
	index = digestOf(idx)
	f.manifests[index] = idx
	return index, platform
}

// TestCheckImagePin exercises the registry verification that
// TestImagePinsResolveUpstream runs against Docker Hub, so its failure modes
// are covered without network access.
func TestCheckImagePin(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	recent := now.Add(-10 * 24 * time.Hour)
	stale := now.Add(-maxImagePinAge - 24*time.Hour)

	tests := []struct {
		name string
		// setup publishes images and returns the pin to check.
		setup     func(t *testing.T, f *fakeRegistry) ImageRef
		wantErr   string // checkImagePin error substring
		wantDrift bool
	}{
		{
			name: "tag still at pin",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, stale, "amd64")
				f.tags["node"] = idx
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
		},
		{
			name: "tag moved, pin recent",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, recent, "amd64")
				f.tags["node"], _ = publishImage(t, f, now, "amd64")
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
		},
		{
			name: "tag moved, pin stale",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, stale, "amd64")
				f.tags["node"], _ = publishImage(t, f, now, "amd64")
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
			wantDrift: true,
		},
		{
			name: "single-platform manifest pin",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				_, man := publishImage(t, f, recent, "amd64")
				f.tags["node"] = man
				return ImageRef{Repository: f.repo, Digest: man, Tag: "node"}
			},
		},
		{
			name: "digest does not exist",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				f.tags["node"], _ = publishImage(t, f, now, "amd64")
				return ImageRef{Repository: f.repo, Digest: "sha256:" + strings.Repeat("0", 64), Tag: "node"}
			},
			wantErr: "does not resolve",
		},
		{
			name: "content does not match digest",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, now, "amd64")
				f.tags["node"] = idx
				f.manifests[idx] = []byte(`{"mediaType":"tampered"}`)
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
			wantErr: "hashes to",
		},
		{
			name: "no linux/amd64 image",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, now, "arm64")
				f.tags["node"] = idx
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
			wantErr: "linux/amd64",
		},
		{
			name: "tag removed",
			setup: func(t *testing.T, f *fakeRegistry) ImageRef {
				idx, _ := publishImage(t, f, now, "amd64")
				return ImageRef{Repository: f.repo, Digest: idx, Tag: "node"}
			},
			wantErr: "resolving tag node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := &fakeRegistry{
				repo:      "vendor/tool",
				manifests: map[string][]byte{},
				tags:      map[string]string{},
				blobs:     map[string][]byte{},
			}
			ref := tt.setup(t, f)
			srv := httptest.NewServer(f)
			t.Cleanup(srv.Close)

			reg := registryClient{http: srv.Client(), authURL: srv.URL + "/token", service: "test", baseURL: srv.URL}
			st, err := reg.checkImagePin(t.Context(), ref)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("checkImagePin error = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("checkImagePin: %v", err)
			}

			drift := imagePinDrift(ref, st, now)
			if gotDrift := drift != nil; gotDrift != tt.wantDrift {
				t.Errorf("imagePinDrift = %v, want drift %v", drift, tt.wantDrift)
			}
			if drift != nil && !strings.Contains(drift.Error(), st.TagDigest) {
				t.Errorf("drift error %q should name the digest to bump to (%s)", drift, st.TagDigest)
			}
		})
	}
}

func TestRegistryGetReportsNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)

	reg := registryClient{http: srv.Client(), baseURL: srv.URL}
	_, _, err := reg.get(t.Context(), http.MethodGet, srv.URL+"/v2/x/manifests/y", "", "")
	if !errors.Is(err, errNotFound) {
		t.Errorf("get error = %v, want errNotFound", err)
	}
}
