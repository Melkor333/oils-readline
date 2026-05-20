// Prototype Shell GUI using FANOS
//
// Each command is allocated a pty for stdin/stdout and a (named) pipe for stderr.
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
	"fmt"
	"io"
	"os"

	//"encoding/json"
	"flag"

	tea "charm.land/bubbletea/v2"

	//  TODO: Once we have chroma highlighting. (Vibecode chroma highlighter from vim highlighter/treesitter maybe?)
	// editor "github.com/ionut-t/goeditor/adapter-bubbletea"

	"log"

	"github.com/Melkor333/oils-readline/fanos"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/Melkor333/oils-readline/tiling"
	"github.com/creack/pty"
)

var Version = "devel"

var (
	versionFlag = flag.Bool("version", false, "Print version and exit")
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

type ExecType int

const (
	// Run Command blocking in the foreground
	Blocking ExecType = iota
	// Alt mode runs commands themselves in normal mode, but the readline itself is in alt mode!
	AltMode
)

type State int

type model struct {
	shell  shell.Shell
	Height int
	Width  int
	layout *tiling.Model
	//highlighter Highlighter
	program *tea.Program
	focus   int // 1 = history, 0 = prompt
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) View() tea.View {
	v := m.layout.View()
	v.AltScreen = true
	return v
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// tea messages first
	//log.Print(msg)
	//log.Printf("%T", msg)
	//log.Printf("focus: %v", m.focus)

	switch msg := msg.(type) {
	case CommandEnteredMsg:
		// Run a command
		// TODO: make each `shell` a widget as well somehow?!
		command := msg.Text
		if len(command) == 0 {
			break // We still let widgets deal with it!
		}

		size, _ := pty.GetsizeFull(os.Stdin)
		cmd, err := m.shell.Command(command, size)
		if err != nil {
			log.Fatal("Can't create new Command!", err)
		}

		cmd.SetOnStdout(func() { m.program.Send(shell.StdoutMsg{Cmd: cmd}) })
		cmd.SetOnStderr(func() { m.program.Send(shell.StderrMsg{Cmd: cmd}) })

		log.Print("Running command")
		return m, tea.Batch(
			func() tea.Msg { return shell.NewCommandMsg{Cmd: cmd} },
			func() tea.Msg { cmd.Run(); return shell.CommandDoneMsg{Cmd: cmd} },
		)

	case tea.WindowSizeMsg:
		log.Print("Resizing")
		m.Height = msg.Height
		m.Width = msg.Width

	// TODO: Should be cast to CommandDone?
	case tea.EnvMsg:
		log.Print("Got env")
	}

	l, cmd := m.layout.Update(msg)
	m.layout = l.(*tiling.Model)
	return m, cmd
}

func main() {
	flag.Parse()

	if versionFlag != nil && *versionFlag {
		fmt.Printf("Oils-Readline version: %s\n", Version)
		os.Exit(0)
	}

	// TODO: Discard logs when debug is off
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s, err := fanos.New()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	} else {
		log.SetOutput(io.Discard)
	}

	model := &model{
		shell: s,
		focus: 1,
	}
	model.layout = tiling.NewModel(newBasicPrompt(s), newHistory())
	defer model.shell.Cancel()

	p := tea.NewProgram(model)
	model.program = p
	go func() {
		model.shell.Wait()
		p.Send(tea.Quit)
	}()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error Running Oils-Readline: %v", err)
		os.Exit(1)
	}
}
