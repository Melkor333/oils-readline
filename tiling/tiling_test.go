package tiling

import (
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

type blockModel struct {
	width  int
	height int
	label  string
	color  string
}

func newBlock(label, color string) *blockModel {
	return &blockModel{label: label, color: color}
}

func (m *blockModel) Init() tea.Cmd { return nil }

func (m *blockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, quitSoon()
}

func (m *blockModel) View() tea.View {
	return tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height).
		Background(lipgloss.Color(m.color)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.label))
}

func quitSoon() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tea.QuitMsg{}
	})
}

func waitForOutput(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	out := tm.Output()
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(2*time.Second))
}

func readTeatestOutput(t *testing.T, tm *teatest.TestModel) []byte {
	t.Helper()
	out := tm.Output()
	bts, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("reading teatest output: %v", err)
	}
	return bts
}

func TestSingleChild(t *testing.T) {
	m := NewModel(newBlock("A", "21"))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestTwoChildrenVertical(t *testing.T) {
	m := NewModel(
		newBlock("LEFT", "21"),
		newBlock("RIGHT", "52"),
	)
	m.SetSplitMode(SplitVertical)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestThreeChildrenVertical(t *testing.T) {
	m := NewModel(
		newBlock("M", "21"),
		newBlock("S1", "52"),
		newBlock("S2", "93"),
	)
	m.SetSplitMode(SplitVertical)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestTwoChildrenHorizontal(t *testing.T) {
	m := NewModel(
		newBlock("TOP", "21"),
		newBlock("BOT", "52"),
	)
	m.SetSplitMode(SplitHorizontal)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestMasterRatio(t *testing.T) {
	m := NewModel(
		newBlock("BIG", "21"),
		newBlock("SMALL", "52"),
	)
	m.SetMasterRatio(0.75)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestNoBorderSingleChild(t *testing.T) {
	m := NewModel(newBlock("SOLO", "21"))

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	waitForOutput(t, tm)
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	golden.RequireEqual(t, readTeatestOutput(t, tm))
}

func TestLayoutDirectRender(t *testing.T) {
	l := New().Size(80, 10).Children("AAAA", "BBBB")
	golden.RequireEqual(t, []byte(l.Render()))
}

func TestLayoutEmpty(t *testing.T) {
	l := New().Size(80, 10)
	result := l.Render()
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestLayoutSingleChild(t *testing.T) {
	l := New().Size(80, 10).Children("HELLO")
	golden.RequireEqual(t, []byte(l.Render()))
}

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

func TestLayoutThreeChildrenDirect(t *testing.T) {
	l := New().Size(80, 24).Children("M", "S1", "S2")
	golden.RequireEqual(t, []byte(l.Render()))
}

func TestLayoutHorizontalDirect(t *testing.T) {
	l := New().Size(80, 24).Split(SplitHorizontal).Children("TOP", "BOT")
	golden.RequireEqual(t, []byte(l.Render()))
}

func TestModelView(t *testing.T) {
	m := NewModel(
		newBlock("A", "21"),
		newBlock("B", "52"),
	)
	m.SetSplitMode(SplitVertical)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	golden.RequireEqual(t, []byte(updated.View().Content))
}

func TestModelViewThreeChildren(t *testing.T) {
	m := NewModel(
		newBlock("M", "21"),
		newBlock("S1", "52"),
		newBlock("S2", "93"),
	)
	m.SetSplitMode(SplitVertical)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	golden.RequireEqual(t, []byte(updated.View().Content))
}

func TestModelViewHorizontal(t *testing.T) {
	m := NewModel(
		newBlock("TOP", "21"),
		newBlock("BOT", "52"),
	)
	m.SetSplitMode(SplitHorizontal)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	golden.RequireEqual(t, []byte(updated.View().Content))
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
			name:   "single_child",
			layout: New().Size(80, 10).Children("HELLO"),
		},
		{
			name:   "vertical_2_children",
			layout: New().Size(80, 24).Children("LEFT", "RIGHT"),
		},
		{
			name:   "vertical_3_children",
			layout: New().Size(80, 24).Children("M", "S1", "S2"),
		},
		{
			name:   "horizontal_2_children",
			layout: New().Size(80, 24).Split(SplitHorizontal).Children("TOP", "BOT"),
		},
		{
			name:   "horizontal_3_children",
			layout: New().Size(80, 24).Split(SplitHorizontal).Children("M", "S1", "S2"),
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
			name:   "four_children_vertical",
			layout: New().Size(80, 24).Children("A", "B", "C", "D"),
		},
		{
			name:   "four_children_horizontal",
			layout: New().Size(80, 24).Split(SplitHorizontal).Children("A", "B", "C", "D"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.layout.Render()
			golden.RequireEqual(t, []byte(result))
		})
	}
}

func TestGoldenModelView(t *testing.T) {
	tests := []struct {
		name     string
		children []tea.Model
		split    SplitMode
		ratio    float64
		w, h     int
	}{
		{
			name:     "single_block",
			children: []tea.Model{newBlock("A", "21")},
			split:    SplitVertical,
			w:        80, h: 24,
		},
		{
			name:     "vertical_split",
			children: []tea.Model{newBlock("L", "21"), newBlock("R", "52")},
			split:    SplitVertical,
			w:        80, h: 24,
		},
		{
			name:     "horizontal_split",
			children: []tea.Model{newBlock("T", "21"), newBlock("B", "52")},
			split:    SplitHorizontal,
			w:        80, h: 24,
		},
		{
			name:     "three_children",
			children: []tea.Model{newBlock("M", "21"), newBlock("S1", "52"), newBlock("S2", "93")},
			split:    SplitVertical,
			w:        80, h: 24,
		},
		{
			name:     "wide_ratio",
			children: []tea.Model{newBlock("BIG", "21"), newBlock("SM", "52")},
			split:    SplitVertical,
			ratio:    0.75,
			w:        120, h: 30,
		},
		{
			name:     "small_size",
			children: []tea.Model{newBlock("A", "21"), newBlock("B", "52")},
			split:    SplitVertical,
			w:        40, h: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(tt.children...)
			m.SetSplitMode(tt.split)
			if tt.ratio != 0 {
				m.SetMasterRatio(tt.ratio)
			}
			updated, _ := m.Update(tea.WindowSizeMsg{Width: tt.w, Height: tt.h})
			view := updated.View()
			golden.RequireEqual(t, []byte(view.Content))
		})
	}
}

type removableModel struct {
	width  int
	height int
	label  string
}

func newRemovable(label string) *removableModel {
	return &removableModel{label: label}
}

func (m *removableModel) Init() tea.Cmd { return nil }

func (m *removableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case triggerRemoveMsg:
		return m, func() tea.Msg { return RemoveSelfMsg{} }
	}
	return m, nil
}

func (m *removableModel) View() tea.View {
	return tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height).
		Background(lipgloss.Color("21")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.label))
}

