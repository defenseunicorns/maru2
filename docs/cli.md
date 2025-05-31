# CLI Maru2

## Usage

<!-- TODO: automate this once a docs site is created -->

```text
A simple task runner

Usage:
  maru2 [flags]

Examples:

maru2 build

maru2 -f ../foo.yaml bar baz -w zab="zaz"

maru2 -f "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo -w message="hello world"


Flags:
  -C, --directory string      Change to directory before doing anything
      --dry-run               Don't actually run anything; just print
  -p, --fetch-policy string   Set fetch policy ("always", "if-not-present", "never") (default "if-not-present")
  -f, --from string           Read location as workflow definition (default "file:tasks.yaml")
  -h, --help                  help for maru2
      --list                  Print list of available tasks and exit
  -l, --log-level string      Set log level (default "info")
  -s, --store string          Set storage directory (default "${HOME}/.maru2/store")
  -t, --timeout duration      Maximum time allowed for execution (default 1h0m0s)
  -V, --version               Print version number and exit
  -w, --with stringToString   Pass key=value pairs to the called task(s) (default [])
```

## Discover available tasks

The `--list` flag can be used to list all the tasks in a Maru2 workflow.

If defined, the `default` task will be listed first. Otherwise, tasks will be listed in alphabetical order.

```sh
$ maru2 --list

Available:

- default
- build
- test
```

## Passing inputs with `--with`

The `--with` flag allows you to pass key-value pairs to tasks. This is particularly useful for tasks that define input parameters.

```sh
$ maru2 deploy --with environment=production --with version=1.2.3
```

These values can then be accessed within the task using the `${{ input "key" }}` syntax:

```yaml
deploy:
  - run: echo "Deploying version ${{ input "version" }} to ${{ input "environment" }}"
```

The `--with` flag accepts values in the format `key=value`. If your value contains spaces or special characters, you should quote the value:

```sh
$ maru2 greet --with message="Hello, World!"
```

You can pass multiple `--with` flags, and they will be combined into a single set of inputs for the task.

## Dry run

The `--dry-run` flag allows you to preview what a workflow would do without actually executing any commands. When this flag is set, Maru2 will:

1. Parse and validate the workflow file
2. Resolve and evaluate all `uses` imports (including remote workflows)
3. Process and template all `with` expressions
4. Print the commands that would be executed
5. Skip the actual execution of any `run` commands

```sh
$ maru2 build --dry-run

$ go build -o bin/app ./cmd/app
```

### Use cases

- **Workflow validation**: Verify that your workflow is correctly structured without running any commands
- **Input validation**: Ensure that all required inputs are provided and correctly formatted
- **Remote workflow inspection**: View the contents of remote workflows before executing them
- **Debugging**: Understand how variables and expressions will be evaluated in your workflow
- **Security review**: Inspect what commands would be executed before running them

### Templating in dry run mode

In dry run mode, template expressions that cannot be evaluated (such as outputs from previous steps that weren't actually run) will be displayed with special formatting:

```sh
$ maru2 template-example --dry-run

$ echo "The value is ❯ from step-id output-key ❮"
```

This allows you to see what parts of your workflow depend on outputs from previous steps.

## "default" task

The task named `default` in a Maru2 workflow is the task that will be run when no task is specified.

```sh
$ maru2
# is equivalent to
$ maru2 default
# but this will only run the 'hello-world' task
$ maru2 hello-world
```

## Run multiple tasks

Like `make`, you can run multiple tasks in a single command.

```sh
$ maru2 task1 task2
```

## Specify a local workflow file

By default, Maru2 will look for a file named `tasks.yaml` in the current directory. You can specify a different location to use with the `--from` or `-f` flag.

```sh
$ maru2 --from path/to/other.yaml
```

## Specify a remote workflow file

Any [`uses` syntax](./syntax.md#run-a-task-from-a-remote-file) is also acceptable as a workflow location.

```sh
# NOTE: referencing remote workflows requires quoting, since the package-url spec leverages reserved shell characters (like # and @)!!!
$ maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo
```

## Fetch policy

The `--fetch-policy` or `-p` flag controls how Maru2 fetches remote workflows:

```sh
$ maru2 --fetch-policy always
```

Available fetch policies:

- `always`: Always fetch remote workflows, even if they exist in the local cache
- `if-not-present`: Only fetch remote workflows if they don't exist in the local cache (default)
- `never`: Never fetch remote workflows, only use the local cache

If you want to re-fetch all references without running any local code, you can use:

```sh
$ maru2 --dry-run --log-level error --fetch-policy always
```

## Shell completions

Like `make`, `maru2` only has a single command. As such, shell completions are not generated in the normal way most Cobra CLI applications are (i.e. `maru2 completion bash`). Instead, you can use the following snippet to generate completions for your shell:

```bash
MARU2_COMPLETION=true maru2 completion bash
```

```zsh
MARU2_COMPLETION=true maru2 completion zsh
```

```fish
MARU2_COMPLETION=true maru2 completion fish
```

```powershell
$env:MARU2_COMPLETION='true'; maru2 completion powershell; $env:MARU2_COMPLETION=$null
```

Completions are only generated when the `MARU2_COMPLETION` environment variable is set to `true`, and the `completion <shell>` arguments are passed to the `maru2` command.

This is because `completion bash|fish|etc...` are valid task names in a Maru2 workflow, so the CLI would attempt to run these tasks. By setting the environment variable, the CLI knows to generate completions instead of running tasks.

> If using `fish` and attempting to perform tab completions w/ a remote workflow, surround your query in both sets of quotes. This is due to the way that Cobra's completion script is generated, the first set of quotes is stripped, and the underlying string will cause a completion error.
>
> ```sh
> $ maru2 --from "'pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml'" [tab][tab]
> # or just use --list to discover tasks
> $ maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" --list
> ```

## Timeout

Maru2 has a default 1hr timeout. To specify a different timeout, see the `-t, --timeout` flag.

## Log level

Maru2 allows you to control the verbosity of its output using the `-l, --log-level` flag. The default log level is `info`.

```sh
$ maru2 build --log-level debug
```

Available log levels (from least to most verbose):

- `error`: Only show errors
- `warn`: Show errors and warnings
- `info`: Show errors, warnings, and informational messages (default)
- `debug`: Show all messages, including debug information

## Store directory

The `--store` or `-s` flag allows you to specify a custom directory for storing cached remote workflows and other Maru2 data:

```sh
$ maru2 --store /path/to/custom/store
```

By default, Maru2 uses `${HOME}/.maru2/store` as the storage directory.
