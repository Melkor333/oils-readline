package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/Melkor333/oils-readline/tiling"
	"github.com/Melkor333/oils-readline/widget"
	"github.com/chalk-ai/bubbline/editline"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockShell struct{}

var S = []shell.Shell{&MockShell{}}

func (m *MockShell) Command(cmd string, size *pty.Winsize) (shell.Command, error) {
	return &MockCommand{}, nil
}

func (m *MockShell) Run(cmd string, ptmx, tty, stderr *os.File) error           { return nil }
func (m *MockShell) GetPrompt() string                                          { return "" }
func (m *MockShell) Cancel()                                                    {}
func (m *MockShell) Complete([][]rune, int, int) (string, editline.Completions) { return "", nil }
func (m *MockShell) Dir() string                                                { return "" }
func (m *MockShell) Wait()                                                      { time.Sleep(time.Millisecond) }

type MockCommand struct {
	state shell.CommandState
}

func (m *MockCommand) Run()                          {}
func (m *MockCommand) Resize(_ *pty.Winsize) error   { return nil }
func (m *MockCommand) CommandLine() string           { return "" }
func (m *MockCommand) Wait()                         {}
func (m *MockCommand) Stdin() io.Writer              { return io.Discard }
func (m *MockCommand) Stdout() string                { return "" }
func (m *MockCommand) Stderr() string                { return "" }
func (m *MockCommand) SetStdout(stdout io.Reader)    {}
func (m *MockCommand) SetStdin(stdin io.Writer)      {}
func (m *MockCommand) SetOnStdout(fn func())         {}
func (m *MockCommand) SetOnStderr(fn func())         {}
func (m *MockCommand) State() shell.CommandState     { return m.state }
func (m *MockCommand) SetState(s shell.CommandState) { m.state = s }

type blockModel struct {
	width  int
	height int
	label  string
	color  string
}

func newBlock(label, color string) *blockModel {
	return &blockModel{label: label, color: color}
}

func (m *blockModel) Init() widget.Action { return tiling.DisplaySelf(0) }

func (m *blockModel) Update(msg tea.Msg) (widget.Model, widget.Action) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, quitSoon()
}

func (m *blockModel) Name() string { return "TEST Blockmodel " }

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

type removableModel struct {
	width  int
	height int
	label  string
}

func newRemovable(label string) *removableModel {
	return &removableModel{label: label}
}

func (m *removableModel) Name() string { return "TEST Removable Model" }

func (m *removableModel) Init() widget.Action { return tiling.DisplaySelf(0) }

func (m *removableModel) Update(msg tea.Msg) (widget.Model, widget.Action) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case triggerRemoveMsg:
		return m, RemoveSelfMsg{}
	}
	return m, nil
}

func (m *removableModel) View() tea.View {
	return tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height).
		Background(lipgloss.Color("5")).
		Align(lipgloss.Center, lipgloss.Center).
		Render(m.label))
}

type triggerRemoveMsg struct{}

func TestChildSelfRemove(t *testing.T) {
	m := NewModel(S, []widget.Model{
		newBlock("A", "5"),
		newRemovable("B"),
		newBlock("C", "7"),
	})
	m2 := NewModel(S, []widget.Model{
		newBlock("A", "5"),
		newBlock("C", "7"),
	})

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	require.Equal(t, 3, len(m.widgets))
	m.Update(removeWidgetMsg{w: m.widgets[1]})

	if len(m.widgets) != 2 {
		t.Fatalf("expected 2 children after removal, got %d", len(m.widgets))
	}

	assert.Equal(t, m2.View().Content, m.View().Content)
}

