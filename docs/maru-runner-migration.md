# maru-runner migration guide

This guide will help you migrate your existing `maru-runner` tasks to the new `maru2` format. While both tools serve similar purposes, `maru2` has a more modern, GitHub Actions-like syntax with improved capabilities.

> [!NOTE]
> This migration guide is for migrating from `maru-runner` to `maru2`'s `v1` schema.
>
> This guide is a living document and _may_ not be 100% accurate in all situations.
>
> Contributions are most welcome!

## Why no migration tool?

Migrating from `maru-runner` to `maru2` is no small task, and one that should be taken with care and consideration.

Additionally, the migration gives workflow authors a chance to redefine the patterns they have been using and complete sweeping/breaking changes to their comfort level; a migration tool would stymie that creativity.

Lastly, this will be the last such time that a pure migration guide will be provided. Since `maru2` has versioned schemas, there will be schema migrations that happen automatically during runtime, as well as schema migrations that can be accomplished via a future migration CLI (probably something like `go run github.com/maru2/cmd/maru2-migrate@main tasks.yaml`).

## Using AI to migrate

The following setup and prompt _should_ get the ball rolling on migrating a given workflow using AI.

1. Download relevant context (or add as remote context via the raw content URLs):

```sh
curl -sS -o maru-readme.md https://raw.githubusercontent.com/defenseunicorns/maru-runner/main/README.md
curl -sS -o maru-runner.schema.json https://raw.githubusercontent.com/defenseunicorns/maru-runner/main/tasks.schema.json
curl -sSO https://raw.githubusercontent.com/defenseunicorns/maru2/main/docs/syntax.md
curl -sSO https://raw.githubusercontent.com/defenseunicorns/maru2/main/docs/cli.md
curl -sSO https://raw.githubusercontent.com/defenseunicorns/maru2/main/docs/maru-runner-migration.md
curl -sSO https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json
```

2. Prompt:

```text
context: read maru-readme.md and maru-runner.schema.json for context on how the old task runner schema and system worked,
now read maru2.schema.json, syntax.md and cli.md for context on how the new task runner schema and system works,
now read maru-runner-migration.md on tips and tricks on how to migrate between maru-runner and maru2

task: migrate tasks.yaml from the old (maru-runner) to the new (maru2 v1), if a property cannot be cleanly migrated, or you are unsure, comment out that property / step as is so the user can make the determination.

validate: execute `maru2 --list -f tasks.yaml` to list all tasks, then execute `maru2 --dry-run -f tasks.yaml <taskname>` for each of the given tasks to ensure correctness of schema and dry run behavior.
```

## Table of Contents

