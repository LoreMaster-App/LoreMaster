package testreporting

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type xmlSuites struct {
	Suites []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Failures  int        `xml:"failures,attr"`
	Errors    int        `xml:"errors,attr"`
	Skipped   int        `xml:"skipped,attr"`
	Time      string     `xml:"time,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	Cases     []xmlCase  `xml:"testcase"`
	Suites    []xmlSuite `xml:"testsuite"`
}

type xmlCase struct {
	Name      string       `xml:"name,attr"`
	ClassName string       `xml:"classname,attr"`
	Time      string       `xml:"time,attr"`
	Failures  []xmlProblem `xml:"failure"`
	Errors    []xmlProblem `xml:"error"`
	Skipped   []xmlProblem `xml:"skipped"`
}

type xmlProblem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// ParseJUnit reads one JUnit XML report. The root may be <testsuites> or a single
// <testsuite>; suites nested in suites become suites of their own, named "outer / inner".
// source is the report's workspace-relative path, kept on each suite. A suite with no tests
// at all is left out.
func ParseJUnit(data []byte, source string) ([]Suite, error) {
	root, err := rootElement(data)
	if err != nil {
		return nil, err
	}
	var top []xmlSuite
	switch root {
	case "testsuites":
		var suites xmlSuites
		if err := xml.Unmarshal(data, &suites); err != nil {
			return nil, fmt.Errorf("not valid JUnit XML: %w", err)
		}
		top = suites.Suites
	case "testsuite":
		var suite xmlSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("not valid JUnit XML: %w", err)
		}
		top = []xmlSuite{suite}
	default:
		return nil, fmt.Errorf("not a JUnit report: the root element is <%s>, expected <testsuites> or <testsuite>", root)
	}

	var suites []Suite
	for _, suite := range top {
		suites = append(suites, flatten(suite, "", source)...)
	}

	return suites, nil
}

// rootElement names the first element of the document.
func rootElement(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return "", errors.New("not a JUnit report: the file has no elements")
		}
		if err != nil {
			return "", fmt.Errorf("not valid JUnit XML: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok {
			return start.Name.Local, nil
		}
	}
}

func flatten(suite xmlSuite, prefix string, source string) []Suite {
	name := strings.TrimSpace(suite.Name)
	if name == "" {
		name = "(unnamed suite)"
	}
	name = prefix + name

	var suites []Suite
	if converted := convert(suite, name, source); converted.Tests > 0 {
		suites = append(suites, converted)
	}
	for _, child := range suite.Suites {
		suites = append(suites, flatten(child, name+" / ", source)...)
	}

	return suites
}

func convert(suite xmlSuite, name string, source string) Suite {
	converted := Suite{Name: name, Source: source, Timestamp: strings.TrimSpace(suite.Timestamp)}
	for _, test := range suite.Cases {
		converted.Cases = append(converted.Cases, convertCase(test))
	}
	if len(converted.Cases) == 0 {
		converted.Tests, converted.Failures, converted.Errors, converted.SkippedCount = suite.Tests, suite.Failures, suite.Errors, suite.Skipped
		converted.Seconds = seconds(suite.Time)

		return converted
	}
	for _, test := range converted.Cases {
		converted.Tests++
		converted.Seconds += test.Seconds
		switch test.Status {
		case Failed:
			converted.Failures++
		case Errored:
			converted.Errors++
		case Skipped:
			converted.SkippedCount++
		case Passed:
		}
	}

	return converted
}

func convertCase(test xmlCase) Case {
	converted := Case{Name: strings.TrimSpace(test.Name), ClassName: strings.TrimSpace(test.ClassName), Seconds: seconds(test.Time), Status: Passed}
	switch {
	case len(test.Errors) > 0:
		converted.Status, converted.Message, converted.Details = Errored, joinMessages(test.Errors), joinDetails(test.Errors)
	case len(test.Failures) > 0:
		converted.Status, converted.Message, converted.Details = Failed, joinMessages(test.Failures), joinDetails(test.Failures)
	case len(test.Skipped) > 0:
		converted.Status, converted.Message = Skipped, joinMessages(test.Skipped)
	}

	return converted
}

func joinMessages(problems []xmlProblem) string {
	var messages []string
	for _, problem := range problems {
		if message := strings.TrimSpace(problem.Message); message != "" {
			messages = append(messages, message)
		}
	}

	return strings.Join(messages, "; ")
}

func joinDetails(problems []xmlProblem) string {
	var details []string
	for _, problem := range problems {
		if text := strings.TrimSpace(problem.Text); text != "" {
			details = append(details, text)
		}
	}

	return strings.Join(details, "\n\n")
}

// seconds reads a duration attribute; some runners write a thousands separator.
func seconds(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(value), ",", ""), 64)
	if err != nil || parsed < 0 {
		return 0
	}

	return parsed
}
