# CLI Maru2

## Usage

```bash
{{< usage >}}
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

## Dry run

When the `--dry-run` flag is set, Maru2 will evaluate `uses` imports and `with` expressions but will _not_
execute any code in `run`.

This allows for debugging, as well as viewing the contents of remote workflows without executing them.

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

## Specify a workflow file

By default, Maru2 will look for a file named `tasks.yaml` in the current directory. You can specify a different file to use with the `--file` or `-f` flag.

```sh
$ maru2 --file path/to/other.yaml
```

## Shell completions

Like `make`, `maru2` only has a single command. As such, shell completions are not generated in the normal way most Cobra CLI applications are (i.e. `maru2 completion bash`). Instead, you can use the following snippet to generate completions for your shell:

{{< tabs items="bash,zsh,fish,powershell" >}}

{{< tab "bash" >}}

```bash
MARU2_COMPLETION=true maru2 completion bash
```

{{< /tab >}}

{{< tab "zsh" >}}

```zsh
MARU2_COMPLETION=true maru2 completion zsh
```

{{< /tab >}}

{{< tab "fish" >}}

```fish
MARU2_COMPLETION=true maru2 completion fish
```

{{< /tab >}}

{{< tab "powershell" >}}

```powershell
$env:MARU2_COMPLETION='true'; maru2 completion powershell; $env:MARU2_COMPLETION=$null
```

{{< /tab >}}

{{< /tabs >}}

Completions are only generated when the `MARU2_COMPLETION` environment variable is set to `true`, and the `completion <shell>` arguments are passed to the `maru2` command.

This is because `completion bash|fish|etc...` are valid task names in a Maru2 workflow, so the CLI would attempt to run these tasks. By setting the environment variable, the CLI knows to generate completions instead of running tasks.

## Timeout

Maru2 has a default 1hr timeout. To specify a different timeout, see the `-t, --timeout` flag.
