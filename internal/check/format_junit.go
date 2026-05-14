package check

import (
	"encoding/xml"
	"fmt"
	"io"
)

// JUnit XML types.

type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Skipped  int             `xml:"skipped,attr,omitempty"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

func formatJUnit(report *CheckReport, w io.Writer) error {
	// Group checks by category.
	byCategory := make(map[CheckCategory][]CheckResult)
	for _, r := range report.Checks {
		byCategory[r.Category] = append(byCategory[r.Category], r)
	}

	var suites junitTestSuites

	for _, cat := range categoryOrder {
		results, ok := byCategory[cat]
		if !ok || len(results) == 0 {
			continue
		}

		suite := junitTestSuite{
			Name:  string(cat),
			Tests: len(results),
		}

		for _, r := range results {
			tc := junitTestCase{
				Name:      r.Name,
				ClassName: string(r.Category),
			}

			switch r.Status {
			case StatusFail:
				suite.Failures++
				msg := r.Message
				if r.Remediation != "" {
					msg += "\n" + r.Remediation
				}
				tc.Failure = &junitFailure{
					Message: r.Message,
					Type:    string(r.Severity),
					Content: msg,
				}
			case StatusSkip:
				suite.Skipped++
				tc.Skipped = &junitSkipped{
					Message: r.Message,
				}
			}

			suite.Cases = append(suite.Cases, tc)
		}

		suites.TestSuites = append(suites.TestSuites, suite)
	}

	// Write XML header.
	fmt.Fprint(w, xml.Header)

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}

	// Trailing newline.
	_, err := fmt.Fprintln(w)
	return err
}
