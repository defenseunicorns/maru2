# Maru2 documentation

Maru2 is a simple, powerful task runner designed to make workflow automation easy and intuitive. Inspired by the simplicity of Makefiles but with modern features like GitHub Actions, Maru2 helps you define, organize, and execute tasks with minimal configuration.

## Quick start

> [!NOTE]
> Use `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables to pull task files from remote GitHub and GitLab destinations using the [package URL spec](https://github.com/package-url/purl-spec).
>
> Example:
>
> ```sh
> export GITHUB_TOKEN=ghxxxxxxxxxx
> maru2 -f "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo -w message="hello world"
> ```

1. **Create a simple workflow file**

   Create a file named `tasks.yaml` in your project root:

   ```yaml
   schema-version: v1
   tasks:
     default:
       steps:
         - run: echo "Hello, Maru2!"
   ```

2. **Run your first task**

   ```sh
   maru2
   ```

   That's it! You've just run your first Maru2 task.

## Documentation navigation

- **[Core Concepts](#core-concepts)**: Understand the fundamental concepts of Maru2.
- **[Example Workflow](#example-workflow)**: See a complete example with explanations.
- **[Workflow Syntax](syntax.md)**: Learn the syntax for defining tasks and workflows.
- **[CLI Documentation](cli.md)**: Master the Maru2 command line interface.
- **[Built-in Tasks](builtins.md)**: Explore the built-in tasks provided by Maru2.
- **[Publishing Workflows](publish.md)**: Learn how to publish workflows as Open Container Initiative (OCI) artifacts.
- **[Configuration](config.md)**: Configure Maru2 with global settings.
- **[Migrating from maru-runner](maru-runner-migration.md)**: Follow the guide for migrating from `maru-runner` to `maru2`.

## Core concepts

Maru2 builds around these simple concepts:

1. **Tasks** - The basic unit of work in Maru2, defined as a series of steps

   ```yaml
   schema-version: v1
   tasks:
     build:
       steps:
         - run: go build -o bin/ ./cmd/app
   ```

2. **Steps** - Individual actions within a task, which can be:
   - Shell commands (`run`)
   - References to other tasks (`uses`)

   ```yaml
   schema-version: v1
   tasks:
     deploy:
       steps:
         - run: echo "Deploying application..." # Shell command step
         - uses: notify # Reference to another task
   ```

3. **Inputs** - Parameters that can be passed to tasks

   ```yaml
   schema-version: v1
   tasks:
     deploy:
       inputs:
         environment:
           description: "Deployment environment"
           default: "staging"
       steps:
         - run: echo "Deploying to ${{ input "environment" }}"
   ```

4. **Outputs** - Values that can be passed between steps

   ```yaml
   schema-version: v1
   tasks:
     get-version:
       steps:
         - run: |
             echo "version=1.0.0" >> $MARU2_OUTPUT
           id: version-step
         - run: echo "Version is ${{ from "version-step" "version" }}"
   ```

## Example workflow

This example demonstrates inputs, task references, and output passing:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json
schema-version: v1

aliases:
  # Local file alias for common tasks
  common:
    path: workflows/common.yaml

tasks:
  # This is the default task that runs when you type just 'maru2'
  default:
    inputs:
      message:
        description: "Message to display"
        default: "Hello, World!"
        required: true
    steps:
      - uses: greet
        with:
          message: |
            ${{ input "message" }}
      # Use a task from an aliased workflow
      - uses: common:cleanup

  # A reusable greeting task
  greet:
    inputs:
      message:
        description: "Message to display"
        required: true
    steps:
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

# Run a task from a local alias
maru2 common:cleanup
```

## Exploring workflows

Maru2 provides powerful tools for understanding and documenting your workflows:

### Discovering available tasks

```sh
# List all available tasks in the current workflow
maru2 --list

# List tasks from a specific workflow file
maru2 --from other-tasks.yaml --list

# List tasks from a remote workflow
maru2 --from "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" --list
```

### Understanding workflow structure

```sh
# Generate detailed explanation of all tasks and their parameters
maru2 --explain

# Explain specific tasks
maru2 --explain greet

# Explain tasks from a different workflow
maru2 --from workflows/deploy.yaml --explain production-deploy
```

The `--explain` command generates comprehensive documentation including:

- Task descriptions and input parameters
- Default values and validation rules
- Task dependencies and relationships
- Alias definitions for remote repositories
- Schema version information

## Advanced features

Maru2 includes powerful features for complex workflows:

- **Conditional execution** - Control step execution with `if` directives
- **Error handling and tracebacks** - Get detailed information about errors
- **Environment variable integration** - Use environment variables as input defaults
- **Remote task execution** - Execute tasks from remote repositories
- **Input validation** - Validate inputs using regular expressions
- **Package URL aliases** - Create shortcuts for remote repositories and local workflow files
- **Local file aliases** - Reference local workflow files with short aliases
- **Aliased task execution** - Run tasks from aliased workflows using `alias:task` syntax
- **Workflow explanation** - Generate detailed documentation of tasks and their parameters with `--explain`

## Next steps

Ready to dive deeper? Continue with:

- [CLI Documentation](cli.md) to learn all the command line options
- [Workflow Syntax](syntax.md) for detailed information on defining tasks
- [Built-in Tasks](builtins.md) to discover pre-defined tasks you can use
