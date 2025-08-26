# Workflow Syntax

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
	rm -rf bin/
```

```yaml
schema-version: v0
tasks:
  default:
    - uses: build

  build:
    - run: |
        CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" ./cmd/maru2

  test:
    - run: go test -v -race -cover -failfast -timeout 3m ./...

  clean:
    - run: rm -rf bin/
```

## Schema Version

Maru2 workflow files require a top-level `schema-version` property:

```yaml
schema-version: v0
```

Currently, only `v0` is supported. This required property enables schema validation and will support future migrations as the workflow syntax evolves.

## Task names and descriptions

Task names must follow the following regex: `^[_a-zA-Z][a-zA-Z0-9_-]*$`.

This means:

1. Task names must start with a letter (a-z, A-Z) or underscore (\_)
2. After the first character, task names can contain letters, numbers, underscores, and hyphens
3. Task names cannot contain spaces or other special characters

Valid task names:

```yaml
schema-version: v0
tasks:
  build: ...
  another-task: ...
  UPPERCASE: ...
  mIxEdCaSe: ...
  WithNumbers123: ...
  _private: ...
```

Invalid task names:

```yaml
schema-version: v0
tasks:
  # Invalid: starts with a number
  1task: ...

  # Invalid: contains a space
  "my task": ...

  # Invalid: contains special characters
  "task@example": ...
```

Note that the same naming rules apply to step IDs. This consistency makes it easier to work with both task names and step IDs throughout your workflows.

## `run` vs `uses`

- `run`: runs a shell command/script
- `uses`: calls another task / executes a builtin

Both can be used interchangeably within a task, and interoperate cleanly with `with`.

### Selecting the Shell for `run` Steps

By default, Maru2 runs shell commands using `sh`. You can specify a different shell for a step using the `shell` field. Supported shells are:

- `sh` (default)
- `bash`
- `pwsh`
- `powershell`

Example:

```yaml
schema-version: v0
tasks:
  build:
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

- `sh`: `sh -e -u -c {script}`
- `bash`: `bash -e -u -o pipefail -c {script}`
- `pwsh`: `pwsh -Command $ErrorActionPreference = 'Stop'; {script}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }`
- `powershell`: `powershell -Command $ErrorActionPreference = 'Stop'; {script}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }`

> **Note:** Support for `pwsh` and `powershell` is experimental and may change in future versions.

## Working directory with `dir`

You can specify a working directory for a step using the `dir` field. This applies to both `run` and `uses` steps.

```yaml
schema-version: v0
tasks:
  build:
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
    - run: sleep 60 # This command runs for 60 seconds
      timeout: 30s # This step will time out after 30 seconds
    - run: echo "This message will not be displayed if the previous step times out"
```

The `timeout` value should be a string representing a duration, such as "30s" for 30 seconds, "1m" for 1 minute, or "1h30m" for 1 hour and 30 minutes.

