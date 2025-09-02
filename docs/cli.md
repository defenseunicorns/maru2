# Maru2 CLI Reference

This guide explains how to use the Maru2 command line interface effectively.

## Basic Usage

```text
maru2 [task] [flags]
```

Without any arguments, Maru2 runs the `default` task from the `tasks.yaml` file in the current directory.

## Common Examples

```sh
# Run the default task
maru2

# Run a specific task
maru2 build

# Run multiple tasks in sequence
maru2 clean build test

# Run a task with input parameters
maru2 deploy --with environment=production --with version=1.2.3

# List all available tasks
maru2 --list

# Use a different workflow file
maru2 --from other-tasks.yaml build

# Run a task from a remote workflow
maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo

# Run a task using an alias (if aliases are defined in the workflow)
maru2 alias-name:task-name
```

## All Available Flags

```text
Flags:
  -C, --directory string      Change to directory before doing anything
      --dry-run               Don't actually run anything; just print
      --fetch-all             Fetch all tasks
  -p, --fetch-policy string   Set fetch policy ("always", "if-not-present", "never") (default "if-not-present")
  -f, --from string           Read location as workflow definition (default "file:tasks.yaml")
      --gc                    Perform garbage collection on the store
  -h, --help                  help for maru2
      --list                  Print list of available tasks and exit
  -l, --log-level string      Set log level (default "info")
  -s, --store string          Set storage directory (default "${HOME}/.maru2/store")
  -t, --timeout duration      Maximum time allowed for execution (default 1h0m0s)
  -V, --version               Print version number and exit
  -w, --with stringToString   Pass key=value pairs to the called task(s) (default [])
```

## Discovering Tasks

The `--list` flag shows you all available tasks in a workflow file.

```sh
$ maru2 --list

Available:

- default
- build
- test
- local-alias:setup
- local-alias:deploy
```

If a `default` task is defined, it's listed first. Otherwise, tasks are displayed in alphabetical order. Tasks from local file aliases are also shown in the format `alias:task-name`.

You can also list tasks from a specific file or remote workflow:

```sh
maru2 --from custom-tasks.yaml --list
maru2 --from "pkg:github/defenseunicorns/maru2@main#examples/web-app.yaml" --list
```

## Passing Inputs to Tasks

Use the `--with` flag to pass input values to tasks:

```sh
maru2 deploy --with environment=production --with version=1.2.3
```

Inside your task definition, access these values using the `${{ input "key" }}` syntax:

```yaml
deploy:
  - run: echo "Deploying version ${{ input "version" }} to ${{ input "environment" }}"
```

### Input Value Formatting

- Basic values: `--with key=value`
- Values with spaces or special characters: `--with key="Hello, World!"`
- Multiple inputs: Use multiple `--with` flags
- Passing outputs from other commands: `--with version=$(git describe --tags)`

Examples:

```sh
# Multiple inputs
$ maru2 notify --with channel=releases --with message="New version deployed"

# Using command output
$ maru2 build --with timestamp=$(date +%s)
```

## Previewing Execution with Dry Run

The `--dry-run` flag lets you preview what commands would execute without actually running them:

```sh
$ maru2 build --dry-run

go build -o bin/app ./cmd/app
```

When you use `--dry-run`, Maru2:

1. Parses and validates the workflow file
2. Resolves all `uses` imports (including remote workflows)
3. Processes all `with` expressions and templates
4. Shows the commands that would run
5. Doesn't actually execute any commands

### Why Use Dry Run?

- **Verify workflows**: Check that your workflow is correctly structured
- **Validate inputs**: Confirm all required inputs are properly provided
- **Inspect remote tasks**: See what remote workflows would do before executing them
- **Debug templates**: Check how variables and expressions would be evaluated
- **Security**: Review commands before execution

### Understanding Template Output in Dry Run

