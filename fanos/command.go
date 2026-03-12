package fanos

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Melkor333/oils-readline/shell"
	"github.com/creack/pty"

	//"github.com/mcpherrinm/multireader"
	//"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

// Implementation of the tea.ExecCommand interface for fanos
type Command struct {
	shell                 *Shell
	commandline           string
	err                   error
	ctx                   context.Context
	Cancel                context.CancelFunc
	stdin, stdout, stderr *os.File
	stdoutBuf, stderrBuf  *strings.Builder
	stdoutMu, stderrMu    sync.Mutex
	// For the Client
	tty      *os.File
	stderrIn *os.File
	wg       *sync.WaitGroup
	lock     *sync.Mutex
}

func (c *Command) CommandLine() string {
	return c.commandline
}

func (c *Command) Stdin() io.Writer {
	return c.stdin
}

func (c *Command) Stdout() string {
	c.stdoutMu.Lock()
	defer c.stdoutMu.Unlock()
	return c.stdoutBuf.String()
}

func (c *Command) Stderr() string {
	c.stderrMu.Lock()
	defer c.stderrMu.Unlock()
	return c.stderrBuf.String()
}

func (c *Command) SetStdout(stdout io.Reader) {
	c.stdout = stdout.(*os.File) // this will panic if it's not an *os.File, maybe add error handling or make c.stdout an io.Reader instead?
}

func (c *Command) SetStdin(stdin io.Writer) {
	c.stdin = stdin.(*os.File) // this will panic if it's not an *os.File, maybe add error handling or make c.stdin an io.Writer instead?
}

func (shell *Shell) Command(commandLine string, size *pty.Winsize) (shell.Command, error) {
	var err error
	var c *Command = new(Command)
	// check errors
	c.commandline = commandLine
	c.shell = shell
	c.stdoutBuf = new(strings.Builder)
	c.stderrBuf = new(strings.Builder)
	c.ctx, c.Cancel = context.WithCancel(context.Background())
	c.wg = new(sync.WaitGroup)
	// Will be closed when the command was executed
	c.wg.Add(1)

	// get a PTY Master & Slave
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}

	c.tty = tty
	pty.Setsize(ptmx, size)

	var descriptor int
	descriptor = int(ptmx.Fd())
	_, err = term.MakeRaw(descriptor)
	if err != nil {
		log.Println("Couldn't make file raw")
	}

	c.stdin = ptmx

	c.stdout = ptmx
	// Read from stdout/stderr into our buffer
	// TODO: the stdoutBuf.Write might require a lock?!
	c.wg.Go(func() {
		buf := make([]byte, 1024*1024) // large buffer
		for {
			count, err := c.stdout.Read(buf)
			if err != nil {
				// TODO: This message is currently unhandled!
				break
				// handle error / EOF
			}
			c.stdoutMu.Lock()
			c.stdoutBuf.Write(buf[:count])
			c.stdoutMu.Unlock()
		}
	})

	c.stderr, c.stderrIn, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	c.wg.Go(func() {
		buf := make([]byte, 100) // large buffer
		for {
			count, err := c.stderr.Read(buf)
			if err != nil {
				// TODO: This message is currently unhandled!
				break
				// handle error / EOF
			}
			c.stderrMu.Lock()
			c.stderrBuf.Write(buf[:count])
			c.stderrMu.Unlock()
		}
	})

	return c, nil
}

func (c *Command) Wait() {
	c.wg.Wait()
}

func (c *Command) Run() {
	// Cleanup the "help" line
	err := c.shell.Run(c.commandline, c.tty, c.tty, c.stderrIn)

	c.wg.Done()
	if err != nil {
		c.err = err
	}
}

func (c *Command) Error() string {
	return "Capturing Exit code is not yet implemented"
}