func TestAddRemoveChild(t *testing.T) {
	m := NewModel(S, []widget.Model{newBlock("A", "5")})
	m.layout.Split(tiling.SplitVertical)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*model)

	m.AddChild(newBlock("B", "6"))

	if len(m.widgets) != 2 {
		t.Fatalf("expected 2 children, got %d", len(m.widgets))
	}

	m.RemoveChild(m.widgets[0])

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(*model)

	if len(m.widgets) != 1 {
		t.Fatalf("expected 1 child, got %d", len(m.widgets))
	}

	fresh := NewModel(S, []widget.Model{newBlock("B", "6")})
	fresh.layout.Split(tiling.SplitVertical)
	updated2, _ := fresh.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fresh = updated2.(*model)

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

			m := NewModel(S, makeModels(expected))

			numOps := rng.Intn(10) + 1
			for range numOps {
				canAdd := len(used) < len(labels)
				canRemove := len(m.widgets) > 0

				if canAdd && (!canRemove || rng.Intn(2) == 0) {
					label := labels[rng.Intn(len(labels))]
					for used[label] {
						label = labels[rng.Intn(len(labels))]
					}
					used[label] = true
					m.AddChild(newBlock(label, "5"))
					expected = append(expected, label)
				} else if canRemove {
					pos := rng.Intn(len(m.widgets))
					delete(used, expected[pos])
					m.RemoveChild(m.widgets[pos])
					expected = append(expected[:pos], expected[pos+1:]...)
				}
			}

			fresh := NewModel(S, makeModels(expected))

			w, h := 80, 24
			updated1, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
			m = updated1.(*model)
			updated2, _ := fresh.Update(tea.WindowSizeMsg{Width: w, Height: h})
			fresh = updated2.(*model)

			assert.Equal(t, m.View().Content, fresh.View().Content)
		})
	}
}

func makeModels(labels []string) []widget.Model {
	models := make([]widget.Model, len(labels))
	for i, l := range labels {
		models[i] = newBlock(l, "5")
	}
	return models
}

// captureMockModel implements KeySink to capture all keypresses and print them.
type captureMockModel struct {
	received        []tea.KeyPressMsg
	alreadyCaptured bool
}

func newCaptureMock() *captureMockModel {
	return &captureMockModel{}
}

func (m *captureMockModel) Name() string { return "TEST Capture Mock Model" }

func (m *captureMockModel) Init() widget.Action {
	return widget.Batch{tiling.DisplaySelf(0), RequestCapture()}
}

func (m *captureMockModel) Update(msg tea.Msg) (widget.Model, widget.Action) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.received = append(m.received, msg)
		log.Printf("capture mock received: %s\n", msg.String())
		if msg.String() == "q" {
			return m, ReleaseCapture()
		}
	}
	if !m.alreadyCaptured {
		m.alreadyCaptured = true
		return m, RequestCapture()
	}
	return m, nil
}

func (m *captureMockModel) View() tea.View {
	return tea.NewView("capture")
}

func TestKeyCapture(t *testing.T) {
	m := NewModel(S, []widget.Model{newCaptureMock()})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// We need to wait for the Init functions to run
	time.Sleep(time.Millisecond * 100)
	// Send ctrl+c — should be routed to the capture mock, not remove the widget
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	// Send 'q' to exit capture mode
	tm.Send(tea.KeyPressMsg{Code: 'q'})

	// Send ctrl+c — should be routed to the capture mock, not remove the widget
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	// Send 'q' to exit capture mode
	tm.Send(tea.KeyPressMsg{Code: 'q'})

	tm.Quit()
	final := tm.FinalModel(t)
	model := final.(*model)

	// The capture mock should have received ctrl+c and q — total 2 keys
	t.Logf("widget count: %v", len(model.widgets))
	for _, w := range model.widgets {
		if cm, ok := w.Model.(*captureMockModel); ok {
			if len(cm.received) != 3 {
				t.Fatalf("expected 4 keypresses, got %d", len(cm.received))
			}
			if cm.received[0].String() != "ctrl+c" {
				t.Errorf("expected first key to be ctrl+c, got %q", cm.received[0].String())
			}
			if cm.received[2].String() != "q" {
				t.Errorf("expected third key to be ctrl+c, got %q", cm.received[2].String())
			}
			return
		}
	}
	t.Fatal("captureMockModel not found in widgets")
}

func TestTeaAddAndRemoveBlock(t *testing.T) {
	m := NewModel(S, nil)
	m.AddChild(newBlock("A", "5"))

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	tm.Quit()
	final := tm.FinalModel(t)
	model := final.(*model)

	if len(model.widgets) != 1 {
		t.Fatalf("expected 1 widgets after add+remove, got %d", len(model.widgets))
	}
}
