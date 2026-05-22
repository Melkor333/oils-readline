package tiling

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// RemoveSelfMsg is a message that a child model can return from its Update
// function (via a command) to request its own removal from the parent tiling
// Model.
type RemoveSelfMsg struct{}

type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{} // Go to main
type BlurMsg struct{}
type FocusMsg struct{}

// removeChildMsg is an internal message to remove a child by its unique ID.
type removeChildMsg struct {
	id uint64
}

// childEntry pairs a child model with a unique ID for stable identity tracking.
type childEntry struct {
	tea.Model
	id uint64
}

// Model is a bubbletea Model that renders a tiling layout of child models.
type Model struct {
	layout   *Layout
	focus    int
	children []childEntry
	nextID   uint64
	width    int
	height   int
}

// NewModel creates a new tiling Model wrapping the given child models.
func NewModel(children ...tea.Model) *Model {
	entries := make([]childEntry, len(children))
	for i, c := range children {
		entries[i] = childEntry{c, uint64(i)}
	}
	return &Model{
		layout:   New(),
		children: entries,
		nextID:   uint64(len(children)),
	}
}

// AddChild appends a child model to the end of the layout.
// It returns the child's Init command.
func (m *Model) AddChild(child tea.Model) tea.Cmd {
	id := m.nextID
	m.nextID++
	m.children = append(m.children, childEntry{child, id})
	return child.Init()
}

// AddChildAt inserts a child model at the given index, shifting existing
// children. It returns the child's Init command, or nil if the index is out of
// range.
func (m *Model) AddChildAt(index int, child tea.Model) tea.Cmd {
	if index < 0 || index > len(m.children) {
		return nil
	}
	id := m.nextID
	m.nextID++
	m.children = append(m.children, childEntry{})
	copy(m.children[index+1:], m.children[index:])
	m.children[index] = childEntry{child, id}
	return child.Init()
}

// RemoveChild removes the child at the given index.
func (m *Model) RemoveChild(index int) {
	if index < 0 || index >= len(m.children) {
		return
	}
	m.children = append(m.children[:index], m.children[index+1:]...)
}

// NumChildren returns the number of children.
func (m *Model) NumChildren() int {
	return len(m.children)
}

func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, c := range m.children {
		cmds = append(cmds, c.Init())
	}
	return tea.Batch(cmds...)
}

func (m *Model) updateChild(i int, msg tea.Msg) (cmd tea.Cmd) {
	newM, cmd := m.children[i].Update(msg)
	m.children[i] = childEntry{newM, m.children[i].id}
	return
}

func (m *Model) updateFocus(i int) (tea.Model, tea.Cmd) {
	old := m.focus
	if i >= len(m.children) {
		m.focus = 0
	} else if i < 0 {
		m.focus = len(m.children) - 1
	} else {
		m.focus = i
	}
	return m, tea.Batch(m.updateChild(old, BlurMsg{}), m.updateChild(m.focus, FocusMsg{}))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// TODO: This should be a "widget manager"
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+space":
			return m.updateFocus(m.focus + 1)
		case "esc":
			return m.updateFocus(len(m.children) - 1)
		}
		return m, m.updateChild(m.focus, msg)

	case RequestFocusNextMsg:
		return m.updateFocus(m.focus + 1)
	case RequestFocusPrevMsg:
		return m.updateFocus(m.focus - 1)
	case RequestFocusMainMsg:
		return m.updateFocus(len(m.children))
	case removeChildMsg:
		for i, c := range m.children {
			if c.id == msg.id {
				m.RemoveChild(i)
				break
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout.Size(msg.Width, msg.Height)

		sizes := m.layout.TileSizes(len(m.children))
		var cmds []tea.Cmd
		for i, child := range m.children {
			var cmd tea.Cmd
			m.updateChild(i, tea.WindowSizeMsg{
				Width:  sizes[i].W,
				Height: sizes[i].H,
			})
			cmds = append(cmds, wrapChildCmd(cmd, child.id))
		}
		return m, tea.Batch(cmds...)
	}

	var cmds []tea.Cmd
	for i, child := range m.children {
		var cmd tea.Cmd
		cmd = m.updateChild(i, msg)
		cmds = append(cmds, wrapChildCmd(cmd, child.id))
	}
	return m, tea.Batch(cmds...)
}

// wrapChildCmd wraps a child's command to intercept RemoveSelfMsg and convert it
// to a removeChildMsg with the correct child ID.
func wrapChildCmd(cmd tea.Cmd, childID uint64) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if _, ok := msg.(RemoveSelfMsg); ok {
			return removeChildMsg{id: childID}
		}
		return msg
	}
}

func (m *Model) View() tea.View {
	var views []string
	for _, child := range m.children {
		views = append(views, child.View().Content)
	}

	// TODO: use tiling layout instead
	return tea.NewView(m.layout.
		Children(views...).
		Render())
}

// SetBorder sets the border style on the underlying layout.
func (m *Model) SetBorder(b lipgloss.Border, style lipgloss.Style) {
	m.layout.Border(b).BorderStyle(style)
}

// SetMasterRatio sets the master area ratio.
func (m *Model) SetMasterRatio(r float64) {
	m.layout.MasterRatio(r)
}

// SetSplitMode sets the split mode.
func (m *Model) SetSplitMode(mode SplitMode) {
	m.layout.Split(mode)
}
