package main

import (
	"log"
	"os"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/Melkor333/oils-readline/tiling"
	"github.com/creack/pty"
)

// RemoveSelfMsg is a message that a child model can return from its Update
// function (via a command) to request its own removal from the parent tiling
// Model.
func widgets(m *model) map[string]func() tea.Cmd {
	return map[string]func() tea.Cmd{
		"SimplePrompt": func() tea.Cmd { return m.AddChild(newBasicPrompt(m.shells[m.shellFocus])) },
		"StdoutLog":    func() tea.Cmd { return m.AddChild(newStdoutViewer()) },
		"ErrorLog":     func() tea.Cmd { return m.AddChild(newStderrViewer()) },
	}
}

type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{} // Go to main

// Sent by a widget to request all input
func RequestCapture() tea.Cmd {
	return func() tea.Msg { return requestCaptureMsg{} }
}

type requestCaptureMsg struct{ id uint64 }

func (msg requestCaptureMsg) Tag(t uint64) tea.Msg { msg.id = t; return msg }

// Sent by a widget to stop receiving all input
func ReleaseCapture() tea.Cmd {
	return func() tea.Msg { return releaseCaptureMsg{} }
}

type releaseCaptureMsg struct{ id uint64 }

func (msg releaseCaptureMsg) Tag(t uint64) tea.Msg { msg.id = t; return msg }

type RemoveSelfMsg struct{ id uint64 }
type removeWidgetMsg struct{ id uint64 }

func (msg RemoveSelfMsg) TargetedMsg() uint64  { return msg.id }
func (msg RemoveSelfMsg) Tag(t uint64) tea.Msg { msg.id = t; return msg }

// removeChildMsg is an internal message to remove a child by its unique ID.
type removeShellMsg struct {
	id uint64
}

// Widget pairs a child model with a unique ID for stable identity tracking.
type Widget struct {
	tea.Model
	id uint64
}

type trackedShell struct {
	shell.Shell
	id uint64
}

type model struct {
	shells      []trackedShell
	shellFocus  int
	nextShellID uint64

	widgets      []Widget
	widgetFocus  int
	nextWidgetID uint64

	history []shell.Command

	layout *tiling.Layout
	Height int
	Width  int

	//highlighter Highlighter
	program *tea.Program

	selecting     bool
	selector      *SelectorWidget
	captureWidget int // index of widget capturing all keys, -1 = none
}

func NewModel(shells []shell.Shell, widgets []tea.Model) *model {
	entries := make([]Widget, len(widgets))
	s := make([]trackedShell, len(shells))
	for i, c := range widgets {
		entries[i] = Widget{c, uint64(i)}
	}
	for i, shell := range shells {
		s[i] = trackedShell{shell, uint64(i)}
	}
	return &model{
		shells:        s,
		nextShellID:   uint64(len(shells)),
		layout:        tiling.New(),
		widgets:       entries,
		nextWidgetID:  uint64(len(widgets)),
		captureWidget: -1,
	}
}

// AddChild appends a child model to the end of the layout.
// It returns the child's Init command.
func (m *model) AddChild(child tea.Model) tea.Cmd {
	id := m.nextWidgetID
	m.nextWidgetID++
	m.widgets = append(m.widgets, Widget{child, id})
	cmd := m.updateChild(len(m.widgets)-1, tea.BlurMsg{})
	return tea.Batch(m.recalculateSizes(), tea.Batch(child.Init(), cmd))
}

func (m *model) AddChildAt(pos int, child tea.Model) tea.Cmd {
	id := m.nextWidgetID
	m.nextWidgetID++
	m.widgets = slices.Insert(m.widgets, pos, Widget{child, id})
	return tea.Batch(m.recalculateSizes(), child.Init())
}

type Cancellable interface {
	Cancel()
}

func (m *model) Cancel() {
	for _, shell := range m.shells {
		shell.Cancel()
	}
	for _, w := range m.widgets {
		if c, ok := w.Model.(Cancellable); ok {
			c.Cancel()

		}
	}
}

