# maru2 (for now)

![GitHub Tag](https://img.shields.io/github/v/tag/defenseunicorns/maru2)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/defenseunicorns/maru2)
[![Go Report Card](https://goreportcard.com/badge/github.com/defenseunicorns/maru2)](https://goreportcard.com/report/github.com/defenseunicorns/maru2)
![GitHub License](https://img.shields.io/github/license/defenseunicorns/maru2)

A simple task runner.

> [!CAUTION]
> This project is still in its early stages. Expect breaking changes.

## Installation

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2@latest
```

or if you like to live dangerously:

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2@main
```

or checkout the latest [release artifacts](https://github.com/defenseunicorns/maru2/releases/latest).

## Documentation

- [CLI](docs/cli.md)
- [Workflow Syntax](docs/syntax.md)
- [Built-in Tasks](docs/builtins.md)

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
