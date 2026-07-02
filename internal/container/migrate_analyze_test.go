package container

import "testing"

// TestQualifyImageName_PrivateRegistry is a regression test for BL-P1-14. The
// previous implementation used LastIndex(":") for the tag and only treated an
// image as already-qualified when the name part contained "." or started
// "localhost/". That corrupted private-registry references carrying a
// host:port, e.g. "registry:5000/myapp:v1" became
// "docker.io/registry:5000/myapp:v1". These cases are NOT covered by the
// existing TestQualifyImageName in migrate_test.go.
func TestQualifyImageName_PrivateRegistry(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		// Registry host detected in the first "/"-segment => left unqualified.
		{"host_port_registry", "registry:5000/myapp:v1", "registry:5000/myapp:v1"},
		{"host_port_registry_no_tag", "registry:5000/myapp", "registry:5000/myapp"},
		{"localhost_port", "localhost:5000/app", "localhost:5000/app"},

		// Unqualified references still get docker.io added.
		{"bare_with_tag", "myapp:v1", "docker.io/library/myapp:v1"},
		{"bare_digest", "nginx@sha256:abc123", "docker.io/library/nginx@sha256:abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := qualifyImageName(tt.image); got != tt.want {
				t.Errorf("qualifyImageName(%q) = %q, want %q", tt.image, got, tt.want)
			}
		})
	}
}
