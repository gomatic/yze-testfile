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
	names := []string{"a.go", "a_test.go", "helper_test.go", "example_test.go", "integration_test.go", "legacy_test.go", "comment_test.go", "notgo.txt"}
	dir := func(string) ([]string, error) { return names, nil }
	file := fakeFiles(map[string]string{
		// A unit test with no source file: the sole genuine orphan.
		"d/helper_test.go": "package a\nimport \"testing\"\nfunc TestHelper(t *testing.T) {}",
		// Examples/benchmarks declare no Test function and are exempt.
		"d/example_test.go": "package a\nfunc ExampleA() {}",
		// A //go:build constraint marks an integration test, which is exempt.
		"d/integration_test.go": "//go:build integration\n\npackage a\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}",
		// The legacy // +build form is equally a build constraint and exempt
		// (the substring check missed it, wrongly flagging legacy orphans).
		"d/legacy_test.go": "// +build integration\n\npackage a\n\nimport \"testing\"\n\nfunc TestLegacy(t *testing.T) {}",
		// A real unit-test orphan whose comment and string literals merely
		// mention //go:build and func Test as text: neither is structural, so
		// the file is still an orphan (the substring check wrongly exempted it).
		"d/comment_test.go": "package a\n\nimport \"testing\"\n\n// mentions //go:build integration and func TestFake as text only\nfunc TestComment(t *testing.T) {\n\t_ = \"//go:build integration\"\n\t_ = \"func TestFake(t *testing.T)\"\n\t_ = t\n}",
	})

	stems := orphanTestStems(dir, file, "d")

	assert.ElementsMatch(t, []string{"helper", "comment"}, stems)
}

func TestOrphanTestStemsReturnsNilOnReadError(t *testing.T) {
	dir := func(string) ([]string, error) { return nil, errors.New("unreadable") }

	assert.Nil(t, orphanTestStems(dir, fakeFiles(nil), "d"))
}

func TestExemptTreatsUnreadableFileAsExempt(t *testing.T) {
	assert.True(t, exempt(fakeFiles(nil), "missing.go"))
}

func TestExemptTreatsUnparseableFileAsExempt(t *testing.T) {
	file := fakeFiles(map[string]string{"d/broken_test.go": "package a\nfunc TestBroken(t *testing.T {"})

	assert.True(t, exempt(file, "d/broken_test.go"))
}

func TestExemptIgnoresMethodsNamedLikeTests(t *testing.T) {
	file := fakeFiles(map[string]string{"d/method_test.go": "package a\ntype S struct{}\nfunc (S) TestThing() {}"})

	assert.True(t, exempt(file, "d/method_test.go"))
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
	assert.Equal(t, "yze/testfile", Registration.RuleID())
	assert.Same(t, Analyzer, Registration.Analyzer)
}
