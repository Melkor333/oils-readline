package main

import (
	"context"
	"errors"
	"log"
	//TODO: no unnecessary depth
	"github.com/Melkor333/oils-readline/internal/term"
	tea "github.com/charmbracelet/bubbletea"
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
	return command

}

func (c *Command) Stdin() (*os.File, error) {
	if c.stdin == nil {
		return nil, errors.New("Need to call command.StdIO first!")
	}

	return c.stdin, nil
}

func (c *Command) Stdout() io.Reader {
	return c.stdout.Reader()
}

func (c *Command) Stderr() io.Reader {
	return c.stderr.Reader()
}

// TODO: A waitgroup is probably overkill :D
func (c *Command) Wait() {
	c.wg.Wait()
}

func (c *Command) Done() {
	c.wg.Done()
}

func (c *Command) StdIO(stdin *os.File, stdout *os.File, stderr io.Reader) {
	// stdout
	c.stdin = stdin
	c.wg.Add(1)
	go func() {
		defer c.stdout.Close()
		defer c.wg.Done()
		io.Copy(c.stdout, stdout)
	}()

	// stderr
	c.wg.Add(1)
	go func() {
		defer c.stderr.Close()
		defer c.wg.Done()
		io.Copy(c.stderr, stderr)
	}()
}

// All the Bubbletea Stuff from here on
type CommandDoneMsg *Command

type LegacyCmd struct {
	shell   Shell
	command *Command
	tty     *os.File
	stderr  *os.File
}

// tty is the slave for the shell
// stderrIn is a separate pipe for the shell (stderrOut already stored)
// TODO: Should this directly be done inside of `StdIO` to simplify?
func SetupIO(c *Command, s Shell, size *pty.Winsize) (tty *os.File, stderrIn *os.File, err error) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}
	stderrOut, stderrIn, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	pty.Setsize(ptmx, size)
	c.StdIO(
		ptmx,
		ptmx,
		stderrOut)

	return tty, stderrIn, err
}

func NewLegacyCmd(command string, shell Shell, size *pty.Winsize) (*LegacyCmd, tea.ExecCallback, error) {
	c := NewCommand(command)
	tty, stderrIn, err := SetupIO(c, shell, size)
	if err != nil {
		return nil, nil, err
	}

	return &LegacyCmd{shell, c, tty, stderrIn}, func(error) tea.Msg { return CommandDoneMsg(c) }, nil

}

func (c *LegacyCmd) Run() error {
	// Cleanup the "help" line
	err := c.shell.Run(c.command.CommandLine, c.tty, c.tty, c.stderr)
	c.command.Done()
	if err != nil {
		return err
	}
	//	term.Restore(descriptor, state)

	// TODO: Capture and return exit code :)
	return nil
}

func (c *LegacyCmd) SetStdin(r io.Reader) {
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

	stdin, err := c.command.Stdin()
	rr, _ := cancelreader.NewReader(r)
	if err != nil {
		log.Println("command StdIO wasn't set!")
	}

	go func() {
		c.command.Wait()
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

func (c *LegacyCmd) SetStdout(w io.Writer) {
	go func() {
		_, err := io.Copy(w, c.command.Stdout())
		if err != nil {
			log.Println(err)
		}
	}()
}

func (c *LegacyCmd) SetStderr(w io.Writer) {
	go func() {
		_, err := io.Copy(w, c.command.Stderr())
		if err != nil {
			log.Println(err)
		}
	}()
}

func (c *LegacyCmd) Error() string {
	return "Capturing Exit code is not yet implemented"
}
