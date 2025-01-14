package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	markdownText = `# XX-1.1: Description from markdown

## Summary

## Procedure
`
	testCode = `// License text line 1.
// License text line 2.

package foo_functional_test

import (
"testing"

"github.com/openconfig/featureprofiles/internal/fptest"
)

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}
`
	rundataCode = `// Code generated by go run tools/addrundata; DO NOT EDIT.
package foo_functional_test

import "github.com/openconfig/featureprofiles/internal/rundata"

func init() {
	rundata.TestPlanID = "YY-1.1"
	rundata.TestDescription = "Description from code"
	rundata.TestUUID = "123e4567-e89b-42d3-8456-426614174000"
}
`
)

var tcopt = cmp.AllowUnexported(testcase{}, parsedData{})

func TestCase_Read(t *testing.T) {
	tests := []struct {
		desc         string
		markdownText string
		testCode     string
		rundataCode  string
		want         testcase
		wantErr      string
	}{{
		desc:    "empty",
		wantErr: "no such file",
	}, {
		desc:         "bad markdown",
		markdownText: "~!@#$%^&*()_+",
		wantErr:      "parse markdown",
	}, {
		desc:         "no tests",
		markdownText: markdownText,
		want: testcase{
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Description from markdown",
			},
		},
	}, {
		desc:         "good markdown and test",
		markdownText: markdownText,
		testCode:     testCode,
		want: testcase{
			pkg: "foo_functional_test",
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Description from markdown",
			},
		},
	}, {
		desc:         "bad rundata",
		markdownText: markdownText,
		testCode:     testCode,
		rundataCode:  "~!@#$%^&*()_+",
		wantErr:      "parse rundata_test",
	}, {
		desc:         "good rundata",
		markdownText: markdownText,
		testCode:     testCode,
		rundataCode:  rundataCode,
		want: testcase{
			pkg: "foo_functional_test",
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Description from markdown",
			},
			existing: &parsedData{
				testUUID:        "123e4567-e89b-42d3-8456-426614174000",
				testPlanID:      "YY-1.1",
				testDescription: "Description from code",
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			testdir := t.TempDir()
			for fname, fdata := range map[string]string{
				"README.md":       test.markdownText,
				"foo_test.go":     test.testCode,
				"rundata_test.go": test.rundataCode,
			} {
				if fdata != "" {
					if err := os.WriteFile(filepath.Join(testdir, fname), []byte(fdata), 0600); err != nil {
						t.Fatalf("Could not write %s: %v", fname, err)
					}
				}
			}

			var got testcase
			err := got.read(testdir)
			if (err == nil) != (test.wantErr == "") || (err != nil && !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("testcase.read got error %v, want error containing %q:", err, test.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(test.want, got, tcopt); diff != "" {
				t.Errorf("testcase.read -want,+got:\n%s", diff)
			}
		})
	}
}

func TestCase_Check(t *testing.T) {
	cases := []struct {
		name string
		tc   testcase
		want int
	}{{
		name: "good",
		tc: testcase{
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Foo Functional Test",
			},
			existing: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Foo Functional Test",
				testUUID:        "123e4567-e89b-42d3-8456-426614174000",
			},
		},
		want: 0,
	}, {
		name: "allbad",
		tc: testcase{
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Description from Markdown",
			},
			existing: &parsedData{
				testPlanID:      "YY-1.1",
				testDescription: "Description from Test",
				testUUID:        "123e4567-e89b-12d3-a456-426614174000",
			},
		},
		want: 3,
	}, {
		name: "noexisting",
		tc: testcase{
			markdown: &parsedData{
				testPlanID:      "XX-1.1",
				testDescription: "Foo Functional Test",
			},
		},
		want: 1,
	}, {
		name: "nodata",
		tc:   testcase{},
		want: 2,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs := c.tc.check()
			t.Logf("Errors from check: %#q", errs)
			if got := len(errs); got != c.want {
				t.Errorf("Number of errors from check got %d, want %d.", got, c.want)
			}
		})
	}
}

func TestCase_Fix(t *testing.T) {
	tc := testcase{
		markdown: &parsedData{
			testPlanID:      "XX-1.1",
			testDescription: "Foo Functional Test",
		},
	}
	if err := tc.fix(); err != nil {
		t.Fatal(err)
	}
	got := tc.fixed
	want := &parsedData{
		testPlanID:      tc.markdown.testPlanID,
		testDescription: tc.markdown.testDescription,
		testUUID:        got.testUUID,
	}
	if diff := cmp.Diff(want, got, tcopt); diff != "" {
		t.Errorf("fixed -want,+got:\n%s", diff)
	}
}

func TestCase_FixUUID(t *testing.T) {
	tc := testcase{
		markdown: &parsedData{
			testPlanID:      "XX-1.1",
			testDescription: "Foo Functional Test",
		},
		existing: &parsedData{
			testUUID: "urn:uuid:123e4567-e89b-42d3-8456-426614174000",
		},
	}
	if err := tc.fix(); err != nil {
		t.Fatal(err)
	}
	got := tc.fixed
	want := &parsedData{
		testPlanID:      tc.markdown.testPlanID,
		testDescription: tc.markdown.testDescription,
		testUUID:        "123e4567-e89b-42d3-8456-426614174000",
	}
	if diff := cmp.Diff(want, got, tcopt); diff != "" {
		t.Errorf("fixed -want,+got:\n%s", diff)
	}
}

func TestCase_Write(t *testing.T) {
	var want, got testcase

	// Prepare a testdir with just README.md
	testdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(testdir, "README.md"), []byte(markdownText), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testdir, "foo_test.go"), []byte(testCode), 0600); err != nil {
		t.Fatal(err)
	}

	// Read, fix, and write.
	if err := want.read(testdir); err != nil {
		t.Fatal(err)
	}
	if err := want.fix(); err != nil {
		t.Fatal(err)
	}
	if err := want.write(testdir); err != nil {
		t.Fatal(err)
	}

	// Read it back to ensure we got the same data.
	if err := got.read(testdir); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want.fixed, got.existing, tcopt); diff != "" {
		t.Errorf("Write then read output differs -want,+got:\n%s", diff)
	}
}