When a template depends on output from previous steps (which aren't actually run in dry run mode), Maru2 shows special formatting:

```sh
$ maru2 template-example --dry-run

echo "The value is ❯ from step-id output-key ❮"
```

This visual indicator helps you identify dynamic parts of your workflow that depend on previous step outputs.

## Task Execution

### The Default Task

When you run `maru2` without specifying a task, it runs the `default` task:

```sh
$ maru2
# Same as
$ maru2 default
```

Creating a `default` task in your workflow provides a convenient entry point for the most common operation.

### Running Specific Tasks

To run a specific task from your workflow:

```sh
maru2 hello-world
```

### Running Multiple Tasks

Like `make`, you can run multiple tasks in sequence:

```sh
maru2 clean build test deploy
```

Tasks are executed in the order specified on the command line, which is useful for creating simple pipelines.

### Running Aliased Tasks

If your workflow defines local file aliases, you can run tasks from those aliased workflows directly:

```sh
# Run a task from a local alias
maru2 common:setup

# Run multiple aliased tasks
maru2 common:setup utils:compile common:deploy
```

The `alias:task` format allows you to reference tasks from aliased workflow files without needing to specify the full file path.

## Working with Workflow Files

### Local Workflow Files

By default, Maru2 looks for a file named `tasks.yaml` in the current directory. To use a different file:

```sh
maru2 --from path/to/other.yaml
maru2 -f custom-workflow.yaml build
```

### Remote Workflow Files

Maru2 can execute tasks directly from remote repositories:

```sh
# Run a task from a GitHub repository
maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo

# With custom input
maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo --with message="Hello from remote!"
```

> **Note**: When referencing remote workflows, you must use quotes since the package-URL spec uses special shell characters like `#` and `@`.

## Managing Remote Workflows

### Fetch Policy

Control how Maru2 retrieves remote workflows with the `--fetch-policy` flag:

```sh
maru2 --fetch-policy always my-task
```

Available policies:

| Policy           | Description                                   |
| ---------------- | --------------------------------------------- |
| `always`         | Always fetch remote workflows, even if cached |
| `if-not-present` | Only fetch if not in cache (default)          |
| `never`          | Never fetch, only use cached workflows        |

### Refreshing Remote Workflows

To update all remote references without executing any tasks:

```sh
maru2 --dry-run --log-level error --fetch-policy always
```

This combination refreshes your cache without running any code.

### Prefetching All Dependencies

Use `--fetch-all` to download all remote dependencies (even ones not in the hot path) before execution:

```sh
maru2 --fetch-all deploy
```

This ensures all dependencies are available, which is useful before going offline or in environments with unreliable connectivity.

## Setting Up Shell Completions

Maru2 supports command completion for various shells, making it easier to discover and use available tasks and options.

### Installation Commands

Choose the command for your shell:

**Bash**:

```bash
maru2 completion bash > ~/.maru2/maru2_completion.bash
echo 'source ~/.maru2/maru2_completion.bash' >> ~/.bashrc
```

**Zsh**:

```zsh
maru2 completion zsh > ~/.maru2/maru2_completion.zsh
echo 'source ~/.maru2/maru2_completion.zsh' >> ~/.zshrc
```

**Fish**:

```fish
maru2 completion fish > ~/.config/fish/completions/maru2.fish
```

**PowerShell**:

```powershell
maru2 completion powershell > $PROFILE.CurrentUserAllHosts
```

### Completion with Remote Workflows

**Fish shell note**: When using tab completion with remote workflows in fish shell, use both sets of quotes:

```sh
# For tab completion in fish shell
maru2 --from "'pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml'" [tab][tab]

# Alternative: use --list to discover available tasks
maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" --list
```

### Completion with Aliased Tasks

Tab completion also works with aliased tasks. If your workflow defines aliases, you'll see them in completion:

```sh
# Tab completion shows both regular and aliased tasks
maru2 [tab][tab]
# Shows: default build test common:setup common:deploy utils:compile

# Complete specific alias
maru2 common:[tab][tab]
# Shows: common:setup common:deploy
```

## Additional Options

### Execution Timeout

Control how long Maru2 will run before timing out:

```sh
maru2 long-task --timeout 2h30m
```

The default timeout is 1 hour. Use standard Go duration format for specifying timeouts.

### Log Verbosity

Adjust the amount of information displayed during execution:

```sh
maru2 build --log-level debug
```

Available log levels:

| Level   | Description                                        |
| ------- | -------------------------------------------------- |
| `error` | Only show errors                                   |
| `warn`  | Show errors and warnings                           |
| `info`  | Show errors, warnings, and info messages (default) |
| `debug` | Show all messages, including debugging information |

### Working Directory

Change to a specific directory before executing any tasks:

```sh
maru2 --directory /path/to/project build
```

This is equivalent to `cd /path/to/project && maru2 build; cd -`.

### Managing the Cache Store

#### Custom Store Location

Set a custom location for cached workflows:

```sh
maru2 --store /path/to/custom/store build
```

By default, Maru2 uses:

- `${HOME}/.maru2/store` (global cache)
- `./.maru2/store` (if it exists in the current directory)

#### Cleaning the Cache

Remove unused workflows from the cache:

```sh
maru2 --gc
```

This frees up disk space by removing cached workflows that are no longer referenced.

## Error handling and traceback

When a step in a Maru2 workflow fails, the error is propagated up the call stack with a traceback that shows the path of execution. This helps you identify where in your workflow the error occurred, especially for complex workflows with nested task calls.

```yaml
tasks:
  fail:
    steps:
      - run: exit 1

  caller:
    steps:
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
