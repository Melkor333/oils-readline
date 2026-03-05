package fanos

import (
	"context"
	"io"
	"log"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"

	//"github.com/mcpherrinm/multireader"
	//"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

// Implementation of the tea.ExecCommand interface for fanos
type Command struct {
	shell                 *Shell
	CommandLine           string
	err                   error
	ctx                   context.Context
	Cancel                context.CancelFunc
	stdin, stdout, stderr *os.File
	// For the Client
	tty      *os.File
	stderrIn *os.File
	wg       *sync.WaitGroup
	lock     *sync.Mutex
}

type CommandDoneMsg tea.Msg

func (c *Command) Stdin() io.Writer {
	return c.stdin
}

func (c *Command) Stdout() io.Reader {
	return c.stdout
}

func (c *Command) Stderr() io.Reader {
	return c.stderr
}

func (shell *Shell) Command(commandLine string, size *pty.Winsize) (*Command, error) {
	var err error
	var c *Command = new(Command)
	// check errors
	c.CommandLine = commandLine
	c.shell = shell
	c.ctx, c.Cancel = context.WithCancel(context.Background())
	c.wg = new(sync.WaitGroup)
	c.wg.Add(1)
	// TODO: somethingsomething TeeReader...?
	//c.stdout = multireader.New()
	//c.stderr = multireader.New()
	// to be unlocked when stdin exists.

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

	// Stdin has to be set by writer
	c.stdin = ptmx
	c.stdout = ptmx

	// Stdout
	//c.wg.Add(1)
	//go func() {
	//	defer c.stdout.Close()
	//	defer c.wg.Done()
	//	io.Copy(c.stdout, ptmx)
	//}()

	// Stderr
	c.stderr, c.stderrIn, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	//c.stderrIn = stderrIn
	//c.wg.Add(1)
	//go func() {
	//	defer c.stderr.Close()
	//	defer c.wg.Done()
	//	io.Copy(c.stderr, stderrOut)
	//}()

	return c, nil
}

// Get a reader to read STDOUT, even AFTER the command finished
//func (c *Command) Stdout() io.Reader {
//	return c.stdout.Reader()
//}

// Get a reader to read STDERR, even AFTER the command finished
//func (c *Command) Stderr() io.Reader {
//	return c.stderr.Reader()
//}

// Wait for command to finish
func (c *Command) Wait() {
	c.wg.Wait()
}

// This is the desired tea.Cmd
func (c *Command) Run() tea.Msg {
	// Cleanup the "help" line
	err := c.shell.Run(c.CommandLine, c.tty, c.tty, c.stderrIn)
	c.wg.Done()
	if err != nil {
		c.err = err
	}

	return CommandDoneMsg(c)
}

func (c *Command) Error() string {
	return "Capturing Exit code is not yet implemented"
}