type triggerRemoveMsg struct{}

func TestChildSelfRemove(t *testing.T) {
	m := NewModel(
		newBlock("A", "21"),
		newRemovable("B"),
		newBlock("C", "93"),
	)
	m.SetSplitMode(SplitVertical)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*Model)

	if m.NumChildren() != 3 {
		t.Fatalf("expected 3 children, got %d", m.NumChildren())
	}

	// Simulate the removeChildMsg that wrapChildCmd generates when a child
	// returns RemoveSelfMsg. Children are assigned IDs 0,1,2 in order.
	updated, _ = m.Update(removeChildMsg{id: 1})
	m = updated.(*Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*Model)

	if m.NumChildren() != 2 {
		t.Fatalf("expected 2 children after removal, got %d", m.NumChildren())
	}

	fresh := NewModel(newBlock("A", "21"), newBlock("C", "93"))
	fresh.SetSplitMode(SplitVertical)
	updated2, _ := fresh.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fresh = updated2.(*Model)

	// TODO: Compare full view?
	if m.View().Content != fresh.View().Content {
		t.Errorf("view after self-removal doesn't match fresh model")
	}
}

func TestWrapChildCmd(t *testing.T) {
	removeCmd := func() tea.Msg { return RemoveSelfMsg{} }
	wrapped := wrapChildCmd(removeCmd, 42)
	msg := wrapped()
	rcm, ok := msg.(removeChildMsg)
	if !ok {
		t.Fatalf("expected removeChildMsg, got %T", msg)
	}
	if rcm.id != 42 {
		t.Errorf("expected id 42, got %d", rcm.id)
	}

	quitCmd := func() tea.Msg { return tea.QuitMsg{} }
	wrapped2 := wrapChildCmd(quitCmd, 7)
	msg2 := wrapped2()
	if _, ok := msg2.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg to pass through, got %T", msg2)
	}

	if wrapChildCmd(nil, 1) != nil {
		t.Errorf("expected nil for nil cmd")
	}
}

