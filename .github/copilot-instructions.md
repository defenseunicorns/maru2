# Copilot Instructions for Maru2

## Repository Overview

**Maru2** is a simple, powerful task runner written in Go that makes workflow automation easy and intuitive. Inspired by the simplicity of Makefiles but with modern features like GitHub Actions, it helps users define, organize, and execute tasks with minimal configuration.

## Core Design Principles & Philosophy

Maru2 development is guided by fundamental principles that prioritize simplicity, performance, and user experience. **All code changes must align with these core principles.**

### Rob Pike's 5 Rules of Programming

These foundational rules guide all performance and algorithmic decisions:

**Rule 1**: You can't tell where a program is going to spend its time. Bottlenecks occur in surprising places, so don't try to second guess and put in a speed hack until you've proven that's where the bottleneck is.

**Rule 2**: Measure. Don't tune for speed until you've measured, and even then don't unless one part of the code overwhelms the rest.

**Rule 3**: Fancy algorithms are slow when n is small, and n is usually small. Fancy algorithms have big constants. Until you know that n is frequently going to be big, don't get fancy. (Even if n does get big, use Rule 2 first.)

**Rule 4**: Fancy algorithms are buggier than simple ones, and they're much harder to implement. Use simple algorithms as well as simple data structures.

**Rule 5**: Data dominates. If you've chosen the right data structures and organized things well, the algorithms will almost always be self-evident. Data structures, not algorithms, are central to programming.

**Key Takeaways**: Pike's rules 1 and 2 restate Tony Hoare's famous maxim "Premature optimization is the root of all evil." Ken Thompson rephrased Pike's rules 3 and 4 as "When in doubt, use brute force." Rules 3 and 4 are instances of the design philosophy KISS. Rule 5 was previously stated by Fred Brooks in The Mythical Man-Month. Rule 5 is often shortened to "write stupid code that uses smart objects."

### Maru2-Specific Design Principles

**Simplicity First**: "Simple things should be simple, complex things should be possible" ~ Alan Kay

- Prioritize straightforward, readable implementations over clever optimizations
- Make common use cases trivial to accomplish
- Ensure advanced features don't complicate basic workflows

**Excellent Shell Script Experience**: The last mile in every effort is paved with `bash`, `sh`, and tears. As such maru2 must make the experience of using embedded scripts excellent.

- Shell script integration should be seamless and intuitive
- Error handling and debugging for shell scripts must be superior
- Output formatting and logging should enhance script readability

**Low Latency Over Complexity**: If choosing between creating an operation with low latency, simple logic that must be chained together, or a singular powerful, yet costly, operation, choose the simple low latency option.

- Prefer multiple fast operations over single slow operations
- Design for composability and pipeline-friendly patterns
- Optimize for startup time and immediate feedback

**Documentation vs Implementation Consistency**: The documentation states how the system _should_ operate. The implementation drives how it _does_. In a conflict between the two, evaluate which behavior is more consistent with the overall system and update the other to reflect that change.

- Neither documentation nor implementation is automatically "correct"
- Evaluate conflicts based on system-wide consistency
- Update the inconsistent component to match the more logical behavior
- Maintain clear, accurate documentation that reflects actual behavior

### High-Level Repository Information

- **Size**: Medium-sized Go project
- **Language**: Go (current version in `go.mod`) (primary), YAML, Markdown, Shell scripts
- **Framework**: Cobra CLI framework with Go modules dependency management
- **Target**: Cross-platform static binaries (Linux, macOS, supports amd64/arm64) with `CGO_ENABLED=0`
- **Status**: Early development - expect breaking changes
- **Testing**: Comprehensive test suite using `testscript` for E2E testing and standard Go tests

## Build Instructions

### Bootstrap & Dependencies

Dependencies are managed via Go modules and downloaded automatically. No manual dependency installation required except for optional linting.

### Build Commands

**Always run `make` before any other operations** to ensure binaries and schemas are up-to-date:

```bash
# Build all binaries and generate schemas (REQUIRED FIRST STEP)
make

# Individual builds
make maru2          # Build main binary + generate schemas
make maru2-publish  # Build publish binary only
make clean          # Remove build artifacts
```

**Critical**: The `make` command generates three schema files:

- `maru2.schema.json` (root-level, for public consumption and IDE integration)
- `schema/v0/schema.json` (version-specific, for internal validation)
- `schema/v1/schema.json` (version-specific, for internal validation)

These files MUST be committed if changed during development.

### Makefile Task Execution

**The Makefile includes a `%:` catch-all rule** that allows running any maru2 task defined in `tasks.yaml`:

