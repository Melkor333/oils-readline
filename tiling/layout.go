package tiling

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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
