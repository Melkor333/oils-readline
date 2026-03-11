package shell

import (
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/chalk-ai/bubbline/editline"
	"github.com/creack/pty"
)

type Shell interface {
	//StdIO(*os.File, *os.File, *os.File) error
	Command(cmd string, size *pty.Winsize) (Command, error)
	Run(cmd string, ptmx, tty, stderr *os.File) error
	Cancel()
	Complete([][]rune, int, int) (string, editline.Completions)
	Dir() string
	Wait()
}

type Command interface {
	Run() tea.Msg
	Stdin() io.Writer
	Stdout() io.Reader
	Stderr() io.Reader
	SetStdout(stdout io.Reader)
	SetStdin(stdin io.Writer)
}
