# Maru2 Documentation

Maru2 is a simple task runner designed to make workflow automation easy and intuitive. Inspired by the simplicity of Makefiles but with modern features, Maru2 helps you define, organize, and execute tasks with minimal configuration.

## Documentation Navigation

- **[Getting Started](#getting-started)** - Start here for an introduction to Maru2
- **[CLI Documentation](cli.md)** - Learn how to use the Maru2 command line interface
- **[Workflow Syntax](syntax.md)** - Understand the syntax for defining tasks and workflows
- **[Built-in Tasks](builtins.md)** - Explore the built-in tasks provided by Maru2
- **[Configuration](config.md)** - Configure Maru2 with global settings

## Getting Started

1. **New to Maru2?** Start with the [Example Workflow](#example-workflow) below to see Maru2 in action.
2. **Installation:** (Documentation coming soon)
3. **Next Steps:** After exploring the example, continue to the [CLI Documentation](cli.md) to learn more.

## Core Concepts

Maru2 is built around a few simple concepts:

1. **Tasks** - The basic unit of work in Maru2, defined as a series of steps
2. **Steps** - Individual actions within a task, which can be shell commands or references to other tasks
3. **Inputs** - Parameters that can be passed to tasks
4. **Outputs** - Values that can be passed between steps

## Example Workflow

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json
inputs:
  message:
    description: "Message to display"
    default: "Hello, World!"
    required: true

tasks:
  default:
    - uses: greet
      with:
        message: "${{ input "message" }}"

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

Maru2 includes powerful features for complex workflows:

- **Conditional execution** - Control step execution with `if` directives
- **Error handling and traceback** - Get detailed information about errors
- **Environment variable integration** - Use environment variables as input defaults
- **Remote task execution** - Execute tasks from remote repositories
- **Input validation** - Validate inputs using regular expressions
- **Package URL aliases** - Create shortcuts for common repositories

For more details on these features, see the [Workflow Syntax](syntax.md) documentation and the [Configuration](config.md) guide.
