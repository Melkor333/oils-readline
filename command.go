package main

import (
	"context"
	"errors"
	"log"
	//TODO: no unnecessary depth
	"github.com/Melkor333/oils-readline/internal/term"
	"github.com/creack/pty"
	"github.com/mcpherrinm/multireader"
	"github.com/muesli/cancelreader"
	"io"
	"os"
	"sync"
)

type Command struct {
	shell          Shell
	CommandLine    string
	err            error
	ctx            context.Context
	Cancel         context.CancelFunc
	stdinMu        sync.Mutex
	stdin          *os.File
	stdout, stderr *multireader.Buffer
	// For the Client
	tty      *os.File
	stderrIn *os.File
	wg       *sync.WaitGroup
	lock     *sync.Mutex
}

// chain -> to which chain to add the command? be here?
func NewCommand(commandLine string, shell Shell, size *pty.Winsize) (*Command, error) {
	var err error
	var c *Command = new(Command)
	// check errors
	c.CommandLine = commandLine
	c.shell = shell
	c.ctx, c.Cancel = context.WithCancel(context.Background())
	c.wg = new(sync.WaitGroup)
	c.wg.Add(1)
	// TODO: somethingsomething TeeReader...?
	c.stdout = multireader.New()
	c.stderr = multireader.New()
	// to be unlocked when stdin exists.
	c.stdinMu.Lock()

	// get a PTY Master & Slave
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}
	c.tty = tty
	pty.Setsize(ptmx, size)

	// Stdin has to be set by writer
	c.stdin = ptmx

	// Stdout
	c.wg.Add(1)
	go func() {
		defer c.stdout.Close()
		defer c.wg.Done()
		io.Copy(c.stdout, ptmx)
	}()

	// Stderr
	stderrOut, stderrIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	c.stderrIn = stderrIn
	c.wg.Add(1)
	go func() {
		defer c.stderr.Close()
		defer c.wg.Done()
		io.Copy(c.stderr, stderrOut)
	}()

	return c, nil

}

// Get a Writer PTY to write into
func (c *Command) Stdin() (*os.File, error) {
	if c.stdin == nil {
		return nil, errors.New("Need to call command.StdIO first!")
	}

	return c.stdin, nil
}

// Get a reader to read STDOUT, even AFTER the command finished
func (c *Command) Stdout() io.Reader {
	return c.stdout.Reader()
}

// Get a reader to read STDERR, even AFTER the command finished
func (c *Command) Stderr() io.Reader {
	return c.stderr.Reader()
}

// Wait for command to finish
func (c *Command) Wait() {
	c.wg.Wait()
}

// Mark command as done
func (c *Command) done() {
	c.wg.Done()
}

// Execute Command
func (c *Command) Run() error {
	// Cleanup the "help" line
	err := c.shell.Run(c.CommandLine, c.tty, c.tty, c.stderrIn)
	c.done()
	if err != nil {
		return err
	}
	//	term.Restore(descriptor, state)

	// TODO: Capture and return exit code :)
	return nil
}

func (c *Command) SetStdin(r io.Reader) {
	// IF the reader is a File (and thus probably a Terminal) we need to put it into Raw mode during execution!
	file, ok := r.(*os.File)
	var oldState *term.State
	var descriptor int
	var err error
	if ok {
		descriptor := int(file.Fd())
		oldState, err = term.MakeRaw(descriptor)
		// If it's a FIFO, we might not be able to make it RAW :)
		if err != nil {
			ok = false
		}
	}

	stdin, err := c.Stdin()
	rr, _ := cancelreader.NewReader(r)
	if err != nil {
		log.Println("command StdIO wasn't set!")
	}

	go func() {
		c.Wait()
		rr.Cancel()
	}()

	go func() {
		if ok {
			defer term.Restore(descriptor, oldState)
		}
		for {
			//_, err := r.Read(buf)
			_, err := io.Copy(stdin, rr)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println(err)
				break
			}
		}
		io.Copy(stdin, rr)
	}()
}

// These are to set BEFORE running the command
func (c *Command) SetStdout(w io.Writer) {
	go func() {
		_, err := io.Copy(w, c.stdout.Reader())
		if err != nil {
			log.Println(err)
		}
	}()
}

// These are to set BEFORE running the command
func (c *Command) SetStderr(w io.Writer) {
	go func() {
		_, err := io.Copy(w, c.stderr.Reader())
		if err != nil {
			log.Println(err)
		}
	}()
}

func (c *Command) Error() string {
	return "Capturing Exit code is not yet implemented"
}