```bash
# Run any maru2 task via make
make <task-name>              # Executes: ./bin/maru2 <task-name>
make <task-name> ARGS="..."   # Executes: ./bin/maru2 <task-name> <ARGS>

# Examples
make hello-world              # Runs the hello-world task
make echo                     # Runs the echo task with default input
make echo ARGS='-w text="Custom message"'  # Runs echo with custom text
```

### Testing

**Multiple testing approaches available**:

```bash
# Option 1: Via maru2 task system (uses tasks.yaml) (recommended for development)
make test                           # Full test suite (short=false)
make test ARGS='-w short=true'      # Skip network tests (short=true)

# Option 2: Direct Go testing
go test -short -v -timeout 3m ./...  # Skip network-dependent tests
go test -race -cover -coverprofile=coverage.out -failfast -timeout 3m ./...  # Full suite

# Run specific E2E tests (testscript-based)
go test ./cmd/ -run TestE2E/<TestName> -v
```

**Test Configuration**:

- Use `-short` flag to skip network-dependent tests
- E2E testing uses `testscript` framework in `/testdata/` - each `.txtar` file defines a complete test scenario
- The `test` task in `tasks.yaml` sets `CGO_ENABLED=1`, uses race detection, and can be customized via maru2's input system
- Prompt the user after finishing a feature for confirmation to run the entire test suite, do not auto run it.

### Linting

**golangci-lint must be installed separately**:

```bash
# Install golangci-lint first
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Then run linting
make lint           # Run linters
make lint-fix       # Run linters with auto-fix
```

### Validation Commands

Always validate changes with this sequence:

1. `make` (rebuild + regenerate schemas)
2. `make test ARGS='-w short=true'` (run core tests)
3. `make lint` (if golangci-lint installed)
4. Check schema files are committed if changed

### Common Build Issues & Workarounds

- **Test failures without `-short`**: Network tests require `GITHUB_TOKEN` environment variable to avoid rate limits
- **golangci-lint not found**: Install separately with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **Schema out of sync**: Run `make` to regenerate, then commit changes
- **Timeout issues**: Use longer timeout for full test suite (`-timeout 5m`)

**Note**: For accessing private repositories or avoiding API rate limits, set `GITHUB_TOKEN` and `GITLAB_TOKEN` environment variables.

## Project Layout

### Architecture Overview

Maru2 follows a modular Go architecture with clear separation of concerns:

```
/cmd/           - CLI entry points and command implementations
  /maru2/       - Main CLI binary
  /maru2-publish/ - Publishing utility binary
  /maru2-schema/  - Schema generation utility
  /internal/    - Example of embedding maru2 in other CLIs
/schema/        - YAML schema definitions (versioned)
  /v0/          - Schema version 0
  /v1/          - Schema version 1
  /generics.go  - Shared schema components
/config/        - Configuration file handling (versioned)
  /v0/          - Current config schema version
/uses/          - Remote task fetching (GitHub, GitLab, OCI)
/builtins/      - Built-in tasks (echo, fetch)
/testdata/      - E2E test scenarios using testscript
/docs/          - Comprehensive documentation
```

### Core Architecture Patterns

**Builtin System**: Built-in tasks are registered in `builtins/registration.go` with a factory pattern. Each builtin implements the `Builtin` interface with an `Execute(ctx context.Context) (map[string]any, error)` method. Use `builtins.Get("name")` to retrieve instances.

**Schema-Driven Validation**: The entire workflow syntax is defined via Go structs in `schema/v0/` and `schema/v1/` that auto-generate JSON schemas. The `WorkFlowSchema()` function creates the main schema, while individual structs use `JSONSchemaExtend()` methods for documentation or behavior that is too complex to represent within the struct tags. Configuration files are also versioned using the same pattern in `config/v0/`.

**Schema Versioning**: Maru2 now supports multiple schema versions (v0 and v1) with separate validation pipelines. The build process generates version-specific schema files for internal validation while maintaining a public schema for IDE integration.

**Remote Uses System**: The `uses/` package implements pluggable fetchers for different protocols (GitHub, GitLab, OCI, HTTP, local files). Each fetcher implements the `Fetcher` interface and is registered via URL scheme detection.

**Testscript E2E Pattern**: E2E tests use `.txtar` archive format in `/testdata/` with `go test ./cmd/ -run TestE2E/<TestName> -v` for individual test execution.

### Key Workflow Concepts

**Step Outputs**: Steps can produce outputs by writing `key=value` pairs to the `$MARU2_OUTPUT` environment variable file. Access outputs from previous steps using `${{ from "step-id" "output-key" }}` syntax. (see `output.go` and `output_test.go`)

