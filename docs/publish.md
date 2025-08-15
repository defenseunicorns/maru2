# Publishing Workflows

Maru2 provides a command to publish your workflows as OCI artifacts to UDS Registry (or any OCI compliant registry for that matter).

> [!WARNING]
> The `maru2-publish` command is currently in **ALPHA**. Expect frequent breaking changes.

<!--
TODO: once out of ALPHA, this doc MAY be merged into ./syntax.md.

At the very minimum, ./syntax.md MUST be updated to showcase the `oci:` uses syntax and query parameters.
-->

## The `maru2-publish` Command

The `maru2-publish` command packs your workflow file(s) and any `uses:` references into an OCI artifact and pushes it to a registry.

### Installation

```sh
go install github.com/defenseunicorns/maru2/cmd/maru2-publish@main
```

### Usage

```sh
maru2-publish <oci-image-reference> --entrypoint ... --entrypoint ...
```

- `--entrypoint`: Relative path(s) to local workflow entrypoints (e.g., `tasks.yaml`).
- `<oci-image-reference>`: The OCI image reference to publish to (e.g., `staging.uds.sh/public/my-workflow:latest`).

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
schema-version: v0
tasks:
  default:
    - uses: file:tasks/helper.yaml?task=hello
```

`tasks/helper.yaml`:

```yaml
schema-version: v0
tasks:
  hello:
    - run: echo "Hello from helper!"
```

To publish this workflow, you would run:

```sh
zarf tools registry login staging.uds.sh/public/my-workflow:latest ...
maru2-publish staging.uds.sh/public/my-workflow:latest -e tasks.yaml
```

### Using Published Workflows

Once published, you can use the workflow in another project with the `oci:` scheme:

```yaml
schema-version: v0
tasks:
  run-published:
    - uses: oci:staging.uds.sh/public/my-workflow:latest
```

By default, this looks for the `file:tasks.yaml` entry in the published manifest and runs the `default` task.

To specify another path use the URL hash:

```yaml
uses: oci:staging.uds.sh/public/my-workflow#file:tasks/helper.yaml
```

The following query parameters are supported:

- `plain-http`: pull via plain HTTP (default: `false`)
- `insecure-skip-tls-verify`: skip TLS checking (default: `false`)
- `task`: specify the task to run (default: `default`)
