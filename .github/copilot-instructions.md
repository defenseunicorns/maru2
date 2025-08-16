# Copilot Instructions for Maru2

## Repository Overview

**Maru2** is a simple, powerful task runner written in Go that makes workflow automation easy and intuitive. Inspired by the simplicity of Makefiles but with modern features like GitHub Actions, it helps users define, organize, and execute tasks with minimal configuration.

### High-Level Repository Information
- **Size**: Medium-sized Go project (~80 files including tests and documentation)
- **Language**: Go 1.24.3 (primary), YAML, Markdown, Shell scripts
- **Framework**: Cobra CLI framework with Go modules dependency management
- **Target**: Cross-platform static binaries (Linux, macOS) with `CGO_ENABLED=0`
- **Status**: Early development - expect breaking changes

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

**Critical**: The `make` command generates `maru2.schema.json` and `schema/v0/schema.json`. These files MUST be committed if changed during development.

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
# Option 1: Direct Go testing (recommended for development)
go test -short -v -timeout 3m ./...  # Skip network-dependent tests
go test -race -cover -coverprofile=coverage.out -failfast -timeout 3m ./...  # Full suite

# Option 2: Via maru2 task system (uses tasks.yaml)
make test                           # Full test suite (short=false)
make test ARGS='-w short=true'      # Skip network tests (short=true)

# Run specific E2E tests
go test ./cmd/ -run TestE2E/<TestName> -v
```

**Important**: The `test` task in `tasks.yaml` provides an alternative testing interface that:
- Sets `CGO_ENABLED=1` (required for race detection)
- Uses the `short` input parameter to control `-short` flag
- Generates coverage reports and uses race detection by default
- Can be customized via maru2's input system

**Test timing**: Full test suite takes ~3 minutes. Use `-short` flag to skip network-dependent tests.

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
2. `go test -short ./...` or `make test ARGS='-w short=true'` (run core tests)
3. `make lint` (if golangci-lint installed)
4. Check schema files are committed if changed

### Common Build Issues & Workarounds

- **Test failures without `-short`**: Network tests require `GITHUB_TOKEN` environment variable
- **golangci-lint not found**: Install separately with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **Schema out of sync**: Run `make` to regenerate, then commit changes
- **Timeout issues**: Use longer timeout for full test suite (`-timeout 5m`)

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
  /v0/          - Current schema version
/config/        - Configuration file handling
/uses/          - Remote task fetching (GitHub, GitLab, OCI)
/builtins/      - Built-in tasks (echo, fetch)
/testdata/      - E2E test scenarios using testscript
/docs/          - Comprehensive documentation
```

### Key Configuration Files

- **`.golangci.yaml`**: Linting configuration with custom rules
- **`go.mod`**: Go module definition and dependencies
- **`Makefile`**: Primary build orchestration with `%:` catch-all rule for task execution
- **`tasks.yaml`**: Example workflow showing maru2 syntax and defining available tasks
- **`maru2.schema.json`**: Auto-generated JSON schema for YAML validation
- **`.goreleaser.yaml`**: Release automation configuration

### GitHub Workflows & CI

Located in `.github/workflows/`:
- **`go.yaml`**: Main CI pipeline (build, test, lint) on push/PR to main
- **`release.yaml`**: Automated releases
- **`nightly-build.yaml`**: Nightly builds

**CI Requirements**:
- All schema files must remain in sync
- Tests must pass on both Linux and macOS
- Linting must pass
- Coverage reporting included

### Validation Pipeline

The CI runs these checks:
1. `make` (build + schema generation)
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
- `github.com/expr-lang/expr` - Fast expression evaluation for dynamic template values (`${{ input "name" }}`)

**Remote Integrations**:
- `github.com/google/go-github/v62` - GitHub API client for fetching remote tasks
- `gitlab.com/gitlab-org/api/client-go` - GitLab API client for GitLab integration
- `oras.land/oras-go/v2` - OCI registry support for artifact-based task distribution

**UI/Logging**:
- `github.com/charmbracelet/lipgloss` - Terminal styling and color output
- `github.com/charmbracelet/log` - Structured, leveled logging with styling
- `github.com/alecthomas/chroma/v2` - Syntax highlighting for code output

**Utilities**:
- `github.com/spf13/afero` - Filesystem abstraction for testability
- `github.com/go-viper/mapstructure/v2` - Clean struct mapping and configuration binding
- `github.com/spf13/cast` - Safe type conversion utilities

**Testing**:
- `github.com/stretchr/testify` - Assertion and testing utilities
- `github.com/rogpeppe/go-internal` - Internal Go tooling support (used for testscript E2E testing)

### Dependency Philosophy & Rules

**CRITICAL RULES**:

1. **Never modify `go.mod` or `go.sum`** - Dependabot automatically handles dependency updates. Agents must not add, remove, or update dependencies.

2. **Leverage Go standard library first** - Before considering external dependencies, always check if functionality exists in the standard library (`net/http`, `encoding/json`, `os`, `path/filepath`, etc.).

3. **No new dependencies** - The current dependency set is intentionally minimal and covers all required functionality. Adding new dependencies requires exceptional justification and maintainer approval.

4. **Prefer standard library solutions**:
   - Use `net/http` for HTTP requests instead of third-party clients
   - Use `encoding/json` for JSON processing
   - Use `os/exec` for command execution
   - Use `path/filepath` for path manipulation
   - Use `strings`, `strconv`, `fmt` for text processing

**Rationale**: Maintaining a minimal dependency surface reduces security risks, improves build reliability, ensures long-term maintainability, and keeps the binary size small for the static binary distribution model.

### Architecture Notes

- **Remote fetching**: Supports GitHub, GitLab, and OCI artifact sources
- **Schema validation**: JSON Schema validation for YAML workflows
- **Template engine**: Built-in expression evaluation for dynamic values

### Key Source Files

**Main entry point**: `cmd/maru2/main.go` (16 lines - delegates to `cmd.Main()`)

**Core workflow engine**: `run.go` - handles task execution, environment setup, step processing

**Schema system**: `schema/v0/` - defines workflow syntax and validation rules

**Built-in tasks**: `builtins/` - implements `builtin:echo` and `builtin:fetch` tasks

**Remote task support**: `uses/` - handles fetching tasks from remote sources

### File Structure Priority

**Root level files**:
- `README.md` - Installation and basic usage
- `Makefile` - Build commands and orchestration
- `go.mod` - Dependencies (Go 1.24.3)
- `tasks.yaml` - Example workflow file
- `maru2.schema.json` - Auto-generated schema

**Documentation** (in `docs/`):
- `README.md` - Comprehensive documentation overview
- `cli.md` - Command-line interface reference
- `syntax.md` - Workflow syntax guide
- `builtins.md` - Built-in task documentation
- `publish.md` - Workflow publishing guide

**Contributing**: `.github/CONTRIBUTING.md` - Development workflow and requirements

## Final Instructions

**Trust these instructions** and only search for additional information if something is incomplete or incorrect. The build and test commands documented here have been validated to work correctly.

**Always start with `make`** when working on this codebase to ensure binaries and schemas are properly generated and synchronized.

For schema changes, **always commit the generated files** after running `make` as they are part of the project's interface.