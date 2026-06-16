package main

import (
	"fmt"
	"log"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/creack/pty"
	"github.com/muesli/reflow/wrap"

	"github.com/Melkor333/oils-readline/shell"
)

type StdoutViewer struct {
	command         shell.Command
	view            viewport.Model
	targetIndex     int
	currentIndex    int
	showStderr      bool
	interactiveMode bool
	exitMenuSelect  menuSelection
	Width           int
	Height          int
}

func (h *StdoutViewer) commandRunning() bool {
	return h.command != nil && (h.command.State() == shell.Queued || h.command.State() == shell.Started)
}

var (
	activeColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // bright green
	inactiveColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("22")) // dark green
	highlightColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
)

type menuSelection int

const (
	menuSelectHidden menuSelection = iota + 2
	menuSelectSendctrlc
	menuSelectExit
	menuSelectCancel
)

func newStdoutViewer() *StdoutViewer {
	return &StdoutViewer{targetIndex: -1, currentIndex: -1, exitMenuSelect: menuSelectHidden}
}

func newStderrViewer() *StdoutViewer {
	return &StdoutViewer{targetIndex: -1, currentIndex: -1, showStderr: true, exitMenuSelect: menuSelectHidden}
}

func (h *StdoutViewer) Init() tea.Cmd {
	return nil
}

func (h *StdoutViewer) WriteStdin(b []byte) (int, error) {
	if h.command == nil {
		return 0, fmt.Errorf("no command")
	}
	return h.command.Stdin().Write(b)
}

func (h *StdoutViewer) IsInteractive() bool {
	return h.interactiveMode
}

func (h *StdoutViewer) requestHistoryEntry(index int) tea.Cmd {
	return func() tea.Msg {
		return shell.RequestHistoryEntryMsg{Index: index}
	}
}

func (h *StdoutViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if h.interactiveMode {
			switch msg.String() {
			case "enter":
				switch h.exitMenuSelect {
				case menuSelectHidden:
					return h, func() tea.Msg {
						h.WriteStdin([]byte{'\n'})
						return nil
					}
				case menuSelectSendctrlc:
					h.exitMenuSelect = menuSelectHidden
					return h, func() tea.Msg {
						h.WriteStdin([]byte{'\x03'})
						return nil
					}
				case menuSelectExit:
					h.interactiveMode = false
					h.exitMenuSelect = menuSelectHidden
					return h, ReleaseCapture()
				case menuSelectCancel:
					h.exitMenuSelect = menuSelectHidden
					return h, nil
				}
				return h, nil
			case "k":
				if h.exitMenuSelect != menuSelectHidden {
					if h.exitMenuSelect == menuSelectSendctrlc {
						return h, nil
					}
					h.exitMenuSelect--
					return h, nil
				}
			case "j":
				if h.exitMenuSelect != menuSelectHidden {
					if h.exitMenuSelect == menuSelectCancel {
						return h, nil
					}
					h.exitMenuSelect += 1
					return h, nil
				}
			case "ctrl+c":
				if h.exitMenuSelect == menuSelectHidden {
					h.exitMenuSelect = menuSelectSendctrlc
					return h, nil
				}
			}
			var key rune
			if msg.ShiftedCode != key {
				key = msg.ShiftedCode
			} else {
				key = msg.Code
			}
			return h, func() tea.Msg {
				h.WriteStdin([]byte{byte(key)})
				return nil
			}
		}
		switch msg.String() {
		case "enter":
			if h.commandRunning() {
				h.interactiveMode = true
				if h.command != nil {
					h.command.Resize(&pty.Winsize{
						Cols: uint16(h.Width),
						Rows: uint16(h.Height),
					})
				}
				return h, RequestCapture()
			}
		case "h":
			if h.targetIndex >= 0 {
				h.targetIndex -= 1
				return h, h.requestHistoryEntry(h.targetIndex)
			}
			return h, h.requestHistoryEntry(h.currentIndex - 1)
		case "l":
			if h.targetIndex >= 0 {
				h.targetIndex += 1
				return h, h.requestHistoryEntry(h.targetIndex)
			}
			return h, h.requestHistoryEntry(h.currentIndex + 1)
		case "s":
			if h.targetIndex == -1 {
				h.targetIndex = h.currentIndex
			} else {
				h.targetIndex = -1
			}
			return h, nil
		case "e":
			h.showStderr = !h.showStderr
			h.updateContent()
			return h, nil
		default:
			var cmd tea.Cmd
			h.view, cmd = h.view.Update(msg)
			return h, cmd
		}
	case shell.CommandMsg:
		h.interactiveMode = false
		h.exitMenuSelect = menuSelectHidden
		if h.targetIndex < 0 {
			h.command = msg.Cmd
			h.currentIndex = -1
			h.view = viewport.New(
				viewport.WithWidth(h.Width),
				viewport.WithHeight(max(0, h.Height-1)),
			)
			h.view.FillHeight = false
			h.updateContent()
		}
		return h, ReleaseCapture()

	case shell.CommandDoneMsg:
		if h.interactiveMode {
			h.interactiveMode = false
			h.exitMenuSelect = menuSelectHidden
		}
		return h, ReleaseCapture()

	case shell.HistoryEntryMsg:
		log.Printf("New Command with index %v: %v", msg.Index, msg)
		if h.targetIndex > msg.Total {
			h.targetIndex = msg.Total
			if msg.Total == msg.Index+1 {
				h.currentIndex = msg.Index
				h.command = msg.Cmd
				h.updateContent()
			}
			return h, nil
		}
		if h.targetIndex == msg.Index || h.targetIndex < 0 {
			h.currentIndex = msg.Index
			h.command = msg.Cmd
			h.updateContent()
		}
		log.Printf("Current: %v; Target %v", h.currentIndex, h.targetIndex)
		return h, nil

	case tea.WindowSizeMsg:
		h.Width = msg.Width
		h.Height = msg.Height
		h.view.SetWidth(msg.Width)
		h.view.SetHeight(max(0, h.Height-1))
		if h.interactiveMode && h.command != nil {
			h.command.Resize(&pty.Winsize{
				Cols: uint16(msg.Width),
				Rows: uint16(msg.Height),
			})
		}
		return h, nil

	case shell.StdoutMsg:
		log.Print("Stdout output received:")
		if h.currentIndex < 0 && h.command == msg.Cmd {
			h.updateContent()
		}
		log.Print(h.view.GetContent())
		return h, nil

	case shell.StderrMsg:
		log.Print("Stderr output received")
		if h.currentIndex < 0 && h.command == msg.Cmd && h.showStderr {
			h.updateContent()
		}
		return h, nil
	}

	return h, nil
}

