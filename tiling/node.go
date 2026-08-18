package tiling

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/widget"
)

// A simple
type node struct {
	children []*node
	parent   *node
	// TODO: make it a func?
	positionFunc SplitFunc
	rectangle    rec
	//content      string
	border   lipgloss.Border
	widget   *widget.Widget
	priority int
}

func (n *node) Update(msg tea.Msg) widget.Action {
	if n.widget != nil {
		return n.widget.Update(msg)
	}
	return nil
}

func newNode(m *widget.Widget, positionFunc SplitFunc) *node {
	node := &node{
		positionFunc: positionFunc,
		widget:       m,
		// Prepare left/Right for binary tree
		// But allow (nested) list-based tiling
		//children: make([]*Node, 2),
	}
	return node
}

func (n *node) setBorder(b lipgloss.Border) {
	n.border = b
}

func (n *node) split(split SplitFunc) {
	n.positionFunc = split
	for _, c := range n.children {
		c.split(split)
	}
}

// TODO: make this a separate func. to e.g. hard limit to 2 elems
func (n *node) addChild(w *widget.Widget, priority int) (*node, widget.Action) {
	child := newNode(w, n.positionFunc)
	child.parent = n
	child.setBorder(n.border)
	child.priority = priority
	n.insertSorted(child)
	cmd := n.position(n.rectangle)
	return child, cmd
}

func (n *node) removeChild(m *widget.Widget) bool {
	for i, c := range n.children {
		if c.widget == m {
			n.children = append(n.children[:i], n.children[i+1:]...)
			return true
		}
		if c.removeChild(m) {
			return true
		}
	}
	return false
}

func (n *node) contains(w *widget.Widget) bool {
	for _, c := range n.children {
		if c.widget == w || c.contains(w) {
			return true
		}
	}
	return false
}

func (n *node) insertSorted(child *node) {
	prio := child.priority
	i := 0
	for i < len(n.children) && n.children[i].priority <= prio {
		i++
	}
	n.children = slices.Insert(n.children, i, child)
}

func (n *node) SetSize(width, height int) {
	n.rectangle.width = width
	n.rectangle.height = height
}

// TODO: Split up into Size and position (2 passes for flutter-like behaviour)
func (n *node) position(available rec) widget.Action {
	n.rectangle = available
	if n.widget != nil {
		var cmd widget.Action
		cmd = n.widget.Update(tea.WindowSizeMsg{
			Width:  n.rectangle.width,
			Height: n.rectangle.height,
		})
		return cmd
	}

	if n.positionFunc == nil {
		return nil
	}
	switch c := len(n.children); c {
	case 0:
		return nil
	case 1:
		return n.children[0].position(available)
	default:
		var cmds widget.Batch
		sizes := n.positionFunc(c, available)
		for c, child := range n.children {
			cmds = append(cmds, child.position(sizes[c]))
		}
		return cmds
	}
}
