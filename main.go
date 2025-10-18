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
package main

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	//"fmt"
	"encoding/json"
	"flag"
	// blatant copy reeflective/readline :')
	"github.com/Melkor333/oils-readline/internal/term"
	"github.com/muesli/cancelreader"
	"github.com/reeflective/readline"
	// TODO: should be in a module ;)
	"github.com/creack/pty"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	shell       Shell
	historyFile = flag.String("historyFile", "$HOME/.local/share/oils/readline-history.json", "Path to the history file")
)

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
	//StdIO(*os.File, *os.File, *os.File) error
	Run(*Command) error
	Cancel()
	Complete(context.Context, CompletionReq) (*CompletionResult, error)
	Dir() string
}

var commands []*Command
var prompt string
var runningCommands *atomic.Int64

func main() {
	var err error
	var command string
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	commands = make([]*Command, 0)

	shell, err = NewFANOSShell()
	defer shell.Cancel()

	if err != nil {
		log.Fatal(err)
	}

	rl := readline.NewShell()
	if err != nil {
		log.Fatal(err)
	}
	var h string
	if !strings.HasPrefix(*historyFile, "$HOME") {
		h = *historyFile
	} else {
		c, err := os.UserHomeDir()
		if err == nil {
			h = c + "/.local/share/oils/history.json"
		}
	}
	os.MkdirAll(filepath.Dir(h), os.ModePerm)
	rl.History.AddFromFile("history", h)

	// Show that a process is still running...
	rl.Prompt.Primary(func() string {
		if len(commands) > 0 {
			n := runningCommands.Load()
			if n > 0 {
				return ">" + strconv.FormatInt(n, 10) + " " + prompt
			}

		}
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

	runningCommands = new(atomic.Int64)
	runningCommands.Store(0)
	updatePrompt(shell)
	descriptor := int(os.Stdin.Fd())
	state, _ := term.GetState(descriptor)
	term.Restore(descriptor, state)
	defer term.Restore(descriptor, state)
	for {
		// readline
		command, err = rl.Readline()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println(err)
		}

		// TODO: Copy isEmpty() function from reeflective/console
		if command == "" || command == "\n" {
			continue
		}

		state, err := term.MakeRaw(descriptor)
		if err != nil {
			return
		}

		c := NewCommand(command)
		go func() {
			err = shell.Run(c)
			if err != nil {
				log.Println(err)
			}
		}()
		runningCommands.Add(1)
		go func() {
			c.Wait()
			runningCommands.Add(-1)
		}()
		commands = append(commands, c)

		// TODO: capture resizes
		size, _ := pty.GetsizeFull(os.Stdin)
		pty.Setsize(c.Stdin(), size)
		go func() {
			_, err := io.Copy(os.Stdout, c.Stdout())
			if err != nil {
				log.Println(err)
			}
		}()
		go func() {
			_, err := io.Copy(os.Stderr, c.Stderr())
			if err != nil {
				log.Println(err)
			}
		}()

		//TODO: err handling
		r, _ := cancelreader.NewReader(os.Stdin)
		go func() {
			c.wg.Wait()
			r.Cancel()
		}()
		for {
			//_, err := r.Read(buf)
			_, err := io.Copy(c.Stdin(), r)
			if err != nil {
				if err == io.EOF {
					break
				}
				break
				//log.Println(err)
			}
		}

		term.Restore(descriptor, state)
		updatePrompt(shell)
		//out.wg.Done()
		//_, err = rl.Printf(out.Stdout)
		//if err != nil {
		//	log.Println(err)
		//}
	}

}

func updatePrompt(s Shell) {
	command := NewCommand("pwd | sed \"s|$[ENV.HOME]|~|\"")
	shell.Run(command)
	command.wg.Wait()
	buf := new(strings.Builder)
	_, err := io.Copy(buf, command.stdout.Reader())
	if err != nil {
		log.Println(err)
	}
	prompt = strings.ReplaceAll(buf.String(), "\r\n", "") + " $ "
}

var runCancel context.CancelFunc = func() {}

//func HandleRun(w http.ResponseWriter, req *http.Request) {
//	output := Run(req.Body)
//	o, err := json.Marshal(output)
//	if err != nil {
//		log.Println(err)
//	}
//	_, err = w.Write(o)
//	if err != nil {
//		log.Println(err)
//	}
//}

var compCancel context.CancelFunc = func() {}

func HandleComplete(w http.ResponseWriter, req *http.Request) {
	var compReq CompletionReq
	err := json.NewDecoder(req.Body).Decode(&compReq)
	if err != nil {
		log.Println(err)
		return
	}
	if compCancel != nil {
		compCancel()
	}
	var compCtx context.Context

	compCtx, compCancel = context.WithCancel(context.Background())
	defer runCancel()

	out, err := shell.Complete(compCtx, compReq)
	if err != nil {
		log.Println(err)
		return
	}
	o, err := json.Marshal(out)
	if err != nil {
		log.Println(err)
	}
	_, err = w.Write(o)
	if err != nil {
		log.Println(err)
	}
}

func HandleCancel(w http.ResponseWriter, req *http.Request) {
	log.Print("Received cancel")
	runCancel()
}
