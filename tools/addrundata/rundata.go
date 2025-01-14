package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type parsedData struct {
	testPlanID      string
	testDescription string
	testUUID        string
}

// markdownRE matches the heading line: `# XX-1.1: Foo Functional Test`
var markdownRE = regexp.MustCompile(`#(.*?):(.*)`)

// parseMarkdown reads parsedData from README.md
func parseMarkdown(r io.Reader) (*parsedData, error) {
	sc := bufio.NewScanner(r)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("missing markdown heading")
	}
	line := sc.Text()
	m := markdownRE.FindStringSubmatch(line)
	if len(m) < 3 {
		return nil, fmt.Errorf("cannot parse markdown: %s", line)
	}
	return &parsedData{
		testPlanID:      strings.TrimSpace(m[1]),
		testDescription: strings.TrimSpace(m[2]),
	}, nil
}

// parseCode reads parsedData from a source code.
func parseCode(r io.Reader) (*parsedData, error) {
	var pd *parsedData
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if line := sc.Text(); line != "func init() {" {
			continue
		}
		pd = new(parsedData)
		if err := pd.parseInit(sc); err != nil {
			return nil, err
		}
		break
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if pd == nil {
		return nil, errors.New("missing func init()")
	}
	return pd, nil
}

// rundataRE matches a line like this: `  rundata.TestUUID = "..."`
var rundataRE = regexp.MustCompile(`\s+rundata\.(\w+) = (".*")`)

// parseInit parses the rundata from the body of func init().
func (pd *parsedData) parseInit(sc *bufio.Scanner) error {
	for sc.Scan() {
		line := sc.Text()
		if line == "}" {
			return nil
		}
		m := rundataRE.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		k := m[1]
		v, err := strconv.Unquote(m[2])
		if err != nil {
			return fmt.Errorf("cannot parse rundata line: %s: %w", line, err)
		}
		switch k {
		case "TestPlanID":
			pd.testPlanID = v
		case "TestDescription":
			pd.testDescription = v
		case "TestUUID":
			pd.testUUID = v
		}
	}
	return errors.New("func init() was not terminated")
}

var tmpl = template.Must(template.New("rundata_test.go").Parse(
	`// Code generated by go run tools/addrundata; DO NOT EDIT.
package {{.Package}}

import "github.com/openconfig/featureprofiles/internal/rundata"

func init() {
{{range .Data}}	rundata.{{.Key}} = {{printf "%q\n" .Value}}{{end -}}
}
`))

// write generates a complete rundata_test.go to the writer.
func (pd *parsedData) write(w io.Writer, pkg string) error {
	tmpl.Execute(w, &struct {
		Package string
		Data    []struct{ Key, Value string }
	}{
		Package: pkg,
		Data: []struct{ Key, Value string }{
			{"TestPlanID", pd.testPlanID},
			{"TestDescription", pd.testDescription},
			{"TestUUID", pd.testUUID},
		},
	})
	return nil
}
