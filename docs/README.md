# Maru2 Documentation

Maru2 is a simple, powerful task runner designed to make workflow automation easy and intuitive. Inspired by the simplicity of Makefiles but with modern features like GitHub Actions, Maru2 helps you define, organize, and execute tasks with minimal configuration.

## Quick Start

> [!NOTE] Use `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables to pull task files from remote GitHub and GitLab destinations using the [package-url spec](https://github.com/package-url/purl-spec).
>
> Example: `GITHUB_TOKEN=ghxxxxxxxxxx
> maru2 -f "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo -w message="hello world"`


1. **Create a simple workflow file**

   Create a file named `tasks.yaml` in your project root:

   ```yaml
   schema-version: v0
   tasks:
     default:
       - run: echo "Hello, Maru2!"
   ```

2. **Run your first task**

   ```sh
   maru2
   ```

   That's it! You've just run your first Maru2 task.

## Documentation Navigation

- **[Core Concepts](#core-concepts)** - Understand the fundamental concepts of Maru2
- **[Example Workflow](#example-workflow)** - See a complete example with explanations
- **[CLI Documentation](cli.md)** - Learn how to use the Maru2 command line interface
- **[Workflow Syntax](syntax.md)** - Understand the syntax for defining tasks and workflows
- **[Publishing Workflows](publish.md)** - Learn how to publish workflows as OCI artifacts
- **[Built-in Tasks](builtins.md)** - Explore the built-in tasks provided by Maru2
- **[Configuration](config.md)** - Configure Maru2 with global settings

## Core Concepts

Maru2 is built around these simple concepts:

1. **Tasks** - The basic unit of work in Maru2, defined as a series of steps

   ```yaml
   schema-version: v0
   tasks:
     build:
       - run: go build -o bin/ ./cmd/app
   ```

2. **Steps** - Individual actions within a task, which can be:
   - Shell commands (`run`)
   - References to other tasks (`uses`)

   ```yaml
   schema-version: v0
   tasks:
     deploy:
       - run: echo "Deploying application..." # Shell command step
       - uses: notify # Reference to another task
   ```

3. **Inputs** - Parameters that can be passed to tasks

   ```yaml
   schema-version: v0
   inputs:
     environment:
       description: "Deployment environment"
       default: "staging"

   tasks:
     deploy:
       - run: echo "Deploying to ${{ input "environment" }}"
   ```

4. **Outputs** - Values that can be passed between steps

   ```yaml
   schema-version: v0
   tasks:
     get-version:
       - run: |
           echo "version=1.0.0" >> $MARU2_OUTPUT
         id: version-step
       - run: echo "Version is ${{ from "version-step" "version" }}"
   ```

## Example Workflow

This example demonstrates inputs, task references, and output passing:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json
schema-version: v0
inputs:
  message:
    description: "Message to display"
    default: "Hello, World!"
    required: true

tasks:
  # This is the default task that runs when you type just 'maru2'
  default:
    - uses: greet
      with:
        message: |
          ${{ input "message" }}

  # A reusable greeting task
  greet:
    - run: echo "stdout=${{ input "message" }}" >> $MARU2_OUTPUT
      id: greeter
    - run: |
        echo "The message was: ${{ from "greeter" "stdout" }}"
```

Run it with:

```sh
# Run the default task with the default message
maru2

# Run with a custom message
maru2 --with message="Hello, Maru2!"

# Run a specific task
maru2 greet --with message="Specific greeting"
```

## Advanced Features

Maru2 includes powerful features for complex workflows:

- **Conditional execution** - Control step execution with `if` directives
- **Error handling and traceback** - Get detailed information about errors
- **Environment variable integration** - Use environment variables as input defaults
- **Remote task execution** - Execute tasks from remote repositories
- **Input validation** - Validate inputs using regular expressions
- **Package URL aliases** - Create shortcuts for common repositories

## Next Steps

Ready to dive deeper? Continue with:

- [CLI Documentation](cli.md) to learn all the command line options
- [Workflow Syntax](syntax.md) for detailed information on defining tasks
- [Built-in Tasks](builtins.md) to discover pre-defined tasks you can use
