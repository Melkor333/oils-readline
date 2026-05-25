package main

import (
	"log"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/Melkor333/oils-readline/shell"
)

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
)

type basicPrompt struct {
	input *textinput.Model
	shell shell.Shell
}

type CommandEnteredMsg struct{ Text string }

func newBasicPrompt(s shell.Shell) *basicPrompt {
	ti := textinput.New()
	ti.SetVirtualCursor(true)
	ti.Placeholder = "Enter command"
	ti.Focus()
	ti.CharLimit = 156
	ti.SetWidth(20)
	ti.Prompt = promptStyle.Render(s.GetPrompt())

	return &basicPrompt{
		input: &ti,
		shell: s,
	}
}

func (bp *basicPrompt) Init() tea.Cmd {
	return nil
}

func (bp *basicPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// don't handle keyboard inputs if we're not focussed!
		if !bp.input.Focused() {
			return bp, nil
		}
		switch msg.String() {
		case "ctrl+c":
			bp.input.Reset()
			return bp, nil
		case "ctrl+d":
			if bp.input.Value() == "" {
				return bp, tea.Quit
			}
			return bp, nil
		case "enter":
			command := bp.input.Value()
			bp.input.Reset()
			bp.input.Blur()
			if len(command) == 0 {
				return bp, nil
			}
			log.Print("Sending CommandEntered")
			return bp, func() tea.Msg { return CommandEnteredMsg{Text: command} }
		}

	case tea.WindowSizeMsg:
		bp.input.SetWidth(msg.Width)
		return bp, nil

	case shell.CommandDoneMsg:
		bp.input.Prompt = promptStyle.Render(bp.shell.GetPrompt())
		return bp, bp.input.Focus()

	case tea.BlurMsg:
		bp.input.Blur()

	case tea.FocusMsg:
		return bp, bp.input.Focus()
	}

	var cmd tea.Cmd
	input, cmd := bp.input.Update(msg)
	bp.input = &input
	return bp, cmd
}

func (bp *basicPrompt) View() tea.View {
	return tea.NewView(strings.Trim(bp.input.View(), "\r\n"))
}
