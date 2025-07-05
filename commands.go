package main

import (
	"context"
	//TODO: no unnecessary depth
	"github.com/mcpherrinm/multireader"
	"io"
	"sync"
)

type Command struct {
	shell       Shell
	CommandLine string
	// TODO: stdout/err should always just be a (buffered) reader!
	err error
	//Id                     int
	ctx            context.Context
	cancel         context.CancelFunc
	stdin          io.Writer
	stdout, stderr *multireader.Buffer
	wg             *sync.WaitGroup
	lock           *sync.Mutex
}

// chain -> to which chain to add the command? be here?
func NewCommand(commandLine string) *Command {
	var command *Command = new(Command)
	// check errors
	command.CommandLine = commandLine
	command.wg = new(sync.WaitGroup)
	command.wg.Add(1)
	// TODO: somethingsomething TeeReader...?
	command.stdout = multireader.New()
	command.stderr = multireader.New()
	// TODO: Create a History object?
	//commMu.Lock()
	// TODO: UUID? even necessary?
	//command.Id = len(commands)
	//commMu.Unlock()
	return command

}
func (c *Command) StdIO(stdin io.Writer, stdout io.Reader, stderr io.Reader) {
	c.stdin = stdin

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

// TODO: create a TeeReader every time!
//func (c *Command) Stdout () *io.Reader {
//
//}
