package shell

import (
	"io"
	"os"

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
