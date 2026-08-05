package main

import (
	"log"
	"os"
	"reflect"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/history"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/Melkor333/oils-readline/tiling"
	"github.com/Melkor333/oils-readline/widget"
	"github.com/creack/pty"
)

// RemoveSelfMsg is a message that a child model can return from its Update
// function (via a command) to request its own removal from the parent tiling
// Model.
func widgets(m *model) map[string]func() tea.Cmd {
	return map[string]func() tea.Cmd{
		"SimplePrompt": func() tea.Cmd { return AddWidget(newBasicPrompt(m.shells[m.shellFocus])) },
		"StdoutLog":    func() tea.Cmd { return AddWidget(newStdoutViewer()) },
		"ErrorLog":     func() tea.Cmd { return AddWidget(newStderrViewer()) },
		"Terminal":     func() tea.Cmd { return AddWidget(newTerminal()) },
	}
}

type RemoveSelfMsg struct{ w *widget.Widget }
type removeWidgetMsg struct{ w *widget.Widget }

func (msg RemoveSelfMsg) TargetedMsg() *widget.Widget  { return msg.w }
func (msg RemoveSelfMsg) Tag(t *widget.Widget) tea.Msg { msg.w = t; return msg }

// removeChildMsg is an internal message to remove a child by its unique ID.
type removeShellMsg struct {
	s shell.Shell
}

type trackedShell struct {
	shell.Shell
	id uint64
}

type model struct {
	shells      []trackedShell
	shellFocus  int
	nextShellID uint64

	widgets []*widget.Widget

	history *history.History

	layout *tiling.Layout

	Height int
	Width  int

	//highlighter Highlighter
	program *tea.Program

	selecting     bool
	selector      *SelectorWidget
	captureWidget *widget.Widget // index of widget capturing all keys, -1 = none
}

func NewModel(shells []shell.Shell, children []tea.Model) *model {
	entries := make([]*widget.Widget, len(children))
	s := make([]trackedShell, len(shells))
	layout := tiling.New()
	for i, c := range children {
		w := &widget.Widget{Model: c}
		entries[i] = w
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
		history:       &history.History{},
	}
	return m
}

// addWidget appends a child model to the widget list (but not the layout).
// It returns the child's Init command.
func (m *model) addWidget(w *widget.Widget) tea.Cmd {
	m.widgets = append(m.widgets, w)
	// TODO: Should the model itself care about the init, or is init dependant of it being added to the model?
	return w.Init()
}

// AddChild appends a child model to the end of the widget list and the layout.
// It returns the child's Init command.
func (m *model) AddChild(child tea.Model) tea.Cmd {
	w := &widget.Widget{child}
	m.widgets = append(m.widgets, w)

	return nil
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

func (m *model) RemoveChild(w *widget.Widget) tea.Cmd {
	if m.captureWidget == w {
		m.captureWidget = nil
	}
	for i, ww := range m.widgets {
		if w == ww {
			m.widgets = append(m.widgets[:i], m.widgets[i+1:]...)
		}
	}
	// TODO: Should it be removed from the layout? Or are these two separate things?
	// A widget should probably remove itself when it's hidden
	// (TODO: being hidden/displayed should spawn a message to the widget? And then it removes itself... Do we want to make it that hard for widgets?)
	return tea.Batch(m.layout.RemoveChild(w), m.recalculateSizes())
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	var shellCmds []tea.Cmd

	for _, shell := range m.shells {
		shellCmds = append(shellCmds,
			func() tea.Msg {
				shell.Wait()
				return removeShellMsg{shell}
			})
	}

	for r, w := range m.widgets {
		log.Printf("Initiating %v %v", r, w)
		cmds = append(cmds, w.Init())
	}
	cmds = append(cmds,
		func() tea.Msg { log.Print("request focus"); return tiling.RequestFocusMainMsg{} },
	)

	// Focus the first widget after init
	return tea.Batch(tea.Batch(shellCmds...), tea.Sequence(cmds...))
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
		v := child.View().Content
		if v != "" {
			views = append(views, v)
		}
	}

	base := m.layout.RenderLayer()

	if m.selecting && m.selector != nil {
		selectorContent := m.selector.View().Content
		selectorLayer := lipgloss.NewLayer(selectorContent).Z(100)
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
	var cmd tea.Cmd
	log.Print(reflect.TypeOf(msg), msg)

	if m.selecting {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			newSel, cmd := m.selector.Update(msg)
			if c, ok := newSel.(*SelectorWidget); ok {
				m.selector = c
			}
			return m, cmd
		}
	}

	// Capture mode: all keypresses go to the capturing widget, bypass dispatch
	if m.captureWidget != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			log.Printf("Send capture to widget")
			_, cmd := m.captureWidget.Update(msg)
			return m, cmd
		}
	}

	// TODO: This means as long as a targetedCmd runs, the widget will still exist, even when deleted from the view?!
	// Maybe we need a way to ensure a deleted widget is not being updated anymore? :thinking:
	if tmsg, ok := msg.(TargetedMsg); ok {
		_, cmd := tmsg.TargetWidget().Update(msg)
		return m, cmd
	}

	msg, cmd = m.history.Dispatch(msg)
	if msg == nil {
		return m, cmd
	}

	// in case history/layout changed thr msg.
	if tmsg, ok := msg.(TargetedMsg); ok {
		_, cmd := tmsg.TargetWidget().Update(msg)
		return m, cmd
	}

	// Second switch: handle all other cases, returning normally.
	switch msg := msg.(type) {
	case releaseCaptureMsg:
		m.captureWidget = nil
		return m, nil

	case requestCaptureMsg:
		m.captureWidget = msg.Widget
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+space":
			m.selecting = true
			m.selector = newWidgetSelector(widgets(m))
			m.selector.width = m.Width
			m.selector.height = m.Height
			return m, m.selector.Init()
		}

	case CloseSelectorMsg:
		m.selecting = false
		m.selector = nil
		return m, nil

	case addWidgetMsg:
		w := &widget.Widget{Model: msg.Model}
		return m, m.addWidget(w)

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

		m.history.Add(cmd)

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
	if tmsg, ok := msg.(TargetedMsg); ok {
		_, cmd := tmsg.TargetWidget().Update(msg)
		return m, cmd
	}

	msg, cmd2 := m.layout.Dispatch(msg)
	cmd = tea.Batch(cmd, cmd2)
	if msg == nil {
		return m, cmd
	}

	var cmds []tea.Cmd
	for _, child := range m.widgets {
		_, cmd := child.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
