package tiling

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/widget"
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
	hor, _ := New(80, 24).Split(SplitHorizontal).AddChildren(0, &widget.Widget{M{"SINGLE"}})
	vert, _ := New(80, 24).Split(SplitVertical).AddChildren(0, &widget.Widget{M{"SINGLE"}})
	horStr := renderLayer(hor.RenderLayer())
	vertStr := renderLayer(vert.RenderLayer())
	golden.RequireEqual(t, []byte(horStr))

	// Horizontal and Vertical should be equal with a single child
	assert.Equal(t, []byte(horStr), []byte(vertStr))
	golden.RequireEqual(t, []byte(horStr))
}

func TestLayoutEmpty(t *testing.T) {
	l := New(10, 10)
	result := renderLayer(l.RenderLayer())
	assert.Equal(t, "          \n          \n          \n          \n          \n          \n          \n          \n          \n          ", result, "should only be newlines and spaces.")
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
		children []*widget.Widget
	}{
		{
			name:     "2_children",
			layout:   New(80, 24),
			children: []*widget.Widget{{M{"ONE"}}, {M{"TWO"}}},
		},
		{
			name:     "3_children",
			layout:   New(80, 24),
			children: []*widget.Widget{{M{"M"}}, {M{"S1"}}, {M{"S2"}}},
		},
		{
			name:     "master_ratio_75",
			layout:   New(80, 24).MasterRatio(0.75),
			children: []*widget.Widget{{M{"BIG"}}, {M{"SMALL"}}},
		},
		{
			name:     "master_ratio_25",
			layout:   New(80, 24).MasterRatio(0.25),
			children: []*widget.Widget{{M{"SMALL"}}, {M{"BIG"}}},
		},
		{
			name:     "small_terminal",
			layout:   New(20, 5),
			children: []*widget.Widget{{M{"A"}}, {M{"B"}}},
		},
		{
			name:     "custom_border_rounded",
			layout:   New(80, 24).BorderStyle(lipgloss.RoundedBorder()),
			children: []*widget.Widget{{M{"A"}}, {M{"B"}}},
		},
		{
			name:     "custom_border_thick",
			layout:   New(80, 24).BorderStyle(lipgloss.ThickBorder()),
			children: []*widget.Widget{{M{"A"}}, {M{"B"}}, {M{"C"}}},
		},
		{
			name:     "double_border",
			layout:   New(80, 24).BorderStyle(lipgloss.DoubleBorder()),
			children: []*widget.Widget{{M{"X"}}, {M{"Y"}}},
		},
		{
			name:     "master_count_2",
			layout:   New(80, 24).MasterCount(2),
			children: []*widget.Widget{{M{"M1"}}, {M{"M2"}}, {M{"S1"}}},
		},
		{
			name:     "four_children",
			layout:   New(80, 24),
			children: []*widget.Widget{{M{"A"}}, {M{"B"}}, {M{"C"}}, {M{"D"}}},
		},
	}

	for _, tt := range tests {
		tt.layout.AddChildren(0, tt.children...)
		t.Run(tt.name+"_horizontal", func(t *testing.T) {
			result := renderLayer(tt.layout.Split(SplitHorizontalWithMain).RenderLayer())
			golden.RequireEqual(t, []byte(result))
		})
		t.Run(tt.name+"_vertical", func(t *testing.T) {
			result := renderLayer(tt.layout.Split(SplitVerticalWithMain).RenderLayer())
			golden.RequireEqual(t, []byte(result))
		})
	}
}
