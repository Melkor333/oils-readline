package main

import (
	"fmt"
	"log"
	"log/slog"
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

func widgets(m *model) map[string]func() widget.Action {
	return map[string]func() widget.Action{
		"SimplePrompt": func() widget.Action {
			return AddWidget(&widget.Widget{Model: newBasicPrompt(m.shells[m.shellFocus].Shell)})
		},
		"StdoutLog": func() widget.Action {
			return AddWidget(&widget.Widget{Model: newStdoutViewer()})
		},
		"ErrorLog": func() widget.Action {
			return AddWidget(&widget.Widget{Model: newStderrViewer()})
		},
		"Terminal": func() widget.Action {
			return AddWidget(&widget.Widget{Model: newTerminal()})
		},
	}
}

type removeWidgetMsg struct{ w *widget.Widget }

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

	widgets  []*widget.Widget
	registry *widget.Registry

	// commandWidgets indexes the widgets spawned per command (the
	// per-command, ephemeral scope). The flat widgets slice remains the
	// runtime truth for dispatch and rendering.
	commandWidgets map[shell.Command]*commandGroup

	history *history.History

	layout *tiling.Layout

	Height int
	Width  int

	//highlighter Highlighter
	program *tea.Program

	selecting     bool
	selector      *SelectorWidget
	captureWidget *widget.Widget // index of widget capturing all keys, -1 = none

	groupSeq int // spawn order counter for command groups
}

// commandGroup is the set of widgets spawned for one command, plus the
// flag marking that the group has displayed itself at least once (which
// triggers the lazy teardown of older groups of the same shell).
type commandGroup struct {
	widgets   []*widget.Widget
	displayed bool
	seq       int
}

func NewModel(shells []shell.Shell, registry *widget.Registry) *model {
	s := make([]trackedShell, len(shells))
	layout := tiling.New(200, 200)
	for i, shell := range shells {
		s[i] = trackedShell{shell, uint64(i)}
	}
	m := &model{
		shells:         s,
		nextShellID:    uint64(len(shells)),
		layout:         layout,
		registry:       registry,
		commandWidgets: make(map[shell.Command]*commandGroup),
		captureWidget:  nil,
		history:        &history.History{},
	}
	if registry != nil {
		for _, sh := range s {
			for _, g := range registry.CreateGlobals(sh.Shell) {
				m.widgets = append(m.widgets, &widget.Widget{Model: g})
			}
		}
	}
	return m
}

// addWidget appends a child model to the widget list (but not the layout).
// It returns the child's Init command.
func (m *model) addWidget(w *widget.Widget, msgs []tea.Msg) widget.Action {
	var cmds []widget.Action
	m.widgets = append(m.widgets, w)
	// TODO: Should the model itself care about the init, or is init dependant of it being added to the model?
	cmds = append(cmds, w.Init())
	for _, msg := range msgs {
		msg := w.Update(msg)
		cmds = append(cmds, msg)
	}
	return widget.Batch(cmds)
}

