# Workflow Syntax

A Maru2 workflow is any YAML file that conforms to the [`maru2` schema](../schema-validation#raw-schema).

Unless specified, the default file name is `tasks.yaml`.

## Structure

Similar to `Makefile`s, a Maru2 workflow is a map of tasks, where each task is a series of steps.

Checkout the comparison below:

{{< tabs items="Makefile,Maru2" >}}

  {{< tab >}}

```makefile {filename="Makefile"}
.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o bin/ -ldflags="-s -w" ./cmd/maru2

test:
	go test -v -race -cover -failfast -timeout 3m ./...

clean:
	rm -rf bin/
```

  {{< /tab >}}
  {{< tab >}}

```yaml {filename="tasks.yaml"}
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

  {{< /tab >}}

{{< /tabs >}}

## Task names

Task names must follow the following regex: `^[_a-zA-Z][a-zA-Z0-9_-]*$`. Try it out below:

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
</script>

### Examples of valid task names

```yaml
build: ...
another-task: ...
UPPERCASE: ...
mIxEdCaSe: ...
WithNumbers123: ...
```

## `run` vs `uses`

- `run`: runs a shell command/script
- `uses`: calls another task / executes a builtin

Both can be used interchangeably within a task, and interoperate cleanly with `with`.

## Passing inputs

`with` is a map of [Tengo](https://github.com/d5/tengo) expressions.

On top of the builtin behavior, Maru2 provides a few additional helpers:

- `input`: the value passed to the task at that key
  - If the task is top-level (called via CLI), `with` values are received from the `--with` flag.
  - If the task is called from another task, `with` values are passed from the calling step.
- `os`, `arch`, `platform`: the current OS, architecture, or platform

{{< tabs items="run,eval" >}}
{{< tab >}}

`with` is then mapped to the steps's environment variables, with key names being transformed to standard environment variable names (uppercase, with underscores).

```yaml {filename="tasks.yaml"}
echo:
  - run: echo "Hello, $NAME, today is $DATE"
    with:
      name: input
      # default to "now" if input is nil
      date: input || "now"
  - run: echo "The current OS is $OS, architecture is $ARCH, platform is $PLATFORM"
    with:
      os: os
      arch: arch
      platform: platform
```

```sh
maru2 echo --with name=$(whoami) --with date=$(date)
```

{{< /tab >}}
{{< tab >}}

`with` values are passed to the Tengo script as global variables at compilation time.

```yaml {filename="tasks.yaml"}
echo:
  - eval: |
      fmt := import("fmt")
      if date == "now" {
        times := import("times")
        date = times.time_format(times.now(), "2006-01-02")
      }
      s := fmt.sprintf("Hello, %s, today is %s", name, date)
      fmt.println(s)
    with:
      name: input
      # default to "now" if input is nil
      date: input || "now"
```

```sh
maru2 echo --with name=$(whoami)
```

{{< /tab >}}
{{< /tabs >}}

## Run another task as a step

Calling another task within the same workflow is as simple as using the task name, similar to Makefile targets.

```yaml {filename="tasks.yaml"}
general-kenobi:
  - run: echo "General Kenobi, you are a bold one"
  - run: echo "$RESPONSE"
    with:
      response: input

hello:
  - run: echo "Hello There!"
  - uses: general-kenobi
    with:
      response: '"Your move"'
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
simple:
  - run: echo "$MESSAGE"
    with:
      message: input
```

```yaml {filename="tasks.yaml"}
echo:
  - uses: file:tasks/echo.yaml?task=simple
    with:
      message: input
```

```sh
maru2 echo --with message="Hello, World!"
```

## Run a task from a remote file

{{< callout emoji="⚠️" >}}
`uses` syntax leverages the [package-url spec](https://github.com/package-url/purl-spec)
{{< /callout >}}

{{< tabs items="GitHub,GitLab,HTTP(S)" >}}

{{< tab >}}

```yaml {filename="tasks.yaml"}
remote-echo:
  - uses: pkg:github/noxsios/maru2@main?task=echo#testdata/simple.yaml
    with:
      message: '"Hello, World!"'
```

{{< /tab >}}

{{< tab >}}

```yaml {filename="tasks.yaml"}
remote-echo:
  - uses: pkg:gitlab/noxsios/maru2@main?task=echo#testdata/simple.yaml
    with:
      message: '"Hello, World!"'
```

{{< /tab >}}

{{< tab >}}

```yaml {filename="tasks.yaml"}
remote-echo:
  - uses: https://raw.githubusercontent.com/noxsios/maru2/main/testdata/simple.yaml?task=echo
    with:
      message: '"Hello, World!"'
```

{{< /tab >}}

{{< /tabs >}}

```sh
maru2 remote-echo
```

## Passing outputs

This leverages the same mechanism as GitHub Actions.

The `id` field is used to reference the output in subsequent steps.

`steps` is a map of step IDs to their outputs. Values can be accessed through either bracket or dot notation.

{{< tabs items="run,eval" >}}
{{< tab >}}

```yaml {filename="tasks.yaml"}
color:
  - run: |
      echo "selected-color=green" >> $VAI_OUTPUT
    id: color-selector
  - run: echo "The selected color is $SELECTED"
    with:
      selected: steps["color-selector"]["selected-color"]
```

{{< /tab >}}
{{< tab >}}

```yaml {filename="tasks.yaml"}
color:
  - eval: |
      color := "green"
      maru2_output["selected-color"] = color
    id: color-selector
  - run: echo "The selected color is $SELECTED"
    with:
      selected: steps["color-selector"]["selected-color"]
```

{{< /tab >}}
{{< /tabs >}}

```sh
maru2 color
```
