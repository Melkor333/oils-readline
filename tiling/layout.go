package tiling

import (
	"image/color"
	"log"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/widget"
)

// displaySelfMsg is sent by a widget (or on its behalf) to add itself to the
// layout.
type displaySelfMsg struct {
	Widget   *widget.Widget
	Priority int
}

func (msg displaySelfMsg) Tag(w *widget.Widget) tea.Msg {
	msg.Widget = w
	return msg
}

// DisplaySelf returns a command that requests the given model be made visible in the layout.
// priority determines the order (lower = first).
func DisplaySelf(priority int) widget.Action {
	return displaySelfMsg{Widget: nil, Priority: priority}
}

// HideSelf returns a command that requests the given model be removed from the layout.
func HideSelf() widget.Action {
	return hideSelfMsg{}
}

type hideSelfMsg struct {
	*widget.Widget
}

// TODO: Allow tea.Model and test for *wisdget.Widget in the widget library.
// To make this independent from  the widget.
// Users can then choose to use a wrapper + Tag or not.
func (msg hideSelfMsg) Tag(w *widget.Widget) tea.Msg {
	return hideSelfMsg{w}
}

// To be used by widgets
// TODO: Deal with them in the Flutter logic
type ErrNotWideEnough error
type ErrNotHighEnough error

// Focus messages
type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{}

// RemoveFocusedMsg is sent when the focused widget should be removed from the
// layout. The model handles removal from the widget list.
type RemoveFocusedMsg struct {
	Model tea.Model
}

type Layout struct {
	tree          *node
	focussed      *node
	Width, Height int

	border        lipgloss.Border
	activeColor   color.Color
	inactiveColor color.Color
}

func New(width, height int) *Layout {
	l := &Layout{
		tree:          newNode(nil, SplitHorizontal),
		border:        lipgloss.NormalBorder(),
		activeColor:   lipgloss.Color("2"),
		inactiveColor: lipgloss.Color("240"),
		Width:         width,
		Height:        height,
	}
	// This must be nil without a child
	l.tree.position(rec{0, 0, width, height})
	return l
}

func (l *Layout) Size(w, h int) (*Layout, widget.Action) {
	l.Width = w
	l.Height = h
	// TODO: return the actions!!
	return l, l.tree.position(rec{0, 0, w, h})
}

func (l *Layout) RemoveChild(w *widget.Widget) widget.Action {
	wasFocused := l.focussed != nil && l.focussed.widget == w

	cmd := l.tree.Update(tea.BlurMsg{})
	l.tree.removeChild(w)
	l.tree.position(l.tree.rectangle)

	if wasFocused {
		if len(l.tree.children) > 0 {
			return widget.Sequence{cmd, l.focus(l.tree.children[0])}
		} else {
			l.focussed = nil
		}
	}

	return cmd
}

func (l *Layout) Split(split SplitFunc) *Layout {
	n := l.tree
	n.split(split)
	n.position(n.rectangle)
	return l
}

func (l *Layout) AddChildren(priority int, mm ...*widget.Widget) (*Layout, widget.Action) {
	var cmds widget.Batch
	var node *node
	for _, m := range mm {
		var cmd widget.Action
		node, cmd = l.tree.addChild(m, priority)
		cmds = append(cmds, cmd)
		if l.focussed == nil {
			l.focus(node)
		} else {
			cmds = append(cmds, node.Update(tea.BlurMsg{}))
		}
	}
	return l, cmds
}

func (l *Layout) Len() int {
	return len(l.tree.children)
}

func (l *Layout) MasterRatio(float64) *Layout {
	// TODO: Implement!
	return l
}

func (l *Layout) MasterCount(int) *Layout {
	// TODO: Implement!
	return l
}

func (l *Layout) BorderStyle(s lipgloss.Border) *Layout {
	l.border = s
	return l
}

// FocusFocusMsg is sent by a widget to request focus on a specific model.
type RequestFocusMsg struct {
	Model tea.Model
}

// focus sets the focused model in the layout.
func (l *Layout) focus(n *node) widget.Action {
	var cmds widget.Sequence
	if l.focussed != nil {
		cmds = append(cmds, l.focussed.Update(tea.BlurMsg{}))
	}

	l.focussed = n

	if l.focussed != nil {
		cmds = append(cmds, l.focussed.Update(tea.FocusMsg{}))
	}
	return cmds
}

// focusNext focuses the next visible child.
func (l *Layout) focusNext() widget.Action {
	if len(l.tree.children) == 0 {
		return nil
	}
	if l.focussed == nil {
		return l.focus(l.tree.children[0])
	}
	for i, c := range l.tree.children {
		if c == l.focussed {
			return l.focus(l.tree.children[(i+1)%len(l.tree.children)])
		}
	}
	return l.focus(l.tree.children[0])
}

// focusPrev focuses the previous visible child.
func (l *Layout) focusPrev() widget.Action {
	if len(l.tree.children) == 0 {
		return nil
	}
	if l.focussed == nil {
		return l.focus(l.tree.children[0])
	}
	for i, c := range l.tree.children {
		if c == l.focussed {
			return l.focus(l.tree.children[(i-1+len(l.tree.children))%len(l.tree.children)])
		}
	}
	return l.focus(l.tree.children[0])
}

// focusFirst focuses the last visible child.
func (l *Layout) focusFirst() widget.Action {
	if len(l.tree.children) == 0 {
		return l.focus(nil)
	}
	return l.focus(l.tree.children[0])
}

// Focused returns the currently focused model, or nil.
func (l *Layout) Focused() *widget.Widget {
	if l.focussed == nil {
		return nil
	}
	return l.focussed.widget
}

func (l *Layout) blurMsg() widget.Action {
	if l.focussed == nil || l.focussed.widget == nil {
		return nil
	}
	cmd := l.focussed.Update(tea.BlurMsg{})
	return cmd
}

func (l *Layout) Dispatch(msg tea.Msg) (tea.Msg, widget.Action) {
	var cmd widget.Action
	switch msg := msg.(type) {
	case displaySelfMsg:
		_, cmd := l.AddChildren(msg.Priority, msg.Widget)
		return nil, cmd
	case hideSelfMsg:
		cmd := l.RemoveChild(msg.Widget)
		return nil, cmd
	case RequestFocusNextMsg:
		return nil, l.focusNext()
	case RequestFocusPrevMsg:
		return nil, l.focusPrev()
	case RequestFocusMainMsg:
		return nil, l.focusFirst()
	case tea.BlurMsg:
		return nil, l.blurMsg()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+j":
			return nil, l.focusNext()
		case "ctrl+k":
			return nil, l.focusPrev()
		case "ctrl+c":
			focused := l.Focused()
			if focused != nil {
				log.Print("Removing focussed Widget")
				cmd := l.RemoveChild(focused)
				return nil, cmd
			}
			log.Print("no widget left. Stopping")
			return nil, tea.Quit
		}
		if l.focussed != nil {
			cmd = l.focussed.widget.Update(msg)
			return nil, cmd
		}
	}
	return msg, nil
}
