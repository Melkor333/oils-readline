package shell

import (
	"io"
	"os"

	"github.com/chalk-ai/bubbline/editline"
	"github.com/creack/pty"
)

type CommandOutputErrorMsg error
type NewCommandMsg struct{ Cmd Command }
type CommandDoneMsg struct{ Cmd Command }
type StdoutMsg struct{ Cmd Command }
type StderrMsg struct{ Cmd Command }

type RequestHistoryEntryMsg struct{ Index int }
type HistoryEntryMsg struct {
	Cmd   Command
	Index int
	Total int
}

type Shell interface {
	//StdIO(*os.File, *os.File, *os.File) error
	Command(cmd string, size *pty.Winsize) (Command, error)
	Run(cmd string, ptmx, tty, stderr *os.File) error
	GetPrompt() string // TODO: Should also return an error?
	Cancel()
	Complete([][]rune, int, int) (string, editline.Completions)
	Dir() string
	Wait()
}

type Command interface {
	Run()
	CommandLine() string
	Wait()
	Stdin() io.Writer
	Stdout() string
	Stderr() string
	SetStdout(stdout io.Reader)
	SetStdin(stdin io.Writer)
	SetOnStdout(fn func())
	SetOnStderr(fn func())
}
