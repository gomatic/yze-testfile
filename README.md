# yze-go-testfile

A [`yze`](https://github.com/gomatic/yze) analyzer (group `go`, category `testing`) enforcing the gomatic Go testing standard that unit-test files are **1:1 with their source**: `<name>_test.go` tests `<name>.go`. It exists because the 100%-coverage gate makes scattered `<name><extra>_test.go` files easy to introduce.

A `_test.go` file without a matching source file is flagged **unless** it is not a unit test:

- it carries a **build constraint** (`//go:build ...`) — i.e. an integration test; or
- it declares **no `Test` functions** — i.e. only examples, benchmarks, or fuzz targets.

The package directory is read from the filesystem, so the rule holds in production (where the driver does not load test files into the analysis pass).

- **Rule:** `yze/go/testfile`
- **Binary:** `cmd/yze-go-testfile` runs it standalone.

Built on the [`go-yze`](https://github.com/gomatic/go-yze) framework.
