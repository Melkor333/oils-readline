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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	//"encoding/json"
	"flag"

	tea "charm.land/bubbletea/v2"

	//"github.com/charmbracelet/lipgloss"
	"charm.land/bubbles/v2/viewport"
	//"github.com/muesli/reflow/wrap"

	//  TODO: Once we have chroma highlighting. (Vibecode chroma highlighter from vim highlighter/treesitter maybe?)
	// editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"log"

	"github.com/Melkor333/oils-readline/fanos"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/creack/pty"

	"strings"
)

var Version = "devel"

var (
	versionFlag = flag.Bool("version", false, "Print version and exit")
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

type ExecType int

const (
	// Run Command blocking in the foreground
	Blocking ExecType = iota
	// Alt mode runs commands themselves in normal mode, but the readline itself is in alt mode!
	AltMode
)

type State int

//const (
//	Reading State = iota
//	Executing
//)

type model struct {
	shell       shell.Shell
	input       textinput.Model
	commandView viewport.Model
	output      string
	Height      int
	Width       int
	highlighter Highlighter
	// state           State
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) View() tea.View {
	// TODO: Show output... ;)
	v := tea.NewView(m.commandView.View() + "\n" + m.input.View())
	return v
}

type CommandOutputMsg string

func CommandOutputToMessage(reader io.Reader) func() tea.Msg {
	return func() tea.Msg {
		buf := new(bytes.Buffer)
		io.Copy(buf, reader)
		return CommandOutputMsg(buf.String())
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// tea messages first
	var cmd tea.Cmd
	log.Print(msg)
	log.Printf("%T", msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			log.Println("ctrl+c!")
			m.input.Reset()
			return m, cmd
		case "ctrl+d":
			if m.input.Value() == "" {
				return m, tea.Quit
			}
			return m, nil
		case "enter":
			// Handle editline messages
			//TODO: Other exec types
			//if m.execType == Blocking
			command := m.input.Value()
			if len(command) == 0 {
				return m, nil
			}

			size, _ := pty.GetsizeFull(os.Stdin)
			cmd, err := m.shell.Command(command, size)
			if err != nil {
				log.Fatal("Can't create new Command!", err)
			}
			// TODO: we also need to capture its output :)

			m.input.Reset()
			return m, tea.Sequence(tea.Batch(cmd.Run, CommandOutputToMessage(cmd.Stdout())))
		}
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case CommandOutputMsg:
		m.commandView.SetContent(string(msg))
		m.commandView.GotoTop()
		m.input.Reset()
		//m.outputs = append(m.outputs, string(msg))
		return m, nil

	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
		//m.input, cmd = m.input.Update(msg)
		m.commandView.SetHeight(msg.Height - lipgloss.Height(m.input.View()))
		m.commandView.SetWidth(msg.Width)
		m.input.SetWidth(m.Width)
		return m, nil

	// Fanos
	// TODO: Should be cast to CommandDone?
	case fanos.CommandDoneMsg:
		return m, nil

	default:
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
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
	}

	ti := textinput.New()
	ti.SetVirtualCursor(true)
	ti.Placeholder = "Enter command"
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(20)
	commandView := viewport.New(
		viewport.WithWidth(20),
		viewport.WithHeight(20),
	)
	commandView.YPosition = 0
	commandView.FillHeight = false
	model := &model{
		input:       ti,
		shell:       s,
		commandView: commandView,
	}
	defer model.shell.Cancel()

	p := tea.NewProgram(model)
	go func() {
		model.shell.Wait()
		p.Send(tea.Quit)
	}()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error Running Oils-Readline: %v", err)
		os.Exit(1)
	}
}

func getPrompt(shell shell.Shell) string {
	// TODO: this should also work in osh :D
	command, err := shell.Command("pwd | sed \"s|$[ENV.HOME]|~|\"", &pty.Winsize{1, 100, 5, 5})
	if err != nil {
		return ""
	}
	buf := new(bytes.Buffer)
	command.SetStdout(buf)
	command.SetStdin(bytes.NewBuffer(nil))
	command.Run() // we don't care about the message

	return strings.ReplaceAll(buf.String(), "\r\n", "") + " $ "
}

var runCancel context.CancelFunc = func() {}
