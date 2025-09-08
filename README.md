# maru2 (for now)

![GitHub Tag](https://img.shields.io/github/v/tag/defenseunicorns/maru2)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/defenseunicorns/maru2)
![GitHub License](https://img.shields.io/github/license/defenseunicorns/maru2)
![CodeQL](https://github.com/defenseunicorns/maru2/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/defenseunicorns/maru2)](https://goreportcard.com/report/github.com/defenseunicorns/maru2)
[![codecov](https://codecov.io/gh/defenseunicorns/maru2/graph/badge.svg?token=IQMK40GAOK)](https://codecov.io/gh/defenseunicorns/maru2)
![CI Status](https://github.com/defenseunicorns/maru2/actions/workflows/go.yaml/badge.svg)

A simple task runner.

> [!CAUTION]
> This project is still in its early stages. Expect breaking changes.

## Installation

via brew:

```sh
brew tap defenseunicorns/tap
brew install maru2
```

via curl:

```sh
curl -s https://raw.githubusercontent.com/defenseunicorns/maru2/main/install.sh | bash
```

via wget:

```sh
wget -q -O - https://raw.githubusercontent.com/defenseunicorns/maru2/main/install.sh | bash
```

via go install:

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2@latest
```

or if you like to live dangerously:

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2@main
```

or checkout the latest [release artifacts](https://github.com/defenseunicorns/maru2/releases/latest).

## Documentation

- [Getting Started](docs/README.md)
- [CLI Reference](docs/cli.md)
- [Workflow Syntax](docs/syntax.md)
- [Publishing Workflows](docs/publish.md)
- [Built-in Tasks](docs/builtins.md)

View CLI usage w/ `maru2 --help`

If you are coming from `maru-runner` / `uds run` and looking to transition, checkout the [migration guide](./docs/maru-runner-migration.md).

If you are looking to embed maru2 into another Cobra CLI, take a look at the example in [`cmd/internal`](./cmd/internal/main.go).

## Contributing

Thanks for taking time out of your day to contribute! Read the [contributing guide](./.github/CONTRIBUTING.md) for more information.

## Schema Validation

Enabling schema validation in VSCode:

```json
    "yaml.schemas": {
        "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json": "tasks.yaml",
    },
```

Per file basis:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json
```
