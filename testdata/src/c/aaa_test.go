package c_test // want `zzz_test.go`

// aaa_test.go is itself exempt (only an Example) and sorts first, so the
// orphan diagnostic for zzz_test.go is anchored on this file's package clause.
func ExampleAaa() {}
