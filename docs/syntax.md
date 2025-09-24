# Workflow Syntax

> [!INFO]
> Looking for `v0` schema docs? They are located on the [`v0.4.0` branch](https://github.com/defenseunicorns/maru2/blob/v0.4.0/docs/syntax.md).

A Maru2 workflow is any YAML file that conforms to the [`maru2` schema](../schema-validation#raw-schema).

Unless specified, the default file name is `tasks.yaml`.

## Structure

Similar to `Makefile`s, a Maru2 workflow is a map of tasks, where each task is a series of steps.

Checkout the comparison below:

```makefile
.DEFAULT_GOAL := build

build:
  CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" ./cmd/maru2

test:
  go test -v -race -cover -failfast -timeout 3m ./...

clean:
  @rm -rf bin/
```

```yaml
schema-version: v1
tasks:
  default:
    steps:
      - uses: build

  build:
    steps:
      - run: go build -o bin/ -ldflags="-s -w" ./cmd/maru2
        env:
          CGO_ENABLED: 0

  test:
    steps:
      - run: go test -v -race -cover -failfast -timeout 3m ./...

  clean:
    steps:
      - run: rm -rf bin/
        show: false
```

## Schema Version

Maru2 workflow files require a top-level `schema-version` property:

```yaml
schema-version: v1
```

Currently, `v1` is the recommended version (with `v0` still supported for backwards compatibility). This required property enables schema validation and will support future migrations as the workflow syntax evolves.

## Tasks

Tasks are the core building blocks of a Maru2 workflow. They are defined as keys within the top-level `tasks` map.

### Task names and descriptions

Task names must follow the following regex: `^[_a-zA-Z][a-zA-Z0-9_-]*$`.

This means:

1. Task names must start with a letter (a-z, A-Z) or underscore (\_)
2. After the first character, task names can contain letters, numbers, underscores, and hyphens
3. Task names cannot contain spaces or other special characters

Valid task names:

```yaml
schema-version: v1
tasks:
  build:
    steps: ...
  another-task:
    steps: ...
  UPPERCASE:
    steps: ...
  mIxEdCaSe:
    steps: ...
  WithNumbers123:
    steps: ...
  _private:
    steps: ...
```

Invalid task names:

```yaml
schema-version: v1
tasks:
  # Invalid: starts with a number
  1task:
    steps: ...

  # Invalid: contains a space
  "my task":
    steps: ...

  # Invalid: contains special characters
  "task@example":
    steps: ...
```

Note that the same naming rules apply to step IDs. This consistency makes it easier to work with both task names and step IDs throughout your workflows.

## Steps

Steps are the individual commands or actions that make up a task. They are executed sequentially within a task.

### `run` vs `uses`

- `run`: Executes a shell command or script.
- `uses`: References another task (in the same file, a local file, or a remote source) or a [built-in task](./builtins.md).

Both can be used interchangeably within a task, and interoperate cleanly with `with`.

### Selecting the Shell for `run` Steps

By default, Maru2 runs shell commands using `sh`. You can specify a different shell for a step using the `shell` field. Supported shells are:

- `sh` (default)
- `bash`
- `pwsh`
- `powershell`

Example:

```yaml
schema-version: v1
tasks:
  build:
    steps:
      # Run this step only on Linux
      - run: echo "Hello from sh on Linux"
        shell: sh
        if: os == "linux"

      # Run this step only on macOS
      - run: echo "Hello from bash on macOS"
        shell: bash
        if: os == "darwin"

      # Run this step only on Windows
      - run: Write-Host "Hello from PowerShell on Windows"
        shell: powershell
        if: os == "windows"
```

The shell field changes how the command is executed:

- `sh`: `sh -e -c {script}`
- `bash`: `bash -e -o pipefail -c {script}`
- `pwsh`: `pwsh -Command $ErrorActionPreference = 'Stop'; {script}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }`
- `powershell`: `powershell -Command $ErrorActionPreference = 'Stop'; {script}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }`

> **Note:** Support for `pwsh` and `powershell` is experimental and may change in future versions.

## Working directory with `dir`

You can specify a working directory for a step using the `dir` field. This applies to both `run` and `uses` steps.

```yaml
schema-version: v1
tasks:
  build:
    steps:
      # Run a command in a specific directory
      - run: npm install
        dir: frontend

      # Use a task in a specific directory
      - uses: test
        dir: backend
```

The `dir` field must be a relative path and cannot be an absolute path. It defaults to the current working directory `maru2` is executed in.

For `run` steps, the command is executed in the specified directory.

For `uses` steps, the referenced task is executed with the working directory set to the specified directory.

## Step Timeout with `timeout`

You can set a maximum duration for a step's execution using the `timeout` field. If the step exceeds this duration, it will be terminated.

```yaml
tasks:
  long-running-task:
    steps:
      - run: sleep 60 # This command runs for 60 seconds
        timeout: 30s # This step will time out after 30 seconds
      - run: echo "This message will not be displayed if the previous step times out"
```

The `timeout` value should be a string representing a duration, such as "30s" for 30 seconds, "1m" for 1 minute, or "1h30m" for 1 hour and 30 minutes.

See [https://pkg.go.dev/time#Duration](https://pkg.go.dev/time#Duration) for more information on supported duration units.

When a step times out, the task will fail, and any subsequent steps that do not explicitly handle failures (e.g., with `if: always()` or `if: failure()`) will be skipped.

## Controlling script display with `show`

By default, Maru2 displays the rendered script before executing it. You can control this behavior using the `show` field:

```yaml
schema-version: v1
tasks:
  example:
    steps:
      - run: echo "This script will be shown"
        # show: true (default)
      - run: echo "This script will be hidden"
        show: false
      - run: echo "This script will be shown again"
```

When `show` is set to `false`, the script content is not displayed before execution, but the command still runs normally and produces output.

> **Note**: In dry-run mode (`--dry-run`), all scripts are shown regardless of the `show` setting to help you preview what would be executed.

## Muting terminal output with `mute`

You can suppress a step's output using the `mute` field. When `mute` is set to `true`, the step's stdout and stderr will not be displayed, though the step will still execute and can still set outputs. The `mute` field only affects command output, not the script display (use `show: false` to hide the script itself).

```yaml
schema-version: v1
tasks:
  example:
    steps:
      - run: echo "Script shown, output hidden"
        mute: true
      - run: echo "Script hidden, output shown"
        show: false
      - run: echo "Both script and output hidden"
        show: false
        mute: true
```

## Defining input parameters

Maru2 allows you to define input parameters for your tasks. These parameters can be required or optional, and can have default values.

```yaml
schema-version: v1
tasks:
  greet:
    inputs:
      # Required input (default behavior)
      name:
        description: "Your name"

      # Optional input with default value
      greeting:
        description: "Greeting to use"
        default: "Hello"
        required: false

      # Required input with default from environment variable
      username:
        description: "Username"
        default-from-env: USER
    steps:
      - run: echo "${{ input "greeting" }}, ${{ input "name" }}! Your username is ${{ input "username" }}."
```

Input parameters have the following properties:

- `description`: A description of the parameter (required)
- `required`: Whether the parameter is required (defaults to `true`)
- `default`: A default value for the parameter
- `default-from-env`: An environment variable to use as the default value. Environment variable names must start with a letter or underscore, and can contain letters, numbers, and underscores (e.g., `MY_ENV_VAR`, `_ANOTHER_VAR`).
- `validate`: A regular expression to validate the parameter value
- `deprecated-message`: A warning message to display when the parameter is used (for deprecated parameters)

See [priority order for default values](#priority-order-for-default-values).

## Passing inputs

On top of the builtin behavior, Maru2 provides a few additional helpers:

- `${{ input "<name>" }}`: calling an input
  - If the task is top-level (called via CLI), `with` values are received from the `--with` flag.
  - If the task is called from another task, `with` values are passed from the calling step.
- `${{ which "<key>" }}`: expands `key` to a registered executable or falls back to $PATH lookup
  - First checks for registered shortcuts (configured via Maru2 wrappers)
  - If no shortcut is found, falls back to searching for the executable in $PATH using `exec.LookPath()`
  - If the executable is not found in either the registration system or $PATH, returns an error that will cause the step to fail
  - There are no `which` shortcuts configured for Maru2 by default, these are left up to wrapper implementations.
  - ex: `${{ which "uds" }} --version` when Maru2 is run as: `uds run foo ...` renders as `/absolute/path/to/uds --version`
  - ex: `${{ which "git" }} status` when no `git` shortcut is registered will find `git` in $PATH and render as `/usr/bin/git status`
  - ex: `${{ which "nonexistent" }} --help` will fail with error `exec: "nonexistent": executable file not found in $PATH`
- `OS`, `ARCH`, `PLATFORM`: the current OS, architecture, or platform

```yaml
schema-version: v1
tasks:
  echo:
    inputs:
      date:
        description: The date
        default: now # default to "now" if input is nil
      name:
        description: Your name
        required: true
    steps:
      - run: echo "Hello, ${{ input "name" }}, today is ${{ input "date" }}"
      - run: echo "The current OS is ${{ .OS }}, architecture is ${{ .ARCH }}, platform is ${{ .PLATFORM }}"
```

```sh
maru2 echo --with name=$(whoami) --with date=$(date)
```

## Defining environment variables

You can set custom environment variables for individual steps using the `env` field. Variable names follow the same rules as task names. Variable values leverage the same input templating engine as `run`.

```yaml
schema-version: v1
tasks:
  deploy:
    inputs:
      deployment-env:
        description: "Deployment environment"
        default: "development"
    steps:
      - id: "build-app"
        name: "Build application with environment config"
        run: |
          npm run build
          echo "build_version=$(git rev-parse --short HEAD)" >> $MARU2_OUTPUT
        env:
          NODE_ENV: ${{ input "deployment-env" }}

      - name: "Deploy application"
        run: |
          echo "Deploying version $BUILD_VERSION to $DEPLOY_TARGET"
          ${{ which "zarf" }} dev deploy .
        env:
          BUILD_VERSION: ${{ from "build-app" "build_version" }}
          DEPLOY_TARGET: ${{ input "deployment-env" }}
          ZARF_NO_PROGRESS: true
          KUBECONFIG: /etc/kubernetes/${{ input "deployment-env" }}-config
```

Environment variables set in the `env` field apply to that specific step. For `run` steps, they only apply to that single step. For `uses` steps, they are passed down to ALL steps in the called task.

When using the `env` field on a `uses` step, those environment variables are templated and passed to all steps within the called task:

```yaml
schema-version: v1
tasks:
  parent-task:
    inputs:
      some-input:
        description: "Some input value"
        required: true
    steps:
      - uses: file:subtask.yaml?task=child-task
        with:
          message: "Hello from parent"
        env:
          PARENT_VAR: "value-from-parent"
          TEMPLATED_VAR: ${{ input "some-input" }}

  # In subtask.yaml, both steps will have access to PARENT_VAR and TEMPLATED_VAR
```

### `env` Restrictions

You cannot set the `PWD` environment variable through the `env` field. Use the [`dir` field](#working-directory-with-dir) instead to control the working directory:

```yaml
tasks:
  example:
    steps:
      - run: pwd
        dir: subdirectory # Use dir field, not env: { PWD: "..." }
```

## Run another task as a step

Calling another task within the same workflow is as simple as using the task name, similar to Makefile targets.

```yaml
schema-version: v1
tasks:
  general-kenobi:
    inputs:
      response:
        description: "Response message"
        required: true
    steps:
      - run: echo "General Kenobi, you are a bold one"
      - run: echo "${{ input "response" }}"

  hello:
    steps:
      - run: echo "Hello There!"
      - uses: general-kenobi
        with:
          response: Your move
        env:
          GREETING_TYPE: "formal"
```

```sh
maru2 hello
```

## Run a task from a local file

Calling a task from a local file uses the format `file:<relative-filepath>?task=<taskname>`.

- The file path is required and cannot be a directory.
- If the task name is not provided, the `default` task is run.

```yaml
schema-version: v1
tasks:
  simple:
    inputs:
      message:
        description: "Message to echo"
        required: true
    steps:
      - run: echo "${{ input "message" }}"
```

```yaml
schema-version: v1
tasks:
  echo:
    inputs:
      message:
        description: "Message to echo"
        required: true
    steps:
      - uses: file:tasks/echo.yaml?task=simple
        with:
          message: ${{ input "message" }}
```

```sh
maru2 echo --with message="Hello, World!"
```

## Run a task from a remote file

If a `uses` reference is not a local task or a `file:` reference, it is parsed as a URL and fetched based on its protocol scheme. If no task is specified in the URL, the `task` query parameter defaults to `default`.

- `pkg:`: leverages the [package-url spec](https://github.com/package-url/purl-spec) to create authenticated Go clients for GitHub / GitLab. Has access to [aliases](package-url-aliases), by default uses `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables for GitHub / GitLab authentication.
- `http:/https:`: leverages standard HTTP GET requests for raw content.
- `oci:`: leverages ORAS and the ALPHA [`maru2-publish`](./publish.md) CLI to fetch. While this feature is currently in ALPHA, the following usage samples for other protocol schemes will generally apply.

examples:

```yaml
schema-version: v1
tasks:
  remote-pkg:
    steps:
      - uses: pkg:github/defenseunicorns/maru2@main?task=echo
  remote-http:
    steps:
      - uses: https://raw.githubusercontent.com/defenseunicorns/maru2/main/testdata/simple.yaml?task=echo
  remote-oci:
    steps:
      - uses: oci:staging.uds.sh/public/my-workflow:latest
```

## Aliases

Maru2 supports defining aliases for package URLs or local paths to create shorthand references for commonly used package types.

Examples of using aliases in workflow files:

```yaml
schema-version: v1
aliases:
  gl:
    type: gitlab
    base-url: https://gitlab.example.com
  gh:
    type: github
  internal:
    type: gitlab
    base-url: https://gitlab.internal.company.com
  local-tasks:
    path: tasks/common.yaml

tasks:
  # Using the full GitHub package URL
  alpha:
    steps:
      - uses: pkg:github/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
        with:
          message: Hello, World!

  # Using the 'gh' alias defined in ~/.maru2/config.yaml
  bravo:
    steps:
      - uses: pkg:gh/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
        with:
          message: Hello, World!

  # Using the 'gl' alias with GitLab
  charlie:
    steps:
      - uses: pkg:gl/noxsios/maru2@main?task=echo#testdata/simple.yaml
        with:
          message: Hello, World!
```

The alias `gl` will be resolved to `gitlab` with the base URL qualifier set to `https://gitlab.example.com`.

Maru2 supports two types of aliases, which can be defined in the global `~/.maru2/config.yaml` file or within a workflow file's `aliases` block.

### Package URL Aliases

Package URL aliases create shortcuts for remote repositories (e.g., GitHub, GitLab). They have the following properties:

- `type` (**required**): The package URL type (`github`, `gitlab`, etc.).
- `base-url` (optional): The base URL for the repository, useful for self-hosted instances.
- `token-from-env` (optional): The name of an environment variable containing an access token.

### Local File Aliases

Local file aliases create shortcuts for local workflow files. They have the following properties:

- `path` (**required**): The relative path to a local workflow file.

Local aliases allow you to create shorthand references to workflow files in your project:

```yaml
schema-version: v1
aliases:
  common:
    path: workflows/common.yaml
  utils:
    path: scripts/utils.yaml

tasks:
  build:
    steps:
      - uses: common:setup
      - uses: utils:compile
```

This is particularly useful for organizing complex projects with multiple workflow files:

```text
my-project/
├── tasks.yaml              # Main workflow
├── workflows/
│   ├── common.yaml         # Shared setup tasks
│   └── deployment.yaml     # Deployment workflows
└── scripts/
    └── utils.yaml          # Utility tasks
```

## Step identification with `id` and `name`

Each step in a Maru2 workflow can have an optional `id` and `name` field:

- `id`: A unique identifier for the step, used to reference outputs from the step in subsequent steps
- `name`: A human-readable description of what the step does

The `id` field must follow the same naming rules as task names: `^[_a-zA-Z][a-zA-Z0-9_-]*$`

```yaml
schema-version: v1
tasks:
  build:
    steps:
      - name: "Install dependencies"
        run: npm install
        id: install
      - name: "Build application"
        run: npm run build
        id: build
```

The `name` field is primarily for documentation purposes and to improve readability of the workflow, while the `id` field is used for referencing outputs.

## Passing outputs

Maru2 allows steps to produce outputs that can be consumed by subsequent steps. This leverages a similar mechanism to GitHub Actions.

To set outputs from a step:

1. Assign an `id` to the step
2. Write to the `$MARU2_OUTPUT` file in the format `key=value`
3. Reference the output in subsequent steps using `${{ from "step-id" "output-key" }}`

```yaml
schema-version: v1
tasks:
  color:
    steps:
      - run: |
          echo "selected-color=green" >> $MARU2_OUTPUT
        id: color-selector
      - run: echo "The selected color is ${{ from "color-selector" "selected-color" }}"
```

```sh
maru2 color

echo "selected-color=green" >> $MARU2_OUTPUT
echo "The selected color is green"
The selected color is green
```

You can set multiple outputs from a single step by writing multiple lines to the `$MARU2_OUTPUT` file:

```yaml
schema-version: v1
tasks:
  multi-output:
    steps:
      - run: |
          echo "name=John" >> $MARU2_OUTPUT
          echo "age=30" >> $MARU2_OUTPUT
          echo "city=New York" >> $MARU2_OUTPUT
        id: user-info
      - run: echo "User ${{ from "user-info" "name" }} is ${{ from "user-info" "age" }} years old and lives in ${{ from "user-info" "city" }}"
```

Outputs are only available to steps that come after the step that sets them. If a step with an ID doesn't write anything to `$MARU2_OUTPUT`, no outputs will be available from that step.

## Default values from environment variables

In addition to static default values, you can specify environment variables as default values for input parameters using the `default-from-env` field.

```yaml
schema-version: v1
tasks:
  hello:
    inputs:
      name:
        description: "Your name"
        default-from-env: USER
    steps:
      - run: echo "Hello, ${{ input "name" }}"
```

```sh
# Uses the USER environment variable as the default value
maru2 hello

echo "Hello, razzle"
Hello, razzle

# Provided input overrides the environment variable
maru2 hello --with name="Jeff"

echo "Hello, Jeff"
Hello, Jeff
```

### Priority Order for Default Values

You can specify both `default` and `default-from-env` for the same input parameter. Maru2 uses the following priority order:

1. **Provided input** (via `--with` flag or `with:` property) - highest priority
2. **Environment variable** (via `default-from-env`) - if the environment variable exists
3. **Static default** (via `default`) - fallback if environment variable doesn't exist
4. **No value** - if none of the above are available and the input is required, an error occurs

```yaml
schema-version: v1
tasks:
  hello:
    inputs:
      ci:
        description: "Am I running in CI?"
        default-from-env: CI
        default: false
    steps:
      - run: echo "CI is ${{ input "ci" }}"
```

## Input validation

Maru2 allows you to validate input parameters using regular expressions. This ensures that inputs meet specific format requirements before the task is executed.

To add validation to an input parameter, use the `validate` field with a regular expression pattern:

```yaml
schema-version: v1
tasks:
  hello:
    inputs:
      name:
        description: "Your name"
        validate: ^\w+$ # Only allow alphanumeric characters and underscores

      version:
        description: "Semantic version"
        validate: ^\d+\.\d+\.\d+$ # Enforce semantic versioning format (e.g., 1.2.3)

      email:
        description: "Email address"
        validate: ^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$ # Basic email validation
    steps:
      - run: echo "Hello, ${{ input "name" }}"
```

When a task is run, Maru2 will validate all inputs against their respective patterns. If validation fails, an error is returned and the task is not executed:

```sh
# fails due to missing input!
maru2 hello

ERRO missing required input: "name"
ERRO at (file:tasks.yaml)

# fails due to invalid input
maru2 hello --with name="Goodbye, World!"

ERRO failed to validate: input=name, value=Goodbye, World!, regexp=^\w+$
ERRO at (file:tasks.yaml)

# succeeds!
maru2 hello --with name="Jeff"

echo "Hello, Jeff"
Hello, Jeff
```

Validation is performed after any default values are applied and before the task is executed. This ensures that even default values must pass validation.

## Conditional execution with `if`

Maru2 supports conditional execution of steps using `if`. `if` statements are [expr](https://github.com/expr-lang/expr) expressions. They have access to all expr stdlib functions, and five extra helper functions:

- `failure()`: Run this step only if a previous step has failed (from timeout, script failure, syntax errors, `SIGINT`, etc...)
- `always()`: Run this step regardless of whether previous steps have succeeded or failed
- `cancelled()`: Run this step _only_ if the task was cancelled (e.g., via `Ctrl+C` or a `SIGINT` signal, `SIGTERM` kills the task entirely).
- `input("name")`: Access an input value by name. Only one argument is allowed. Returns the value of the input (which may be a string, number, or boolean), or `nil` if the input doesn't exist.
- `from("step-id", "output-key")`: Access an output from a previous step. Only two arguments are allowed: the step ID and the output key. Returns the output value, or `nil` if the step or output key doesn't exist.

Go's `runtime` helper constants are also available- `os`, `arch`, `platform`: the current OS, architecture, or platform.

> **Note**: The behavior of `input()` and `from()` in `if` expressions differs from their behavior in templates (like `${{ input "name" }}`). In `if` expressions, these functions return `nil` when values don't exist, allowing you to check for missing values gracefully. In templates, missing values cause errors and prevent the step from executing.

By default (without an `if` directive), steps will only run if all previous steps have succeeded.

> **Note**: In dry-run mode, steps with `if` conditions that evaluate to `false` will still be executed (with a warning) to help you preview the complete workflow execution path.

```yaml
schema-version: v1
tasks:
  example:
    inputs:
      text:
        description: Some text to echo
        default: foo
    steps:
      - run: echo "This step always runs first"
      - run: exit 1 # This step will fail
      - run: echo "This step will be skipped because the previous step failed"
      - if: failure()
        run: echo "This step runs because a previous step failed"
      - if: always()
        run: echo "This step always runs, regardless of previous failures"
      - if: len(input("text")) > 5
        run: echo "I only run when ${{ input "text" }} has a len greater than 5"
```

```sh
maru2 example >/dev/null

echo "This step always runs first"
exit 1
echo "This step runs because a previous step failed"
echo "This step always runs, regardless of previous failures"

ERRO exit status 1
ERRO at example[1] (file:tasks.yaml)
```

## CI Environment Integration

Maru2 provides optional enhanced output formatting when running in CI environments to improve log readability and organization.

### Output Grouping with `collapse`

When the `collapse` property is set to `true` on a task, Maru2 automatically groups the task's output in supported CI environments.
No other configuration is required - the output grouping feature activates automatically when these environments are detected.

- GitHub Actions (`GITHUB_ACTIONS=true`): Uses `::group::` and `::endgroup::` commands to create collapsible log sections
- GitLab CI (`GITLAB_CI=true`): Uses section markers to create collapsible log sections

```yaml
schema-version: v1
tasks:
  build:
    description: "Build the application"
    collapse: true
    steps:
      - run: echo "Installing dependencies..."
      - run: npm install
      - run: echo "Building application..."
      - run: npm run build
```

In GitHub Actions, this produces:

```text
::group::build: Build the application
Installing dependencies...
npm install output...
Building application...
npm run build output...
::endgroup::
```

In local/non-CI environments, the `collapse` property has no effect and output is displayed normally.

Nested tasks with their own `collapse: true` property will not create additional nested groups within an already collapsed section.

While this is supported in GitLab, it is not in GitHub and consistency is better in this case.
