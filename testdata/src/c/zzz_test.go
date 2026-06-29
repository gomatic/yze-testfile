package c_test

import "testing"

// TestZzz is a genuine unit-test orphan: no zzz.go source, no build tag, and it
// declares a Test function. The package has no non-test .go file (GoFiles empty),
// so it is delivered only through the external-test pass; the orphan must still
// be reported there even though the empty base-package pass is skipped.
func TestZzz(t *testing.T) { _ = t }
