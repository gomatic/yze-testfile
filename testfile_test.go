package testfile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

// fakeFiles returns a fileReader serving the given contents, erroring on unknown
// paths.
func fakeFiles(contents map[string]string) fileReader {
	return func(path string) ([]byte, error) {
		if content, ok := contents[path]; ok {
			return []byte(content), nil
		}
		return nil, errors.New("no such file")
	}
}

func TestOrphanTestStemsClassifiesEveryCase(t *testing.T) {
	names := []string{"a.go", "a_test.go", "helper_test.go", "example_test.go", "integration_test.go", "notgo.txt"}
	dir := func(string) ([]string, error) { return names, nil }
	file := fakeFiles(map[string]string{
		"d/helper_test.go":      "package a\nimport \"testing\"\nfunc TestHelper(t *testing.T) {}",
		"d/example_test.go":     "package a\nfunc ExampleA() {}",
		"d/integration_test.go": "//go:build integration\npackage a\nfunc TestX() {}",
	})

	stems := orphanTestStems(dir, file, "d")

	assert.Equal(t, []string{"helper"}, stems)
}

func TestOrphanTestStemsReturnsNilOnReadError(t *testing.T) {
	dir := func(string) ([]string, error) { return nil, errors.New("unreadable") }

	assert.Nil(t, orphanTestStems(dir, fakeFiles(nil), "d"))
}

func TestExemptTreatsUnreadableFileAsExempt(t *testing.T) {
	assert.True(t, exempt(fakeFiles(nil), "missing.go"))
}

func TestOSReadDirNames(t *testing.T) {
	names, err := osReadDirNames("testdata/src/a")
	require.NoError(t, err)
	assert.Contains(t, names, "a.go")

	_, err = osReadDirNames("testdata/does-not-exist")
	require.Error(t, err)
}

func TestRunReportsOrphanUnitTestFiles(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "a")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, Registration.Validate())
	assert.Equal(t, "yze/go/testfile", Registration.RuleID())
	assert.Same(t, Analyzer, Registration.Analyzer)
}
