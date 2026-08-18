package widget

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type targetedMsg struct{ w *Widget }
type untargetedMsg struct{}

func (msg targetedMsg) Tag(t *Widget) tea.Msg { msg.w = t; return msg }

// --- WrapAction ---

func TestWrapAction(t *testing.T) {
	w := &Widget{}

	tests := []struct {
		name   string
		action Action
		check  func(t *testing.T, result Action)
	}{
		{
			name: "nil",
			check: func(t *testing.T, result Action) {
				assert.Nil(t, result)
			},
		},
		{
			name:   "Batch/tagged+untagged+nil",
			action: Batch{targetedMsg{}, untargetedMsg{}, nil},
			check: func(t *testing.T, result Action) {
				b, ok := result.(Batch)
				require.True(t, ok)
				require.Len(t, b, 3)
				assert.Nil(t, b[2])

				tm, ok := b[0].(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)

				_, ok = b[1].(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name:   "Batch/nested",
			action: Batch{Batch{targetedMsg{}}, untargetedMsg{}},
			check: func(t *testing.T, result Action) {
				b, ok := result.(Batch)
				require.True(t, ok)

				inner, ok := b[0].(Batch)
				require.True(t, ok)

				tm, ok := inner[0].(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name:   "Sequence/tagged+untagged+nil",
			action: Sequence{targetedMsg{}, untargetedMsg{}, nil},
			check: func(t *testing.T, result Action) {
				s, ok := result.(Sequence)
				require.True(t, ok)
				require.Len(t, s, 3)
				assert.Nil(t, s[2])

				tm, ok := s[0].(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)

				_, ok = s[1].(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name:   "Sequence/nested",
			action: Sequence{Sequence{targetedMsg{}}, untargetedMsg{}},
			check: func(t *testing.T, result Action) {
				s, ok := result.(Sequence)
				require.True(t, ok)

				inner, ok := s[0].(Sequence)
				require.True(t, ok)

				tm, ok := inner[0].(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name:   "tea.Cmd/tagged",
			action: tea.Cmd(func() tea.Msg { return targetedMsg{} }),
			check: func(t *testing.T, result Action) {
				wrapped, ok := result.(tea.Cmd)
				require.True(t, ok)

				msg := wrapped()
				tm, ok := msg.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name:   "tea.Cmd/untagged",
			action: tea.Cmd(func() tea.Msg { return untargetedMsg{} }),
			check: func(t *testing.T, result Action) {
				wrapped := result.(tea.Cmd)
				msg := wrapped()
				_, ok := msg.(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name:   "tea.Cmd/nilBody",
			action: tea.Cmd(func() tea.Msg { return nil }),
			check: func(t *testing.T, result Action) {
				wrapped := result.(tea.Cmd)
				assert.Nil(t, wrapped())
			},
		},
		{
			name:   "func() tea.Msg/aliasedAsCmd",
			action: func() tea.Msg { return targetedMsg{} },
			check: func(t *testing.T, result Action) {
				// func() tea.Msg is identical to tea.Cmd, captured as tea.Cmd
				wrapped, ok := result.(tea.Cmd)
				require.True(t, ok)

				msg := wrapped()
				tm, ok := msg.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name:   "plainMsg/tagged",
			action: targetedMsg{},
			check: func(t *testing.T, result Action) {
				tm, ok := result.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name:   "plainMsg/untagged",
			action: untargetedMsg{},
			check: func(t *testing.T, result Action) {
				_, ok := result.(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name:   "StopProcessingMsg",
			action: StopProcessingMsg{},
			check: func(t *testing.T, result Action) {
				_, ok := result.(StopProcessingMsg)
				assert.True(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, WrapAction(tt.action, w))
		})
	}
}

func TestWrapAction_differentWidget(t *testing.T) {
	w2 := &Widget{}
	result := WrapAction(targetedMsg{}, w2)
	tm, ok := result.(targetedMsg)
	require.True(t, ok)
	assert.Equal(t, w2, tm.w)
}

// --- wrapMsg ---

func TestWrapMsg(t *testing.T) {
	w := &Widget{}

	tests := []struct {
		name string
		msg  tea.Msg
		check func(t *testing.T, result tea.Msg)
	}{
		{
			name: "nil",
			check: func(t *testing.T, result tea.Msg) {
				assert.Nil(t, result)
			},
		},
		{
			name: "BatchMsg",
			msg: tea.BatchMsg{
				func() tea.Msg { return targetedMsg{} },
				func() tea.Msg { return untargetedMsg{} },
				func() tea.Msg { return nil },
			},
			check: func(t *testing.T, result tea.Msg) {
				b, ok := result.(tea.BatchMsg)
				require.True(t, ok)
				require.Len(t, b, 3)

				msg1 := b[0]()
				tm, ok := msg1.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)

				msg2 := b[1]()
				_, ok = msg2.(untargetedMsg)
				assert.True(t, ok)

				assert.Nil(t, b[2]())
			},
		},
		{
			name: "TaggedMsg",
			msg:  targetedMsg{},
			check: func(t *testing.T, result tea.Msg) {
				tm, ok := result.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name: "plainMsg",
			msg:  untargetedMsg{},
			check: func(t *testing.T, result tea.Msg) {
				_, ok := result.(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name: "StopProcessingMsg",
			msg:  StopProcessingMsg{},
			check: func(t *testing.T, result tea.Msg) {
				_, ok := result.(StopProcessingMsg)
				assert.True(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, wrapMsg(tt.msg, w))
		})
	}
}

// --- wrapTeaCmd ---

func TestWrapTeaCmd(t *testing.T) {
	w := &Widget{}

	tests := []struct {
		name string
		cmd  tea.Cmd
		check func(t *testing.T, wrapped tea.Cmd)
	}{
		{
			name: "nil",
			check: func(t *testing.T, wrapped tea.Cmd) {
				assert.Nil(t, wrapped)
			},
		},
		{
			name: "tagged",
			cmd:  func() tea.Msg { return targetedMsg{} },
			check: func(t *testing.T, wrapped tea.Cmd) {
				msg := wrapped()
				tm, ok := msg.(targetedMsg)
				require.True(t, ok)
				assert.Equal(t, w, tm.w)
			},
		},
		{
			name: "untagged",
			cmd:  func() tea.Msg { return untargetedMsg{} },
			check: func(t *testing.T, wrapped tea.Cmd) {
				msg := wrapped()
				_, ok := msg.(untargetedMsg)
				assert.True(t, ok)
			},
		},
		{
			name: "nilReturn",
			cmd:  func() tea.Msg { return nil },
			check: func(t *testing.T, wrapped tea.Cmd) {
				assert.Nil(t, wrapped())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, wrapTeaCmd(tt.cmd, w))
		})
	}
}

// --- Widget.Init ---

func TestWidget_Init_wrapsAction(t *testing.T) {
	w := &Widget{Model: &testModel{initAction: targetedMsg{}}}
	result := w.Init()
	tm, ok := result.(targetedMsg)
	require.True(t, ok)
	assert.Equal(t, w, tm.w)
}

func TestWidget_Init_nil(t *testing.T) {
	w := &Widget{Model: &testModel{initAction: nil}}
	assert.Nil(t, w.Init())
}

func TestWidget_Init_Batch(t *testing.T) {
	w := &Widget{Model: &testModel{initAction: Batch{targetedMsg{}, untargetedMsg{}}}}
	result := w.Init()
	b, ok := result.(Batch)
	require.True(t, ok)
	tm, ok := b[0].(targetedMsg)
	require.True(t, ok)
	assert.Equal(t, w, tm.w)
}

// --- Widget.Update ---

func TestWidget_Update_wrapsAction(t *testing.T) {
	w := &Widget{Model: &testModel{updateAction: targetedMsg{}}}
	result := w.Update(untargetedMsg{})
	tm, ok := result.(targetedMsg)
	require.True(t, ok)
	assert.Equal(t, w, tm.w)
}

func TestWidget_Update_nil(t *testing.T) {
	assert.Nil(t, (*Widget)(nil).Update(untargetedMsg{}))
}

func TestWidget_Update_nilAction(t *testing.T) {
	w := &Widget{Model: &testModel{updateAction: nil}}
	assert.Nil(t, w.Update(untargetedMsg{}))
}

func TestWidget_Update_untaggedMsg(t *testing.T) {
	w := &Widget{Model: &testModel{updateAction: untargetedMsg{}}}
	result := w.Update(untargetedMsg{})
	_, ok := result.(untargetedMsg)
	assert.True(t, ok)
}

// --- helpers ---

type testModel struct {
	initAction   Action
	updateAction Action
	name         string
}

func (m *testModel) Init() Action                   { return m.initAction }
func (m *testModel) View() tea.View                 { return tea.View{} }
func (m *testModel) Update(tea.Msg) (Model, Action) { return m, m.updateAction }
func (m *testModel) Name() string                   { return m.name }