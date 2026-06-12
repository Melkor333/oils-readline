package shell

import (
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/chalk-ai/bubbline/editline"
	"github.com/creack/pty"
)

type CommandState int32

const (
	Ready CommandState = iota
	Queued
	Started
	Stopped
)

type CommandOutputErrorMsg error
type NewCommandMsg struct{ Cmd Command }
type CommandDoneMsg struct{ Cmd Command }
type StdoutMsg struct{ Cmd Command }
type StderrMsg struct{ Cmd Command }

type RequestHistoryEntryMsg struct {
	Index int
	Id    uint64
}

var _ TaggedMsg = RequestHistoryEntryMsg{}

func (msg RequestHistoryEntryMsg) Tag(id uint64) tea.Msg {
	msg.Id = id
	return msg
}

type TaggedMsg interface {
	Tag(uint64) tea.Msg
}

type TargetedMsg interface {
	TargetWidget() uint64
}

type HistoryEntryMsg struct {
	Cmd   Command
	Index int
	Total int
	Id    uint64
}

func (msg HistoryEntryMsg) TargetWidget() uint64 { return msg.Id }

var _ TargetedMsg = HistoryEntryMsg{}

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
	State() CommandState
	SetState(CommandState)
	Resize(size *pty.Winsize) error
}
