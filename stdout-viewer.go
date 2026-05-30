package main

import (
	"fmt"
	"log"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/Melkor333/oils-readline/shell"
)

type StdoutViewer struct {
	command      shell.Command
	view         viewport.Model
	isFocussed   bool
	targetIndex  int
	currentIndex int
	showStderr   bool
	Width        int
	Height       int
}

var (
	brightGreenGut = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // bright green
	darkGreenGut   = lipgloss.NewStyle().Foreground(lipgloss.Color("22")) // dark green
	redCmdLine     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
)

func newStdoutViewer() *StdoutViewer {
	return &StdoutViewer{targetIndex: -1, currentIndex: -1}
}

func newStderrViewer() *StdoutViewer {
	return &StdoutViewer{targetIndex: -1, currentIndex: -1, showStderr: true}
}

func (h *StdoutViewer) Init() tea.Cmd {
	return nil
}

func (h *StdoutViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if h.isFocussed {
			switch msg.String() {
			case "h":
				if h.currentIndex == -1 {
					h.currentIndex = 0
				} else if h.currentIndex > 0 {
					h.currentIndex--
				} else {
					return h, nil
				}
				return h, func() tea.Msg {
					return shell.RequestHistoryEntryMsg{Index: h.currentIndex}
				}
			case "l":
				if h.currentIndex >= 0 {
					h.currentIndex++
					return h, func() tea.Msg {
						return shell.RequestHistoryEntryMsg{Index: h.currentIndex}
					}
				}
				return h, nil
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
		}

	case shell.NewCommandMsg:
		if h.targetIndex == -1 {
			h.command = msg.Cmd
			h.currentIndex = -1
			h.view = viewport.New(
				viewport.WithWidth(h.Width),
				viewport.WithHeight(max(0, h.Height-1)),
			)
			h.view.FillHeight = false
			h.view.LeftGutterFunc = selected
			h.updateContent()
		}
		return h, nil

	case shell.HistoryEntryMsg:
		if h.targetIndex == msg.Index || h.targetIndex == -1 {
			h.currentIndex = msg.Index
			h.command = msg.Cmd
			h.updateContent()
		}
		return h, nil

	case tea.WindowSizeMsg:
		h.Width = msg.Width
		h.Height = msg.Height
		h.view.SetWidth(msg.Width)
		h.view.SetHeight(max(0, h.Height-1))
		return h, nil

	case shell.StdoutMsg:
		log.Print("Stdout output received")
		if h.currentIndex == -1 && h.command == msg.Cmd {
			h.updateContent()
		}
		return h, nil

	case shell.StderrMsg:
		log.Print("Stderr output received")
		if h.currentIndex == -1 && h.command == msg.Cmd && h.showStderr {
			h.updateContent()
		}
		return h, nil

	case tea.BlurMsg:
		h.view.LeftGutterFunc = unselected
		h.isFocussed = false

	case tea.FocusMsg:
		h.view.LeftGutterFunc = selected
		h.isFocussed = true
		return h, nil
	}

	return h, nil
}

func (h *StdoutViewer) View() tea.View {
	if h.command == nil {
		return tea.NewView("")
	}

	cmdLine := h.command.CommandLine()
	if h.showStderr {
		cmdLine = redCmdLine.Render(cmdLine)
	}

	sticky := darkGreenGut
	if h.targetIndex != h.currentIndex {
		sticky = brightGreenGut
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
	output = strings.Trim(output, "\r\n")
	h.view.SetContent(output)
}

func unselected(ctx viewport.GutterContext) string {
	return brightGreenGut.Render("│")
}

func selected(ctx viewport.GutterContext) string {
	return darkGreenGut.Render("│")
}
