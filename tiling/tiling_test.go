package tiling

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/stretchr/testify/assert"
)

func TestDistribute(t *testing.T) {
	sizes := distribute(10, 3)
	if len(sizes) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(sizes))
	}
	total := 0
	for _, s := range sizes {
		total += s
	}
	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}
}

func TestSingleChild(t *testing.T) {
	hor := New().Size(80, 24).Split(SplitHorizontal).Children("SINGLE")
	vert := New().Size(80, 24).Split(SplitVertical).Children("SINGLE")
	golden.RequireEqual(t, []byte(hor.Render()))

	// Horizontal and Vertical should be equal with a single child
	assert.Equal(t, []byte(hor.Render()), []byte(vert.Render()))
	golden.RequireEqual(t, []byte(hor.Render()))
}

func TestLayoutEmpty(t *testing.T) {
	l := New().Size(80, 10)
	result := l.Render()
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestTileSizes(t *testing.T) {
	l := New().Size(80, 24)

	sizes := l.TileSizes(1)
	if len(sizes) != 1 || sizes[0].W != 80 || sizes[0].H != 24 {
		t.Errorf("unexpected sizes for 1 child: %v", sizes)
	}

	sizes = l.TileSizes(2)
	if len(sizes) != 2 {
		t.Fatalf("expected 2 sizes, got %d", len(sizes))
	}
	totalW := sizes[0].W + sizes[1].W
	if totalW > 80 {
		t.Errorf("total width %d exceeds 80", totalW)
	}

	sizes = l.TileSizes(3)
	if len(sizes) != 3 {
		t.Fatalf("expected 3 sizes, got %d", len(sizes))
	}
}

func TestGoldenRender(t *testing.T) {
	tests := []struct {
		name   string
		layout *Layout
	}{
		{
			name:   "2_children",
			layout: New().Size(80, 24).Children("ONE", "TWO"),
		},
		{
			name:   "3_children",
			layout: New().Size(80, 24).Children("M", "S1", "S2"),
		},
		{
			name:   "master_ratio_75",
			layout: New().Size(80, 24).MasterRatio(0.75).Children("BIG", "SMALL"),
		},
		{
			name:   "master_ratio_25",
			layout: New().Size(80, 24).MasterRatio(0.25).Children("SMALL", "BIG"),
		},
		{
			name:   "small_terminal",
			layout: New().Size(20, 5).Children("A", "B"),
		},
		{
			name:   "custom_border_rounded",
			layout: New().Size(80, 24).Border(lipgloss.RoundedBorder()).Children("A", "B"),
		},
		{
			name:   "custom_border_thick",
			layout: New().Size(80, 24).Border(lipgloss.ThickBorder()).Children("A", "B", "C"),
		},
		{
			name:   "double_border",
			layout: New().Size(80, 24).Border(lipgloss.DoubleBorder()).Children("X", "Y"),
		},
		{
			name:   "master_count_2",
			layout: New().Size(80, 24).MasterCount(2).Children("M1", "M2", "S1"),
		},
		{
			name:   "four_children",
			layout: New().Size(80, 24).Children("A", "B", "C", "D"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_horizontal", func(t *testing.T) {
			result := tt.layout.Split(SplitHorizontal).Render()
			golden.RequireEqual(t, []byte(result))
		})
		t.Run(tt.name+"_vertical", func(t *testing.T) {
			result := tt.layout.Split(SplitVertical).Render()
			golden.RequireEqual(t, []byte(result))
		})
	}
}