// AddChild appends a child model to the end of the widget list and the layout.
// It returns the child's Init command.
func (m *model) AddChild(child widget.Model) widget.Action {
	w := &widget.Widget{Model: child}
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

// RemoveChild is the single choke point for removing a widget. It notifies
// the widget (so it can release goroutines and other resources), releases
// keyboard capture if needed, and removes the widget from the widget list,
// the command index and the layout.
func (m *model) RemoveChild(w *widget.Widget) widget.Action {
	if m.captureWidget == w {
		m.captureWidget = nil
	}
	var remove widget.Action
	if r, ok := w.Model.(widget.Teardownable); ok {
		remove = r.Teardown()
	}
	for i, ww := range m.widgets {
		if w == ww {
			m.widgets = append(m.widgets[:i], m.widgets[i+1:]...)
			break
		}
	}
	for cmd, group := range m.commandWidgets {
		for i, ww := range group.widgets {
			if w == ww {
				group.widgets = append(group.widgets[:i], group.widgets[i+1:]...)
				break
			}
		}
		if len(group.widgets) == 0 {
			delete(m.commandWidgets, cmd)
		}
	}
	return widget.Batch{m.layout.RemoveChild(w), m.recalculateSizes(), remove}
}

// spawnCommandGroup creates the per-command widget group via the registry,
// indexes it and binds it to the command. It returns the widgets' init
// actions.
func (m *model) spawnCommandGroup(cmd shell.Command) widget.Action {
	group := &commandGroup{seq: m.groupSeq}
	m.groupSeq++
	m.commandWidgets[cmd] = group
	if m.registry == nil {
		return nil
	}
	var addCmds widget.Batch
	for _, cw := range m.registry.CreateForCommand(cmd) {
		w := &widget.Widget{Model: cw}
		group.widgets = append(group.widgets, w)
		addCmds = append(addCmds, m.addWidget(w, []tea.Msg{shell.CommandMsg{Cmd: cmd}}))
	}
	return addCmds
}

// commandForWidget finds the command and group a widget belongs to.
func (m *model) commandForWidget(w *widget.Widget) (shell.Command, *commandGroup, bool) {
	for cmd, group := range m.commandWidgets {
		for _, ww := range group.widgets {
			if w == ww {
				return cmd, group, true
			}
		}
	}
	return nil, nil, false
}

// removeOtherCommandGroups tears down all command groups of the shell that
// were spawned before cmd's group, once cmd's group displays itself for the
// first time. It is the lazy counterpart of D1: old output stays visible
// until there is genuinely new output to show.
func (m *model) removeOtherCommandGroups(cmd shell.Command) widget.Action {
	own, ok := m.commandWidgets[cmd]
	if !ok {
		return nil
	}
	var stale []*widget.Widget
	for otherCmd, group := range m.commandWidgets {
		if otherCmd == cmd || group.seq >= own.seq || otherCmd.Shell() != cmd.Shell() {
			continue
		}
		stale = append(stale, group.widgets...)
	}
	var actions widget.Batch
	for _, w := range stale {
		actions = append(actions, m.RemoveChild(w))
	}
	return actions
}

func (m *model) Init() tea.Cmd {
	var actions widget.Sequence
	var shellCmds []tea.Cmd

	for _, shell := range m.shells {
		shellCmds = append(shellCmds,
			func() tea.Msg {
				shell.Wait()
				return removeShellMsg{shell.Shell}
			})
	}

	for _, w := range m.widgets {
		log.Printf("Initiating widget: %v", w.Name())
		actions = append(actions, w.Init())
	}
	actions = append(actions, tiling.RequestFocusMainMsg{})
	return tea.Batch(tea.Batch(shellCmds...), m.update(actions))
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

func (m *model) update(action widget.Action) tea.Cmd {
	for {
		if action == nil {
			return nil
		}
		slog.Debug("model manages action", "action", fmt.Sprintf("%T", action))
		if sequence, ok := action.(widget.Sequence); ok {
			slog.Debug("captured Sequence")
			var cmds []tea.Cmd
			for _, _action := range sequence {
				cmds = append(cmds, m.update(_action))
			}
			// TODO: don't just run each action, but each action & it's cmds in order?
			return tea.Sequence(cmds...)
		}
		if batch, ok := action.(widget.Batch); ok {
			slog.Debug("captured Batch")
			var cmds []tea.Cmd
			for _, _action := range batch {
				cmds = append(cmds, m.update(_action))
			}
			return tea.Batch(cmds...)
		}
		if cmd, ok := action.(tea.Cmd); ok {
			slog.Debug("captured cmd")
			return cmd
		}
		// If the func is only ever treated as `tea.Msg` it lands here.
		if cmd, ok := action.(func() tea.Msg); ok {
			slog.Debug("captured untyped cmd")
			return cmd
		}
		_, action = m.Update(tea.Msg(action))
	}
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
			return m, m.update(cmd)
		}
	}

	// Capture mode: all keypresses go to the capturing widget, bypass dispatch
	if m.captureWidget != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			log.Printf("Send capture to widget")
			cmd := m.captureWidget.Update(msg)
			return m, m.update(cmd)
		}
	}

	// TODO: This means as long as a targetedCmd runs, the widget will still exist, even when deleted from the view?!
	// Maybe we need a way to ensure a deleted widget is not being updated anymore? :thinking:
	if tmsg, ok := msg.(TargetedMsg); ok {
		cmd := tmsg.TargetWidget().Update(msg)
		return m, m.update(cmd)
	}

	msg, cmd = m.history.Dispatch(msg)
	if msg == nil {
		return m, m.update(cmd)
	}

	// in case history/layout changed the msg.
	if tmsg, ok := msg.(TargetedMsg); ok {
		cmd := tmsg.TargetWidget().Update(msg)
		return m, m.update(cmd)
	}

	switch msg := msg.(type) {
	case tiling.DisplaySelfMsg:
		// Lazy group replacement: the first display of a command group
		// tears down the older command groups of the same shell, so old
		// output stays visible until there is genuinely new output.
		if command, group, ok := m.commandForWidget(msg.Widget); ok && !group.displayed {
			group.displayed = true
			cmd = tea.Batch(cmd, m.update(m.removeOtherCommandGroups(command)))
		}
		break // proceed with normal dispatch, which adds the widget to the layout

	case removeShellMsg:
		var stale []*widget.Widget
		for command, group := range m.commandWidgets {
			if command.Shell() != msg.s {
				continue
			}
			stale = append(stale, group.widgets...)
		}
		var actions widget.Batch
		for _, w := range stale {
			actions = append(actions, m.RemoveChild(w))
		}
		return m, m.update(actions)

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
			return m, m.update(m.selector.Init())
		}

	case CloseSelectorMsg:
		m.selecting = false
		m.selector = nil
		return m, nil

	case addWidgetMsg:
		log.Print("requested widget!")
		return m, m.update(m.addWidget(msg.Widget, msg.Msgs))

	case removeWidgetMsg:
		return m, m.update(m.RemoveChild(msg.w))

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

		addCmds := m.spawnCommandGroup(cmd)

		log.Print("Running command")
		return m, tea.Sequence(
			m.update(addCmds),
			func() tea.Msg { return shell.CommandMsg{Cmd: cmd} },
			func() tea.Msg { cmd.Run(); return shell.CommandDoneMsg{Cmd: cmd} },
		)

	case tea.EnvMsg:
		log.Print("Got env from tea process")
	}

	// TODO: This means as long as a targetedCmd runs, the widget will still exist, even when deleted from the view?!
	// Maybe we need a way to ensure a deleted widget is not being updated anymore? :thinking:
	if tmsg, ok := msg.(TargetedMsg); ok {
		action := tmsg.TargetWidget().Update(msg)
		return m, m.update(action)
	}

	// almost exclusively handles keypressmsg
	msg, action := m.layout.Dispatch(msg)
	_cmd := m.update(action)
	cmd = tea.Batch(cmd, _cmd)
	if msg == nil {
		return m, cmd
	}

	var cmds []tea.Cmd
	for _, child := range m.widgets {
		cmds = append(cmds, m.update(child.Update(msg)))
	}
	return m, tea.Batch(cmds...)
}
