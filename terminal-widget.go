package main

import (
	"fmt"
	"log"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/Melkor333/oils-readline/history"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/Melkor333/oils-readline/tiling"
)

type Terminal struct {
	command         shell.Command
	term            vt.Terminal
	position        int
	targetIndex     int
	currentIndex    int
	interactiveMode bool
	exitMenuSelect  menuSelection
	Width           int
	Height          int
}

func (h *Terminal) commandRunning() bool {
	return h.command != nil && (h.command.State() == shell.Queued || h.command.State() == shell.Started)
}

func newTerminal() *Terminal {
	return &Terminal{targetIndex: -1, currentIndex: -1, exitMenuSelect: menuSelectHidden, term: vt.NewSafeEmulator(10, 10)}
}

func (h *Terminal) Init() tea.Cmd {
	return tiling.DisplaySelf()
}

func (h *Terminal) WriteStdin(b []byte) (int, error) {
	if h.command == nil {
		return 0, fmt.Errorf("no command")
	}
	return h.command.Stdin().Write(b)
}

func (h *Terminal) IsInteractive() bool {
	return h.interactiveMode
}

func (h *Terminal) requestHistoryEntry(index int) tea.Cmd {
	return func() tea.Msg {
		return history.RequestHistoryEntryMsg{Index: index}
	}
}

func (h *Terminal) flushOutput() {
	h.term = vt.NewSafeEmulator(h.Width, h.Height-1)
}

func (h *Terminal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
						Rows: uint16(h.Height - 1), // TODO: -height of command prompt
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
		}
	case shell.CommandMsg:
		if h.targetIndex < 0 {
			h.interactiveMode = false
			h.position = 0
			h.exitMenuSelect = menuSelectHidden
			h.command = msg.Cmd
			h.currentIndex = -1
			emuW, emuH := h.Width, max(0, h.Height-1)
			if emuW > 0 && emuH > 0 && h.command.State() == shell.Started {
				h.command.Resize(&pty.Winsize{Cols: uint16(emuW), Rows: uint16(emuH)})
				h.term = vt.NewSafeEmulator(h.Width, h.Height-1)
			}
			h.flushOutput()
			h.updateContent()
		}
		//h.command.SetStdout(h.t.InputPipe())
		return h, ReleaseCapture()

	case shell.CommandDoneMsg:
		if h.interactiveMode {
			h.interactiveMode = false
			h.exitMenuSelect = menuSelectHidden
		}
		h.updateContent()
		return h, ReleaseCapture()

	case history.HistoryEntryMsg:
		log.Printf("New Command with index %v: %v", msg.Index, msg)
		if h.targetIndex > msg.Total {
			h.targetIndex = msg.Total
			if msg.Total == msg.Index+1 {
				h.currentIndex = msg.Index
				h.command = msg.Cmd
				h.flushOutput()
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
		if h.term != nil && h.Width > 0 && h.Height > 0 {
			h.term.Resize(h.Width, h.Height-1)
		}
		if h.command != nil && h.commandRunning() {
			h.command.Resize(&pty.Winsize{
				Cols: uint16(msg.Width),
				Rows: uint16(msg.Height - 1), // TODO: -height of command prompt
			})
		}
		return h, nil

	case shell.StdoutMsg:
		log.Print("Stdout output received:")
		if h.currentIndex < 0 && h.command == msg.Cmd {
			h.updateContent()
		}
		stdout := h.command.Stdout()
		h.term.WriteString(stdout[h.position:])
		h.position = len(stdout)
		return h, nil
	}

	return h, nil
}

func (h *Terminal) View() tea.View {

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
		return tea.NewView(fmt.Sprintf("%v %s\n%s", i, cmdLine, h.term.String()))
	}
	return tea.NewView(cmdLine + "\n" + h.term.Render())
}

func (h *Terminal) updateContent() {
	if h.command == nil {
		return
	}
	output := h.command.Stdout()

	h.term.Write([]byte(output[h.position:]))
	h.position = len(output)
}
