package shell

import (
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/chalk-ai/bubbline/editline"
	"github.com/creack/pty"
)

type CommandOutputErrorMsg error
type NewCommandMsg struct{ Cmd Command }
type CommandDoneMsg struct{ Cmd Command }
type StdoutMsg struct{ Cmd Command }
type StderrMsg struct{ Cmd Command }
type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{} // Go to main

type Widget interface {
	Init() tea.Cmd
	Update(tea.Msg) (Widget, tea.Cmd)
	View() string
	Blur()
	Focus() tea.Cmd // because some set a virtual cursor
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
