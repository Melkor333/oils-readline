package main

import (
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/stretchr/testify/assert"
)

type fakeCommand struct {
	commandLine string
	stdout      string
	stderr      string
	state       shell.CommandState
}

func (f *fakeCommand) Run()                            {}
func (f *fakeCommand) CommandLine() string             { return f.commandLine }
func (f *fakeCommand) Wait()                           {}
func (f *fakeCommand) Stdin() io.Writer                { return io.Discard }
func (f *fakeCommand) Stdout() string                  { return f.stdout }
func (f *fakeCommand) Stderr() string                  { return f.stderr }
func (f *fakeCommand) SetStdout(stdout io.Reader)      {}
func (f *fakeCommand) SetStdin(stdin io.Writer)        {}
func (f *fakeCommand) SetOnStdout(fn func())           {}
func (f *fakeCommand) SetOnStderr(fn func())           {}
func (f *fakeCommand) State() shell.CommandState       { return f.state }
func (f *fakeCommand) SetState(s shell.CommandState)   { f.state = s }

func newFakeCmd(cmdLine, output string) shell.Command {
	return &fakeCommand{commandLine: cmdLine, stdout: output}
}

func updateStdoutViewer(t *testing.T, h tea.Model, msg tea.Msg) *StdoutViewer {
	t.Helper()
	result, _ := h.Update(msg)
	return result.(*StdoutViewer)
}

func TestStdoutViewerShowsLastCommand(t *testing.T) {
	h := newStdoutViewer()

	cmd1 := newFakeCmd("echo hello", "hello\n")
	cmd2 := newFakeCmd("echo world", "world\n")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd1})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd1})

	assert.Equal(t, "echo hello", h.command.CommandLine())
	fullView := h.View().Content
	assert.Contains(t, fullView, "echo hello")
	assert.Contains(t, fullView, "hello")

	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd2})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd2})

	assert.Equal(t, "echo world", h.command.CommandLine())
	fullView = h.View().Content
	assert.Contains(t, fullView, "echo world")
	assert.Contains(t, fullView, "world")
}

func TestStdoutViewerStdoutUpdatesContent(t *testing.T) {
	h := newStdoutViewer()

	cmd := newFakeCmd("ls", "")
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})

	fullView := h.View().Content
	assert.Contains(t, fullView, "ls")
	assert.NotContains(t, fullView, "file.txt")

	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})

	fullView = h.View().Content
	assert.Contains(t, fullView, "ls")
}

func TestStdoutViewerReplacesPreviousCommand(t *testing.T) {
	h := newStdoutViewer()

	cmd1 := newFakeCmd("echo first", "first\n")
	cmd2 := newFakeCmd("echo second", "second\n")
	cmd3 := newFakeCmd("echo third", "third\n")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd1})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd1})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd2})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd2})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd3})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd3})

	assert.Equal(t, "echo third", h.command.CommandLine())
	fullView := h.View().Content
	assert.Contains(t, fullView, "third")
	assert.NotContains(t, fullView, "first")
	assert.NotContains(t, fullView, "second")
}

func TestStdoutViewerStdoutForOlderCommandIgnored(t *testing.T) {
	h := newStdoutViewer()

	cmd1 := newFakeCmd("echo old", "old\n")
	cmd2 := newFakeCmd("echo new", "new\n")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd1})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd2})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd1})

	assert.Equal(t, "echo new", h.command.CommandLine())
	assert.NotContains(t, h.View().Content, "old")
}

func TestStdoutViewerViewEmpty(t *testing.T) {
	h := newStdoutViewer()
	v := h.View()
	assert.Equal(t, "", v.Content)
}

func TestStdoutViewerCommandAlwaysVisible(t *testing.T) {
	h := newStdoutViewer()

	longOutput := ""
	for range 50 {
		longOutput += "line\n"
	}
	cmd := newFakeCmd("my-cmd", longOutput)

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 5})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, tea.FocusMsg{})

	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'j'})
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'j'})
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'j'})

	fullView := h.View().Content
	assert.Contains(t, fullView, "my-cmd", "command line must remain visible after scrolling")
}

func TestStdoutViewerScrollingWithJK(t *testing.T) {
	h := newStdoutViewer()

	longOutput := ""
	for i := range 50 {
		longOutput += "line " + string(rune('A'+i%26)) + "\n"
	}

	cmd := newFakeCmd("big-command", longOutput)

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 5})
	h = updateStdoutViewer(t, h, tea.FocusMsg{})

	initialY := h.view.YOffset()

	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'j'})
	assert.Greater(t, h.view.YOffset(), initialY, "j should scroll down")

	newY := h.view.YOffset()
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'k'})
	assert.Less(t, h.view.YOffset(), newY, "k should scroll back up")
}

func TestStdoutViewerFocusBlur(t *testing.T) {
	h := newStdoutViewer()

	cmd := newFakeCmd("echo test", "test\n")
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})

	h = updateStdoutViewer(t, h, tea.FocusMsg{})
	assert.True(t, h.isFocussed)

	h = updateStdoutViewer(t, h, tea.BlurMsg{})
	assert.False(t, h.isFocussed)
}

