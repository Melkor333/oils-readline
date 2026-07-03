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

type TaggedMsg interface {
	Tag(w *Widget) tea.Msg
}

type TargetedMsg interface {
	TargetWidget() *Widget
}

type RequestFocusPrevMsg struct{}
type RequestFocusNextMsg struct{}
type RequestFocusMainMsg struct{} // Go to main

// Sent by a widget to request all input
func RequestCapture() tea.Cmd {
	return func() tea.Msg { return requestCaptureMsg{} }
}

type requestCaptureMsg struct{ w *Widget }

func (msg requestCaptureMsg) Tag(w *Widget) tea.Msg { msg.w = w; return msg }

// Sent by a widget to stop receiving all input
func ReleaseCapture() tea.Cmd {
	return func() tea.Msg { return releaseCaptureMsg{} }
}

type releaseCaptureMsg struct{ w *Widget }

func (msg releaseCaptureMsg) Tag(w *Widget) tea.Msg { msg.w = w; return msg }

type RemoveSelfMsg struct{ w *Widget }
type removeWidgetMsg struct{ w *Widget }

func (msg RemoveSelfMsg) TargetedMsg() *Widget  { return msg.w }
func (msg RemoveSelfMsg) Tag(t *Widget) tea.Msg { msg.w = t; return msg }

// removeChildMsg is an internal message to remove a child by its unique ID.
type removeShellMsg struct {
	s shell.Shell
}

// Widget pairs a child model with a unique ID for stable identity tracking.
type Widget struct {
	tea.Model
}

type trackedShell struct {
	shell.Shell
	id uint64
}

type model struct {
	shells      []trackedShell
	shellFocus  int
	nextShellID uint64

	widgets     []*Widget
	widgetFocus *Widget

	history      []shell.Command
	historyIndex int // -1 = not viewing history; 0..len-1 = viewing specific entry

	layout *tiling.Layout
	Height int
	Width  int

	//highlighter Highlighter
	program *tea.Program

	selecting     bool
	selector      *SelectorWidget
	captureWidget *Widget // index of widget capturing all keys, -1 = none
}

func NewModel(shells []shell.Shell, children []tea.Model) *model {
	entries := make([]*Widget, len(children))
	s := make([]trackedShell, len(shells))
	layout := tiling.New()
	for i, c := range children {
		w := &Widget{c}
		entries[i] = w
		layout.Children(w)
	}
	for i, shell := range shells {
		s[i] = trackedShell{shell, uint64(i)}
	}
	m := &model{
		shells:        s,
		nextShellID:   uint64(len(shells)),
		layout:        layout,
		widgets:       entries,
		captureWidget: nil,
	}
	if len(children) > 0 {
		m.widgetFocus = m.widgets[0]
	}
	return m
}

// AddChild appends a child model to the end of the layout.
// It returns the child's Init command.
func (m *model) AddChild(child tea.Model) tea.Cmd {
	return m.AddChildAt(len(m.widgets), child)
}

func (m *model) AddChildAt(pos int, child tea.Model) tea.Cmd {
	w := &Widget{child}
	m.widgets = slices.Insert(m.widgets, pos, w)

	// Make the first added widget focussed
	if len(m.widgets) == 1 {
		m.widgetFocus = w
	}
	m.layout.AddChildAt(pos, w)
	_, cmd := w.Update(tea.BlurMsg{})
	return tea.Batch(m.recalculateSizes(), tea.Batch(wrapChildCmd(w.Model.Init(), w), cmd))
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
		return removeShellMsg{shell}
	}
}

func (m *model) RemoveChild(w *Widget) tea.Cmd {
	if m.captureWidget == w {
		m.captureWidget = nil
	}
	for i, ww := range m.widgets {
		if w == ww {
			// Go to previous widget
			if m.widgetFocus == w {
				if len(m.widgets) > 0 {
					m.widgetFocus = m.widgets[max(i-1, 0)]
				} else {
					// In case there is no widget left
					m.widgetFocus = nil
				}
			}
			m.widgets = append(m.widgets[:i], m.widgets[i+1:]...)
		}
	}
	m.layout.RemoveChild(w)
	return m.recalculateSizes()
}

// wrapChildCmd wraps a child's command to intercept RemoveSelfMsg and convert it
// to a removeChildMsg with the correct child ID.
func wrapChildCmd(cmd tea.Cmd, w *Widget) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if t, ok := msg.(TaggedMsg); ok {
			log.Printf("Tagged message for %v!", w)
			msg = t.Tag(w)
		}
		return msg
	}
}

func (m *model) updateFocus(i int) (tea.Model, tea.Cmd) {
	old := m.widgetFocus
	if i >= len(m.widgets) {
		m.widgetFocus = m.widgets[0]
	} else if i < 0 {
		m.widgetFocus = m.widgets[len(m.widgets)-1]
	} else {
		m.widgetFocus = m.widgets[i]
	}
	_, focusCmd := m.widgetFocus.Update(tea.FocusMsg{})
	_, blurCmd := old.Update(tea.BlurMsg{})
	return m, tea.Sequence(blurCmd, focusCmd)
}

