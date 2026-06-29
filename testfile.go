// Package testfile provides a go/analysis analyzer enforcing the gomatic Go
// testing standard that unit-test files are 1:1 with their source files:
// <name>_test.go tests <name>.go. A _test.go file without a matching source file
// is allowed only when it is not a unit test — it carries a build constraint
// (integration tests) or declares no Test functions (examples, benchmarks, fuzz
// targets).
package testfile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
)

// Injected filesystem operations, so the analyzer's error and decision paths are
// testable without a real directory tree.
type (
	dirReader  func(dir string) ([]string, error)
	fileReader func(path string) ([]byte, error)
)

var (
	readDir  dirReader  = osReadDirNames
	readFile fileReader = os.ReadFile
)

// Analyzer reports unit-test files that are not 1:1 with a source file.
var Analyzer = &analysis.Analyzer{
	Name: "testfile",
	Doc:  "reports _test.go unit-test files without a matching source file",
	Run:  run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "testfile",
	Group:      "go",
	Categories: []goyze.Category{"testing"},
	URL:        "https://docs.gomatic.dev/yze/go/testfile",
	Analyzer:   Analyzer,
}

// run reports each orphan unit-test file in the package directory.
func run(pass *analysis.Pass) (any, error) {
	dir := filepath.Dir(pass.Fset.File(pass.Files[0].Pos()).Name())
	for _, stem := range orphanTestStems(readDir, readFile, dir) {
		pass.Reportf(pass.Files[0].Name.Pos(), "test file %s_test.go has no source file %s.go; unit tests must be 1:1 with their source (give integration tests a build tag, or keep only examples/benchmarks/fuzz)", stem, stem)
	}
	return nil, nil
}

// orphanTestStems returns the stems of unit-test files lacking a source file.
func orphanTestStems(dir dirReader, file fileReader, path string) []string {
	names, err := dir(path)
	if err != nil {
		return nil
	}
	var stems []string
	for _, name := range names {
		if stem, ok := orphan(file, path, name, names); ok {
			stems = append(stems, stem)
		}
	}
	return stems
}

// orphan reports a test file's stem when it is a unit test with no source file.
func orphan(file fileReader, dir, name string, names []string) (string, bool) {
	stem, ok := testStem(name)
	if !ok || slices.Contains(names, stem+".go") {
		return "", false
	}
	if exempt(file, filepath.Join(dir, name)) {
		return "", false
	}
	return stem, true
}

// testStem returns the stem of a _test.go file name.
func testStem(name string) (string, bool) {
	if !strings.HasSuffix(name, "_test.go") {
		return "", false
	}
	return strings.TrimSuffix(name, "_test.go"), true
}

// exempt reports whether a test file is not a unit test: it carries a build
// constraint, or declares no Test functions.
func exempt(file fileReader, path string) bool {
	content, err := file(path)
	if err != nil {
		return true
	}
	text := string(content)
	return strings.Contains(text, "//go:build") || !strings.Contains(text, "func Test")
}

// osReadDirNames lists the file names in a directory.
func osReadDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}
