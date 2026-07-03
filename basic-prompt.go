package main

import (
	"log"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/Melkor333/oils-readline/shell"
)

var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	waitingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

type basicPrompt struct {
	input    *textarea.Model
	shell    shell.Shell
	focussed bool
	waiting  bool
}

type CommandEnteredMsg struct{ Text string }

func newBasicPrompt(s shell.Shell) *basicPrompt {
	ti := textarea.New()
	ti.SetVirtualCursor(true)
	ti.Placeholder = "Enter command"
	ti.Focus()
	ti.CharLimit = 156
	ti.Prompt = ""
	//ti.Prompt = promptStyle.Render(s.GetPrompt())

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
		bp.input.SetHeight(msg.Height)
		_, cmd := bp.input.Update(msg)
		return bp, cmd

	case shell.CommandMsg:
		if msg.Cmd.State() == shell.Queued || msg.Cmd.State() == shell.Started {
			bp.waiting = true
			bp.input.Placeholder = "busy..."
			//bp.input.Prompt = waitingStyle.Render("busy... ")
			return bp, nil
		}
		return bp, nil

	case shell.CommandDoneMsg:
		bp.waiting = false
		bp.input.Placeholder = "Enter command"
		//bp.input.Prompt = promptStyle.Render("")
		if bp.focussed {
			return bp, bp.input.Focus()
		}
		return bp, nil

	case tea.BlurMsg:
		bp.focussed = false
		bp.input.Blur()

	case tea.FocusMsg:
		bp.focussed = true
		if !bp.waiting {
			return bp, bp.input.Focus()
		}
		return bp, nil
	}

	var cmd tea.Cmd
	input, cmd := bp.input.Update(msg)
	bp.input = &input
	return bp, cmd
}

func (bp *basicPrompt) View() tea.View {
	return tea.NewView(strings.Trim(bp.input.View(), "\r\n"))
}
