# maru2 (for now)

A simple task runner. Imagine GitHub actions and Makefile had a baby.

> [!CAUTION]
> This project is still in its early stages. Expect breaking changes.

## Installation

> [!IMPORTANT]
> While this repo is still private, you will have to setup auth via `.netrc`
>
> ```bash
> cat > ~/.netrc <<EOF
> machine github.com
> login x-oauth-basic
> password $(gh auth token)
> EOF
> chmod 600 ~/.netrc
> ```

```sh
GOPRIVATE=github.com/defenseunicorns/maru2 \
go install github.com/defenseunicorns/maru2/cmd/maru2@latest
```

or if you like to live dangerously:

```sh
GOPRIVATE=github.com/defenseunicorns/maru2 \
go install github.com/defenseunicorns/maru2/cmd/maru2@main
```

## Documentation

- [CLI](docs/cli.md)
- [Workflow Syntax](docs/syntax.md)
- [Built-in Tasks](docs/builtins.md)

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
