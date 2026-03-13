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

	//"github.com/charmbracelet/lipgloss"
	//"github.com/muesli/reflow/wrap"

	//  TODO: Once we have chroma highlighting. (Vibecode chroma highlighter from vim highlighter/treesitter maybe?)
	// editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
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

type CommandOutputErrorMsg error
type CommandDoneMsg shell.Command

var (
	promptStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	brightGreenGut = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // bright green
	darkGreenGut   = lipgloss.NewStyle().Foreground(lipgloss.Color("22")) // dark green
)

type model struct {
	shell           shell.Shell
	input           textinput.Model
	commands        []shell.Command
	viewports       []viewport.Model
	focusedViewport int // -1 = input focused, 0+ = viewport index
	Height          int
	Width           int
	highlighter     Highlighter
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) NewViewport(prompt string) {
	commandView := viewport.New(
		viewport.WithWidth(20),
		//viewport.WithHeight(20),
	)
	commandView.YPosition = 0
	commandView.FillHeight = false
	m.viewports = append(m.viewports, commandView)
}

func (m *model) View() tea.View {
	// TODO: Show output... ;)
	var strs []string
	log.Print(m.commands)
	for i, command := range m.commands {
		log.Print(command)
		gutStyle := brightGreenGut
		if m.focusedViewport == i {
			gutStyle = darkGreenGut
		}
		m.viewports[i].LeftGutterFunc = func(ctx viewport.GutterContext) string {
			return gutStyle.Render("│")
		}
		content := command.CommandLine() + "\n" + strings.Trim(command.Stdout(), "\r\n")
		m.viewports[i].SetHeight(lipgloss.Height(content))
		m.viewports[i].SetWidth(m.Width)
		m.viewports[i].SetContent(content)
		strs = append(strs, m.viewports[i].View())
	}
	strs = append(strs, strings.Trim(m.input.View(), "\r\n"))
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, strs...))
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
				m.shell.Cancel()
				return m, tea.Quit
			}
			return m, nil
		case "ctrl+space":
			if m.focusedViewport >= 0 {
				m.focusedViewport = -1
				return m, m.input.Focus()
			} else if len(m.viewports) > 0 {
				m.input.Blur()
				if m.focusedViewport == -1 {
					m.focusedViewport = len(m.viewports) - 1
				}
				return m, nil
			}
			return m, nil
		case "esc":
			if m.focusedViewport >= 0 {
				m.focusedViewport = -1
				return m, m.input.Focus()
			}
		case "up", "k":
			if m.focusedViewport >= 0 && len(m.viewports) > 0 {
				if m.focusedViewport > 0 {
					m.focusedViewport--
				}
				return m, nil
			}
		case "down", "j":
			if m.focusedViewport >= 0 && len(m.viewports) > 0 {
				if m.focusedViewport < len(m.viewports)-1 {
					m.focusedViewport++
				}
				return m, nil
			}
		case "enter":
			if m.focusedViewport >= 0 {
				return m, nil
			}
			// Handle editline messages
			//TODO: Other exec types
			//if m.execType == Blocking
			command := m.input.Value()
			if len(command) == 0 {
				return m, nil
			}

			size, _ := pty.GetsizeFull(os.Stdin)
			cmd, err := m.shell.Command(command, size)
			m.commands = append(m.commands, cmd)
			m.NewViewport(command)
			if err != nil {
				log.Fatal("Can't create new Command!", err)
			}
			// TODO: we also need to capture its output :)

			m.input.Reset()
			m.input.Blur()
			return m, func() tea.Msg { cmd.Run(); return CommandDoneMsg(cmd) }
		}
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		log.Print("Resizing")
		m.Height = msg.Height
		m.Width = msg.Width
		//m.input, cmd = m.input.Update(msg)
		//m.commandView.SetHeight(msg.Height - lipgloss.Height(m.input.View()))
		// TODO: Also set the height (Also set the height?)
		for _, el := range m.viewports {
			el.SetWidth(msg.Width)
		}
		m.input.SetWidth(m.Width)
		return m, nil

	// Fanos
	// TODO: Should be cast to CommandDone?
	case CommandDoneMsg:
		log.Print("Command done!")
		m.input.Prompt = getPrompt(m.shell)
		return m, m.input.Focus()

	// TODO: Should be cast to CommandDone?
	case tea.EnvMsg:
		log.Print("Got env")
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
	} else {
		log.SetOutput(io.Discard)
	}

	ti := textinput.New()
	ti.SetVirtualCursor(true)
	ti.Placeholder = "Enter command"
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(20)
	ti.Prompt = getPrompt(s)
	model := &model{
		input:           ti,
		shell:           s,
		focusedViewport: -1,
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
	log.Print("Getting prompt")
	command, err := shell.Command("pwd | sed \"s|$[ENV.HOME]|~|\"", &pty.Winsize{1, 100, 5, 5})
	if err != nil {
		return ""
	}
	command.Run() // we don't care about the message
	command.Wait()
	log.Print("Got prompt")

	return promptStyle.Render(strings.ReplaceAll(command.Stdout(), "\n", "") + " $ ")
}
