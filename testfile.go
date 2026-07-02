// Package testfile provides a go/analysis analyzer enforcing the gomatic Go
// testing standard that unit-test files are 1:1 with their source files:
// <name>_test.go tests <name>.go. A _test.go file without a matching source file
// is allowed only when it is not a unit test — it carries a build constraint
// (integration tests) or declares no Test functions (examples, benchmarks, fuzz
// targets).
package testfile

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
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
	dirReader  func(dir dirPath) ([]string, error)
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
	Categories: []goyze.Category{"testing"},
	URL:        "https://docs.gomatic.dev/yze/testfile",
	Analyzer:   Analyzer,
}

// run reports each orphan unit-test file in the package directory.
//
// A package whose only Go files are external test files (GoFiles empty,
// XTestGoFiles non-empty — e.g. an examples directory holding only Example
// functions) is delivered by the driver as a base-package pass with no syntax
// files. There is no file to anchor a directory or report on, and the package's
// test files are analyzed in their own pass; this pass is a no-op.
func run(pass *analysis.Pass) (any, error) {
	if len(pass.Files) == 0 {
		return nil, nil
	}
	dir := filepath.Dir(pass.Fset.File(pass.Files[0].Pos()).Name())
	for _, stem := range orphanTestStems(readDir, readFile, dirPath(dir)) {
		pass.Reportf(
			pass.Files[0].Name.Pos(),
			"test file %s_test.go has no source file %s.go; unit tests must be 1:1 with their source (give integration tests a build tag, or keep only examples/benchmarks/fuzz)",
			stem,
			stem,
		)
	}
	return nil, nil
}

// dirPath is the filesystem path of the analyzed package's directory.
type dirPath string

// orphanTestStems returns the stems of unit-test files lacking a source file.
func orphanTestStems(dir dirReader, file fileReader, path dirPath) []string {
	names, err := dir(path)
	if err != nil {
		return nil
	}
	var stems []string
	for _, name := range names {
		if stem, ok := orphan(file, path, fileName(name), names); ok {
			stems = append(stems, stem)
		}
	}
	return stems
}

// orphan reports a test file's stem when it is a unit test with no source file.
func orphan(file fileReader, dir dirPath, name fileName, names []string) (string, bool) {
	stem, ok := testStem(name)
	if !ok || slices.Contains(names, stem+".go") {
		return "", false
	}
	if exempt(file, filePath(filepath.Join(string(dir), string(name)))) {
		return "", false
	}
	return stem, true
}

// fileName is a bare file name within the package directory.
type fileName string

// testStem returns the stem of a _test.go file name.
func testStem(name fileName) (string, bool) {
	if !strings.HasSuffix(string(name), "_test.go") {
		return "", false
	}
	return strings.TrimSuffix(string(name), "_test.go"), true
}

// filePath is the filesystem path of a test file under exemption inspection.
type filePath string

// exempt reports whether a test file is not a unit test: it carries a build
// constraint (a //go:build or legacy // +build line), or declares no Test
// functions. The file is parsed so that build-constraint lines and Test
// function declarations are recognized structurally — never by substring, which
// both misses the legacy // +build form and misfires on text appearing inside a
// comment or string literal.
func exempt(file fileReader, path filePath) bool {
	content, err := file(string(path))
	if err != nil {
		return true
	}
	parsed, err := parser.ParseFile(
		token.NewFileSet(),
		string(path),
		content,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		return true
	}
	return hasBuildConstraint(parsed) || !hasTestFunc(parsed)
}

// hasBuildConstraint reports whether the file carries a build constraint, in
// either the //go:build or the legacy // +build form, before its package clause.
func hasBuildConstraint(f *ast.File) bool {
	for _, group := range f.Comments {
		if constrains(group, f.Package) {
			return true
		}
	}
	return false
}

// constrains reports whether a comment group holds a build-constraint line that
// precedes the package clause at pkg.
func constrains(group *ast.CommentGroup, pkg token.Pos) bool {
	for _, c := range group.List {
		if c.Pos() < pkg && (constraint.IsGoBuild(c.Text) || constraint.IsPlusBuild(c.Text)) {
			return true
		}
	}
	return false
}

// hasTestFunc reports whether the file declares at least one Test function.
func hasTestFunc(f *ast.File) bool {
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && isTestFunc(fn) {
			return true
		}
	}
	return false
}

// isTestFunc reports whether a declaration is a top-level Test function (a
// TestXxx free function, not a method).
func isTestFunc(fn *ast.FuncDecl) bool {
	return fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test")
}

// osReadDirNames lists the file names in a directory.
func osReadDirNames(dir dirPath) ([]string, error) {
	entries, err := os.ReadDir(string(dir))
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}
