# Developing maru2

> First read the [Contributing Guide](../.github/CONTRIBUTING.md).
>
> Then read the [E2E tests documentation](../testdata/README.md).
>
> The [copilot instructions](../.github/copilot-instructions.md) are also a good read on repo structure. Be sure to keep this file up-to-date so the LLMs can stay sane.

## Core design principles:

1. [Rob Pike's 5 Rules of Programming](https://users.ece.utexas.edu/~adnan/pike.html)
1. "Simple things should be simple, complex things should be possible" ~ Alan Kay
1. Take in `interface`s, return `struct`s.
1. Build upon existing, well defined systems versus defining replacements.

## Building

The [`Makefile`](../Makefile) has all of the necessary targets. Run `make help` / read the Makefile to see what you need to do in order to build and run maru2.

The quickest way to start fresh and then compile everything fully:

```bash
make clean
make -j all
```

If you are looking for Go docs, the best way to view them is `go doc -http`, `go doc -all . | grep -C 10 <your query>`, or reading the relevant source code.

## Dependency Updates

Dependabot || Renovate should take care of dependency updates.

Maru2's dependencies were carefully selected, as such most dependency updates should be painless and a quick approval.

The only caveat is [`github.com/google/go-github`](https://github.com/google/go-github/releases) as that library does major version increases regularly, and there is no current programatic way with the Go CLI to check for major version increases [issue here](https://github.com/golang/go/issues/67420).

Either automate w/ a GitHub CI cronjob (maybe Renovate has this builtin?), or manually check the repo every now and then. The client has no [non-Google dependencies](https://github.com/google/go-github/blob/master/go.mod), so I am not too worried about manually merging dependency updates.

There is a `v2` of most of `charmbracelet`'s libraries coming soon, so keep a lookout for that as well.

## Releases

Release-Please and GoReleaser handle 99% of releases.

To unstick the Release-Please release PR, add and remove the PR from the `Unstick CI` milestone, that will kick off CI and allow for your approval to be mergable.

Release-Please handles the Git tag, GitHub release and CHANGELOG; GoReleaser handles building and publishing the binaries and creating the PR on the [Defense Unicorns Homebrew Tap repository](https://github.com/defenseunicorns/homebrew-tap). Releases are not fully finished until the generated PR is approved and merged on that repository.

When debugging GoReleaser I found the following useful:

```bash
goreleaser release --snapshot --clean --skip=publish
```

## Testing

Run individual tests w/ your preferred flavor of `go test -run ...`.

When running the entire suite, most of the time use the following:

```bash
make test ARGS="-w short=true"
# or
maru2 test -w short=true
```

This skips tests that call the GitHub and GitLab APIs, keeping you from 1. running into auth/429 errors, and 2: speeds up the test suite a little.

If you _do_ run w/o `short=true`, ensure your `GITHUB_TOKEN`/`GITLAB_TOKEN` are set so you don't run into said 429s.

Read [the E2E testing guide](../testdata/README.md) for information on adding / updating E2E tests.

After running `make test`/`maru2 test`, check coverage using `go tool cover -html=coverage.out` or `go tool -func=coverage.out`.

## Creating a new major schema

1. `cp -r schema/v1 schema/v2`
1. Update the `SchemaVersion` and `SchemaURL` in `schema/v2/workflow.go`
1. Add the schema to the meta generator `schema.go`
1. Add a new Make target for the schema
1. Use an LLM to change over all of the current `v1` references to the new `v2` schema objects.
1. Update the `Migrate` method in `schema/v2/migrate.go` to handle `v0 -> v1 -> v2`.
1. (perform similar steps as above if cutting a new version of the `config` schema)
1. Start modifying the schema to your heart's desire!
1. Note that only the schema is versioned, the runtime is _not_. Take care that any new behavior works well in the old system (prefer building opt-in enhancements versus replacing behavior).
1. The top-level Go files in this project (`run.go`, `with.go`, etc...) are the core runtime files. Any changes made to these files should be done with the utmost scrutiny and test coverage.

## Adding a new property to the schema

1. Add the new property to the relevant type, include the `json` struct tag
1. Update the `JSONSchemaExtend` method to include the new property in the generated schema, match the type to the Go type as needed. Schema configuration exists in this method versus struct tags due to sometimes requiring `fmt.Sprintf` or other programmatic configurations.
1. Run `make`, the Makefile auto tracks all relevant files to re-generate the schemas.
1. Commit the changes.

## Creating more builtins

Read [the builtins guide](../builtins/README.md).

## Things that are (mostly) set and forget

- `install.sh`: the convenience script will probably never need to be updated aside from adding / removing CLIs using the `BINARIES` variable.
- `**/main.go`: all of the `main.go`s are extrememely minimal and will probably never need changes.

## Being kind to embedders

As you make changes to the `*Main()` functions in `cmd`, be sure to keep [`cmd/internal/main.go`](../cmd/internal/main.go) up to date with the latest and most preferred way to embed Maru2 as a Cobra CLI. Other Unicorns will most certainly appreciate that.

## Thanks

Thanks for choosing to develop / contribute to Maru2!