func (h *StdoutViewer) View() tea.View {

	if h.command == nil {
		return tea.NewView("")
	}

	if h.interactiveMode {
		log.Print("Interactive mode")
		if h.exitMenuSelect != menuSelectHidden {
			sendCtrlc := activeColor.Border(lipgloss.ASCIIBorder()).Render("Sendctrlc")
			exit := activeColor.Border(lipgloss.ASCIIBorder()).Render("Exit interactive mode")
			cancel := activeColor.Border(lipgloss.ASCIIBorder()).Render("Cancel")
			switch h.exitMenuSelect {
			case menuSelectSendctrlc:
				sendCtrlc = highlightColor.Border(lipgloss.ASCIIBorder()).Render("Sendctrlc")
			case menuSelectExit:
				exit = highlightColor.Border(lipgloss.ASCIIBorder()).Render("Exit interactive mode")
			case menuSelectCancel:
				cancel = highlightColor.Border(lipgloss.ASCIIBorder()).Render("Cancel")
			}
			return tea.NewView(lipgloss.JoinVertical(lipgloss.Center, sendCtrlc, exit, cancel))
		}
	}

	cmdLine := h.command.CommandLine()
	if h.showStderr {
		cmdLine = highlightColor.Render(cmdLine)
	}

	if h.commandRunning() {
		cmdLine = activeColor.Render("● ") + cmdLine
	}

	if h.interactiveMode {
		cmdLine = cmdLine + " " + highlightColor.Render("[interactive]")
	}

	sticky := inactiveColor
	if h.targetIndex != h.currentIndex || h.targetIndex < 0 {
		sticky = activeColor
	}
	if h.currentIndex >= 0 {
		i := sticky.Render(fmt.Sprintf("[%d]", h.currentIndex))
		return tea.NewView(fmt.Sprintf("%v %s\n%s", i, cmdLine, h.view.View()))
	}
	return tea.NewView(cmdLine + "\n" + h.view.View())
}

func (h *StdoutViewer) updateContent() {
	if h.command == nil {
		return
	}
	output := h.command.Stdout()
	if h.showStderr {
		output = h.command.Stderr()
	}
	output = wrap.String(output, h.Width)
	h.view.SetContent(output)
}
