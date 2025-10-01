# Built-in tasks

Maru2 provides several built-in tasks that you can use in your workflows.
Reference these using the `builtin:` prefix in the `uses` field.

## Echo

The `echo` built-in task simply outputs the provided text.

```yaml
schema-version: v1
tasks:
  echo-example:
    steps:
      - uses: builtin:echo
        with:
          text: "Hello, World!"
```

```sh
maru2 echo-example

Hello, World!
```

Outputs:

- `stdout`: The echoed text

## Fetch

The `fetch` built-in task makes HTTP requests and returns the response.

```yaml
schema-version: v1
tasks:
  fetch-example:
    inputs:
      token:
        description: "API token"
        required: true
    steps:
      - uses: builtin:fetch
        with:
          url: "https://api.example.com/data"
          method: "GET" # Optional, defaults to GET
          timeout: "30s" # Optional, defaults to 30 seconds
          headers: # Optional
            Content-Type: application/json
            Accept: application/json
            Authorization: Bearer ${{ input "token" }}
```

Outputs:

- `body`: The response body as a string

The `fetch` built-in is useful for integrating with external APIs or services from your workflow.
