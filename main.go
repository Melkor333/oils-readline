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
	"bufio"
	"os"
	"context"
	"strconv"
	"sync/atomic"
	//"fmt"
	"encoding/json"
	"flag"
	"github.com/reeflective/readline"
	"io"
	"log"
	"net/http"
	"strings"
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
	//StdIO(*os.File, *os.File, *os.File) error
	Run(*Command) error
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
	if err != nil {
		log.Fatal(err)
	}

	rl := readline.NewShell()
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
	for {
		// readline
		command, err = rl.Readline()
		if err != nil {
			log.Println(err)
		}

		// TODO: Copy isEmpty() function from reeflective/console
		if command == "" || command == "\n" {
			continue
		}

		// FANOS
		c := NewCommand(command)
		go func() {
			err = shell.Run(c)
			if err != nil {
				log.Println(err)
			}
		}()
		runningCommands.Add(1)
		go func() {
			c.wg.Wait()
			runningCommands.Add(-1)
		}()
		commands = append(commands, c)

		go func() {
			r := bufio.NewReader(c.stdout.Reader())
			for {
				l, err := r.ReadByte()
				os.Stdout.Write([]byte{l})
				//rl.PrintTransientf(l)
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Println(err)
				}
			}
			updatePrompt(shell)
		}()

		go func() {
			r := bufio.NewReader(c.stderr.Reader())
			for {
				b, err := r.ReadByte()
				os.Stdout.Write([]byte{b})
				//rl.PrintTransientf(l)
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Println(err)
				}
			}
			updatePrompt(shell)
		}()

		r := bufio.NewReader(os.Stdin)
		for {
			b, err := r.ReadByte()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println(err)
			}
			_, err = c.stdin.Write([]byte{b})
			if err != nil {
				//log.Println(err)
				break
			}
		}
		c.wg.Wait()

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
	prompt = strings.Replace(buf.String(), "\r\n", "", -1) + " $ "
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
