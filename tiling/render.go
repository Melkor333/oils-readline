package tiling

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// TODO: Add bit to mark active Node?
// Border boundary calculation
const (
	bitL = 1 << iota
	bitR
	bitU
	bitD
)

func (n *node) Render() *lipgloss.Layer {
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

	// seems unnecessary! :)
	// might be required by individual widgets?!
	//box := lipgloss.NewStyle().
	//	Width(n.rectangle.width).
	//	MaxWidth(n.rectangle.width).
	//	Height(n.rectangle.height).
	//	MaxHeight(n.rectangle.height).
	//	Render(content)

	var l *lipgloss.Layer
	l = lipgloss.NewLayer(content).X(n.rectangle.x).Y(n.rectangle.y).Z(5)
	return l
}

func (l *Layout) RenderLayer() *lipgloss.Layer {
	content := l.tree.Render()
	content.AddLayers(l.calculateBorders())
	return content
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
	var leafs []*node
	middles := []*node{rNode}
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

// Example:
//
//	var roundedBorder = Border{
//	 	Top:          "─",
//	 	Bottom:       "─",
//	 	Left:         "│",
//	 	Right:        "│",
//	 	TopLeft:      "╭",
//	 	TopRight:     "╮",
//	 	BottomLeft:   "╰",
//	 	BottomRight:  "╯",
//	 	MiddleLeft:   "├",
//	 	MiddleRight:  "┤",
//	 	Middle:       "┼",
//	 	MiddleTop:    "┬",
//	 	MiddleBottom: "┴",
//	 }
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
