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

### Testing

**Run tests in short mode to avoid network dependencies**:

```bash
# Recommended: Run without network tests
go test -short -v -timeout 3m ./...

# Full test suite (requires GITHUB_TOKEN environment variable)
go test -race -cover -coverprofile=coverage.out -failfast -timeout 3m ./...

# Run specific E2E tests
go test ./cmd/ -run TestE2E/<TestName> -v
```

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
2. `go test -short ./...` (run core tests)
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
- **`Makefile`**: Primary build orchestration
- **`tasks.yaml`**: Example workflow showing maru2 syntax
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

### Dependencies & Architecture Notes

- **External deps**: Uses `github.com/spf13/cobra` for CLI, `github.com/goccy/go-yaml` for YAML parsing
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