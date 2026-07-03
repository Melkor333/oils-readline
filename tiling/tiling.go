package tiling

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type SplitMode int

const (
	Vertical SplitMode = iota
	Horizontal
)

const (
	Left  = 0
	Right = 1
)

type rec struct {
	x, y, width, height int
}

func SplitVertical(c int, available rec) (positions []rec) {
	// we want a border between each 2 nodes
	w := available.width - c + 1
	extra := w % c
	w = w / c
	for range c {
		r := available
		r.width = w
		// width + border
		available.x += w + 1

		if extra > 0 {
			// +1 extra
			r.width += 1
			available.x += 1
			extra--
		}
		positions = append(positions, r)
	}
	return positions
}

func SplitHorizontal(c int, available rec) (positions []rec) {
	// we want a border between each 2 nodes
	h := available.height - c + 1
	extra := h % c
	h = h / c
	for range c {
		r := available
		r.height = h
		// height + border
		available.y += h + 1

		if extra > 0 {
			// +1 extra
			r.height += 1
			// height + border + extra
			available.y += 1
			extra--
		}
		positions = append(positions, r)
	}
	return positions
}

func SplitHorizontalWithMain(c int, available rec) (positions []rec) {
	big := SplitHorizontal(2, available)
	positions = append(positions, big[0])
	small := SplitVertical(c-1, big[1])
	return append(positions, small...)
}

func SplitVerticalWithMain(c int, available rec) (positions []rec) {
	big := SplitVertical(2, available)
	positions = append(positions, big[0])
	small := SplitHorizontal(c-1, big[1])
	return append(positions, small...)
}

// TODO: Add bit to mark active Node?
// Border boundary calculation
const (
	bitL = 1 << iota
	bitR
	bitU
	bitD
)

// var roundedBorder = Border{
//  	Top:          "─",
//  	Bottom:       "─",
//  	Left:         "│",
//  	Right:        "│",
//  	TopLeft:      "╭",
//  	TopRight:     "╮",
//  	BottomLeft:   "╰",
//  	BottomRight:  "╯",
//  	MiddleLeft:   "├",
//  	MiddleRight:  "┤",
//  	Middle:       "┼",
//  	MiddleTop:    "┬",
//  	MiddleBottom: "┴",
//  }

func borderMap(bs lipgloss.Border) map[int]string {
	return map[int]string{
		0:                         " ",
		bitD:                      bs.Top,
		bitU:                      bs.Bottom,
		bitL:                      bs.Left,
		bitR:                      bs.Right,
		bitL | bitR:               bs.Left,
		bitU | bitD:               bs.Bottom,
		bitR | bitD:               bs.TopLeft,
		bitL | bitD:               bs.TopRight,
		bitR | bitU:               bs.BottomLeft,
		bitL | bitU:               bs.BottomRight,
		bitL | bitR | bitD:        bs.MiddleTop,
		bitL | bitR | bitU:        bs.MiddleBottom,
		bitR | bitU | bitD:        bs.MiddleLeft,
		bitL | bitU | bitD:        bs.MiddleRight,
		bitL | bitR | bitU | bitD: bs.Middle,
	}
}

func (l *Layout) calculateBorders() *lipgloss.Layer {
	rNode := l.tree
	root := l.tree.rectangle

	if root.width == 0 || root.height == 0 {
		return lipgloss.NewLayer("")
	}

	// either 1 or 0 children will result in the same..
	if len(rNode.children) < 2 {
		return lipgloss.NewLayer(lipgloss.NewStyle().Width(root.width).Height(root.height).Render("")).X(root.x).Y(root.y)
	}

	// Get a list of all leaf nodes. We only need to calculate leaf nodes.
	// Or do we? :D
	var leafs []*Node
	middles := []*Node{rNode}
	for c := 0; c < len(middles); c++ {
		middle := middles[c]
		for _, child := range middle.children {
			if len(child.children) > 0 {
				middles = append(middles, child)
			} else {
				leafs = append(leafs, child)
			}
		}
	}

	// Calculate the edges for each leaf node
	bitMask := make([]int, (root.width)*(root.height))
	for _, c := range leafs {
		child := c.rectangle

		// calculate the rectangle for the border
		// Since we don't draw a border at the edge, we know if a child is at the edge just by checking if it is on the edge.
		// We also need to adjust border height/width for each applicable border
		var left, right, top, bottom bool
		borderW := child.width
		borderH := child.height
		borderX := child.x
		borderY := child.y

		if child.x > root.x {
			left = true
			borderX--
			borderW++
		}
		if child.x+child.width < root.x+root.width {
			right = true
			borderW++
		}
		if child.y > root.y {
			top = true
			borderY--
			borderH++
		}
		if child.y+child.height < root.y+root.height {
			// width is one more
			borderH++
			bottom = true
		}

		line := root.width

		start := borderX + (root.width * borderY)
		if top || bottom {
			for c := range borderW {
				if top {
					bitMask[start+c] |= bitD
				}
				if bottom {
					bitMask[start+(line*(borderH-1))+c] |= bitU
				}
			}
		}

		if left || right {
			for c := range borderH {
				if left {
					bitMask[start+(line*(c))] |= bitR
				}
				if right {
					bitMask[start+(line*(c))+borderW-1] |= bitL
				}
			}
		}
	}

	return lipgloss.NewLayer(maskToBorder(bitMask, l.border, root.width, root.height)).Z(1)
}

func maskToBorder(mask []int, borderStyle lipgloss.Border, width int, height int) string {
	var border strings.Builder
	bm := borderMap(borderStyle)
	for y := range height {
		if y > 0 {
			border.WriteRune('\n')
		}
		for x := range width {
			border.WriteString(bm[mask[(y*width)+x]])
		}
	}
	return border.String()
}

