package a

import "testing"

// TestComment is a real unit test with no comment.go source file, so it is an
// orphan. The string literals below mention //go:build and func Test as text
// only; neither is a structural build constraint or declaration, so the file is
// still flagged — the substring check wrongly exempted it on the //go:build text.
func TestComment(t *testing.T) {
	_ = "//go:build integration"
	_ = "func TestFake(t *testing.T)"
	_ = t
}
