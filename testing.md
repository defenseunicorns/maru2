---
title: "Chasing Test Coverage in maru2"
#sub_title: (in presenterm!)
author: razzle
theme:
  name: tokyonight-storm
---

## Introduction

Test coverage is often times seen as a project maturity, project "safety", and code-quality metric.

It has even used in some DoD organizations / documents as a pass-fail check
on whether a project can be released / adopted.

The following are some lessons learned, and some cool strategies I picked up in my journey writing the test suite for maru2.

<!-- end_slide -->

## Libraries used

- end-to-end: `github.com/rogpeppe/go-internal/testscript`
<!-- pause -->
- unit: `github.com/stretchr/testify`
<!-- pause -->
- in-memory OCI registry: `github.com/olareg/olareg`
<!-- pause -->
- in-memory HTTP server: `net/http/httptest`
<!-- pause -->
- network test control flow: `testing.Short()`/`go test -short=true|false`
<!-- pause -->

```go
func TestGitHubFetcher(t *testing.T) {
	t.Run("basic fetch", func(t *testing.T) {
		t.Parallel()
		if testing.Short() {
			t.Skip("skipping tests that require network access")
		}
		...
	}
}
```

<!-- end_slide -->

## Types of Tests

- **normal table tests** for pure functions / simple operations
  - [](if_test.go)
  - [](log_test.go)
  - [](output_test.go)
<!-- pause -->
- **table tests w/ complex setup and validation** for testing stateful operations
  - [](uses/store_test.go)
  - [](uses_test.go)
<!-- pause -->
- **f tests** for operations w/ complex setup and behavior that is not condusive to a table test
  - [](uses/http_test.go)
  - [](uses/oci_test.go)
<!-- pause -->
- **simple tests** for operations even too simple for a table test, or so hard to test I only want a vibe check
  - [](schema_test.go)
<!-- pause -->
- **end-to-end tests** for flag parsing, CLI exit status', and logging UX
  - [](cmd/root_test.go)
  - [](testdata/call-local.txtar)
  - [](testdata/completion.txtar)

<!-- end_slide -->

## Don't Chase

<!-- pause -->
- chasing test coverage as a _number_ is a fool's errand.
<!-- pause -->
- testing increases confidence in consistency of behavior, not correctness
<!-- pause -->
- testing increases code quality as a second order effect

<!-- end_slide -->

## Tests should start simple

Writing tests for a feature **may** be difficult, but it should never be **confusing**.

<!-- pause -->

If a function, class struct, etc... is too confusing / complex to test the following cases cleanly:

- success
- failure
- empty (default)

the code is prob too convoluted or has too many layers of abstraction.

AI is actually a pretty good canary for this.

Given a function and its surrounding context, as well as its usage in the codebase, AI should be able to generate the aforementioned tests at a minimum.

<!-- end_slide -->

## Tests are your first consumer

Tests are the first time in a codebase you can act as a consumer of your own SDK.

At a glance, you should be able to figure out what a function does and its boundaries just by looking at the tests.

If you can't use your own code, no one else will.

<!-- pause -->

If you are having to create test setup / teardown that makes you uncomfortable, look to refactor.

Code is very stylistic, and it benefits to be consistent in both writing and testing code. If two functions perform similar, but unrelated operations, their tests should probably look similar as well.

<!-- end_slide -->

## misc. learnings

- `assert.Contains` vs `require.EqualError`
- not everything needs `t.Parallel`, and some things cannot be run in parallel (`t.Setenv`, `t.Chdir`)
- leverage `<module>_test` module to avoid circular dependency issues
  - [](uses/oci_test.go)
- AI does a decent job generating test cases, but a pretty poor job at generating testing logic
- dependency injection via interfaces cleans up a lot of testing boilerplate
- if your unit tests are solid enough, your end-to-end tests should really just be integration tests
- DRY doesnt matter as much when writing tests as long as you are testing at different layers
- learn `go tool cover`
  - `go tool cover -func=coverage.out`
  - `go tool cover -html=coverage.out`

<!-- end_slide -->

<!--jump_to_middle-->

## That's all folks!
