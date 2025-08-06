# Oils-readline

Proof-of-concept [oils](https://oils.pub/) readline as a separate process.

It makes use of [Oils Headless mode](https://oils.pub/release/latest/doc/headless.html) which means that every executed command receives a separate Stdin, stdout and stderr.
This is combined with [reeflective/readline](https://github.com/reeflective/readline) which provides a clean readline interface.

This allows us to have golang tool to handle user input, but forward inputs for commands cleanly to them.
We know exactly what output comes from which command.
We can manage IO for "background" processes completely independently.
We can forward output to any other program we like.

# Build and run
Build with:
```shell
go build
```

Run (depends on oils, currently not builtin. [TODO](https://web.archive.org/web/20220509043908/https://0xcf9.org/2021/06/22/embed-and-execute-from-memory-with-golang/)):

```shell
# Bash-like
./oils-readline -oil_path $(which osh)
# New YSH
./oils-readline -oil_path $(which ysh)
```

## Current state

Current state is implement basic necessities to actually be comparable to libc readline.
Afterwards the idea is to implement highlighting, advanced completion using carapace, an improved history including the output of preexisting commands, forwarding of output to other processes (e.g. display images in an actual image viewer), etc.
