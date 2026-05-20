package tiling

import (
	"math"

	"charm.land/lipgloss/v2"
)

// SplitMode controls how children are arranged.
type SplitMode int

const (
	// SplitVertical splits the area vertically (left-right), with the master
	// area on the left and the stack on the right. This is the default
	// awesome-wm style layout.
	SplitVertical SplitMode = iota
	// SplitHorizontal splits the area horizontally (top-bottom), with the
	// master area on top and the stack on the bottom.
	SplitHorizontal
)

// Layout is a list-based tiling layout manager inspired by awesome-wm.
// It arranges a list of lipgloss strings into a tiled layout with borders
// between them, returning a single lipgloss string.
type Layout struct {
	width  int
	height int

	borderStyle lipgloss.Style
	border      lipgloss.Border

	// masterRatio is the proportion of space given to the master area (0.0-1.0).
	masterRatio float64
	// masterCount is the number of children in the master area.
	masterCount int

	splitMode SplitMode

	children []string
}

// New creates a new Layout with sensible defaults matching awesome-wm's
// default tiling behavior: vertical split, one master window, 50/50 ratio.
func New() *Layout {
	return &Layout{
		masterRatio: 0.5,
		masterCount: 1,
		splitMode:   SplitVertical,
		border:      lipgloss.NormalBorder(),
		borderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

// Size sets the total available width and height for the layout.
func (l *Layout) Size(w, h int) *Layout {
	l.width = w
	l.height = h
	return l
}

// Width sets the total available width.
func (l *Layout) Width(w int) *Layout {
	l.width = w
	return l
}

// Height sets the total available height.
func (l *Layout) Height(h int) *Layout {
	l.height = h
	return l
}

// MasterRatio sets the proportion of space given to the master area.
// The value should be between 0.0 and 1.0.
func (l *Layout) MasterRatio(r float64) *Layout {
	l.masterRatio = math.Max(0, math.Min(1, r))
	return l
}

// MasterCount sets the number of children in the master area.
func (l *Layout) MasterCount(n int) *Layout {
	l.masterCount = n
	return l
}

// Split sets the split mode (SplitVertical or SplitHorizontal).
func (l *Layout) Split(mode SplitMode) *Layout {
	l.splitMode = mode
	return l
}

// BorderStyle sets the style for borders between tiles.
func (l *Layout) BorderStyle(s lipgloss.Style) *Layout {
	l.borderStyle = s
	return l
}

// Border sets the border characters used between tiles.
func (l *Layout) Border(b lipgloss.Border) *Layout {
	l.border = b
	return l
}

// Children sets the child lipgloss strings to be tiled.
func (l *Layout) Children(c ...string) *Layout {
	l.children = c
	return l
}

// Render computes the tiled layout and returns a single lipgloss string.
func (l *Layout) Render() string {
	if len(l.children) == 0 {
		return ""
	}

	if len(l.children) == 1 {
		return l.placeChild(l.children[0], l.width, l.height)
	}

	masterN := min(l.masterCount, len(l.children))
	stackN := len(l.children) - masterN

	hasBorder := stackN > 0

	var masterArea, stackArea string

	if l.splitMode == SplitVertical {
		masterArea, stackArea = l.renderVertical(masterN, stackN, hasBorder)
	} else {
		masterArea, stackArea = l.renderHorizontal(masterN, stackN, hasBorder)
	}

	if stackN == 0 {
		return masterArea
	}

	if l.splitMode == SplitVertical {
		borderStr := l.borderStyle.Render(string(l.border.Left))
		borderW := lipgloss.Width(borderStr)
		gap := lipgloss.NewStyle().Width(borderW).Height(l.height).Render(borderStr)
		return lipgloss.JoinHorizontal(lipgloss.Top, masterArea, gap, stackArea)
	}

	borderLine := l.borderStyle.Render(l.horizontalBorderLine(l.width))
	gap := lipgloss.NewStyle().Width(l.width).Height(1).Render(borderLine)
	return lipgloss.JoinVertical(lipgloss.Left, masterArea, gap, stackArea)
}

func (l *Layout) renderVertical(masterN, stackN int, hasBorder bool) (string, string) {
	borderW := 0
	if hasBorder {
		borderW = lipgloss.Width(l.borderStyle.Render(string(l.border.Left)))
	}

	availW := l.width - borderW
	masterW := int(float64(availW) * l.masterRatio)
	stackW := availW - masterW

	if masterW < 0 {
		masterW = 0
	}
	if stackW < 0 {
		stackW = 0
	}

	masterArea := l.tileColumn(l.children[:masterN], masterW, l.height)
	stackArea := ""
	if stackN > 0 {
		stackArea = l.tileColumn(l.children[masterN:], stackW, l.height)
	}

	return masterArea, stackArea
}

func (l *Layout) renderHorizontal(masterN, stackN int, hasBorder bool) (string, string) {
	borderH := 0
	if hasBorder {
		borderH = 1
	}

	availH := l.height - borderH
	masterH := int(float64(availH) * l.masterRatio)
	stackH := availH - masterH

	if masterH < 0 {
		masterH = 0
	}
	if stackH < 0 {
		stackH = 0
	}

	masterArea := l.tileRow(l.children[:masterN], l.width, masterH)
	stackArea := ""
	if stackN > 0 {
		stackArea = l.tileRow(l.children[masterN:], l.width, stackH)
	}

	return masterArea, stackArea
}

// tileColumn tiles children vertically within a column of given width and height.
func (l *Layout) tileColumn(children []string, width, height int) string {
	if len(children) == 0 {
		return ""
	}
	if len(children) == 1 {
		return l.placeChild(children[0], width, height)
	}

	borderH := l.borderSize()

	availH := max(height-borderH*(len(children)-1), 0)

	sizes := distribute(availH, len(children))

	var parts []string
	for i, child := range children {
		h := sizes[i]
		part := l.placeChild(child, width, h)
		parts = append(parts, part)

		if i < len(children)-1 {
			borderLine := l.borderStyle.Render(l.horizontalBorderLine(width))
			parts = append(parts, borderLine)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// tileRow tiles children horizontally within a row of given width and height.
func (l *Layout) tileRow(children []string, width, height int) string {
	if len(children) == 0 {
		return ""
	}
	if len(children) == 1 {
		return l.placeChild(children[0], width, height)
	}

	borderW := l.borderSize()

	availW := max(width-borderW*(len(children)-1), 0)

	sizes := distribute(availW, len(children))

	var parts []string
	for i, child := range children {
		w := sizes[i]
		part := l.placeChild(child, w, height)
		parts = append(parts, part)

		if i < len(children)-1 {
			borderCol := l.borderStyle.Render(l.verticalBorderLine(height))
			parts = append(parts, borderCol)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// placeChild places a child string within a tile of the given dimensions,
// centering it both horizontally and vertically.
func (l *Layout) placeChild(child string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, child)
}

func (l *Layout) borderSize() int {
	if l.splitMode == SplitVertical {
		return lipgloss.Width(l.borderStyle.Render(string(l.border.Left)))
	}
	return 1
}

func (l *Layout) horizontalBorderLine(width int) string {
	left := l.border.MiddleLeft
	right := l.border.MiddleRight
	if left == "" {
		left = l.border.Left
	}
	if right == "" {
		right = l.border.Right
	}

	mid := l.border.Middle
	if mid == "" {
		mid = "─"
	}

	target := max(width-lipgloss.Width(right), 0)

	line := left
	for lipgloss.Width(line) < target {
		line += mid
	}
	line += right

	// Truncate to exact display width using lipgloss.
	return lipgloss.NewStyle().Width(width).Render(line)
}

func (l *Layout) verticalBorderLine(height int) string {
	middle := l.border.Left
	if middle == "" {
		middle = "│"
	}

	lines := make([]string, height)
	for i := range lines {
		lines[i] = middle
	}

	return lipgloss.JoinVertical(0, lines...)
}

// TileSize holds the width and height of a single tile.
type TileSize struct {
	W, H int
}

// distribute splits total into n roughly equal parts, distributing
// remainder cells to the first elements.
func distribute(total, n int) []int {
	if n <= 0 {
		return nil
	}
	sizes := make([]int, n)
	base := total / n
	remainder := total % n
	for i := range sizes {
		sizes[i] = base
		if i < remainder {
			sizes[i]++
		}
	}
	return sizes
}

// TileSizes calculates the tile dimensions for n children based on the
// current layout configuration. Returns one TileSize per child.
func (l *Layout) TileSizes(n int) []TileSize {
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []TileSize{{W: l.width, H: l.height}}
	}

	masterN := min(l.masterCount, n)
	stackN := n - masterN
	hasBorder := stackN > 0

	if l.splitMode == SplitVertical {
		return l.tileSizesVertical(masterN, stackN, n, hasBorder)
	}
	return l.tileSizesHorizontal(masterN, stackN, n, hasBorder)
}

func (l *Layout) tileSizesVertical(masterN, stackN, total int, hasBorder bool) []TileSize {
	borderW := 0
	if hasBorder {
		borderW = lipgloss.Width(l.borderStyle.Render(string(l.border.Left)))
	}

	availW := l.width - borderW
	masterW := int(float64(availW) * l.masterRatio)
	stackW := availW - masterW
	if masterW < 0 {
		masterW = 0
	}
	if stackW < 0 {
		stackW = 0
	}

	sizes := make([]TileSize, total)
	for i := range masterN {
		sizes[i] = TileSize{W: masterW, H: l.height}
	}

	if stackN > 0 {
		borderH := l.borderSize()
		availH := max(l.height-borderH*(stackN-1), 0)
		heights := distribute(availH, stackN)
		for i := range stackN {
			sizes[masterN+i] = TileSize{W: stackW, H: heights[i]}
		}
	}

	return sizes
}

func (l *Layout) tileSizesHorizontal(masterN, stackN, total int, hasBorder bool) []TileSize {
	borderH := 0
	if hasBorder {
		borderH = 1
	}

	availH := l.height - borderH
	masterH := int(float64(availH) * l.masterRatio)
	stackH := availH - masterH
	if masterH < 0 {
		masterH = 0
	}
	if stackH < 0 {
		stackH = 0
	}

	sizes := make([]TileSize, total)
	for i := range masterN {
		sizes[i] = TileSize{W: l.width, H: masterH}
	}

	if stackN > 0 {
		borderW := l.borderSize()
		availW := max(l.width-borderW*(stackN-1), 0)
		widths := distribute(availW, stackN)
		for i := range stackN {
			sizes[masterN+i] = TileSize{W: widths[i], H: stackH}
		}
	}

	return sizes
}
