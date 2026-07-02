package testfile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
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
	names := []string{
		"a.go",
		"a_test.go",
		"helper_test.go",
		"example_test.go",
		"integration_test.go",
		"legacy_test.go",
		"comment_test.go",
		"notgo.txt",
	}
	dir := func(dirPath) ([]string, error) { return names, nil }
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
	dir := func(dirPath) ([]string, error) { return nil, errors.New("unreadable") }

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

// TestRunSkipsPackageWithoutFiles pins the contract that a pass carrying no
// syntax files is a no-op. The driver delivers such a pass for a package whose
// only Go files are external test files (an examples-only directory); run must
// not index pass.Files[0]. Without the guard this panics with index out of
// range, which is the cmd-cat/examples crash this test exists to prevent.
func TestRunSkipsPackageWithoutFiles(t *testing.T) {
	result, err := run(&analysis.Pass{})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestRunSkipsExternalTestOnlyExamplesPackage reproduces the cmd-cat/examples
// layout end-to-end: a directory of only external test files declaring only
// Example functions. The base package has zero syntax files; the analyzer must
// run clean and report nothing, since every example is exempt.
func TestRunSkipsExternalTestOnlyExamplesPackage(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "b")
}

// TestRunReportsOrphanInExternalTestOnlyPackage proves the empty-pass guard does
// not blind detection: a genuine unit-test orphan living in a package with no
// non-test source (delivered only via the external-test pass) is still flagged.
func TestRunReportsOrphanInExternalTestOnlyPackage(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "c")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, Registration.Validate())
	assert.Equal(t, "yze/testfile", Registration.RuleID())
	assert.Same(t, Analyzer, Registration.Analyzer)
}
