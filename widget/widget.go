package widget

import (
	"fmt"
	"log/slog"

	tea "charm.land/bubbletea/v2"
)

type WidgetOption func(Widget) (Widget, error)
type DispatchFunc func(Widget, tea.Msg) (Widget, tea.Cmd)
type StopProcessingMsg struct{}

// A Tagged Message is targeted at a specific widget.
type TaggedMsg interface {
	Tag(w *Widget) tea.Msg
}

// Action is either a tea.Cmd or a tea.Msg
// For now Msgs are directly passed back to the `model.Update`
// Since go has not a proper sum type, I don't see a better way to solve this
// Without going into messy generics territory
type Action any
type Batch []Action
type Sequence []Action

type Model interface {
	Init() Action
	View() tea.View
	Update(tea.Msg) (Model, Action)
	Name() string
}

// Teardownable is an optional interface a Model can implement to release
// resources (goroutines, file handles) when the parent removes it. Teardown
// is called synchronously; whatever Action it returns is batched by the
// caller and may be executed asynchronously.
type Teardownable interface {
	Teardown() Action
}

type Widget struct{ Model }

// Wrap a
func WrapAction(action Action, w *Widget) Action {
	if action == nil {
		return nil
	}
	if batch, ok := action.(Batch); ok {
		for i, _action := range batch {
			batch[i] = WrapAction(_action, w)
		}
		return batch
	}
	if sequence, ok := action.(Sequence); ok {
		for i, _action := range sequence {
			sequence[i] = WrapAction(_action, w)
		}
		return sequence
	}

	// if a function produces a `tea.Cmd`, that's captured here.
	if _cmd, ok := action.(tea.Cmd); ok {
		slog.Debug("captured Cmd", "action", fmt.Sprintf("%T", _cmd))
		return wrapTeaCmd(_cmd, w)
	}
	// If the func is only ever treated as `tea.Msg` it lands here.
	if _cmd, ok := action.(func() tea.Msg); ok {
		slog.Debug("captured untyped Cmd", "action", fmt.Sprintf("%T", _cmd))
		return wrapTeaCmd(_cmd, w)
	}
	slog.Debug("captured msg", "action", fmt.Sprintf("%T", action))
	return wrapMsg(action, w)
}

// wrapMsg wraps a message and makes sure it's properly tagged
func wrapMsg(msg tea.Msg, w *Widget) tea.Msg {
	switch m := msg.(type) {
	case tea.BatchMsg:
		for i, c := range m {
			m[i] = wrapTeaCmd(c, w)
		}
		return m
	case TaggedMsg:
		slog.Debug("tagged message!", "widget", w)
		return m.Tag(w)
	}
	return msg
}

// WrapChildCmd wraps a child's command to intercept tagged Msgs and properly tag them
// This is run "around" the childcmd
func wrapTeaCmd(cmd tea.Cmd, w *Widget) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		return wrapMsg(msg, w)
	}
}

func (w *Widget) Init() Action {
	return WrapAction(w.Model.Init(), w)
}

func (w *Widget) Update(msg tea.Msg) (cmd Action) {
	if w == nil {
		return nil
	}
	w.Model, cmd = w.Model.Update(msg)
	return WrapAction(cmd, w)
}