func (m *model) AddShell(shell shell.Shell) tea.Cmd {
	id := m.nextShellID
	m.nextShellID++
	m.shells = append(m.shells, trackedShell{shell, id})
	return func() tea.Msg {
		shell.Wait()
		return removeShellMsg{uint64(id)}
	}
}

func (m *model) RemoveChild(index int) tea.Cmd {
	if index < 0 || index >= len(m.widgets) {
		return nil
	}
	if m.captureWidget == index {
		m.captureWidget -= 1
	}
	m.widgets = append(m.widgets[:index], m.widgets[index+1:]...)
	if m.widgetFocus >= len(m.widgets) {
		m.widgetFocus = len(m.widgets) - 1
	}
	return m.recalculateSizes()
}

// wrapChildCmd wraps a child's command to intercept RemoveSelfMsg and convert it
// to a removeChildMsg with the correct child ID.
func wrapChildCmd(cmd tea.Cmd, childID uint64) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if t, ok := msg.(shell.TaggedMsg); ok {
			log.Printf("Tagged message for %v!", childID)
			msg = t.Tag(childID)
		}
		return msg
	}
}

func (m *model) updateFocus(i int) (tea.Model, tea.Cmd) {
	old := m.widgetFocus
	if i >= len(m.widgets) {
		m.widgetFocus = 0
	} else if i < 0 {
		m.widgetFocus = len(m.widgets) - 1
	} else {
		m.widgetFocus = i
	}
	return m, tea.Batch(m.updateChild(old, tea.BlurMsg{}), m.updateChild(m.widgetFocus, tea.FocusMsg{}))
}

func (m *model) updateChild(i int, msg tea.Msg) tea.Cmd {
	if i < 0 || i >= len(m.widgets) { // TODO: assert?
		return nil
	}
	newM, cmd := m.widgets[i].Update(msg)
	m.widgets[i] = Widget{newM, m.widgets[i].id}
	return wrapChildCmd(cmd, m.widgets[i].id)
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i, shell := range m.shells {
		cmds = append(cmds,
			func() tea.Msg {
				shell.Wait()
				return removeShellMsg{uint64(i)}
			})
	}
	for _, c := range m.widgets {
		log.Printf("Initiating")
		cmds = append(cmds, c.Init())
	}
	return tea.Batch(cmds...)
}

func (m *model) recalculateSizes() tea.Cmd {
	sizes := m.layout.TileSizes(len(m.widgets))
	var cmds []tea.Cmd
	for i, child := range m.widgets {
		var cmd tea.Cmd
		m.updateChild(i, tea.WindowSizeMsg{
			Width:  sizes[i].W,
			Height: sizes[i].H,
		})
		cmds = append(cmds, wrapChildCmd(cmd, child.id))
	}
	return tea.Batch(cmds...)
}

