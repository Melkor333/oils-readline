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
	"os"
	"sync/atomic"

	//"encoding/json"
	"flag"

	"github.com/chalk-ai/bubbline/editline"
	tea "github.com/charmbracelet/bubbletea"

	//"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/viewport"
	//"github.com/muesli/reflow/wrap"
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
	shell           shell.Shell
	rl              *editline.Model
	commandView     viewport.Model
	prompt          string
	commands        []tea.ExecCommand
	lastCommand     tea.ExecCommand
	runningCommands *atomic.Int64
	Height          int
	Width           int
	execType        ExecType
	lastLines       int
	highlighter     Highlighter
	// state           State
}

func newModel(e ExecType) model {
	s, err := fanos.New()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: resize?!
	rl := editline.New(80, 20)
	rl.AutoComplete = s.Complete
	//func(entireInput [][]rune, line, col int) (msg string, comp editline.Completions) {
	//	log.Println(entireInput, line, col)
	//	log.Println("\n\n\n\n\n")
	//	return "", editline.SimpleWordsCompletion([]string{"hello world", "goobye world"}, "hello", 3, line, col)
	//}

	var _ shell.Command = &fanos.Command{}
	rl.Prompt = getPrompt(s)
	rl.Highlighter = NewHighlighter().Highlight
	rl.Reset()

	runningCommands := new(atomic.Int64)
	runningCommands.Store(0)

	return model{
		lastLines:       0,
		shell:           s,
		rl:              rl,
		runningCommands: runningCommands,
		execType:        e,
	}
}

func (m model) Init() tea.Cmd {
	if m.execType == AltMode {
		m.execType = AltMode
		return tea.Batch(m.rl.Init(), tea.EnterAltScreen)
	}
	return m.rl.Init()
}

func (m model) View() string {
	return m.rl.View()
	// Use this with more modes:
	//if m.execType == Blocking || m.execType == AltMode {
	//return m.rl.View()
	//}

	//s += lipgloss.JoinHorizontal(lipgloss.Top, focusedModelStyle.Render(fmt.Sprintf("%4s", m.timer.View())), modelStyle.Render(m.spinner.View()))
	//s += helpStyle.Render(fmt.Sprintf("\ntab: focus next • n: new %s • q: exit\n", model))
}

// Update the readline and properly store the returned model
// This requires some funky magic, because rl.Update() only returns the generic tea.Model interface
func (m model) UpdateReadline(msg tea.Msg) (model, tea.Cmd) {
	rl, cmd := m.rl.Update(msg)
	// They return an interface...
	m.rl = rl.(*editline.Model)
	m.lastLines = strings.Count(m.rl.View(), "\n")
	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// tea messages first
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.rl.Reset()
			m, cmd = m.UpdateReadline(msg)
			return m, cmd
		case "ctrl+d":
			if m.rl.Value() == "" {
				return m, tea.Quit
			}
		case "ctrl+ ":
			if m.execType == Blocking {
				m.execType = AltMode
				return m, tea.EnterAltScreen
			} else {
				m.execType = Blocking
				return m, tea.ExitAltScreen
			}
		}

	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
		m.rl.SetSize(m.Width, m.Height-20)
		m.rl.Reset()

	// Handle editline messages
	case editline.InputCompleteMsg:
		//TODO: Other exec types
		//if m.execType == Blocking
		m, _ = m.UpdateReadline(msg)
		command := m.rl.Value()
		m.rl.Reset()

		size, _ := pty.GetsizeFull(os.Stdin)
		exec, err := m.shell.Command(command, size)
		if err != nil {
			log.Fatal("Can't create new Command!", err)
		}
		//m.state = Executing
		m.rl.Blur()
		m.rl.AddHistoryEntry(command)

		// returns nil
		if m.execType == AltMode {
			return m, tea.Sequence(tea.ExitAltScreen, tea.Exec(exec, fanos.CommandCallback(exec)), tea.EnterAltScreen)
		}

		// TODO: The following stuff should be printed by the `cmd` before executing the process :)
		// To make sure we don't have a concurrency issue with an in-between `View`

		// https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797
		// Doesn't work with tea.Printf :(
		// cleanup last readline
		//for range m.lastLines {
		//	fmt.Print("\033[1A\033[2K")
		//}
		////fmt.Printf("\033[1F")
		//// Print blured with short help
		//fmt.Print(model.View())
		//// Go to beginning of help line, remove until end of screen
		//fmt.Print("\r\033[0J")
		return m, tea.Exec(exec, fanos.CommandCallback(exec))

	// Fanos
	// TODO: Should be cast to CommandDone?
	case fanos.CommandDoneMsg:
		//m.state = Executed
		m.rl.Prompt = getPrompt(m.shell)
		m.rl.Focus()
		m.rl.Reset()
		m, cmd = m.UpdateReadline(msg)
		// TODO: history!
		//m.commands = append(m.commands, msg)
		//m.lastCommand = msg
		//buf := wrap.NewWriter(m.Width)

		//_, err := io.Copy(buf, m.lastCommand.Stdout())
		//if err != nil {
		//	log.Println(err)
		//}
		//m.commandView = viewport.New(m.Width, m.Height-20)
		//m.commandView.SetContent(buf.String())
		return m, cmd
	default:
	}

	// pass to readline if nothing else
	m, cmd = m.UpdateReadline(msg)
	m.lastLines = strings.Count(m.rl.View(), "\n")

	return m, cmd
}

func main() {
	flag.Parse()

	if versionFlag != nil && *versionFlag {
		fmt.Printf("Oils-Readline version: %s\n", Version)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	model := newModel(Blocking)
	defer model.shell.Cancel()

	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Printf("Error Running Oils-Readline: %v", err)
		os.Exit(1)
	}
}

//TODO: Add history.json file
//var h string
//if !strings.HasPrefix(*historyFile, "$HOME") {
//	h = *historyFile
//} else {
//	c, err := os.UserHomeDir()
//	if err == nil {
//		h = c + "/.local/share/oils/history.json"
//	}
//}
//os.MkdirAll(filepath.Dir(h), os.ModePerm)
//rl.History.AddFromFile("history", h)

func getPrompt(shell shell.Shell) string {
	// TODO: this should also work in osh :D
	command, err := shell.Command("pwd | sed \"s|$[ENV.HOME]|~|\"", &pty.Winsize{1, 100, 5, 5})
	if err != nil {
		return ""
	}
	buf := new(bytes.Buffer)
	command.SetStdout(buf)
	command.SetStdin(bytes.NewBuffer(nil))
	err = command.Run() // TODO: fixme, and the buffer thingy above me as well
	//defer command.Cancel()
	//if err != nil {
	//	log.Println(err)
	//	return ""
	//}
	//_, err = io.Copy(buf, command.Stdout())
	if err != nil {
		log.Println(err)
	}
	return strings.ReplaceAll(buf.String(), "\r\n", "") + " $ "
}

var runCancel context.CancelFunc = func() {}
