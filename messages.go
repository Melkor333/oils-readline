package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Melkor333/oils-readline/widget"
)

// A Targeted Message is an interface used by messages that should be sent to a specific widget
// When a message matches the interface, it will be sent to the widget it contains
// TODO: Should this take a generic `tea.Model` or is Widget really necessary?
type TargetedMsg interface {
	TargetWidget() *widget.Widget
}

type addWidgetMsg struct {
	Model tea.Model
}

// AddWidget is a func that returns a command to add a new widget to the Message Loop
func AddWidget(m tea.Model) tea.Cmd {
	return func() tea.Msg { return addWidgetMsg{m} }
}

// Sent by a widget to request all keyboard inputs
func RequestCapture() tea.Cmd {
	return func() tea.Msg { return requestCaptureMsg{} }
}

type requestCaptureMsg struct{ Widget *widget.Widget }

func (msg requestCaptureMsg) Tag(w *widget.Widget) tea.Msg { msg.Widget = w; return msg }

// Sent by a widget to stop receiving all keyboard input
func ReleaseCapture() tea.Cmd {
	return func() tea.Msg { return releaseCaptureMsg{} }
}

type releaseCaptureMsg struct{ widget *widget.Widget }

// Implement the widget.Targeted interface
func (msg releaseCaptureMsg) Tag(w *widget.Widget) tea.Msg { msg.widget = w; return msg }