type SplitFunc func(count int, available rec) []rec

type Node struct {
	children []*Node
	parent   *Node
	// TODO: make it a func?
	positionFunc SplitFunc
	rectangle    rec
	//content      string
	border lipgloss.Border
	model  tea.Model
}

func (n *Node) SetBorder(b lipgloss.Border) {
	n.border = b
}

func (n *Node) Parent() *Node {
	return n.parent
}

func (n *Node) PositionFunc() SplitFunc {
	return n.positionFunc
}

func (n *Node) RemoveChild(m tea.Model) bool {
	for i, c := range n.children {
		if c.model == m {
			n.children = append(n.children[:i], n.children[i+1:]...)
			return true
		}
		if c.RemoveChild(m) {
			return true
		}
	}
	return false
}

func (n *Node) SetSize(width, height int) {
	n.rectangle.width = width
	n.rectangle.height = height
}

type SimpleNode struct {
	Node
	text string
}

func (n *SimpleNode) Render() string {
	return n.text
}

//func (n *SimpleNode) Constraint(width, height int) (int, int, error) {
//	n.BaseNode.SetSize(width, height)
//	return width, height, nil
//}

//type Node interface {
//	// Implemented by baseNode
//	SetBorder(lipgloss.Border)
//	Border() lipgloss.Border
//	rectangle rec
//	//BaseNode() *BaseNode
//	Split(SplitFunc)
//	Children() []*Node
//	// TODO: Enhance with constraint
//	Position(rec)
//	Parent() *Node
//	SetParent(*Node)
//	SetPositionFunc(SplitFunc)
//	PositionFunc() SplitFunc
//	// TODO: Rename to AddChild
//	AddChildren(*Node)
//	//TODO
//	// RemoveChild(Node)
//
//	// To be implemented by wrapping node
//	Render() *lipgloss.Layer
//	//Size() (width, height int)
//	//Contraint(availableWidth, availableHeight int) (width, height int, err error)
//}

type NewNodeFunc func(*Node) *Node

type ErrNotWideEnough error
type ErrNotHighEnough error

// TODO: Split up into Size and Position (2 passes for flutter)
func (n *Node) Position(available rec) tea.Cmd {
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
		return n.children[0].Position(available)
	default:
		var cmds []tea.Cmd
		sizes := n.positionFunc(c, available)
		for c, child := range n.children {
			cmds = append(cmds, child.Position(sizes[c]))
		}
		return tea.Batch(cmds...)
	}
}

func newNode(m tea.Model, positionFunc SplitFunc) *Node {
	node := &Node{
		positionFunc: positionFunc,
		model:        m,
		// Prepare left/Right for binary tree
		// But allow (nested) list-based tiling
		//children: make([]*Node, 2),
	}
	return node
}

// TODO: make this a separate func. to e.g. hard limit to 2 elems
func (n *Node) AddChild(model tea.Model) (*Node, tea.Cmd) {
	child := newNode(model, n.positionFunc)
	child.parent = n
	child.SetBorder(n.border)
	n.children = append(n.children, child)
	cmd := n.Position(n.rectangle)
	return child, cmd
}

type Layout struct {
	tree          *Node
	focussed      *Node
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
	l.tree.Position(rec{0, 0, w, h})
	return l
}

func (l *Layout) RemoveChild(m tea.Model) {
	l.tree.RemoveChild(m)
	l.tree.Position(l.tree.rectangle)
}

func (l *Layout) Split(split SplitFunc) *Layout {
	n := l.tree
	n.Split(split)
	n.Position(n.rectangle)
	return l
}

//func (l *Layout) widgetCount() int {
//	if l.tree == nil {
//		return 0
//	}
//	c := 0
//	a := []*Node{l.tree}
//	for {
//		if c > len(a) {
//			break
//		}
//		cur := a[c]
//		for _, child := range cur.children {
//			if child != nil {
//				a = append(a, child)
//			}
//		}
//		c++
//	}
//	return c
//}

func (l *Layout) Children(mm ...tea.Model) (*Layout, tea.Cmd) {
	var cmds []tea.Cmd
	for _, m := range mm {
		var cmd tea.Cmd
		_, cmd = l.tree.AddChild(m)
		cmds = append(cmds, cmd)
	}
	return l, tea.Batch(cmds...)
}

func (l *Layout) AddChildAt(pos int, m tea.Model) (*Node, tea.Cmd) {
	n := l.tree
	child := newNode(m, n.positionFunc)
	child.parent = n
	child.SetBorder(n.border)
	n.children = append(n.children[:pos], append([]*Node{child}, n.children[pos:]...)...)
	cmd := n.Position(n.rectangle)
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

func (l *Layout) Layer() *lipgloss.Layer {
	content := l.tree.Render()
	content.AddLayers(l.calculateBorders())
	return content
}

func (n *Node) Render() *lipgloss.Layer {
	// Collect Child Layers if there are any
	if len(n.children) > 0 {
		l := lipgloss.NewLayer("")
		for _, c := range n.children {
			l.AddLayers(c.Render())
		}
		return l
	}

	content := ""
	if n.model != nil {
		content = n.model.View().Content
	}

	box := lipgloss.NewStyle().
		Width(n.rectangle.width).
		MaxWidth(n.rectangle.width).
		Height(n.rectangle.height).
		MaxHeight(n.rectangle.height).
		Render(content)

	var l *lipgloss.Layer
	l = lipgloss.NewLayer(box).X(n.rectangle.x).Y(n.rectangle.y).Z(5)
	return l
}

func (n *Node) Split(split SplitFunc) {
	n.positionFunc = split
	for _, c := range n.children {
		c.Split(split)
	}
}