See [https://pkg.go.dev/time#Duration](https://pkg.go.dev/time#Duration) for more information on supported duration units.

When a step times out, the task will fail, and any subsequent steps that do not explicitly handle failures (e.g., with `if: always()` or `if: failure()`) will be skipped.

## Defining input parameters

Maru2 allows you to define input parameters for your tasks. These parameters can be required or optional, and can have default values.

```yaml
schema-version: v0
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

tasks:
  greet:
    - run: echo "${{ input "greeting" }}, ${{ input "name" }}! Your username is ${{ input "username" }}."
```

Input parameters have the following properties:

- `description`: A description of the parameter (required)
- `required`: Whether the parameter is required (defaults to `true`)
- `default`: A default value for the parameter
- `default-from-env`: An environment variable to use as the default value. Environment variable names must start with a letter or underscore, and can contain letters, numbers, and underscores (e.g., `MY_ENV_VAR`, `_ANOTHER_VAR`).
- `validate`: A regular expression to validate the parameter value
- `deprecated-message`: A warning message to display when the parameter is used (for deprecated parameters)

Note that `default` and `default-from-env` are mutually exclusive - you can only specify one of them for a given input parameter.

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
schema-version: v0
inputs:
  date:
    description: The date
    default: now # default to "now" if input is nil

tasks:
  echo:
    - run: echo "Hello, ${{ input "name" }}, today is ${{ input "date" }}"
    - run: echo "The current OS is ${{ .OS }}, architecture is ${{ .ARCH }}, platform is ${{ .PLATFORM }}"
```

```sh
maru2 echo --with name=$(whoami) --with date=$(date)
```

## Defining environment variables

You can set custom environment variables for individual steps using the `env` field. Variable names follow the same rules as task names. Variable values leverage the same input templating engine as `run`.

```yaml
schema-version: v0
inputs:
  deployment-env:
    description: "Deployment environment"
    default: "development"

tasks:
  deploy:
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

### Restrictions and Behavior

**PWD Restriction**: You cannot set the `PWD` environment variable through the `env` field. Use the [`dir` field]() instead to control the working directory:

```yaml
tasks:
  example:
    - run: pwd
      dir: subdirectory # Use dir field, not env: { PWD: "..." }
```

**Variable Precedence**: Step-level environment variables can override existing environment variables from the system or parent process.

**Scope**: Environment variables set in the `env` field only apply to that specific step. They do not persist to subsequent steps unless explicitly passed through outputs or inputs.

## Run another task as a step

Calling another task within the same workflow is as simple as using the task name, similar to Makefile targets.

```yaml
schema-version: v0
tasks:
  general-kenobi:
    - run: echo "General Kenobi, you are a bold one"
    - run: echo "${{ input "response" }}"

  hello:
    - run: echo "Hello There!"
    - uses: general-kenobi
      with:
        response: Your move
```

```sh
maru2 hello
```

## Run a task from a local file

Calling a task from a local file takes two arguments: the file path (required) and the task name (optional).

`file:<relative filepath>?task=<taskname>`

If the filepath is a directory, `tasks.yaml` is appended to the path.

If the task name is not provided, the `default` task is run.

```yaml
schema-version: v0
tasks:
  simple:
    - run: echo "${{ input "message" }}"
```

```yaml
schema-version: v0
tasks:
  echo:
    - uses: file:tasks/echo.yaml?task=simple
      with:
        message: ${{ input "message" }}
```

```sh
maru2 echo --with message="Hello, World!"
```

## Run a task from a remote file

If a `uses` reference is not within the workflow, nor a `file:` reference, it is parsed as a URL, then fetched based upon the URL protocol scheme. If no task is specifed, the `task` query parameter is set to `default`.

- `pkg:`: leverages the [package-url spec](https://github.com/package-url/purl-spec) to create authenticated Go clients for GitHub / GitLab. Has access to [aliases](package-url-aliases), by default uses `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables for GitHub / GitLab authentication.
- `http:/https:`: leverages standard HTTP GET requests for raw content.
- `oci:`: leverages ORAS and the ALPHA [`maru2-publish`](./publish.md) CLI to fetch. While this feature is currently in ALPHA, the following usage samples for other protocol schemes will generally apply.

examples:

```yaml
schema-version: v0
tasks:
  remote-pkg:
    - uses: pkg:github/defenseunicorns/maru2@main?task=echo
  remote-http:
    - uses: https://raw.githubusercontent.com/defenseunicorns/maru2/main/testdata/simple.yaml?task=echo
  remote-oci:
    - uses: oci:staging.uds.sh/public/my-workflow:latest
```

### Package URL Aliases

Maru2 supports defining aliases for package URLs to create shorthand references for commonly used package types.

You can define aliases for package URLs to simplify references to frequently used repositories or to set default qualifiers.

If a version is not specified in a `pkg` URL, it defaults to `main`.

Examples of using aliases in workflow files:

```yaml
schema-version: v0
aliases:
  gl:
    type: gitlab
    base: https://gitlab.example.com
  gh:
    type: github
  internal:
    type: gitlab
    base: https://gitlab.internal.company.com

tasks:
  # Using the full GitHub package URL
  alpha:
    - uses: pkg:github/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!

  # Using the 'gh' alias defined in ~/.maru2/config.yaml
  bravo:
    - uses: pkg:gh/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!

  # Using the 'gl' alias with GitLab
  charlie:
    - uses: pkg:gl/noxsios/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

The alias `gl` will be resolved to `gitlab` with the base URL qualifier set to `https://gitlab.example.com`.

An alias has the following properties:

- `alias_name`: A short name you want to use as an alias
- `type`: The actual package URL type (github, gitlab, etc.) - this is required
- `base`: (Optional) Base URL for the repository (useful for self-hosted instances)
- `token-from-env`: (Optional) Environment variable name containing an access token. Environment variable names must start with a letter or underscore, and can contain letters, numbers, and underscores (e.g., `MY_ENV_VAR`, `_ANOTHER_VAR`).

You can also override qualifiers defined in the alias by specifying them in the package URL:

```yaml
schema-version: v0
tasks:
  remote-echo:
    - uses: pkg:gl/noxsios/maru2@main?base=https://other-gitlab.com&task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

## Step identification with `id` and `name`

Each step in a Maru2 workflow can have an optional `id` and `name` field:

- `id`: A unique identifier for the step, used to reference outputs from the step in subsequent steps
- `name`: A human-readable description of what the step does

The `id` field must follow the same naming rules as task names: `^[_a-zA-Z][a-zA-Z0-9_-]*$`

```yaml
schema-version: v0
tasks:
  build:
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
schema-version: v0
tasks:
  color:
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
schema-version: v0
tasks:
  multi-output:
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
schema-version: v0
inputs:
  name:
    description: "Your name"
    default-from-env: USER

tasks:
  hello:
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

If the specified environment variable is not set, an error will be returned:

```sh
# With NON_EXISTENT_ENV_VAR not set
maru2 hello

ERRO environment variable "NON_EXISTENT_ENV_VAR" not set and no input provided for "name"
ERRO at (file:tasks.yaml)
```

Note that `default` and `default-from-env` are mutually exclusive - you can only specify one of them for a given input parameter.

## Input validation

Maru2 allows you to validate input parameters using regular expressions. This ensures that inputs meet specific format requirements before the task is executed.

To add validation to an input parameter, use the `validate` field with a regular expression pattern:

```yaml
schema-version: v0
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

tasks:
  hello:
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
- `input("name")`: Access an input value by name. Only one argument is allowed. Returns the value of the input, which may be a string, number, or boolean.
- `from("step-id", "output-key")`: Access an output from a previous step. Only two arguments are allowed: the step ID and the output key.

Go's `runtime` helper constants are also available- `os`, `arch`, `platform`: the current OS, architecture, or platform.

By default (without an `if` directive), steps will only run if all previous steps have succeeded.

```yaml
schema-version: v0
inputs:
  text:
    description: Some text to echo
    default: foo

tasks:
  example:
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
