package tiling

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

type coord struct {
	x, y, w, h int
}

type M struct{ string }

func (M) Init() tea.Cmd                         { return nil }
func (m M) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m M) View() tea.View                      { return tea.NewView(m.string) }

func TestPosition(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		n    []string
		f    SplitFunc
		rect rec
		out  []coord
	}{
		// TODO: Add test cases.
		{
			"testsingle",
			[]string{"single"},
			SplitVertical,
			rec{0, 0, 10, 10},
			[]coord{{0, 0, 10, 10}},
		},
		{
			"Test two vertical",
			[]string{"one", "two"},
			SplitVertical,
			rec{0, 0, 10, 10},
			[]coord{{0, 0, 5, 10}, {6, 0, 4, 10}},
		},
		{
			"Test two horizontal",
			[]string{"one", "two"},
			SplitHorizontal,
			rec{0, 0, 10, 10},
			[]coord{{0, 0, 10, 5}, {0, 6, 10, 4}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newNode(nil, tt.f)
			for _, c := range tt.n {
				n.AddChild(M{c})
			}
			n.Position(tt.rect)
			for c, _child := range n.children {
				child := _child.rectangle
				assert.Equal(t, tt.out[c].x, child.x, "x needs to match")
				assert.Equal(t, tt.out[c].y, child.y, "y needs to match")
				assert.Equal(t, tt.out[c].w, child.width, "width needs to match")
				assert.Equal(t, tt.out[c].h, child.height, "height needs to match")
			}
		})
	}
}
