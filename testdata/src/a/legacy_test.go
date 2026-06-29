// +build integration

package a

import "testing"

// TestLegacy is an integration test guarded by the legacy // +build form. It has
// no legacy.go source file, but the build constraint exempts it from the rule —
// the substring check missed this form and wrongly flagged such orphans.
func TestLegacy(t *testing.T) { _ = t }