**Shell Selection**: Steps support different shells via the `shell` property: `sh`, `bash`, `powershell`, `pwsh`. Default behavior varies by command content.

**Conditional Execution**: Use `if` property with expr expressions. Built-in functions include `failure()`, `always()`, `cancelled()`, `input("name")`, and `from("step-id", "key")`.

**Package URLs**: Remote tasks use package-url (purl) spec format like `pkg:github/owner/repo@version#path/to/tasks.yaml` which supports aliases for shorthand references. `oci`, `file`, `http` and `https` are also supported. No matter what, a `uses` field _must_ be a proper URL with a protocol scheme.

**OCI Artifact Support**: Maru2 supports distributing and consuming workflows as OCI artifacts in container registries. This enables workflows to be versioned, cached, and distributed through existing container infrastructure. See the `maru2-publish` CLI and the `uses/oci.go` files for more information.

**Input Validation**: Inputs support regex validation via the `validate` property to enforce format constraints before task execution.

### Key Configuration Files

- **`.golangci.yaml`**: Linting configuration with custom rules
- **`go.mod`**: Go module definition and dependencies
- **`Makefile`**: Primary build orchestration with `%:` catch-all rule for task execution
- **`tasks.yaml`**: Example workflow showing maru2 syntax and defining available tasks
- **`maru2.schema.json`**: Auto-generated JSON schema for YAML validation
- **`.goreleaser.yaml`**: Release automation configuration
- **`codecov.yaml`**: Code coverage configuration
- **`release-please-config.json`**: Release automation configuration

### GitHub Workflows & CI

Located in `.github/workflows/`:

- **`go.yaml`**: Main CI pipeline (build, test, lint) on push/PR to main
- **`release.yaml`**: Automated releases
- **`nightly-build.yaml`**: Nightly builds
- **`commitlint.yaml`**: Commit message linting

**CI Requirements**:

- All schema files must remain in sync (v0, v1, and public schemas)
- Tests must pass on both Linux and macOS
- Linting must pass
- Coverage reporting included
- Fuzz testing on schema patterns included

### Validation Pipeline

The CI runs these checks:

1. `make` (build + schema generation for all versions)
2. Schema sync validation (`git diff --exit-code`)
3. `go test -race -cover` with coverage reporting
4. `golangci-lint run`
5. Fuzz testing on schema patterns

## Dependencies & Dependency Management

### Core Dependencies Overview

Maru2 maintains a **minimal dependency footprint** with carefully selected, well-maintained libraries:

**CLI Framework**:

- `github.com/spf13/cobra` - Industry-standard CLI framework with subcommands and flag parsing
- `github.com/spf13/pflag` - POSIX-compliant command-line flag parsing

**YAML Processing**:

- `github.com/goccy/go-yaml` - High-performance YAML parser with better error reporting than gopkg.in/yaml

**Schema & Validation**:

- `github.com/invopop/jsonschema` - JSON Schema generation from Go structs
- `github.com/xeipuuv/gojsonschema` - JSON Schema validation for YAML workflows

**Template/Expression Engine**:

- `text/template` - Go's standard template engine for script interpolation (`${{ input "name" }}` syntax)
- `github.com/expr-lang/expr` - Fast expression evaluation for conditional `if` statements

**Remote Integrations**:

- `github.com/google/go-github/v62` - GitHub API client for fetching remote tasks
- `gitlab.com/gitlab-org/api/client-go` - GitLab API client for GitLab integration
- `oras.land/oras-go/v2` - OCI registry support for artifact-based task distribution
- `github.com/package-url/packageurl-go` - Package URL (purl) parsing for remote task references
- `github.com/olareg/olareg` - OCI local registry implementation for testing
- `github.com/opencontainers/image-spec` - OCI image specification support for artifact handling

**UI/Logging**:

- `github.com/charmbracelet/lipgloss` - Terminal styling and color output
- `github.com/charmbracelet/log` - Structured, leveled logging with styling
- `github.com/alecthomas/chroma/v2` - Syntax highlighting for code output

**Utilities**:

- `github.com/spf13/afero` - Filesystem abstraction for testability
- `github.com/go-viper/mapstructure/v2` - Clean struct mapping and configuration binding
- `github.com/spf13/cast` - Safe type conversion utilities
- `github.com/muesli/termenv` - Terminal environment detection and feature support

**Testing**:

- `github.com/stretchr/testify` - Assertion and testing utilities
- `github.com/rogpeppe/go-internal` - Internal Go tooling support (used for testscript E2E testing)

### Dependency Philosophy & Rules

**CRITICAL RULES**:

1. **Never modify `go.mod` or `go.sum`** - Dependabot automatically handles dependency updates. Agents must not add, remove, or update dependencies.

