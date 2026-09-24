package packaging_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// field returns the first capture of re in content, failing when absent.
func field(t *testing.T, content, file string, re *regexp.Regexp) string {
	t.Helper()
	m := re.FindStringSubmatch(content)
	if m == nil {
		t.Fatalf("%s: no match for %s", file, re)
	}
	return m[1]
}

// readFile returns the file with CRLF line endings normalized: a Windows
// checkout converts these text files (.gitattributes has text=auto).
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// versionParts parses "X.Y.Z" into comparable integers.
func versionParts(t *testing.T, v string) [3]int {
	t.Helper()
	var out [3]int
	parts := strings.SplitN(strings.SplitN(v, "-", 2)[0], ".", 3)
	if len(parts) != 3 {
		t.Fatalf("malformed version %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("malformed version %q: %v", v, err)
		}
		out[i] = n
	}
	return out
}

// TestAURPackageIntegrity is the regression test for W187: the committed
// PKGBUILD disabled integrity checking (sha256sums=SKIP) and .SRCINFO had
// drifted from it. Every source must carry a real checksum, and .SRCINFO (what
// the AUR actually reads) must describe the same package as PKGBUILD.
func TestAURPackageIntegrity(t *testing.T) {
	t.Parallel()
	pkgbuild := readFile(t, filepath.Join("aur", "PKGBUILD"))
	srcinfo := readFile(t, filepath.Join("aur", ".SRCINFO"))

	pkgver := field(t, pkgbuild, "PKGBUILD", regexp.MustCompile(`(?m)^pkgver=(\S+)$`))
	if got := field(t, srcinfo, ".SRCINFO", regexp.MustCompile(`(?m)^\tpkgver = (\S+)$`)); got != pkgver {
		t.Errorf(".SRCINFO pkgver = %q, PKGBUILD pkgver = %q", got, pkgver)
	}

	for _, arch := range []string{"x86_64", "aarch64"} {
		sum := field(t, pkgbuild, "PKGBUILD", regexp.MustCompile(`(?m)^sha256sums_`+arch+`=\('([^']*)'\)$`))
		if !sha256Hex.MatchString(sum) {
			t.Errorf("PKGBUILD sha256sums_%s = %q, want a sha256 (never SKIP)", arch, sum)
		}
		if got := field(t, srcinfo, ".SRCINFO", regexp.MustCompile(`(?m)^\tsha256sums_`+arch+` = (\S+)$`)); got != sum {
			t.Errorf(".SRCINFO sha256sums_%s = %q, PKGBUILD has %q", arch, got, sum)
		}
		src := field(t, srcinfo, ".SRCINFO", regexp.MustCompile(`(?m)^\tsource_`+arch+` = (\S+)$`))
		if want := "/releases/download/v" + pkgver + "/qsdev_" + pkgver + "_"; !strings.Contains(src, want) {
			t.Errorf(".SRCINFO source_%s = %q, want it to contain %q", arch, src, want)
		}
	}

	// VERSION names the next release, so the package may lag it but never
	// lead it (a pkgver ahead of VERSION cannot have been published).
	next := strings.TrimSpace(readFile(t, filepath.Join("..", "VERSION")))
	p, n := versionParts(t, pkgver), versionParts(t, next)
	for i := range p {
		if p[i] != n[i] {
			if p[i] > n[i] {
				t.Errorf("PKGBUILD pkgver %s is ahead of VERSION %s", pkgver, next)
			}
			break
		}
	}
}
