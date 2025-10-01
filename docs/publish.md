# Publishing workflows

Maru2 provides a command to publish your workflows as Open Container Initiative (OCI) artifacts to [UDS Registry](https://registry.defenseunicorns.com/) or any OCI-compliant registry.

> [!WARNING]
> The `maru2-publish` command is currently in **alpha** status. Expect frequent breaking changes.

<!--
TODO: once out of ALPHA, this doc MAY be merged into ./syntax.md.

At the very minimum, ./syntax.md MUST be updated to showcase the `oci:` uses syntax and query parameters.
-->

## The `maru2-publish` command

The `maru2-publish` command is a standalone utility for publishing workflows. It packs your workflow files and any local `uses:` references into an OCI artifact and pushes it to a registry.

### Installation

Installation of `maru2-publish` is automatically handled when installing `maru2` via brew, curl or wget.

via go install:

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2-publish@main
```

### Usage

```sh
maru2-publish <oci-image-reference> --entrypoint ... --entrypoint ...
```

- `--entrypoint`: Relative paths to local workflow entrypoints (for example, `tasks.yaml`).
- `<oci-image-reference>`: The OCI image reference to publish to (for example, `staging.uds.sh/public/my-workflow:latest`).

### Example

Consider the following project structure:

```plaintext
.
├── tasks.yaml
└── tasks/
    └── helper.yaml
```

`tasks.yaml`:

```yaml
schema-version: v1
tasks:
  default:
    steps:
      - uses: file:tasks/helper.yaml?task=hello
```

`tasks/helper.yaml`:

```yaml
schema-version: v1
tasks:
  hello:
    steps:
      - run: echo "Hello from helper!"
```

To publish this workflow, you would run:

```sh
# login to the registry w/ your preferred client
zarf tools registry login staging.uds.sh/public/my-workflow:latest ...

# publish
maru2-publish staging.uds.sh/public/my-workflow:latest -e tasks.yaml
```

### Using published workflows

Once published, you can use the workflow in another project with the `oci` scheme:

```yaml
schema-version: v1
tasks:
  run-published:
    steps:
      - uses: oci:staging.uds.sh/public/my-workflow:latest
```

By default, this looks for the `file:tasks.yaml` entry in the published manifest and runs the `default` task.

To specify another path use the URL hash:

```yaml
uses: oci:staging.uds.sh/public/my-workflow#file:tasks/helper.yaml
```

Supported query parameters:

- `plain-http`: pull via plain HTTP (default: `false`)
- `insecure-skip-tls-verify`: skip Transport Layer Security (TLS) checking (default: `false`)
- `task`: specify the task to run (default: `default`)
