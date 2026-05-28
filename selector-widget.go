package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type CloseSelectorMsg struct{}

type SelectorWidget struct {
	choices []string
	funcs   []func() tea.Cmd
	cursor  int
	width   int
	height  int
}

func newWidgetSelector(elems map[string]func() tea.Cmd) *SelectorWidget {
	w := &SelectorWidget{
		cursor: 0,
	}
	for name, f := range elems {
		w.choices = append(w.choices, name)
		w.funcs = append(w.funcs, f)
	}
	return w
}

func (sw *SelectorWidget) Init() tea.Cmd {
	return nil
}

func (sw *SelectorWidget) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if sw.cursor > 0 {
				sw.cursor--
			}
		case "down", "j":
			if sw.cursor < len(sw.choices)-1 {
				sw.cursor++
			}
		case "enter", " ":
			return sw, tea.Batch(
				sw.funcs[sw.cursor](),
				func() tea.Msg { return CloseSelectorMsg{} },
			)
		case "esc":
			return sw, func() tea.Msg { return CloseSelectorMsg{} }
		}
	case tea.WindowSizeMsg:
		sw.width = msg.Width
		sw.height = msg.Height
	}
	return sw, nil
}

func (sw *SelectorWidget) View() tea.View {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	title := titleStyle.Render("Select Widget")
	var items []string
	for i, choice := range sw.choices {
		if i == sw.cursor {
			items = append(items, cursorStyle.Render("> "+choice))
		} else {
			items = append(items, itemStyle.Render("  "+choice))
		}
	}
	list := lipgloss.JoinVertical(lipgloss.Left, items...)
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", list)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2).
		Render(content)

	centered := lipgloss.Place(sw.width, sw.height, lipgloss.Center, lipgloss.Center, dialog)
	return tea.NewView(centered)
}
