package main

import (
	"log"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/Melkor333/oils-readline/shell"
)

type history struct {
	commands        []shell.Command
	viewports       []viewport.Model
	focusedViewport int // -1 = nothing focused, 0+ = viewport index
	Width           int
	program         *tea.Program
}

func newHistory() *history {
	return &history{
		focusedViewport: -1,
	}
}

func (h *history) Init() tea.Cmd {
	return nil
}

func (h *history) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+space":
			if h.focusedViewport >= 0 {
				h.focusedViewport = -1
				h.updateGutters()
				return h.inputFocusCmd(), true
			} else if len(h.viewports) > 0 {
				if h.focusedViewport == -1 {
					h.focusedViewport = len(h.viewports) - 1
				}
				h.updateGutters()
				return nil, true
			}
			return nil, true
		case "esc":
			if h.focusedViewport >= 0 {
				h.focusedViewport = -1
				h.updateGutters()
				return h.inputFocusCmd(), true
			}
		case "up", "k":
			if h.focusedViewport >= 0 && len(h.viewports) > 0 {
				if h.focusedViewport > 0 {
					h.focusedViewport--
					h.updateGutters()
				}
				return nil, true
			}
		case "down", "j":
			if h.focusedViewport >= 0 && len(h.viewports) > 0 {
				if h.focusedViewport < len(h.viewports)-1 {
					h.focusedViewport++
					h.updateGutters()
				}
				return nil, true
			}
		case "enter":
			if h.focusedViewport >= 0 {
				return nil, true
			}
		}

	case tea.WindowSizeMsg:
		h.Width = msg.Width
		for _, vp := range h.viewports {
			vp.SetWidth(msg.Width)
		}

	case CommandDoneMsg:
		h.focusedViewport = -1
		h.updateGutters()

	case StdoutMsg:
		log.Print("Stdout output received")
		for i, c := range h.commands {
			if c == msg.Cmd {
				h.updateViewportContent(i)
				break
			}
		}

	case StderrMsg:
		log.Print("Stderr output received")
		for i, c := range h.commands {
			if c == msg.Cmd {
				h.updateViewportContent(i)
				break
			}
		}
	}

	return nil, false
}

func (h *history) View() string {
	var strs []string
	for i := range h.viewports {
		strs = append(strs, h.viewports[i].View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, strs...)
}

func (h *history) Add(cmd shell.Command) {
	h.commands = append(h.commands, cmd)

	commandView := viewport.New(
		viewport.WithWidth(h.Width),
	)
	commandView.YPosition = 0
	commandView.FillHeight = false
	h.viewports = append(h.viewports, commandView)

	h.updateViewportContent(len(h.viewports) - 1)
	h.updateGutters()
}

func (h *history) Focused() bool {
	return h.focusedViewport >= 0
}

func (h *history) updateViewportContent(i int) {
	if i < 0 || i >= len(h.commands) {
		return
	}
	command := h.commands[i]
	content := command.CommandLine() + "\n" + strings.Trim(command.Stdout(), "\r\n")
	h.viewports[i].SetHeight(lipgloss.Height(content))
	h.viewports[i].SetContent(content)
}

func (h *history) updateGutters() {
	for i := range h.viewports {
		gutStyle := brightGreenGut
		if h.focusedViewport == i {
			gutStyle = darkGreenGut
		}
		style := gutStyle
		h.viewports[i].LeftGutterFunc = func(ctx viewport.GutterContext) string {
			return style.Render("│")
		}
	}
}

func (h *history) inputFocusCmd() tea.Cmd {
	return func() tea.Msg { return focusInputMsg{} }
}

type focusInputMsg struct{}