func (m *model) View() tea.View {
	var views []string
	for _, child := range m.widgets {
		views = append(views, child.View().Content)
	}

	base := m.layout.Children(views...).Layer()

	if m.selecting && m.selector != nil {
		selectorContent := m.selector.View().Content
		selectorLayer := lipgloss.NewLayer(selectorContent).Z(1)
		result := lipgloss.NewCompositor(base, selectorLayer)
		v := tea.NewView(result.Render())
		v.AltScreen = true
		return v
	}

	v := tea.NewView(lipgloss.NewCompositor(base).Render())
	v.AltScreen = true
	return v
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// First switch: transform messages inline by updating msg, so the new
	// message reaches children in the same loop iteration (no extra MVU roundtrip).
	log.Printf("message: %T, %v", msg, msg)
	switch typedMsg := msg.(type) {
	case shell.RequestHistoryEntryMsg:
		if typedMsg.Index < 0 {
			msg = shell.HistoryEntryMsg{
				Cmd:   m.history[len(m.history)-1],
				Index: len(m.history) - 1,
				Total: len(m.history),
				Id:    typedMsg.Id,
			}
		} else if typedMsg.Index < len(m.history) {
			msg = shell.HistoryEntryMsg{
				Cmd:   m.history[typedMsg.Index],
				Index: typedMsg.Index,
				Total: len(m.history),
				Id:    typedMsg.Id,
			}
		} else {
			return m, nil
		}
	}

	// Second switch: handle all other cases, returning normally.
	switch msg := msg.(type) {
	case releaseCaptureMsg:
		m.captureWidget = -1
		return m, nil

	case requestCaptureMsg:
		for i, w := range m.widgets {
			if w.id == msg.id {
				m.captureWidget = i
			}
		}

	case tea.KeyPressMsg:
		if m.selecting {
			newSel, cmd := m.selector.Update(msg)
			if c, ok := newSel.(*SelectorWidget); ok {
				m.selector = c
			}
			return m, cmd
		}

		// Capture mode: all keypresses go to the capturing widget
		if m.captureWidget >= 0 && m.captureWidget < len(m.widgets) {
			log.Printf("Send capture to widget")
			return m, m.updateChild(m.captureWidget, msg)
		}

		switch msg.String() {
		case "ctrl+l":
			return m.updateFocus(m.widgetFocus + 1)
		case "ctrl+h":
			return m.updateFocus(m.widgetFocus - 1)
		case "ctrl+space":
			m.selecting = true
			m.selector = newWidgetSelector(widgets(m))
			m.selector.width = m.Width
			m.selector.height = m.Height
			return m, m.selector.Init()
		case "ctrl+c":
			if len(m.widgets) > 0 {
				log.Printf("Removing child %v", m.widgetFocus)
				return m, m.RemoveChild(m.widgetFocus)
			}
			return m, func() tea.Msg { return tea.Quit() }
		}

		// Keypresses only go to the currently focussed widget
		if len(m.widgets) > 0 {
			return m, m.updateChild(m.widgetFocus, msg)
		}
		return m, nil

	case CloseSelectorMsg:
		m.selecting = false
		m.selector = nil
		return m, nil

	case RequestFocusNextMsg:
		return m.updateFocus(m.widgetFocus + 1)
	case RequestFocusPrevMsg:
		return m.updateFocus(m.widgetFocus - 1)
	case RequestFocusMainMsg:
		return m.updateFocus(len(m.widgets))
	case removeWidgetMsg:
		for i, c := range m.widgets {
			if c.id == msg.id {
				m.RemoveChild(i)
				break
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.layout.Size(msg.Width, msg.Height)
		if m.selecting && m.selector != nil {
			m.selector.width = msg.Width
			m.selector.height = msg.Height
		}
		return m, m.recalculateSizes()
	case CommandEnteredMsg:
		// Run a command
		// TODO: make each `shell` a widget as well somehow?!
		command := msg.Text
		if len(command) == 0 {
			break // We still let widgets deal with it!
		}

		size, _ := pty.GetsizeFull(os.Stdin)
		cmd, err := m.shells[m.shellFocus].Command(command, size)
		if err != nil {
			log.Fatal("Can't create new Command!", err)
		}
		cmd.SetState(shell.Queued)

		cmd.SetOnStdout(func() { m.program.Send(shell.StdoutMsg{Cmd: cmd}) })
		cmd.SetOnStderr(func() { m.program.Send(shell.StderrMsg{Cmd: cmd}) })

		m.history = append(m.history, cmd)

		log.Print("Running command")
		return m, tea.Batch(
			func() tea.Msg { return shell.NewCommandMsg{Cmd: cmd} },
			func() tea.Msg { cmd.Run(); return shell.CommandDoneMsg{Cmd: cmd} },
		)

	// TODO: Should be cast to CommandDone?
	case tea.EnvMsg:
		log.Print("Got env from tea process")
	}

	if tmsg, ok := msg.(shell.TargetedMsg); ok {
		id := tmsg.TargetWidget()
		for i, c := range m.widgets {
			if c.id == id {
				return m, m.updateChild(i, msg)
			}
		}
		return m, nil
	}

	var cmds []tea.Cmd
	for i, child := range m.widgets {
		var cmd tea.Cmd
		cmd = m.updateChild(i, msg)
		cmds = append(cmds, wrapChildCmd(cmd, child.id))
	}
	return m, tea.Batch(cmds...)
}
