# builtins

What is a Maru2 `builtin:`? Anything that satisfies the `Builtin` interface:

```go
// registration.go

// Builtin defines the interface for built-in tasks (builtin:echo, builtin:fetch)
//
// Implementations must be structs to support configuration binding via mapstructure.
// The Execute method receives context and returns outputs that can be accessed by subsequent steps
type Builtin interface {
	Execute(ctx context.Context) (map[string]any, error)
}
```

See [basic.go](basic.go) for some good examples of how a builtin is structured.

[wacky_structs.go](wacky_structs.go) can be removed once there are more complex usages of the Builtin system, it only exists for test coverage of schema generation.

## Schema generation

Anything registered to the `_registrations` variable will be accessed to generate the JSON schema for that builtin.

For registrations native to Maru2, add them to the `_registrations` variable.

For third party extensions, use the `Register` function to register your builtin, then call `maru2.WorkflowSchema(version string)` to generate and export your new schema with your registered builtin.
