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
	"context"
	//"fmt"
	"encoding/json"
	"flag"
	"strings"
	//"github.com/buildkite/terminal-to-html/v3"
	"github.com/reeflective/readline"
	"log"
	"net/http"
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

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error

	commands = make([]*Command, 0)

	shell, err = NewFANOSShell()
	if err != nil {
		log.Fatal(err)
	}

	//http.HandleFunc("/run", HandleRun)
	////http.HandleFunc("/complete", HandleComplete)
	//http.HandleFunc("/cancel", HandleCancel)
	//http.HandleFunc("/status", HandleStatus)
	//http.Handle("/", http.FileServer(http.Dir("./web")))
	//log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
	var command string

	shell, _ = NewFANOSShell()
	rl := readline.NewShell()
	// Show that a process is still running...
	rl.Prompt.Primary(func() string {
		if len(commands) > 0 {
			if commands[len(commands)-1].Status == "" {
				return ">1" + prompt

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
	updatePrompt(&shell)
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
		go shell.Run(c)
		commands = append(commands, c)

		go func() {
			//out.wg.Wait()
			rl.Printf(commands[len(commands)-1].Stdout)
			updatePrompt(&shell)
		}()
		
		//out.wg.Done()
		//_, err = rl.Printf(out.Stdout)
		//if err != nil {
		//	log.Println(err)
		//}
	}

}

func updatePrompt(s *Shell) {
		command := NewCommand("pwd | sed \"s|$[ENV.HOME]|~|\"")
		shell.Run(command)
		prompt = strings.TrimSuffix(command.Stdout, "\n") + " $ "

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
