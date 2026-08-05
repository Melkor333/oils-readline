package tiling

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	model    tea.Model
	priority int
}

func (n *node) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if n.model != nil {
		n.model, cmd = n.model.Update(msg)
		return cmd
	}
	return nil
}

func newNode(m tea.Model, positionFunc SplitFunc) *node {
	node := &node{
		positionFunc: positionFunc,
		model:        m,
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
func (n *node) addChild(model tea.Model, priority int) (*node, tea.Cmd) {
	child := newNode(model, n.positionFunc)
	child.parent = n
	child.setBorder(n.border)
	child.priority = priority
	n.insertSorted(child)
	cmd := n.position(n.rectangle)
	return child, cmd
}

func (n *node) removeChild(m tea.Model) bool {
	for i, c := range n.children {
		if c.model == m {
			n.children = append(n.children[:i], n.children[i+1:]...)
			return true
		}
		if c.removeChild(m) {
			return true
		}
	}
	return false
}

func (n *node) contains(m tea.Model) bool {
	for _, c := range n.children {
		if c.model == m || c.contains(m) {
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
func (n *node) position(available rec) tea.Cmd {
	n.rectangle = available
	if n.model != nil {
		var cmd tea.Cmd
		n.model, cmd = n.model.Update(tea.WindowSizeMsg{
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
		var cmds []tea.Cmd
		sizes := n.positionFunc(c, available)
		for c, child := range n.children {
			cmds = append(cmds, child.position(sizes[c]))
		}
		return tea.Batch(cmds...)
	}
}
