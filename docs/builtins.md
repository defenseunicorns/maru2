# Built-in Tasks

Maru2 provides several built-in tasks that can be used in your workflows. These are referenced using the `builtin:` prefix in the `uses` field.

## Echo

The `echo` built-in task simply outputs the provided text.

```yaml {filename="tasks.yaml"}
echo-example:
  - uses: builtin:echo
    with:
      text: "Hello, World!"
```

```sh
maru2 echo-example

Hello, World!
```

Outputs:
- `stdout`: The text that was echoed

## Fetch

The `fetch` built-in task makes HTTP requests and returns the response.

```yaml {filename="tasks.yaml"}
fetch-example:
  - uses: builtin:fetch
    with:
      url: "https://api.example.com/data"
      method: "GET"  # Optional, defaults to GET
      timeout: "30s"  # Optional, defaults to 30 seconds
      headers:  # Optional
        Content-Type: application/json
        Authorization: Bearer ${{ input "token" }}
```

Outputs:
- `body`: The response body as a string

The `fetch` built-in is useful for integrating with external APIs or services from your workflow.
