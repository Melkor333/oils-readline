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
	command    shell.Command
	view       viewport.Model
	isFocussed bool
	Width      int
	Height     int
}

var (
	brightGreenGut = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // bright green
	darkGreenGut   = lipgloss.NewStyle().Foreground(lipgloss.Color("22")) // dark green
)

func newHistory() *history {
	return &history{}
}

func (h *history) Init() tea.Cmd {
	return nil
}

func (h *history) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if h.isFocussed {
			var cmd tea.Cmd
			h.view, cmd = h.view.Update(msg)
			return h, cmd
		}

	case shell.NewCommandMsg:
		h.command = msg.Cmd
		h.view = viewport.New(
			viewport.WithWidth(h.Width),
			viewport.WithHeight(max(0, h.Height-1)),
		)
		h.view.FillHeight = false
		h.view.LeftGutterFunc = selected
		h.updateContent()
		return h, nil

	case tea.WindowSizeMsg:
		h.Width = msg.Width
		h.Height = msg.Height
		h.view.SetWidth(msg.Width)
		h.view.SetHeight(max(0, h.Height-1))
		return h, nil

	case shell.StdoutMsg:
		log.Print("Stdout output received")
		if h.command == msg.Cmd {
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

func (h *history) View() tea.View {
	if h.command == nil {
		return tea.NewView("")
	}
	return tea.NewView(h.command.CommandLine() + "\n" + h.view.View())
}

func (h *history) updateContent() {
	if h.command == nil {
		return
	}
	output := strings.Trim(h.command.Stdout(), "\r\n")
	h.view.SetContent(output)
}

func unselected(ctx viewport.GutterContext) string {
	return brightGreenGut.Render("│")
}

func selected(ctx viewport.GutterContext) string {
	return darkGreenGut.Render("│")
}
