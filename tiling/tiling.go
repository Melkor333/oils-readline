package tiling

import (
	"math"
	"strings"

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

// Layer computes the tiled layout and returns a Layer with separate
// widget and border sub-layers.
func (l *Layout) Layer() *lipgloss.Layer {
	if len(l.children) == 0 {
		return lipgloss.NewLayer("")
	}

	if len(l.children) == 1 {
		return lipgloss.NewLayer(l.placeChild(l.children[0], l.width, l.height))
	}

	masterN := min(l.masterCount, len(l.children))
	stackN := len(l.children) - masterN
	hasBorder := stackN > 0

	var widgets, borders []*lipgloss.Layer

	if l.splitMode == SplitVertical {
		widgets, borders = l.buildVerticalLayers(masterN, stackN, hasBorder)
	} else {
		widgets, borders = l.buildHorizontalLayers(masterN, stackN, hasBorder)
	}

	widgetLayer := lipgloss.NewLayer("", widgets...)
	borderLayer := lipgloss.NewLayer("", borders...)

	return lipgloss.NewLayer("", widgetLayer, borderLayer)
}

func (l *Layout) buildVerticalLayers(masterN, stackN int, hasBorder bool) ([]*lipgloss.Layer, []*lipgloss.Layer) {
	borderW := 0
	if hasBorder {
		borderW = lipgloss.Width(l.borderStyle.Render(string(l.border.Left)))
	}

	availW := l.width - borderW
	masterW := max(int(float64(availW)*l.masterRatio), 0)
	stackW := max(availW-masterW, 0)

	var widgets, borders []*lipgloss.Layer

	// Master column
	masterHeights := columnHeights(l.height, masterN)
	y := 0
	for i := range masterN {
		content := l.placeChild(l.children[i], masterW, masterHeights[i])
		widgets = append(widgets, lipgloss.NewLayer(content).X(0).Y(y))
		y += masterHeights[i]
		if i < masterN-1 {
			line := l.borderStyle.Render(l.horizontalBorderLine(masterW))
			borders = append(borders, lipgloss.NewLayer(line).X(0).Y(y).Z(1))
			y++
		}
	}

	// Stack column
	stackX := masterW + borderW
	stackHeights := columnHeights(l.height, stackN)
	y = 0
	for i := range stackN {
		content := l.placeChild(l.children[masterN+i], stackW, stackHeights[i])
		widgets = append(widgets, lipgloss.NewLayer(content).X(stackX).Y(y))
		y += stackHeights[i]
		if i < stackN-1 {
			line := l.borderStyle.Render(l.horizontalBorderLine(stackW))
			borders = append(borders, lipgloss.NewLayer(line).X(stackX).Y(y).Z(1))
			y++
		}
	}

	// Vertical border between master and stack
	if hasBorder {
		col := l.borderStyle.Render(l.verticalBorderLine(l.height))
		borders = append(borders, lipgloss.NewLayer(col).X(masterW).Y(0).Z(1))
	}

	return widgets, borders
}

func (l *Layout) buildHorizontalLayers(masterN, stackN int, hasBorder bool) ([]*lipgloss.Layer, []*lipgloss.Layer) {
	borderH := 0
	if hasBorder {
		borderH = 1
	}

	availH := l.height - borderH
	masterH := max(int(float64(availH)*l.masterRatio), 0)
	stackH := max(availH-masterH, 0)

	borderW := lipgloss.Width(l.borderStyle.Render(string(l.border.Left)))

	var widgets, borders []*lipgloss.Layer

	// Master row
	masterWidths := rowWidths(l.width, masterN, borderW)
	x := 0
	for i := range masterN {
		content := l.placeChild(l.children[i], masterWidths[i], masterH)
		widgets = append(widgets, lipgloss.NewLayer(content).X(x).Y(0))
		x += masterWidths[i]
		if i < masterN-1 {
			col := l.borderStyle.Render(l.verticalBorderLine(masterH))
			borders = append(borders, lipgloss.NewLayer(col).X(x).Y(0).Z(1))
			x += borderW
		}
	}

	// Stack row
	stackY := masterH + borderH
	stackWidths := rowWidths(l.width, stackN, borderW)
	x = 0
	for i := range stackN {
		content := l.placeChild(l.children[masterN+i], stackWidths[i], stackH)
		widgets = append(widgets, lipgloss.NewLayer(content).X(x).Y(stackY))
		x += stackWidths[i]
		if i < stackN-1 {
			col := l.borderStyle.Render(l.verticalBorderLine(stackH))
			borders = append(borders, lipgloss.NewLayer(col).X(x).Y(stackY).Z(1))
			x += borderW
		}
	}

	// Horizontal border between master and stack
	if hasBorder {
		line := l.borderStyle.Render(l.horizontalBorderLine(l.width))
		borders = append(borders, lipgloss.NewLayer(line).X(0).Y(masterH).Z(1))
	}

	return widgets, borders
}

func columnHeights(totalH, n int) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{totalH}
	}
	availH := max(totalH-(n-1), 0)
	return distribute(availH, n)
}

func rowWidths(totalW, n, borderW int) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{totalW}
	}
	availW := max(totalW-borderW*(n-1), 0)
	return distribute(availW, n)
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
	b := l.border.Bottom
	if b == "" {
		b = "─"
	}

	line := strings.Repeat(b, width/lipgloss.Width(b))

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
// TODO: Should also return x/y placement coordinates and borders
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
