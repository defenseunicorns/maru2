# vision

Ideal workflow state: 90% is leveraging `builtins` / `uds-common` tasks.

Without trading speed, as much behavior is validated before any code is executed to reduce side effects.

Some simple examples (used in E2E tests) can be found in [testdata](../testdata)

A basic implementation of a few builtins + unit tests: [builtins/basic.go](../builtins/basic.go)

## Simple shell scripts w/ input type validation

```yaml
name:
  description: "Name to greet"
  # <-- if no default is given, the CLI will error out if the input is accessed but not provided

hour:
  default: 5 # <-- note this type here, inputs will attempt to cast to this type if they mismatch

greet:
  - run: echo "Hello, ${{ input "name" }}, it is ${{ input "hour" }} o'clock"
```

```bash
$ maru2 greet

ERRO error calling input: input "name" does not exist in [date hour]
     traceback (most recent call first)=
     |   at greet[0] (file:test.yaml)
     #   ^-- rich error stack to see the exact location the error occurred
     #   ^-- <task>[<step index>] (<location>)

$ maru2 greet --dry-run

WARN input "name" was not provided, available: [date hour]
echo "Hello, ❯ input name ❮, it is 5 o'clock"
             # ^-- this is colorized in amber for the CLI to make it pop
```

```bash
$ maru2 greet -w name=Jeff

Hello, Jeff, it is 5 o'clock
```

```bash
$ maru2 greet -w name=Jeff -w hour=wrong

ERRO unable to cast "wrong" of type string to uint64
```

## Calling remote tasks

```yaml
setup:
  - uses: pkg:github/defenseunicorns/uds-common@v1.0.0#tasks/setup.yaml?task=k3d-full-cluster
    with:
      insecure-keycloak-admin: true
  - uses: file:../wait.yaml
    with:
      cluster-up: true
  - uses: builtin:get-keycloak-credentials
    if: ${{ env "CI" }} == false
    id: creds # <-- id is required for `from`
  - run: |
      echo "username: ${{ from "creds" "user" }}"
      echo "password: ${{ from "creds" "password"}}"
```

## Zarf Package create + publish

```yaml
create-pkg:
  - uses: builtin:zarf-package-create
    with:
      path: ${{ inputs "path" }}
      arch: ${{ inputs "arch" }}
      flavors:  ${{ inputs "flavors" }}
      log-level: warning

  - if: failure
    # ^-- runs if any previous step failed, does not check return error
    run: |
      echo "Unable to create zarf package at ${{ inputs "path" }}"
      rm ${{ inputs "path" }}/*.tar.zst

  - if: ${{ inputs "publish-to" | len | isNotEmpty }}
    # ^-- simple if conditional based upon inputs
    uses: builtin:zarf-package-publish
    with:
      path: ${{ inputs "path" }}
      arch: ${{ inputs "arch" }}
      flavors:  ${{ inputs "flavors" }}
      to: ${{ inputs "publish-to" }}
```

```bash
maru2 create-pkg -w path=. -w arch=all -w flavors=all -w publish-to=ghcr.io/defenseunicorns/...
```

## Passing ouputs around

```yaml
pass:
  - run: echo "foo=${{ input "foo" }}" >> $MARU2_OUTPUT

bar:
  - uses: pass
    id: 0
    with:
      foo: bar
  - run: echo ${{ from "0" "foo" }}
```
