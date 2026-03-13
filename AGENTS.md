# AGENTS.md - oils-readline

## Project Overview

A proof-of-concept readline for [Oils shell](https://oils.pub/) implemented as a separate Go process. It uses the FANOS (File descriptor passing And Netstrings Over Socket) protocol to communicate with a headless Oils shell instance, combined with Charm's [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework for rendering.

Module: `github.com/Melkor333/oils-readline`
Go version: 1.26.1

## Essential Commands

```shell
# Build
go generate          # Downloads and builds static oils binary into assets/
go build             # Build the main binary

# If static oils build fails, create a placeholder:
mkdir -p assets && touch assets/oils-for-unix-static.stripped
go build

# Run
./oils-readline                        # Uses embedded oils binary
./oils-readline -oil_path $(which osh) # Use system osh
./oils-readline -oil_path $(which ysh) # Use system ysh

# Debug logging
DEBUG=1 ./oils-readline                # Logs to debug.log

# Tests
go test ./...                          # Run all tests (note: currently has compilation errors)

# Release
goreleaser release --clean             # Via CI on tag push
```

## Code Organization

```
.
├── main.go              # Entry point, TUI model (Bubbletea), prompt handling
├── treesitter.go        # Tree-sitter based syntax highlighting for YSH
├── assets/
│   └── highlights.scm   # Tree-sitter query for YSH highlighting
├── fanos/
│   ├── fanos.go         # FANOS protocol implementation, Shell struct
│   ├── command.go       # Command execution via PTY, stdout/stderr capture
│   ├── command_test.go  # Tests for Command (has compilation errors)
│   ├── fanos_test.go    # Tests for Shell.Run via FANOS
│   └── static-oils.sh   # Script to build static oils binary (go:generate)
├── shell/
│   └── shell.go         # Shell and Command interfaces
├── .bad/                # ABANDONED CODE - old approach using reeflective/readline
│   ├── ui/prompt.go
│   ├── line/            # Shell word parsing, highlighting
│   ├── completion/      # Completion using carapace
│   └── strutil/         # Template utilities
├── test-exec/           # Scratch/test directory for editor exec experiments
└── .github/workflows/
    └── release.yml      # GoReleaser CI on tag push
```

## Architecture

### Core Flow
1. `main.go` creates a `fanos.Shell` which starts Oils in `--headless` mode
2. Communication uses Unix socketpair with FANOS netstring protocol
3. Commands are executed by opening a PTY, then sending `EVAL <command>` + file descriptors via FANOS
4. Stdout is captured from the PTY master; stderr from a separate pipe
5. The Bubbletea TUI renders command output and manages user input

### Key Interfaces (`shell/shell.go`)
- `Shell`: `Command()`, `Run()`, `Cancel()`, `Complete()`, `Dir()`, `Wait()`
- `Command`: `Run()`, `Wait()`, `Stdin()`, `Stdout()`, `Stderr()`, `CommandLine()`

### PTY & Command Execution (`fanos/command.go`)
- Each command gets a PTY (pty.Open()) with raw terminal mode
- Stdout read in 1MB buffer goroutine; stderr in 100-byte buffer
- Uses `sync.Mutex` to protect stdout/stderr buffers for concurrent reads

### FANOS Protocol (`fanos/fanos.go`)
- Socketpair (AF_UNIX, SOCK_STREAM) for bidirectional fd-passing
- Messages are netstrings: `<length>:<data>,`
- Sends file descriptors via `syscall.Sendmsg` with `syscall.UnixRights`
- Currently embeds a static Oils binary; falls back to system binary via `-oil_path`

## Code Patterns & Conventions

### Naming
- Fields in structs use lowercase (unexported): `commandline`, `stdoutBuf`, `stderrMu`
- The `Command` struct field is `commandline` (not `CommandLine`) - the accessor method is `CommandLine()`
- Interfaces defined in `shell/` package, implementations in `fanos/`

### Dependencies (Charm ecosystem)
- `charm.land/bubbletea/v2` - TUI framework (v2 API)
- `charm.land/bubbles/v2` - UI components (textinput, viewport)
- `charm.land/lipgloss/v2` - Styling
- `github.com/chalk-ai/bubbline` - Readline interface (computil, editline)
- `github.com/creack/pty` - PTY handling
- `github.com/tree-sitter/go-tree-sitter` + `go-tree-sitter-highlight` - Syntax highlighting

### Debug Logging
- Controlled by `DEBUG` environment variable
- When set: logs to `debug.log` via `tea.LogToFile()`
- When unset: `log.SetOutput(io.Discard)` suppresses all logs
- Use `log.Print()` / `log.Printf()` for debug statements

### Error Handling
- Many TODOs indicate incomplete error handling (exit codes, PTY read errors)
- `Command.Error()` currently returns a placeholder string
- Shell stderr is `io.Discard` - not captured

## Known Issues & Gotchas

### Test Compilation Errors
`fanos/command_test.go` has errors at lines 84 and 155: uses `CommandLine` (exported) but the struct field is `commandline` (unexported). These tests have empty test cases so they don't run, but they won't compile.

### Unused Dependencies
- `github.com/mcpherrinm/multireader` is imported but unused (commented out in imports)

### PTY Limitations (documented in main.go)
- Programs needing `/dev/tty` (like `less`) won't work normally - they read from stderr
- `sudo` needs `-S` flag to read from stdin
- Workaround: `alias less="less 2<&0"`

### Gorelease Build Requirements
- `CGO_ENABLED=1` required for go-tree-sitter-highlight
- Only builds for linux/amd64
- Version injected via ldflags: `-X main.Version={{ .Version }}`

### `.bad/` Directory
Contains abandoned code from an earlier approach using `github.com/reeflective/readline`:
- Shell word parsing with quote handling
- Cobra-based command highlighting
- Carapace completion engine integration
- **Do not import or use** - dependencies are not in go.mod

### Current Branch State
Branch `broken-refactor` suggests active refactoring. Check `git log` for context.

## Release Process

Push a semver tag (`v*.*.*`) to trigger `.github/workflows/release.yml`:
1. GoReleaser runs `go mod tidy` and `go generate ./...`
2. Builds with CGO enabled, stripped binary
3. Creates GitHub release with archives and packages (deb, rpm, apk, termux.deb)
