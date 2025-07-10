# Workflow Syntax

A Maru2 workflow is any YAML file that conforms to the [`maru2` schema](../schema-validation#raw-schema).

Unless specified, the default file name is `tasks.yaml`.

## Structure

Similar to `Makefile`s, a Maru2 workflow is a map of tasks, where each task is a series of steps.

Checkout the comparison below:

```makefile {filename="Makefile"}
.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" ./cmd/maru2

test:
	go test -v -race -cover -failfast -timeout 3m ./...

clean:
	rm -rf bin/
```

```yaml {filename="tasks.yaml"}
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

## Task names and descriptions

Task names must follow the following regex: `^[_a-zA-Z][a-zA-Z0-9_-]*$`.

This means:

1. Task names must start with a letter (a-z, A-Z) or underscore (\_)
2. After the first character, task names can contain letters, numbers, underscores, and hyphens
3. Task names cannot contain spaces or other special characters

<!-- Try it out below:

<input spellcheck="false" placeholder="some-task" id="task-name-regex" />
<span id="regex-result"></span>

<script type="module" defer>
  const input = document.getElementById('task-name-regex');
  const result = document.getElementById('regex-result');
  input.addEventListener('input', () => {
    const regex = /^[_a-zA-Z][a-zA-Z0-9_-]*$/;
    if (input.value === '') {
      result.textContent = '';
      return;
    }
    const valid = regex.test(input.value);
    result.textContent = valid ? '✅' : '❌';
  });
</script> -->

Valid task names:

```yaml
build: ...
another-task: ...
UPPERCASE: ...
mIxEdCaSe: ...
WithNumbers123: ...
_private: ...
```

Invalid task names:

```yaml
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

## Working directory with `dir`

You can specify a working directory for a step using the `dir` field. This applies to both `run` and `uses` steps.

```yaml {filename="tasks.yaml"}
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

## Defining input parameters

Maru2 allows you to define input parameters for your tasks. These parameters can be required or optional, and can have default values.

```yaml {filename="tasks.yaml"}
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

- `${{ input <name> }}`: calling an input
  - If the task is top-level (called via CLI), `with` values are received from the `--with` flag.
  - If the task is called from another task, `with` values are passed from the calling step.
- `os`, `arch`, `platform`: the current OS, architecture, or platform

```yaml {filename="tasks.yaml"}
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

## Run another task as a step

Calling another task within the same workflow is as simple as using the task name, similar to Makefile targets.

```yaml {filename="tasks.yaml"}
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

```yaml {filename="tasks/echo.yaml"}
tasks:
  simple:
    - run: echo "${{ input "message" }}"
```

```yaml {filename="tasks.yaml"}
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

> [!IMPORTANT]
> `uses` syntax leverages the [package-url spec](https://github.com/package-url/purl-spec)

### Package URL Aliases

Maru2 supports defining aliases for package URLs to create shorthand references for commonly used package types. For detailed information on setting up and using aliases, see the [Configuration](./config.md#aliases-configuration) document.

You can define aliases for package URLs to simplify references to frequently used repositories or to set default qualifiers. Aliases are defined in the global configuration file located at `~/.maru2/config.yaml`.

If a version is not specified in a `pkg` URL, it defaults to `main`.

Example configuration with aliases:

```yaml {filename="~/.maru2/config.yaml"}
aliases:
  gl:
    type: gitlab
    base: https://gitlab.example.com
  gh:
    type: github
  internal:
    type: gitlab
    base: https://gitlab.internal.company.com
```

Examples of using aliases in workflow files:

```yaml {filename="tasks.yaml"}
tasks:
  # Using the full GitHub package URL
  remote-echo:
    - uses: pkg:github/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

```yaml {filename="tasks.yaml"}
tasks:
  # Using the 'gh' alias defined in ~/.maru2/config.yaml
  remote-echo:
    - uses: pkg:gh/defenseunicorns/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

```yaml {filename="tasks.yaml"}
tasks:
  # Using the 'gl' alias with GitLab
  remote-echo:
    - uses: pkg:gl/noxsios/maru2@main?task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

The alias `gl` will be resolved to `gitlab` with the base URL qualifier set to `https://gitlab.example.com`.

You can also override qualifiers defined in the alias by specifying them in the package URL:

```yaml {filename="tasks.yaml"}
tasks:
  remote-echo:
    - uses: pkg:gl/noxsios/maru2@main?base=https://other-gitlab.com&task=echo#testdata/simple.yaml
      with:
        message: Hello, World!
```

This will use `https://other-gitlab.com` as the base URL instead of the one defined in the alias.

```yaml {filename="tasks.yaml"}
tasks:
  remote-echo:
    - uses: https://raw.githubusercontent.com/defenseunicorns/maru2/main/testdata/simple.yaml?task=echo
      with:
        message: Hello, World!
```

```sh
maru2 remote-echo
```

## Step identification with `id` and `name`

Each step in a Maru2 workflow can have an optional `id` and `name` field:

- `id`: A unique identifier for the step, used to reference outputs from the step in subsequent steps
- `name`: A human-readable description of what the step does

The `id` field must follow the same naming rules as task names: `^[_a-zA-Z][a-zA-Z0-9_-]*$`

```yaml {filename="tasks.yaml"}
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

```yaml {filename="tasks.yaml"}
tasks:
  color:
    - run: |
        echo "selected-color=green" >> $MARU2_OUTPUT
      id: color-selector
    - run: echo "The selected color is ${{ from "color-selector" "selected-color" }}"
```

```sh
maru2 color

$ echo "selected-color=green" >> $MARU2_OUTPUT
$ echo "The selected color is green"
The selected color is green
```

You can set multiple outputs from a single step by writing multiple lines to the `$MARU2_OUTPUT` file:

```yaml {filename="tasks.yaml"}
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

$ echo "Hello, razzle"
Hello, razzle

# Provided input overrides the environment variable
maru2 hello --with name="Jeff"

$ echo "Hello, Jeff"
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

$ echo "Hello, Jeff"
Hello, Jeff
```

Validation is performed after any default values are applied and before the task is executed. This ensures that even default values must pass validation.

## Conditional execution with `if`

Maru2 supports conditional execution of steps using `if`. `if` statements are [expr](github.com/expr-lang/expr) expressions. They have access to `from`, `inputs`, all expr stdlib functions, and two extra helper functions:

- `failure()`: Run this step only if a previous step has failed
- `always()`: Run this step regardless of whether previous steps have succeeded or failed

By default (without an `if` directive), steps will only run if all previous steps have succeeded.

```yaml
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
    - if: len(input.text) > 5
      run: echo "I only run when ${{ input "text" }} has a len greater than 5"
```

```sh
maru2 example >/dev/null

$ echo "This step always runs first"
$ exit 1
$ echo "This step runs because a previous step failed"
$ echo "This step always runs, regardless of previous failures"

ERRO exit status 1
ERRO at example[1] (file:tasks.yaml)
```

## Error handling and traceback

When a step in a Maru2 workflow fails, the error is propagated up the call stack with a traceback that shows the path of execution. This helps you identify where in your workflow the error occurred, especially for complex workflows with nested task calls.

```yaml {filename="tasks.yaml"}
tasks:
  fail:
    - run: exit 1

  caller:
    - run: echo "Starting workflow"
    - uses: fail
    - run: echo "This step will be skipped"
```

```sh
maru2 caller

$ echo "Starting workflow"
Starting workflow
$ exit 1

ERRO exit status 1
  traceback (most recent call first)=
  │ at fail[0] (file:tasks.yaml)
  │ at caller[1] (file:tasks.yaml)
```

The traceback shows that the error occurred in the first step (`[0]`) of the `fail` task, which was called from the second step (`[1]`) of the `caller` task.

This traceback information is particularly valuable when debugging complex workflows with multiple levels of task nesting or when using remote tasks.
