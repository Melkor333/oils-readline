package main

import (
	"context"
	//TODO: no unnecessary depth
	"github.com/mcpherrinm/multireader"
	"io"
	"sync"
	"os"
)

type Command struct {
	shell       Shell
	CommandLine string
	// TODO: stdout/err should always just be a (buffered) reader!
	err error
	//Id                     int
	ctx            context.Context
	Cancel         context.CancelFunc
	stdinMu        sync.Mutex
	stdin          *os.File
	stdout, stderr *multireader.Buffer
	wg             *sync.WaitGroup
	lock           *sync.Mutex
}

// chain -> to which chain to add the command? be here?
func NewCommand(commandLine string) *Command {
	var command *Command = new(Command)
	// check errors
	command.CommandLine = commandLine
	command.ctx, command.Cancel = context.WithCancel(context.Background())
	command.wg = new(sync.WaitGroup)
	command.wg.Add(1)
	// TODO: somethingsomething TeeReader...?
	command.stdout = multireader.New()
	command.stderr = multireader.New()
	// to be unlocked when stdin exists.
	command.stdinMu.Lock()
	// TODO: Create a History object?
	//commMu.Lock()
	// TODO: UUID? even necessary?
	//command.Id = len(commands)
	//commMu.Unlock()
	return command

}

func (c *Command) Stdin() *os.File {
	// TODO: don't jse alock to wait...
	c.stdinMu.Lock()
	defer c.stdinMu.Unlock()
	return c.stdin
}

func (c *Command) Stdout() io.Reader {
	return c.stdout.Reader()
}

func (c *Command) Stderr() io.Reader {
	return c.stderr.Reader()
}

func (c *Command) Wait() {
	c.wg.Wait()
}

func (c *Command) Done() {
	c.wg.Done()
}

func (c *Command) StdIO(stdin *os.File, stdout *os.File, stderr io.Reader) {
	c.stdin = stdin
	c.stdinMu.Unlock()

	// stdout
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.stdout.Close()
		io.Copy(c.stdout, stdout)
	}()

	// stderr
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.stderr.Close()
		io.Copy(c.stderr, stderr)
	}()
}
