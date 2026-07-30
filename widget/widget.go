package widget

import (
	"log"

	tea "charm.land/bubbletea/v2"
)

// RemoveSelfMsg is a message that a child model can return from its Update
// function (via a command) to request its own removal from the parent tiling
// Model.
//func widgets(m *model) map[string]func() tea.Cmd {
//	return map[string]func() tea.Cmd{
//		"SimplePrompt": func() tea.Cmd { return m.AddChild(newBasicPrompt(m.shells[m.shellFocus])) },
//		"StdoutLog":    func() tea.Cmd { return m.AddChild(newStdoutViewer()) },
//		"ErrorLog":     func() tea.Cmd { return m.AddChild(newStderrViewer()) },
//		"Terminal":     func() tea.Cmd { return m.AddChild(newTerminal()) },
//	}
//}

// A Tagged Message is targeted at a specific widget.
type TaggedMsg interface {
	Tag(w *Widget) tea.Msg
}

// Widget pairs a child model with a unique ID for stable identity tracking.
type Widget struct {
	tea.Model
}

// WrapChildCmd wraps a child's command to intercept RemoveSelfMsg and convert it
// to a removeChildMsg with the correct child ID.
func WrapChildCmd(cmd tea.Cmd, w *Widget) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if t, ok := msg.(TaggedMsg); ok {
			log.Printf("Tagged message for %v!", w)
			msg = t.Tag(w)
		}
		return msg
	}
}

func (w *Widget) Init() tea.Cmd {
	return WrapChildCmd(w.Model.Init(), w)
}
func (w *Widget) Update(msg tea.Msg) (m tea.Model, cmd tea.Cmd) {
	// TODO: Is it ever possible that `w` is nil??
	if w == nil {
		return nil, nil
	}
	w.Model, cmd = w.Model.Update(msg)
	return w, WrapChildCmd(cmd, w)
}
