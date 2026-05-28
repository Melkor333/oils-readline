package main

import (
	"flag"
	"log"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/Melkor333/oils-readline/shell"
)

var (
	historyFile = flag.String("historyFile", "$HOME/.local/share/oils/readline-history.json", "Path to the history file")
)

type history struct {
	commands        []shell.Command
	views           []viewport.Model
	focusedViewport int
	isFocussed      bool
	Width           int
}

var (
	brightGreenGut = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // bright green
	darkGreenGut   = lipgloss.NewStyle().Foreground(lipgloss.Color("22")) // dark green
)

// TODO: history should only track one shell?
func newHistory() *history {
	return &history{
		focusedViewport: 0,
	}
}

func (h *history) Init() tea.Cmd {
	return nil
}

func (h *history) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if h.focusedViewport > 0 {
				h.views[h.focusedViewport].LeftGutterFunc = unselected
				h.focusedViewport--
				h.views[h.focusedViewport].LeftGutterFunc = selected
			}
			return h, nil
		case "down", "j":
			if h.focusedViewport < len(h.views)-1 {
				h.views[h.focusedViewport].LeftGutterFunc = unselected
				h.focusedViewport++
				h.views[h.focusedViewport].LeftGutterFunc = selected
			}
			return h, nil
		case "ctrl+c":
			return h, func() tea.Msg { return RequestFocusMainMsg{} }
		}

	case shell.NewCommandMsg:
		h.Add(msg.Cmd)
		return h, nil

	case tea.WindowSizeMsg:
		h.Width = msg.Width
		for _, view := range h.views {
			view.SetWidth(msg.Width)
		}
		return h, nil

	case shell.StdoutMsg:
		log.Print("Stdout output received")
		for i, c := range h.commands {
			if c == msg.Cmd {
				h.updateViewportContent(i)
				break
			} else {
				log.Printf("%q is not command %q", c, msg.Cmd)
			}
		}
		return h, nil

	case tea.BlurMsg:
		if len(h.views) == 0 {
			return h, nil
		}
		h.views[h.focusedViewport].LeftGutterFunc = unselected
		h.isFocussed = false

	case tea.FocusMsg:
		if len(h.views) == 0 {
			return h, nil
		}
		h.views[h.focusedViewport].LeftGutterFunc = selected
		h.isFocussed = true
		return h, nil
		// TODO: A view for stderr!
		//case shell.StderrMsg:
		//	log.Print("Stderr output received")
		//	for i, c := range h.commands {
		//		if c == msg.Cmd {
		//			h.updateViewportContent(i)
		//			break
		//		}
		//	}
	}

	return h, nil
}

func (h *history) View() tea.View {
	var strs []string
	for _, view := range h.views {
		strs = append(strs, view.View())
	}
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, strs...))
}

func (h *history) Add(cmd shell.Command) {
	h.commands = append(h.commands, cmd)

	commandView := viewport.New(
		viewport.WithWidth(h.Width),
	)
	//commandView.YPosition = 0
	commandView.FillHeight = false
	if len(h.views) > 0 {
		h.views[len(h.views)-1].LeftGutterFunc = unselected
	}
	commandView.LeftGutterFunc = selected
	h.views = append(h.views, commandView)
	h.focusedViewport = len(h.views) - 1

	h.updateViewportContent(len(h.views) - 1)
}

func unselected(ctx viewport.GutterContext) string {
	return brightGreenGut.Render("│")
}

func selected(ctx viewport.GutterContext) string {
	return darkGreenGut.Render("│")
}

func (h *history) updateViewportContent(i int) {
	if i < 0 || i >= len(h.commands) {
		return
	}
	command := h.commands[i]
	content := command.CommandLine() + "\n" + strings.Trim(command.Stdout(), "\r\n")
	h.views[i].SetHeight(lipgloss.Height(content))
	h.views[i].SetContent(content)
}
