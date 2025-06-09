// Prototype Shell UI, for the Go sh package and external shell interpreters.
//
// Each command is allocated a pty for stdout and a (named) pipe for stderr.
// An anonymous pipe would be better, but would require fd passing.
//
// This doesn't work for every command:
//   - If `less` can't open `/dev/tty`, it READS from stderr! Not stdin.
//     (because stdin might be the read end of a pipe)
//     alias less="less 2<&0" works, but wouldn't work in a pipe.
//   - sudo reads from /dev/tty by default, but you can tell it to use stdin
//     with `sudo -S`. alias sudo="sudo -S" works.
//
// Apparently according to POSIX, stderr is supposed to be open for both
// reading and writing...
//
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/creack/pty"
	"github.com/reeflective/readline"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
)

var (
	fifo  = flag.Bool("fifo", true, "Use named fifo instead of anonymous pipe")
	debug = flag.Bool("debug", false, "Watch and live reload typescript")
)

var shell Shell

type CompletionReq struct {
	Text string
	Pos  int
}

type Completion struct {
	Label        string `json:"label"`
	DisplayLabel string `json:"displayLabel,omitempty"`
	Detail       string `json:"detail,omitempty"`
	Info         string `json:"info,omitempty"`
	Apply        string `json:"apply,omitempty"`
	Type         string `json:"type,omitempty"`
	Boost        int    `json:"boost,omitempty"`
	Section      string `json:"section,omitempty"`
}

type CompletionResult struct {
	From    int          `json:"from"`
	To      int          `json:"to,omitempty"`
	Options []Completion `json:"options"`
}

type Shell interface {
	StdIO(*os.File, *os.File, *os.File) error
	Run(context.Context, string) error
	Complete(context.Context, CompletionReq) (*CompletionResult, error)
	Dir() string
}

type CommandOut struct {
	Dir                  string
	Stdout, Stderr       string
	Err                  error
}

var prompt string

func main() {
	flag.Parse()
	//log.SetFlags(log.LstdFlags | log.Lshortfile)

	var command string
	var err error
	var out, pwd CommandOut

	shell, _ = NewFANOSShell()
	rl := readline.NewShell()
	rl.Prompt.Primary(func() string {
		return prompt
	})
	// TODOS:
	// defaults for inputrc from console
	// handle interrupts like EOF!
	// put output stderr in the hints?
	// Autocomplete
	// Highlight bash -> pygments (if `osh...?`
	// LONGTERM:
	// go back to recent command/cycle through/search, etc.
	for {
		// Update prompt
		pwd, err = Run("pwd | sed \"s|$[ENV.HOME]|~|\"")
		if err != nil {
			log.Println(err)
		}
		prompt = strings.TrimSuffix(pwd.Stdout, "\n\n") + " $ "

		// readline
		command, err = rl.Readline()
		if err != nil {
			log.Println(err)
		}
		if command == "" {
			continue
		}

		// FANOS
		out, err = Run(command)
		if err != nil {
			log.Println(err)
		}

		fmt.Println(out.Stdout)
		//_, err = rl.Printf(out.Stdout)
		//if err != nil {
		//	log.Println(err)
		//}
	}

}

var runMu sync.Mutex
var runCancel context.CancelFunc = func() {}

func Run(command string) (CommandOut, error) {
	var output CommandOut
	runMu.Lock()
	defer runMu.Unlock()
	var stdout, stderr bytes.Buffer
	var runCtx context.Context

	runCtx, runCancel = context.WithCancel(context.Background())
	defer runCancel()

	ptmx, pts, err := pty.Open()
	if err != nil {
		log.Println(err)
		return output, err
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}()
	go io.Copy(&stdout, ptmx)

	var pipe *os.File
	if *fifo {
		dir := os.TempDir()
		pipeName := path.Join(dir, "errpipe")
		syscall.Mkfifo(pipeName, 0600)
		// If you open only the read side, then you need to open with O_NONBLOCK
		// and clear that flag after opening.
		//	pipe, err := os.OpenFile(pipeName, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
		pipe, err = os.OpenFile(pipeName, os.O_RDWR, 0600)
		if err != nil {
			log.Println(err)
			return output, err
		}
		defer func() {
			pipe.Close()
			os.Remove(pipeName)
			os.Remove(dir)
		}()
		go io.Copy(&stderr, pipe)
	} else {
		var rdPipe *os.File
		rdPipe, pipe, err = os.Pipe()
		if err != nil {
			log.Println(err)
			return output, err
		}
		go func() {
			io.Copy(&stderr, rdPipe)
			rdPipe.Close()
			pipe.Close()
		}()
	}

	// Reset stdio of runner before running a new command
	err = shell.StdIO(nil, pts, pipe)
	if err != nil {
		log.Println(err)
		return output, err
	}
	err = shell.Run(runCtx, command)
	if err != nil {
		log.Println(err)
	}

	output.Dir = shell.Dir()
	output.Stdout = stdout.String()
	output.Stderr = stderr.String()
	output.Err = err
	return output, nil
}
