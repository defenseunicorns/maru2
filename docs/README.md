# Maru2 Documentation

Maru2 is a simple task runner designed to make workflow automation easy and intuitive.

## Getting Started

- [CLI Documentation](cli.md) - Learn how to use the Maru2 command line interface
- [Workflow Syntax](syntax.md) - Understand the syntax for defining tasks and workflows
- [Built-in Tasks](builtins.md) - Explore the built-in tasks provided by Maru2

## Core Concepts

Maru2 is built around a few simple concepts:

1. **Tasks** - The basic unit of work in Maru2, defined as a series of steps
2. **Steps** - Individual actions within a task, which can be shell commands or references to other tasks
3. **Inputs** - Parameters that can be passed to tasks
4. **Outputs** - Values that can be passed between steps

## Example Workflow

```yaml
# Define inputs
message:
  description: "Message to display"
  default: "Hello, World!"

# Define tasks
default:
  - uses: greet

greet:
  - run: echo "${{ input "message" }}"
```

Run it with:

```sh
maru2
# or with a custom message
maru2 --with message="Hello, Maru2!"
```

## Advanced Features

- Conditional execution with `if` directives
- Error handling and traceback
- Environment variable integration
- Remote task execution
- Input validation

For more details on these features, see the [Workflow Syntax](syntax.md) documentation.
