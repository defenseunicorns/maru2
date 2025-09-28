# testdata

This directory defines E2E test files for `maru2` leveraging <https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript>.

Regular E2E tests are run via [`cmd/root_test.go` `TestE2E`](../cmd/root_test.go).

E2E tests the fetch "external" resources are run via [`cmd/fetch_test.go` `TestFetchE2E`](../cmd/fetch_test.go).

E2E tests for `maru2-publish` are run via [`cmd/publish_test.go` `TestPublishE2E`](../cmd/publish_test.go).

To run individual tests:

```sh
go test ./cmd/ -run TestE2E/<test>
go test ./cmd/ -run TestPublishE2E/<test>
go test ./cmd/ -run TestFetchE2E/<test>

# e.g.
go test ./cmd/ -run TestE2E/version -v # <- add -v if you want extra verbosity / to see STDOUT and STDERR
```

E2E tests _should_ primarily concern themselves w/ flag parsing, exit codes, logging and general CLI UX.

To update the "golden files" embedded in the tests automatically, use:

```sh
maru2 test -w update-scripts=true
```

then commit the changes.
