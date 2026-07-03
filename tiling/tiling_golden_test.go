package tiling

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
)

//func TestDistribute(t *testing.T) {
//	sizes := distribute(10, 3)
//	if len(sizes) != 3 {
//		t.Fatalf("expected 3 elements, got %d", len(sizes))
//	}
//	total := 0
//	for _, s := range sizes {
//		total += s
//	}
//	if total != 10 {
//		t.Errorf("expected total 10, got %d", total)
//	}
//}

func renderLayer(l *lipgloss.Layer) string {
	return lipgloss.NewCompositor(l).Render()
}

func TestSingleChild(t *testing.T) {
	hor, _ := New().Size(80, 24).Split(SplitHorizontal).Children(M{"SINGLE"})
	vert, _ := New().Size(80, 24).Split(SplitVertical).Children(M{"SINGLE"})
	horStr := renderLayer(hor.Layer())
	vertStr := renderLayer(vert.Layer())
	golden.RequireEqual(t, []byte(horStr))

	// Horizontal and Vertical should be equal with a single child
	assert.Equal(t, []byte(horStr), []byte(vertStr))
	golden.RequireEqual(t, []byte(horStr))
}

func TestLayoutEmpty(t *testing.T) {
	l := New().Size(80, 10)
	result := renderLayer(l.Layer())
	assert.Equal(t, "\n\n\n\n\n\n\n\n\n", result, "should only be newlines.")
}

//func TestTileSizes(t *testing.T) {
//	l := New().Size(80, 24)
//
//	sizes := l.TileSizes(1)
//	if len(sizes) != 1 || sizes[0].W != 78 || sizes[0].H != 22 {
//		t.Errorf("unexpected sizes for 1 child: %v", sizes)
//	}
//
//	sizes = l.TileSizes(2)
//	if len(sizes) != 2 {
//		t.Fatalf("expected 2 sizes, got %d", len(sizes))
//	}
//	totalW := sizes[0].W + sizes[1].W
//	if totalW > 78 {
//		t.Errorf("total width %d exceeds 78 (80-2 outer borders)", totalW)
//	}
//	if sizes[0].H != 22 || sizes[1].H != 22 {
//		t.Errorf("expected height 22 for both, got %d and %d", sizes[0].H, sizes[1].H)
//	}
//
//	sizes = l.TileSizes(3)
//	if len(sizes) != 3 {
//		t.Fatalf("expected 3 sizes, got %d", len(sizes))
//	}
//}

func TestGoldenRender(t *testing.T) {
	tests := []struct {
		name     string
		layout   *Layout
		children []tea.Model
	}{
		{
			name:     "2_children",
			layout:   New().Size(80, 24),
			children: []tea.Model{M{"ONE"}, M{"TWO"}},
		},
		{
			name:     "3_children",
			layout:   New().Size(80, 24),
			children: []tea.Model{M{"M"}, M{"S1"}, M{"S2"}},
		},
		{
			name:     "master_ratio_75",
			layout:   New().Size(80, 24).MasterRatio(0.75),
			children: []tea.Model{M{"BIG"}, M{"SMALL"}},
		},
		{
			name:     "master_ratio_25",
			layout:   New().Size(80, 24).MasterRatio(0.25),
			children: []tea.Model{M{"SMALL"}, M{"BIG"}},
		},
		{
			name:     "small_terminal",
			layout:   New().Size(20, 5),
			children: []tea.Model{M{"A"}, M{"B"}},
		},
		{
			name:     "custom_border_rounded",
			layout:   New().Size(80, 24).BorderStyle(lipgloss.RoundedBorder()),
			children: []tea.Model{M{"A"}, M{"B"}},
		},
		{
			name:     "custom_border_thick",
			layout:   New().Size(80, 24).BorderStyle(lipgloss.ThickBorder()),
			children: []tea.Model{M{"A"}, M{"B"}, M{"C"}},
		},
		{
			name:     "double_border",
			layout:   New().Size(80, 24).BorderStyle(lipgloss.DoubleBorder()),
			children: []tea.Model{M{"X"}, M{"Y"}},
		},
		{
			name:     "master_count_2",
			layout:   New().Size(80, 24).MasterCount(2),
			children: []tea.Model{M{"M1"}, M{"M2"}, M{"S1"}},
		},
		{
			name:     "four_children",
			layout:   New().Size(80, 24),
			children: []tea.Model{M{"A"}, M{"B"}, M{"C"}, M{"D"}},
		},
	}

	for _, tt := range tests {
		tt.layout.Children(tt.children...)
		t.Run(tt.name+"_horizontal", func(t *testing.T) {
			result := renderLayer(tt.layout.Split(SplitHorizontalWithMain).Layer())
			golden.RequireEqual(t, []byte(result))
		})
		t.Run(tt.name+"_vertical", func(t *testing.T) {
			result := renderLayer(tt.layout.Split(SplitVerticalWithMain).Layer())
			golden.RequireEqual(t, []byte(result))
		})
	}
}
