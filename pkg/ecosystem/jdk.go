package ecosystem

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// ErrUnsupportedJDKVersion is returned when a configured Java version does not
// name a JDK major release that qsdev can provision from nixpkgs.
var ErrUnsupportedJDKVersion = errors.New("unsupported JDK version")

// SupportedJDKMajors lists the JDK major releases packaged in the pinned
// nixpkgs as jdk<major>, newest first. Non-LTS releases are removed from
// nixpkgs once they reach end of life (jdk23 and jdk24 are throw-aliases that
// fail evaluation), so only releases that evaluate belong here.
var SupportedJDKMajors = []int{25, 21, 17, 11, 8}

// DefaultJDKMajor is provisioned when no Java version is configured.
const DefaultJDKMajor = 21

// jdkVersionRe matches the version spellings found in .java-version files and
// on the command line: an optional vendor/distribution prefix ("temurin-",
// "openjdk64-", "graalvm-ce-"), then the version, whose leading "1." marks the
// legacy scheme ("1.8", "1.8.0_292"). Suffixes ("17.0.2+8", "21-ea",
// "17.0.2-tem") are ignored.
var jdkVersionRe = regexp.MustCompile(`^(?:[a-z][a-z0-9_.-]*-)?(\d+)(?:\.(\d+))?`)

// JDKMajor normalizes a Java version string to its major release number. It
// returns false when no major version can be read from it.
func JDKMajor(version string) (int, bool) {
	m := jdkVersionRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(version)))
	if m == nil {
		return 0, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	if major == 1 && m[2] != "" {
		// Legacy scheme: 1.8 is Java 8.
		if major, err = strconv.Atoi(m[2]); err != nil {
			return 0, false
		}
	}
	return major, true
}

// JDKPackage maps a configured Java version to the nixpkgs JDK attribute
// (e.g. "17.0.2" -> "jdk17", "1.8" -> "jdk8"). An empty version selects
// DefaultJDKMajor. A version whose major release is not in SupportedJDKMajors
// is an error rather than a silent substitution, so the shell never provides a
// JDK other than the one the project asked for.
func JDKPackage(version string) (string, error) {
	if strings.TrimSpace(version) == "" {
		return fmt.Sprintf("jdk%d", DefaultJDKMajor), nil
	}
	major, ok := JDKMajor(version)
	if !ok || !slices.Contains(SupportedJDKMajors, major) {
		return "", fmt.Errorf("%w %q: supported versions are %s", ErrUnsupportedJDKVersion, version, supportedJDKList())
	}
	return fmt.Sprintf("jdk%d", major), nil
}

// JDKWizardOptions returns select options for every supported JDK release.
func JDKWizardOptions() []WizardOption {
	options := make([]WizardOption, 0, len(SupportedJDKMajors))
	for _, major := range SupportedJDKMajors {
		options = append(options, WizardOption{
			Label: fmt.Sprintf("JDK %d", major),
			Value: strconv.Itoa(major),
		})
	}
	return options
}

// supportedJDKList renders SupportedJDKMajors for messages.
func supportedJDKList() string {
	names := make([]string, len(SupportedJDKMajors))
	for i, major := range SupportedJDKMajors {
		names[i] = strconv.Itoa(major)
	}
	return strings.Join(names, ", ")
}