func TestStdoutViewerWindowSizeUpdate(t *testing.T) {
	h := newStdoutViewer()

	cmd := newFakeCmd("echo hi", "hi\n")
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 120, Height: 30})
	assert.Equal(t, 120, h.Width)
	assert.Equal(t, 30, h.Height)
}

func TestStdoutViewerMultipleStdoutUpdates(t *testing.T) {
	h := newStdoutViewer()

	cmd := newFakeCmd("stream-cmd", "")
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})

	assert.NotContains(t, h.view.View(), "chunk1")

	cmd.(*fakeCommand).stdout = "chunk1\n"
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})
	assert.Contains(t, h.view.View(), "chunk1")

	cmd.(*fakeCommand).stdout = "chunk1\nchunk2\n"
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})
	assert.Contains(t, h.view.View(), "chunk2")
}

func TestStdoutViewerToggleStderr(t *testing.T) {
	h := newStdoutViewer()

	cmd := &fakeCommand{commandLine: "my-cmd", stdout: "out\n", stderr: "err\n"}
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})

	assert.False(t, h.showStderr)
	assert.Contains(t, h.view.View(), "out")
	assert.NotContains(t, h.view.View(), "err")

	h = updateStdoutViewer(t, h, tea.FocusMsg{})
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'e'})

	assert.True(t, h.showStderr)
	assert.Contains(t, h.view.View(), "err")
	assert.NotContains(t, h.view.View(), "out")

	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'e'})

	assert.False(t, h.showStderr)
	assert.Contains(t, h.view.View(), "out")
	assert.NotContains(t, h.view.View(), "err")
}

func TestStdoutViewerStderrRedCommandLine(t *testing.T) {
	h := newStdoutViewer()

	cmd := &fakeCommand{commandLine: "fail-cmd", stdout: "ok\n", stderr: "oops\n"}
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, shell.StdoutMsg{Cmd: cmd})

	h = updateStdoutViewer(t, h, tea.FocusMsg{})
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'e'})

	fullView := h.View().Content
	assert.Contains(t, fullView, "fail-cmd")
	assert.Contains(t, fullView, "oops")
}

func TestStdoutViewerStderrMsgUpdatesContent(t *testing.T) {
	h := newStdoutViewer()
	cmd := &fakeCommand{commandLine: "cmd", stdout: "", stderr: ""}
	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})

	h = updateStdoutViewer(t, h, tea.FocusMsg{})
	h = updateStdoutViewer(t, h, tea.KeyPressMsg{Code: 'e'})

	cmd.stderr = "error!\n"
	h = updateStdoutViewer(t, h, shell.StderrMsg{Cmd: cmd})
	assert.Contains(t, h.view.View(), "error!")
}

func TestStdoutViewerRunningState(t *testing.T) {
	h := newStdoutViewer()
	cmd := newFakeCmd("echo hi", "hi\n")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Ready state — not running
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	assert.False(t, h.commandRunning(), "should not be running when state is Ready")

	// Started state
	cmd.SetState(shell.Started)
	assert.True(t, h.commandRunning(), "should be running when state is Started")

	// Queued state
	cmd.SetState(shell.Queued)
	assert.True(t, h.commandRunning(), "should be running when state is Queued")

	// Stopped state
	cmd.SetState(shell.Stopped)
	assert.False(t, h.commandRunning(), "should not be running when state is Stopped")
}

func TestStdoutViewerRunningIndicatorInView(t *testing.T) {
	h := newStdoutViewer()
	cmd := newFakeCmd("echo hi", "hi\n")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	cmd.SetState(shell.Started)

	fullView := h.View().Content
	assert.Contains(t, fullView, "echo hi", "command line should be visible")
	assert.Contains(t, fullView, "●", "running indicator should be present when Started")

	cmd.SetState(shell.Stopped)

	fullView = h.View().Content
	assert.NotContains(t, fullView, "●", "running indicator should disappear when Stopped")
}

func TestStdoutViewerInteractiveModeRequiresRunning(t *testing.T) {
	h := newStdoutViewer()
	cmd := newFakeCmd("sleep 1", "")

	h = updateStdoutViewer(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateStdoutViewer(t, h, shell.NewCommandMsg{Cmd: cmd})
	h = updateStdoutViewer(t, h, tea.FocusMsg{})

	// Started state — should enter interactive mode
	cmd.SetState(shell.Started)
	result, cmd1 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result.(*StdoutViewer)
	assert.True(t, h.interactiveMode, "should enter interactive mode while running")
	assert.NotNil(t, cmd1, "should return a command (RequestCapture)")

	// Cancel interactive mode
	h.interactiveMode = false

	// Stopped state — should NOT enter interactive mode
	cmd.SetState(shell.Stopped)
	result2, cmd2 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result2.(*StdoutViewer)
	assert.False(t, h.interactiveMode, "should NOT enter interactive mode when not running")
	assert.Nil(t, cmd2, "should NOT return RequestCapture when not running")
}