- [Key Differences](#key-differences)
- [Basic Structure](#basic-structure)
- [Variables to Inputs](#variables-to-inputs)
- [Tasks and Actions](#tasks-and-actions)
- [Command Execution](#command-execution)
- [Conditional Execution](#conditional-execution)
- [Directory and Shell Control](#directory-and-shell-control)
- [Timeouts and Retries](#timeouts-and-retries)
- [Includes to Uses](#includes-to-uses)
- [Task Inputs and Reusable Tasks](#task-inputs-and-reusable-tasks)
- [Command Line Usage](#command-line-usage)
- [Complete Example](#complete-example)
- [Enhanced Features in maru2](#enhanced-features-in-maru2)

## Key Differences

| Feature               | maru-runner                          | maru2                              |
| --------------------- | ------------------------------------ | ---------------------------------- |
| Configuration file    | `tasks.yaml`                         | `tasks.yaml` (same)                |
| Schema                | Unversioned                          | Versioned and validated            |
| Command structure     | List of tasks with actions           | Map of tasks with inputs and steps |
| Variable system       | `variables` section + `setVariables` | Task-level `inputs` + step outputs |
| Command execution     | `cmd` key                            | `run` key                          |
| Task references       | `task` key                           | `uses` key                         |
| Includes              | `includes` imports                   | `uses` with URL format             |
| Conditional execution | Limited via `text/template`          | Advanced expressions with `if`     |
| Environment variables | `env` list                           | Explicit `export` in shell         |
| Shell selection       | `shell` object with OS-specific keys | `shell` enum with simple options   |
| Wait for resources    | Built-in wait conditions             | Not yet implemented                |

## Basic Structure

### maru-runner

```yaml
variables:
  - name: FOO
    default: foo

tasks:
  - name: example
    actions:
      - cmd: echo "Hello World"
```

### maru2

```yaml
schema-version: v1
tasks:
  example:
    inputs:
      foo:
        description: "Example input"
        default: "foo"
    steps:
      - run: echo "Hello World"
```

Key differences:

- `maru2` requires a [`schema-version` field](./syntax.md#schema-version)
- Tasks in `maru2` are defined as objects, not a list of objects with `name` properties
- In v1, inputs are defined at the task level, not the workflow level
- Task steps in `maru2` are in a `steps` array under each task

## Variables to Inputs

### maru-runner

```yaml
variables:
  - name: FOO
    default: foo
  - name: BAR
    default: bar

tasks:
  - name: example
    actions:
      - cmd: echo "${FOO}"
```

### maru2

```yaml
schema-version: v1
tasks:
  example:
    inputs:
      foo:
        description: "input foo"
        default: "foo"
      bar:
        description: "input bar"
        default: "bar"
    steps:
      - run: echo "${{ input "foo" }}"
```

Key differences:

- Variables are now defined as [`inputs`](./syntax.md#defining-input-parameters) with more descriptive properties
- Inputs are by default `required: true` and task scoped
- Inputs are weakly type safe when a `default` is set.
  - i.e. if a default is of type `int`, all callers must pass a value that can be coerced to an `int`
- Inputs can be any primitive type (`string`, `int`, `bool`)
- Access inputs using [`${{ input "input-name" }}`](./syntax.md#passing-inputs) expression syntax
- Environment variables can be used as defaults with [`default-from-env: ENV_VAR_NAME`](./syntax.md#default-values-from-environment-variables)
- Input validation is possible with [`validate: "regex-pattern"`](./syntax.md#input-validation)
- Inputs are automapped to `$INPUT_NAME` environment variables (where `NAME` is the uppercase input name)
- The `sensitive` property is not currently implemented, if this is a requirement, please open an issue

## Tasks and Actions

### maru-runner

```yaml
tasks:
  - name: example
    actions:
      - cmd: echo "First step"
      - cmd: echo "Second step"
      - task: another-task

  - name: another-task
    actions:
      - cmd: echo "This is another task"
```

### maru2

```yaml
schema-version: v1
tasks:
  example:
    steps:
      - run: echo "First step"
        name: "Optional step description"
      - run: echo "Second step"
      - uses: another-task

  another-task:
    steps:
      - run: echo "This is another task"
```

Key differences:

- In `maru2`, tasks are defined as objects with keys (not in a list with `name`)
- The task name is the object key in the [`tasks` map](./syntax.md#task-names-and-descriptions), not a `name` property
- `maru2` has an optional [`name` property](./syntax.md#step-identification-with-id-and-name) at the step level for human-readable descriptions
- `maru-runner`'s `actions` are now a `steps` array under each task
- `task` references become [`uses` references](./syntax.md#run-vs-uses)

## Command Execution

### maru-runner

```yaml
tasks:
  - name: example
    actions:
      - cmd: echo "Hello"
        mute: true
        dir: ./some-dir
        env:
          - FOO=bar
```

### maru2

```yaml
schema-version: v1
tasks:
  example:
    steps:
      - run: echo "Hello"
        dir: ./some-dir
        env:
          FOO: bar
        mute: true
```

Key differences:

- `cmd` becomes [`run`](./syntax.md#run-vs-uses)
- `maru2` provides [`id`](./syntax.md#step-identification-with-id-and-name) for step references (required for output access)
- Output capture is done differently (see below)
- `envPath` property is not currently implemented, if this is a requirement, please open an issue

## Capturing and Using Outputs

### maru-runner

```yaml
tasks:
  - name: set-and-use
    actions:
      - cmd: echo "value"
        setVariables:
          - name: MY_VAR
      - cmd: echo "The value is ${MY_VAR}"
```

### maru2

```yaml
schema-version: v1
tasks:
  set-and-use:
    steps:
      - run: |
          echo "Calculating value..."
          echo "my-value=value" >> $MARU2_OUTPUT
          echo "foo=bar" >> $MARU2_OUTPUT
        id: step-one
      - run: echo "The value is ${{ from "step-one" "my-value" }}"
```

Key differences:

- In `maru2`, outputs are written to [`$MARU2_OUTPUT`](./syntax.md#passing-outputs) with `key=value` format
- In `maru2`, multiple outputs can be captured from a single `run`
- Outputs are referenced using [`${{ from "step-id" "output-key" }}`](./syntax.md#passing-outputs)
- Each step needs an [`id`](./syntax.md#step-identification-with-id-and-name) to reference its outputs

## Conditional Execution

### maru-runner

Limited conditional execution support.

### maru2

```yaml
schema-version: v1
tasks:
  conditional:
    inputs:
      enable-feature:
        description: "Enable the feature"
        default: false
    steps:
      - run: echo "This always runs"
      - run: echo "This only runs if an input is true"
        if: input("enable-feature") == true
      - run: echo "This runs if the previous step failed"
        if: failure()
```

Key differences:

- `maru2` supports rich expressions with the [`if` property](./syntax.md#conditional-execution-with-if)
- Built-in functions include [`failure()`, `always()`, `cancelled()`](./syntax.md#conditional-execution-with-if), and more
- Input values can be checked with [`input("name")`](./syntax.md#conditional-execution-with-if)
- Step outputs can be used in conditions with [`from("step-id", "output-key")`](./syntax.md#conditional-execution-with-if)

## Directory and Shell Control

### maru-runner

```yaml
tasks:
  - name: example
    actions:
      - cmd: echo "Running in a directory"
        dir: ./some/path
      - cmd: echo "OS-specific shell"
        shell:
          windows: powershell
          linux: bash
          darwin: bash
```

### maru2

```yaml
schema-version: v1
tasks:
  example:
    steps:
      - run: echo "Running in a directory"
        dir: ./some/path
      - run: echo "Using bash explicitly"
        shell: bash
```

Key differences:

- [`dir`](./syntax.md#working-directory-with-dir) works similarly in both
- `maru-runner` uses OS-specific shell configuration (windows/linux/darwin keys)
- `maru2` adds explicit [`shell`](./syntax.md#selecting-the-shell-for-run-steps) control with simple options: `sh`, `bash`, `pwsh`, `powershell`

## Timeouts and Retries

### maru-runner

```yaml
tasks:
  - name: with-retry
    actions:
      - cmd: some-flaky-command
        maxRetries: 3
        maxTotalSeconds: 60
      - wait:
          network:
            protocol: http
            address: localhost:8080
            code: 200
```

### maru2

```yaml
schema-version: v1
tasks:
  with-timeout:
    steps:
      - run: some-long-command
        timeout: "60s"
```

Key differences:

- `maru-runner`'s `maxRetries` property is not currently implemented in maru2, if this is a requirement, please open an issue
- `maxTotalSeconds` becomes [`timeout`](./syntax.md#step-timeout-with-timeout) with duration string format (e.g., "30s", "1m", "1h")
- `maru-runner`'s `wait` functionality for network and cluster resources is not yet implemented in maru2

## Includes to Uses

### maru-runner

```yaml
includes:
  - local: ./path/to/tasks.yaml
  - remote: https://raw.githubusercontent.com/defenseunicorns/maru-runner/main/tasks.yaml

tasks:
  - name: include-example
    actions:
      - task: local:some-task
      - task: remote:other-task
```

### maru2

```yaml
schema-version: v1
tasks:
  include-example:
    steps:
      - uses: file:path/to/tasks.yaml?task=some-task
      - uses: pkg:github/defenseunicorns/maru-runner?task=other-task
```

or:

```yaml
schema-version: v1
aliases:
  local:
    path: ./path/to/tasks.yaml
tasks:
  include-example:
    steps:
      - uses: local:some-task
      - uses: pkg:github/defenseunicorns/maru-runner?task=other-task
```

Key differences:

- Instead of defining `includes` and using prefixes, `maru2` uses [URL-style references](./syntax.md#run-a-task-from-a-remote-file)
- Format is `protocol:path?task=task-name` (similar to package URLs)
- Supported protocols: [`file:`](./syntax.md#run-a-task-from-a-local-file), [`http:`, `https:`, `pkg:github`, `pkg:gitlab`](./syntax.md#run-a-task-from-a-remote-file), [`builtin:`](./builtins.md), [`oci:`](./publish.md)
- Built-in tasks like `builtin:echo` and `builtin:fetch` provide common functionality
- Support for local path [`aliases`](./syntax.md#local-file-aliases)

## Command Line Usage

### maru-runner

```bash
# Run a task
maru run example

# Run with variables
maru run example --set FOO=bar

# List available tasks
maru --list
```

### maru2

```bash
# Run a task
maru2 example

# Run with inputs
maru2 example -w foo=bar

# List available tasks
maru2 --list

# Run remote directly
maru2 -f "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo -w message="hello world"
```

Key differences:

- `maru2` doesn't require the `run` command ([task name comes first](./cli.md#basic-usage))
- Inputs are set with [`-w key=value`](./cli.md#passing-inputs-to-tasks) (short for `--with`)
- Task listing uses the [`--list` flag](./cli.md#discovering-tasks)
- See [all available flags to learn more](./cli.md#all-available-flags)

## Complete Example

### maru-runner Example

```yaml
variables:
  - name: FOO
    default: foo
  - name: URL
    description: "A URL to check"
    default: "https://example.com"

tasks:
  - name: default
    actions:
      - cmd: echo "run default task"

  - name: example
    actions:
      - task: set-variable
      - task: echo-variable
      - wait:
          network:
            protocol: https
            address: ${URL}
            code: 200

  - name: set-variable
    actions:
      - cmd: echo "bar"
        setVariables:
          - name: FOO

  - name: echo-variable
    actions:
      - cmd: echo "${FOO}"
```

### maru2 Equivalent

```yaml
schema-version: v1
tasks:
  default:
    steps:
      - run: echo "run default task"

  example:
    steps:
      - uses: set-variable
        id: set-var
      - uses: echo-variable
        with:
          value: ${{ from "set-var" "foo" }}

  set-variable:
    steps:
      - run: |
          echo "Generating value..."
          echo "foo=bar" >> $MARU2_OUTPUT

  echo-variable:
    inputs:
      value:
        description: "Value to echo"
        required: true
    steps:
      - uses: builtin:echo
        with:
          text: ${{ input "value" }}
```

## Authentication for Remote Tasks

Both tools support authentication for remote resources, but with different approaches:

### maru-runner

```bash
gh auth token | maru auth login raw.githubusercontent.com --token-stdin
```

### maru2

By default, maru2 uses `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables to pull task files from remote GitHub and GitLab destinations using the [package-url spec](https://github.com/package-url/purl-spec).

It additionally supports a flexible alias system:

```yaml
schema-version: v1
aliases:
  pb:
    type: gitlab
    token-from-env: PEANUT_BUTTER
```

Then in your tasks:

```yaml
tasks:
  remote-example:
    steps:
      - uses: pkg:pb/strawberry/jam@main?task=example
```

The token is pulled from the `PEANUT_BUTTER` environment variable automatically. See [package URL aliases](./syntax.md#package-url-aliases) for more.

## Feature Comparison

| Feature                       | maru-runner       | maru2                                                                              |
| ----------------------------- | ----------------- | ---------------------------------------------------------------------------------- |
| Task composition              | ✅                | ✅ ([task references](./syntax.md#run-another-task-as-a-step))                     |
| Variable support              | ✅                | ✅ (as [inputs](./syntax.md#defining-input-parameters))                            |
| Shell commands                | ✅                | ✅ ([run commands](./syntax.md#run-vs-uses))                                       |
| Remote includes               | ✅                | ✅ ([improved URL format](./syntax.md#run-a-task-from-a-remote-file))              |
| Task inputs                   | ✅                | ✅ ([improved structure](./syntax.md#defining-input-parameters))                   |
| Conditional execution         | Limited           | ✅ ([rich expressions](./syntax.md#conditional-execution-with-if))                 |
| Shell selection               | OS-specific       | ✅ ([non OS-specific](./syntax.md#selecting-the-shell-for-run-steps))              |
| Wait for resources            | ✅                | ❌ (not yet implemented)                                                           |
| Output variables              | Limited           | ✅ ([structured outputs](./syntax.md#passing-outputs))                             |
| Built-in tasks                | ❌                | ✅ ([builtin:echo, builtin:fetch](./builtins.md))                                  |
| OCI registry support          | ❌                | ✅ ([OCI artifacts](./publish.md))                                                 |
| JSON Schema validation        | ❌                | ✅ ([schema validation](./syntax.md#schema-version))                               |
| Input validation              | ❌                | ✅ ([regex validation](./syntax.md#input-validation))                              |
| Package URL aliases           | ❌                | ✅ ([custom repository shortcuts](./syntax.md#package-url-aliases))                |
| Step timeout control          | ✅ (only seconds) | ✅ ([duration-based timeouts](./syntax.md#step-timeout-with-timeout))              |
| Step identification           | ❌                | ✅ ([ID and name properties](./syntax.md#step-identification-with-id-and-name))    |
| Error handling functions      | ❌                | ✅ ([failure(), always(), cancelled()](./syntax.md#conditional-execution-with-if)) |
| Dry run capability            | ❌                | ✅ ([preview execution](./cli.md#previewing-execution-with-dry-run))               |
| Environment variable defaults | ❌                | ✅ ([default-from-env](./syntax.md#default-values-from-environment-variables))     |
| Task-level input scoping      | ~                 | ✅ ([task-level inputs](./syntax.md#defining-input-parameters))                    |
| Script display control        | ❌                | ✅ ([show property](./syntax.md#controlling-script-display-with-show))             |

## Enhanced Features in maru2

When migrating from `maru-runner` to `maru2`, you'll gain access to several powerful new features:

### Rich Expression System

Maru2 introduces a comprehensive expression language for conditions with support for logical operators, string manipulation, and mathematical operations. See the [conditional execution documentation](./syntax.md#conditional-execution-with-if) for more details.

### Structured Outputs

Unlike maru-runner's simple variable capture, maru2 allows capturing multiple structured outputs from a single step. Learn more about [passing outputs between steps](./syntax.md#passing-outputs).

### Built-in Tasks

Maru2 includes pre-defined tasks for common operations, reducing the need for custom shell scripts. See the [built-in tasks documentation](./builtins.md) for available options.

### OCI Registry Support

Publish and consume workflows as OCI artifacts, enabling versioning, caching, and distributing workflows through container registries. See the [publishing workflows documentation](./publish.md) for details.

### Package URL Aliases

Create shorthand references for frequently used repositories with flexible authentication options. Learn more about [package URL aliases](./syntax.md#package-url-aliases).

### Input Validation

Validate inputs with regular expressions to ensure they meet specific format requirements before execution. See [input validation documentation](./syntax.md#input-validation) for details.

### Schema Validation

Enable real-time validation during editing with JSON Schema support, providing immediate feedback for syntax errors. Learn more about [schema validation](./syntax.md#schema-version).

### Script Display Control

Control whether scripts are displayed before execution with the `show` property, providing cleaner output for production workflows while maintaining full visibility during dry-run development. See [script display control documentation](./syntax.md#controlling-script-display-with-show) for details.

## Conclusion

Migrating from `maru-runner` to `maru2` involves restructuring your tasks file, but the core concepts remain similar. The main benefits of `maru2` include:

1. More GitHub Actions-like syntax for familiarity
2. Better [expression support](./syntax.md#conditional-execution-with-if) for conditional execution
3. Improved [step output handling](./syntax.md#passing-outputs)
4. [Built-in tasks](./builtins.md) for common operations
5. More flexible [remote task inclusion system](./syntax.md#run-a-task-from-a-remote-file)
6. [Schema validation](./syntax.md#schema-version) for better error detection
7. [Command line interface](./cli.md) improvements
8. Better typing system with input validation

Note that some features from `maru-runner` such as the `wait` functionality are not yet implemented in `maru2`. If you require these features, please open an issue on the project repository.
