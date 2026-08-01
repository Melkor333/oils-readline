package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Melkor333/oils-readline/widget"
)

type TargetedMsg interface {
	TargetWidget() *widget.Widget
}

// AddWidgetMsg is a message that adds a widget to the model's widget list
// (but not the layout). The widget will be sent a DisplaySelfMsg to add it
// to the layout.
type addWidgetMsg struct {
	Model tea.Model
}

func AddWidget(m tea.Model) tea.Cmd {
	return func() tea.Msg { return addWidgetMsg{m} }
}

type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{} // Go to main

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
