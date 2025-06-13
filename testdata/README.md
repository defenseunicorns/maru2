# testdata

This directory defines E2E test files for `maru2` leveraging <https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript>.

All tests are run via [`cmd/root_test.go` `TestE2E`](../cmd/root_test.go).

To run individual tests:

```sh
go test ./cmd/ -run TestE2E/<Test>

# e.g.
go test ./cmd/ -run TestE2E/version -v # <- add -v if you want extra verbosity / to see STDOUT and STDERR
```

E2E tests _should_ primarily concern themselves w/ flag parsing, exit codes, logging and general CLI UX.
