package tiling

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/widget"
)

// displaySelfMsg is sent by a widget (or on its behalf) to add itself to the
// layout.
type displaySelfMsg struct {
	Model tea.Model
}

func (msg displaySelfMsg) Tag(w *widget.Widget) tea.Msg {
	msg.Model = w
	return msg
}

// DisplaySelf returns a command that requests the model be added to the layout.
func DisplaySelf() tea.Cmd {
	return func() tea.Msg { return displaySelfMsg{} }
}

// HideSelf returns a command that requests the model be removed from the layout.
func HideSelf() tea.Cmd {
	return func() tea.Msg { return hideSelfMsg{} }
}

// hideSelfMsg is sent by a widget to remove itself from the layout while
// keeping it in the widget list.
type hideSelfMsg struct {
	Model tea.Model
}

func (msg hideSelfMsg) Tag(w *widget.Widget) tea.Msg {
	msg.Model = w
	return msg
}

// To be used by widgets
type ErrNotWideEnough error
type ErrNotHighEnough error

type Layout struct {
	tree          *node
	focussed      *node
	Width, Height int

	//splitMode   SplitMode
	//focusIndex int

	border        lipgloss.Border
	activeColor   color.Color
	inactiveColor color.Color

	//children []string
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

func (l *Layout) RemoveChild(m tea.Model) {
	l.tree.removeChild(m)
	l.tree.position(l.tree.rectangle)
}

// Display adds a model to the layout tree at the end of the child list.
// It is idempotent — if the model is already in the tree, it does nothing.
func (l *Layout) Display(m tea.Model) tea.Cmd {
	if l.tree.contains(m) {
		return nil
	}
	_, cmd := l.AddChildAt(l.Len(), m)
	return cmd
}

// Hide removes a model from the layout tree.
func (l *Layout) Hide(m tea.Model) {
	l.RemoveChild(m)
}

func (l *Layout) Split(split SplitFunc) *Layout {
	n := l.tree
	n.split(split)
	n.position(n.rectangle)
	return l
}

func (l *Layout) Children(mm ...tea.Model) (*Layout, tea.Cmd) {
	var cmds []tea.Cmd
	for _, m := range mm {
		var cmd tea.Cmd
		_, cmd = l.tree.addChild(m)
		cmds = append(cmds, cmd)
	}
	return l, tea.Batch(cmds...)
}

func (l *Layout) AddChildAt(pos int, m tea.Model) (*node, tea.Cmd) {
	n := l.tree
	child := newNode(m, n.positionFunc)
	child.parent = n
	child.setBorder(n.border)
	n.children = append(n.children[:pos], append([]*node{child}, n.children[pos:]...)...)
	cmd := n.position(n.rectangle)
	return child, cmd
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

// Dispatch handles display/hide messages for the layout.
func (l *Layout) Dispatch(msg tea.Msg) (tea.Msg, tea.Cmd) {
	switch msg := msg.(type) {
	case displaySelfMsg:
		return nil, l.Display(msg.Model)
	case hideSelfMsg:
		l.Hide(msg.Model)
		return nil, nil
	}
	return msg, nil
}
