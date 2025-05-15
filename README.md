# maru2 (for now)

A simple task runner. Imagine GitHub actions and Makefile had a baby.

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