func TestAddRemoveChild(t *testing.T) {
	m := NewModel(newBlock("A", "21"))
	m.SetSplitMode(SplitVertical)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*Model)

	m.AddChild(newBlock("B", "52"))

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*Model)

	if m.NumChildren() != 2 {
		t.Fatalf("expected 2 children, got %d", m.NumChildren())
	}

	m.RemoveChild(0)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*Model)

	if m.NumChildren() != 1 {
		t.Fatalf("expected 1 child, got %d", m.NumChildren())
	}

	fresh := NewModel(newBlock("B", "52"))
	fresh.SetSplitMode(SplitVertical)
	updated2, _ := fresh.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fresh = updated2.(*Model)

	// TODO: Compare full view?
	if m.View().Content != fresh.View().Content {
		t.Errorf("view doesn't match expected")
	}
}

func TestPropertyAddRemoveChildren(t *testing.T) {
	labels := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	for trial := range 100 {
		t.Run(fmt.Sprintf("trial_%d", trial), func(t *testing.T) {
			rng := rand.New(rand.NewSource(int64(trial)))

			initialCount := rng.Intn(3) + 1
			var expected []string
			used := make(map[string]bool)

			for range initialCount {
				label := labels[rng.Intn(len(labels))]
				for used[label] {
					label = labels[rng.Intn(len(labels))]
				}
				used[label] = true
				expected = append(expected, label)
			}

			m := NewModel(makeModels(expected)...)
			m.SetSplitMode(SplitVertical)

			numOps := rng.Intn(10) + 1
			for range numOps {
				canAdd := len(used) < len(labels)
				canRemove := m.NumChildren() > 0

				if canAdd && (!canRemove || rng.Intn(2) == 0) {
					label := labels[rng.Intn(len(labels))]
					for used[label] {
						label = labels[rng.Intn(len(labels))]
					}
					used[label] = true
					pos := rng.Intn(m.NumChildren() + 1)
					m.AddChildAt(pos, newBlock(label, "21"))
					expected = insertStringAt(expected, pos, label)
				} else if canRemove {
					pos := rng.Intn(m.NumChildren())
					delete(used, expected[pos])
					m.RemoveChild(pos)
					expected = append(expected[:pos], expected[pos+1:]...)
				}
			}

			fresh := NewModel(makeModels(expected)...)
			fresh.SetSplitMode(SplitVertical)

			w, h := 80, 24
			updated1, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			m = updated1.(*Model)
			updated2, _ := fresh.Update(tea.WindowSizeMsg{Width: w, Height: h})
			fresh = updated2.(*Model)

			// TODO: Compare full view?
			if m.View().Content != fresh.View().Content {
				t.Errorf("views differ for children %v", expected)
			}
		})
	}
}

func makeModels(labels []string) []tea.Model {
	models := make([]tea.Model, len(labels))
	for i, l := range labels {
		models[i] = newBlock(l, "21")
	}
	return models
}

func insertStringAt(s []string, i int, v string) []string {
	s = append(s, "")
	copy(s[i+1:], s[i:])
	s[i] = v
	return s
}
