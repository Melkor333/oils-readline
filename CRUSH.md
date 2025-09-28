# CRUSH.md

## Project Overview
This project is a prototype Shell UI for Go's `sh` package and external shell interpreters. It manages command execution using pseudoterminals (PTY) and pipes for handling standard input/output.

## Build and Execution
- **Build Command**: To compile the project, use: `go build`.
- **Running**: Execute the compiled binary directly after building to use the shell.

## Testing
- **Test Framework**: It appears there are no existing tests for unit testing. To add tests, create files with the `_test.go` suffix, and use Go's built-in `go test` command to run them.
- **Running a Single Test**: Once test files are created, use `go test -run TestName` to run specific tests.

## Linting
- **Lint Command**: For Go projects, it's common to use `golint` or `go vet`. Install these tools and run via `golint ./...` or `go vet ./...` for linting checks.

## Code Style Guidelines

### Imports
- Group imports into standard libraries, third-party packages, and local packages.
- Use standard Go import practices without unnecessary depth in import paths.

### Formatting
- Follow `gofmt` for code formatting to ensure consistency.
- Keep code clean and avoid excessive depth in package hierarchy.

### Types and Naming Conventions
- Use `CamelCase` for type names and `mixedCaps` or `mixedCapsWithInternalUnderscores` for variable names.
- Keep naming clear and reflective of the variable's purpose.

### Error Handling
- Use error handling idioms recommended by Go, like returning errors for functions that can fail.
- Important to check all errors returned by functions to avoid unexpected behavior.

### Comments
- Use comments to describe the purpose and exported functions or types. Keep comments concise and relevant.
- Avoid unnecessary comments that describe what code is doing (prefer code readability over comments).

## .gitignore
Ensure that the `.crush` directory is added to the `.gitignore` file if not present, to avoid committing personal configurations or temporary files.
