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
	Model    tea.Model
	Priority int
}

func (msg displaySelfMsg) Tag(w *widget.Widget) tea.Msg {
	msg.Model = w
	return msg
}

// DisplaySelf returns a command that requests the given model be made visible in the layout.
// priority determines the order (lower = first).
func DisplaySelf(priority int) tea.Cmd {
	return func() tea.Msg { return displaySelfMsg{Model: nil, Priority: priority} }
}

// HideSelf returns a command that requests the given model be removed from the layout.
func HideSelf() tea.Cmd {
	return func() tea.Msg { return hideSelfMsg{} }
}

type hideSelfMsg struct {
	Model tea.Model
}

// TODO: Allow tea.Model and test for *wisdget.Widget in the widget library.
// To make this independent from  the widget.
// Users can then choose to use a wrapper + Tag or not.
func (msg hideSelfMsg) Tag(w *widget.Widget) tea.Msg {
	msg.Model = w
	return msg
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

func New() *Layout {
	return &Layout{
		tree:          newNode(nil, SplitHorizontal),
		border:        lipgloss.NormalBorder(),
		activeColor:   lipgloss.Color("2"),
		inactiveColor: lipgloss.Color("240"),
	}
}

func (l *Layout) Size(w, h int) *Layout {
	l.Width = w
	l.Height = h
	l.tree.position(rec{0, 0, w, h})
	return l
}

func (l *Layout) RemoveChild(m tea.Model) tea.Cmd {
	wasFocused := l.focussed != nil && l.focussed.model == m

	cmd := l.tree.Update(tea.BlurMsg{})
	l.tree.removeChild(m)
	l.tree.position(l.tree.rectangle)

	if wasFocused {
		if len(l.tree.children) > 0 {
			return tea.Sequence(cmd, l.focus(l.tree.children[0]))
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

func (l *Layout) AddChildren(priority int, mm ...tea.Model) (*Layout, tea.Cmd) {
	var cmds []tea.Cmd
	var node *node
	for _, m := range mm {
		var cmd tea.Cmd
		node, cmd = l.tree.addChild(m, priority)
		cmds = append(cmds, cmd)
		if l.focussed == nil {
			l.focus(node)
		} else {
			cmds = append(cmds, node.Update(tea.BlurMsg{}))
		}
	}
	return l, tea.Batch(cmds...)
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
func (l *Layout) focus(n *node) tea.Cmd {
	var cmds []tea.Cmd
	if l.focussed != nil {
		cmds = append(cmds, l.focussed.Update(tea.BlurMsg{}))
	}

	l.focussed = n

	if l.focussed != nil {
		cmds = append(cmds, l.focussed.Update(tea.FocusMsg{}))
	}
	return tea.Sequence(cmds...)
}

// focusNext focuses the next visible child.
func (l *Layout) focusNext() tea.Cmd {
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
func (l *Layout) focusPrev() tea.Cmd {
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
func (l *Layout) focusFirst() tea.Cmd {
	if len(l.tree.children) == 0 {
		return l.focus(nil)
	}
	return l.focus(l.tree.children[0])
}

// Focused returns the currently focused model, or nil.
func (l *Layout) Focused() tea.Model {
	if l.focussed == nil {
		return nil
	}
	return l.focussed.model
}

func (l *Layout) blurMsg() tea.Cmd {
	if l.focussed == nil || l.focussed.model == nil {
		return nil
	}
	cmd := l.focussed.Update(tea.BlurMsg{})
	return cmd
}

func (l *Layout) Dispatch(msg tea.Msg) (tea.Msg, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case displaySelfMsg:
		_, cmd := l.AddChildren(msg.Priority, msg.Model)
		return nil, cmd
	case hideSelfMsg:
		cmd := l.RemoveChild(msg.Model)
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
			l.focussed.model, cmd = l.focussed.model.Update(msg)
			return nil, cmd
		}
	}
	return msg, nil
}