func (w *Widget) Update(msg tea.Msg) (m tea.Model, cmd tea.Cmd) {
	// TODO: Is it ever possible that `w` is nil??
	if w == nil {
		return nil, nil
	}
	w.Model, cmd = w.Model.Update(msg)
	return w, wrapChildCmd(cmd, w)
}

func (m *model) broadcastCommand(cmd shell.Command) tea.Cmd {
	msg := shell.CommandMsg{Cmd: cmd}
	var cmds []tea.Cmd
	for _, w := range m.widgets {
		_, cmd := w.Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *model) nextHistory() tea.Cmd {
	if len(m.history) == 0 {
		return nil
	}
	if m.historyIndex < 0 || m.historyIndex >= len(m.history)-1 {
		// Already at newest or not viewing history — stay at newest
		return nil
	}
	m.historyIndex++
	return m.broadcastCommand(m.history[m.historyIndex])
}

func (m *model) prevHistory() tea.Cmd {
	if len(m.history) == 0 {
		return nil
	}
	if m.historyIndex < 0 {
		m.historyIndex = len(m.history) - 1
	} else if m.historyIndex > 0 {
		m.historyIndex--
	} else {
		// Already at oldest entry
		return nil
	}
	return m.broadcastCommand(m.history[m.historyIndex])
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, shell := range m.shells {
		cmds = append(cmds,
			func() tea.Msg {
				shell.Wait()
				return removeShellMsg{shell}
			})
	}
	for _, w := range m.widgets {
		log.Printf("Initiating")
		cmds = append(cmds, wrapChildCmd(w.Model.Init(), w))
		if w == m.widgetFocus {
			_, cmd := w.Update(tea.FocusMsg{})
			cmds = append(cmds, cmd)
		} else {
			_, cmd := w.Update(tea.BlurMsg{})
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *model) recalculateSizes() tea.Cmd {
	return nil
	//sizes := m.layout.TileSizes(len(m.widgets))
	//var cmds []tea.Cmd
	//for i, child := range m.widgets {
	//	var cmd tea.Cmd
	//	m.updateChild(i, tea.WindowSizeMsg{
	//		Width:  sizes[i].W,
	//		Height: sizes[i].H,
	//	})
	//	cmds = append(cmds, wrapChildCmd(cmd, child.id))
	//}
	//return tea.Batch(cmds...)
}

func (m *model) View() tea.View {
	var views []string
	for _, child := range m.widgets {
		views = append(views, child.View().Content)
	}

	// TODO
	//m.layout.Focus(m.widgetFocus)
	base := m.layout.Layer()

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
		m.captureWidget = nil
		return m, nil

	case requestCaptureMsg:
		m.captureWidget = msg.w
		return m, nil

	case tea.KeyPressMsg:
		if m.selecting {
			newSel, cmd := m.selector.Update(msg)
			if c, ok := newSel.(*SelectorWidget); ok {
				m.selector = c
			}
			return m, cmd
		}

		// Capture mode: all keypresses go to the capturing widget
		if m.captureWidget != nil {
			log.Printf("Send capture to widget")
			_, cmd := m.captureWidget.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "esc":
			m.historyIndex = -1
			return m, nil
		case "ctrl+j":
			for i, w := range m.widgets {
				if w == m.widgetFocus {
					return m.updateFocus(i + 1)
				}
			}
			return m.updateFocus(0)
		case "ctrl+k":
			for i, w := range m.widgets {
				if w == m.widgetFocus {
					return m.updateFocus(i - 1)
				}
			}
			return m.updateFocus(0)
		case "ctrl+l":
			return m, m.nextHistory()
		case "ctrl+h":
			return m, m.prevHistory()
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
			_, cmd := m.widgetFocus.Update(msg)
			return m, cmd
		}
		return m, nil

	case CloseSelectorMsg:
		m.selecting = false
		m.selector = nil
		return m, nil

	case RequestFocusNextMsg:
		for i, w := range m.widgets {
			if w == m.widgetFocus {
				return m.updateFocus(i + 1)
			}
		}
		return m.updateFocus(0)
	case RequestFocusPrevMsg:
		for i, w := range m.widgets {
			if w == m.widgetFocus {
				return m.updateFocus(i - 1)
			}
		}
		return m.updateFocus(0)
	case RequestFocusMainMsg:
		return m.updateFocus(len(m.widgets))
	case removeWidgetMsg:
		m.RemoveChild(msg.w)
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
		m.historyIndex = len(m.history) - 1

		log.Print("Running command")
		return m, tea.Batch(
			func() tea.Msg { return shell.CommandMsg{Cmd: cmd} },
			func() tea.Msg { cmd.Run(); return shell.CommandDoneMsg{Cmd: cmd} },
		)

	case tea.EnvMsg:
		log.Print("Got env from tea process")
	}

	// TODO: This means as long as a targetedCmd runs, the widget will still exist, even when deleted from the view?!
	// Maybe we need a way to ensure a deleted widget is not being updated anymore? :thinking:
	// Or do we not care at all? Probably
	if tmsg, ok := msg.(TargetedMsg); ok {
		_, cmd := tmsg.TargetWidget().Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
	for _, child := range m.widgets {
		_, cmd := child.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