2. **Leverage Go standard library first** - Before considering external dependencies, always check if functionality exists in the standard library (`net/http`, `encoding/json`, `os`, `path/filepath`, etc.).

3. **No new dependencies** - The current dependency set is intentionally minimal and covers all required functionality. Adding new dependencies requires exceptional justification and maintainer approval.

**Rationale**: Maintaining a minimal dependency surface reduces security risks, improves build reliability, ensures long-term maintainability, and keeps the binary size small for the static binary distribution model.

### Go Development Best Practices

**Code Discovery & Documentation**:

1. **Always use `go doc` for precision** - Before writing new functionality, use `go doc <package>.<Type>` or `go doc <package>.<Function>` to understand:
   - Function signatures and return types
   - Interface definitions and requirements
   - Struct fields and methods
   - Usage examples and behavior
   - Package-level documentation

   ```bash
   # Examples
   go doc fmt.Printf           # Function documentation
   go doc http.Handler         # Interface definition
   go doc context.Context      # Type and methods
   go doc encoding/json        # Package overview
   ```

2. **Leverage Go's built-in tooling**:
   - `go doc -all <package>` - Show all exported symbols
   - `go doc -src <symbol>` - Show source code

**Code Quality & Review Strategies**:

1. **Follow Go idioms and conventions**:
   - Use receiver names that are short and consistent (e.g., `c *Client`, not `client *Client`)
   - Prefer composition over inheritance
   - Handle errors explicitly, don't ignore them
   - Use meaningful variable names, avoid abbreviations
   - Keep functions small and focused on single responsibility

2. **Error handling best practices**:
   - Errors should be checked the vast majority of the time, exceptions do exist: `if err != nil { return err }`
   - Wrap errors with context only if the underlying error does not provide enough information: `fmt.Errorf("failed to process %s: %w", name, err)`
   - Use sentinel errors for expected conditions, but sparingly: `var ErrNotFound = errors.New("not found")`

3. **Testing strategies**:
   - Write table-driven tests for multiple scenarios
   - Use `testify/require` for assertions that deal with error handling
   - Use `testify/assert` for all other assertions
   - Mock external dependencies using interfaces
   - Test both happy path and error conditions

4. **Memory and performance considerations**:
   - Use `strings.Builder` for string concatenation in loops
   - Prefer `bytes.Buffer` for binary data manipulation
   - Be mindful of goroutine leaks, always provide context cancellation
   - Use `sync.Pool` for frequently allocated objects

5. **Communication**:
   - Be brutally honest in your feedback, avoiding fluff wording and words of encouragement. High quality code is your objective, not assuaging the concerns of your reader.
   - If you are ever confused, or something does not look right, first stop and think, then if you are still confused pause and ask for clarification.
   - Avoid exclamatory openers, affirmative interjections, enthusiastic affirmations, or conversational aknowledgements.
   - Communicate in clear and concise prose that relays your intent without verbosity.

### Architecture Notes

- **Remote fetching**: Supports HTTP, GitHub, GitLab, and OCI artifact sources
- **Schema validation**: JSON Schema validation for YAML workflows with versioned schemas
- **Template engine**: Built-in expression evaluation for dynamic values

### Key Source Files

**Main entry point**: `cmd/maru2/main.go` (16 lines - delegates to `cmd.Main()`)

**Core workflow engine**: `run.go` - handles task execution, environment setup, step processing

**Schema system**: `schema/v0/` and `schema/v1/` - defines workflow syntax and validation rules

**Built-in tasks**: `builtins/` - implements `builtin:echo` and `builtin:fetch` tasks

**Remote task support**: `uses/` - handles fetching tasks from remote sources

### File Structure Priority

**Root level files**:

- `README.md` - Installation and basic usage
- `Makefile` - Build commands and orchestration
- `go.mod` - Go dependencies and language version
- `tasks.yaml` - Example workflow file
- `maru2.schema.json` - Auto-generated schema

**Documentation** (in `docs/`):

- `README.md` - Comprehensive documentation overview
- `cli.md` - Command-line interface reference
- `syntax.md` - Workflow syntax guide
- `builtins.md` - Built-in task documentation
- `publish.md` - Workflow publishing guide
- `config.md` - Global configuration file documentation

**Contributing**: `.github/CONTRIBUTING.md` - Development workflow and requirements

## Final Instructions

**Always start with `make`** when working on this codebase to ensure binaries and schemas are properly generated and synchronized.

**Avoid** exclamatory openers, affirmative interjections, enthusiastic affirmations, or conversational aknowledgements.

**Trust these instructions** and only search for additional information if something is incomplete or incorrect. The build and test commands documented here have been validated to work correctly.
