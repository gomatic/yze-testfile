package b_test

// example_test.go has no source file but declares only an Example, so it is
// exempt. Its directory has no non-test .go file, so the driver delivers a
// base-package pass with zero syntax files: run must treat that as a no-op
// rather than panic on pass.Files[0]. Regression for the cmd-cat/examples crash.
func ExampleThing() {}
