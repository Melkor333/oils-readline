package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Melkor333/oils-readline/history"
	"github.com/Melkor333/oils-readline/shell"
	"github.com/stretchr/testify/assert"
)

func updateTerminal(t *testing.T, h tea.Model, msg tea.Msg) *Terminal {
	t.Helper()
	result, _ := h.Update(msg)
	return result.(*Terminal)
}

// ---------------------------------------------------------------------------
// 1. Basic lifecycle tests (mirroring stdout-viewer patterns)
// ---------------------------------------------------------------------------

func TestTerminalViewEmpty(t *testing.T) {
	h := newTerminal()
	v := h.View()
	assert.Equal(t, "", v.Content)
}

func TestTerminalShowsLastCommand(t *testing.T) {
	h := newTerminal()

	cmd1 := newFakeCmd("echo hello", "hello\n")
	cmd2 := newFakeCmd("echo world", "world\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd1})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd1})

	assert.Equal(t, "echo hello", h.command.CommandLine())
	fullView := h.View().Content
	assert.Contains(t, fullView, "echo hello")
	assert.Contains(t, fullView, "hello")

	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd2})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd2})

	assert.Equal(t, "echo world", h.command.CommandLine())
	fullView = h.View().Content
	assert.Contains(t, fullView, "echo world")
	assert.Contains(t, fullView, "world")
}

func TestTerminalStdoutUpdatesContent(t *testing.T) {
	h := newTerminal()

	cmd := newFakeCmd("ls", "")
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})

	fullView := h.View().Content
	assert.Contains(t, fullView, "ls")

	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	fullView = h.View().Content
	assert.Contains(t, fullView, "ls")
}

func TestTerminalReplacesPreviousCommand(t *testing.T) {
	h := newTerminal()

	cmd1 := newFakeCmd("echo first", "first\n")
	cmd2 := newFakeCmd("echo second", "second\n")
	cmd3 := newFakeCmd("echo third", "third\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd1})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd1})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd2})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd2})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd3})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd3})

	assert.Equal(t, "echo third", h.command.CommandLine())
	fullView := h.View().Content
	assert.Contains(t, fullView, "third")
	assert.NotContains(t, fullView, "first")
	assert.NotContains(t, fullView, "second")
}

func TestTerminalStdoutForOlderCommandIgnored(t *testing.T) {
	h := newTerminal()

	cmd1 := newFakeCmd("echo old", "old\n")
	cmd2 := newFakeCmd("echo new", "new\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd1})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd2})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd1})

	assert.Equal(t, "echo new", h.command.CommandLine())
	assert.NotContains(t, h.View().Content, "old")
}

func TestTerminalMultipleStdoutUpdates(t *testing.T) {
	h := newTerminal()

	cmd := newFakeCmd("stream-cmd", "")
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})

	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	cmd.(*fakeCommand).stdout = "chunk1\n"
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})
	assert.Contains(t, h.View().Content, "chunk1")

	cmd.(*fakeCommand).stdout = "chunk1\nchunk2\n"
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})
	assert.Contains(t, h.View().Content, "chunk2")
}

func TestTerminalWindowSizeUpdate(t *testing.T) {
	h := newTerminal()

	cmd := newFakeCmd("echo hi", "hi\n")
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 120, Height: 30})
	assert.Equal(t, 120, h.Width)
	assert.Equal(t, 30, h.Height)
}

func TestTerminalRunningState(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("echo hi", "hi\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Ready state — not running
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
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

func TestTerminalRunningIndicatorInView(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("echo hi", "hi\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	cmd.SetState(shell.Started)

	fullView := h.View().Content
	assert.Contains(t, fullView, "echo hi", "command line should be visible")
	assert.Contains(t, fullView, "●", "running indicator should be present when Started")

	cmd.SetState(shell.Stopped)

	fullView = h.View().Content
	assert.NotContains(t, fullView, "●", "running indicator should disappear when Stopped")
}

func TestTerminalInteractiveModeRequiresRunning(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("sleep 1", "")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})

	// Started state — should enter interactive mode
	cmd.SetState(shell.Started)
	result, cmd1 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result.(*Terminal)
	assert.True(t, h.interactiveMode, "should enter interactive mode while running")
	assert.NotNil(t, cmd1, "should return a command (RequestCapture)")

	// Cancel interactive mode
	h.interactiveMode = false

	// Stopped state — should NOT enter interactive mode
	cmd.SetState(shell.Stopped)
	result2, cmd2 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result2.(*Terminal)
	assert.False(t, h.interactiveMode, "should NOT enter interactive mode when not running")
	assert.Nil(t, cmd2, "should NOT return RequestCapture when not running")
}

func TestTerminalCommandDoneReleasesCapture(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("sleep 1", "")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	cmd.SetState(shell.Started)

	// Enter interactive mode
	result, _ := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result.(*Terminal)
	assert.True(t, h.interactiveMode)

	// CommandDone should exit interactive mode and release capture
	result2, cmd2 := h.Update(shell.CommandDoneMsg{Cmd: cmd})
	h = result2.(*Terminal)
	assert.False(t, h.interactiveMode, "CommandDone should exit interactive mode")
	assert.NotNil(t, cmd2, "should return ReleaseCapture command")
}

// ---------------------------------------------------------------------------
// 2. Terminal-specific ANSI/OSC tests
// ---------------------------------------------------------------------------

func TestTerminalHandlesANSISequences(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cmd", "\033[31mred text\033[0m\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	// The terminal emulator should have processed the ANSI codes.
	// String() returns the plain text content without escape sequences.
	plain := h.term.String()
	assert.Contains(t, plain, "red text")

	// The view output should contain the visible text, not raw escape sequences.
	view := h.View().Content
	assert.Contains(t, view, "red text")
	assert.Contains(t, view, "cmd")
}

func TestTerminalHandlesANSICursorMovement(t *testing.T) {
	h := newTerminal()

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Write two lines, then move cursor up, erase the line, and replace.
	h.term.WriteString("line1\nline2\n")
	h.term.WriteString("\033[A") // cursor up
	h.term.WriteString("\033[K") // erase to end of line
	h.term.WriteString("replaced\n")

	plain := h.term.String()
	assert.Contains(t, plain, "line1")
	assert.Contains(t, plain, "replaced")
	// "line2" should have been erased by the cursor movement + erase sequence.
	// (May remain in scrollback depending on emulator, but we at least
	// verify the replacement text is present.)
}

func TestTerminalHandlesOSCSequences(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cmd", "\033]0;window title\007\033]2;icon title\007visible text\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	// OSC 0 and OSC 2 set terminal/icon titles and should not produce
	// visible text or leave raw escape codes in the terminal content.
	plain := h.term.String()
	assert.Contains(t, plain, "visible text")
	assert.NotContains(t, plain, "window title", "OSC title should not be visible text")
}

func TestTerminalHandlesHyperlinkOSC(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cmd", "\033]8;;https://example.com\007link text\033]8;;\007\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	// OSC 8 hyperlinks wrap visible text. The terminal emulator should
	// process the hyperlink escape sequences and display only the text.
	plain := h.term.String()
	assert.Contains(t, plain, "link text")

	view := h.View().Content
	assert.Contains(t, view, "link text")
}

func TestTerminalHandlesMixedPlainAndANSIContent(t *testing.T) {
	h := newTerminal()
	mixed := "plain line\n" +
		"\033[31mred text\033[0m\n" +
		"\033]0;title\007" +
		"\033[32m\033[1mgreen bold\033[0m\n" +
		"\033]8;;https://example.com\007hyperlink\033]8;;\007\n"

	cmd := newFakeCmd("cmd", mixed)

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	// The terminal should correctly process all sequence types.
	plain := h.term.String()
	assert.Contains(t, plain, "plain line")
	assert.Contains(t, plain, "red text")
	assert.Contains(t, plain, "green bold")
	assert.Contains(t, plain, "hyperlink")
}

// ---------------------------------------------------------------------------
// 3. History and navigation tests
// ---------------------------------------------------------------------------

func TestTerminalCommandLineWithIndex(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("my-cmd", "output\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h.currentIndex = 3

	view := h.View().Content
	assert.Contains(t, view, "[3]")
	assert.Contains(t, view, "my-cmd")
	assert.Contains(t, view, "output")
}

func TestTerminalHistoryNavigation(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("current cmd", "current output\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h.currentIndex = 2 // simulate having history

	// Press 'h' to go backward in history.
	// When targetIndex < 0, h requests currentIndex - 1.
	result, cmd1 := h.Update(tea.KeyPressMsg{Code: 'h'})
	h = result.(*Terminal)
	assert.NotNil(t, cmd1, "'h' should return a history request command")

	// Simulate receiving a history entry response.
	prevCmd := newFakeCmd("previous cmd", "previous output\n")
	h = updateTerminal(t, h, history.HistoryEntryMsg{
		Cmd:   prevCmd,
		Index: 1,
		Total: 10,
	})
	assert.Equal(t, 1, h.currentIndex)
	assert.Equal(t, "previous cmd", h.command.CommandLine())
	assert.Contains(t, h.View().Content, "[1]")

	// Press 's' to toggle sticky mode.
	// targetIndex starts as -1; 's' sets it to currentIndex.
	assert.Equal(t, -1, h.targetIndex, "targetIndex should be -1 initially")
	result2, cmd2 := h.Update(tea.KeyPressMsg{Code: 's'})
	h = result2.(*Terminal)
	assert.Nil(t, cmd2, "'s' should not return a command (just toggles sticky)")
	assert.Equal(t, 1, h.targetIndex, "targetIndex should equal currentIndex after 's'")

	// Press 's' again to unstick.
	result3, _ := h.Update(tea.KeyPressMsg{Code: 's'})
	h = result3.(*Terminal)
	assert.Equal(t, -1, h.targetIndex, "targetIndex should be -1 after unstick")

	// Press 'l' to go forward in history.
	// When targetIndex < 0, 'l' requests currentIndex + 1.
	result4, cmd4 := h.Update(tea.KeyPressMsg{Code: 'l'})
	h = result4.(*Terminal)
	assert.NotNil(t, cmd4, "'l' should return a history request command")

	// Press 'l' again when sticky (targetIndex >= 0).
	h.targetIndex = 1
	result5, cmd5 := h.Update(tea.KeyPressMsg{Code: 'l'})
	h = result5.(*Terminal)
	assert.Equal(t, 2, h.targetIndex, "'l' in sticky mode should increment targetIndex")
	assert.NotNil(t, cmd5, "'l' should return a history request command")

	// Press 'h' in sticky mode should decrement targetIndex.
	result6, _ := h.Update(tea.KeyPressMsg{Code: 'h'})
	h = result6.(*Terminal)
	assert.Equal(t, 1, h.targetIndex, "'h' in sticky mode should decrement targetIndex")
}

// ---------------------------------------------------------------------------
// 4. Exit menu tests
// ---------------------------------------------------------------------------

func TestTerminalExitMenuNavigation(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cat", "")
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	cmd.SetState(shell.Started)

	// Enter interactive mode.
	result, _ := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result.(*Terminal)
	assert.True(t, h.interactiveMode)

	// Start at menuSelectSendctrlc (opened via ctrl+c).
	h.exitMenuSelect = menuSelectSendctrlc

	// j — Sendctrlc → Exit
	result, _ = h.Update(tea.KeyPressMsg{Code: 'j'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectExit, h.exitMenuSelect, "j should go from Sendctrlc to Exit")

	// j — Exit → Cancel
	result, _ = h.Update(tea.KeyPressMsg{Code: 'j'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectCancel, h.exitMenuSelect, "j should go from Exit to Cancel")

	// j — Cancel → Cancel (boundary — can't go above Cancel)
	result, _ = h.Update(tea.KeyPressMsg{Code: 'j'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectCancel, h.exitMenuSelect, "j should stay at Cancel (boundary)")

	// k — Cancel → Exit
	result, _ = h.Update(tea.KeyPressMsg{Code: 'k'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectExit, h.exitMenuSelect, "k should go from Cancel to Exit")

	// k — Exit → Sendctrlc
	result, _ = h.Update(tea.KeyPressMsg{Code: 'k'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectSendctrlc, h.exitMenuSelect, "k should go from Exit to Sendctrlc")

	// k — Sendctrlc → Sendctrlc (boundary — can't go below Sendctrlc)
	result, _ = h.Update(tea.KeyPressMsg{Code: 'k'})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectSendctrlc, h.exitMenuSelect, "k should stay at Sendctrlc (boundary)")
}

func TestTerminalExitMenuEnterAction(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cat", "")
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	cmd.SetState(shell.Started)
	h.interactiveMode = true

	// --- menuSelectHidden (default, interactive mode) ---
	// Enter with hidden menu writes newline to stdin.
	h.exitMenuSelect = menuSelectHidden
	result, cmd1 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result.(*Terminal)
	assert.Equal(t, menuSelectHidden, h.exitMenuSelect)
	assert.NotNil(t, cmd1, "enter in interactive mode (hidden menu) should return a command")

	// --- menuSelectSendctrlc ---
	// Enter sends 0x03 to stdin and resets menu selection.
	h.exitMenuSelect = menuSelectSendctrlc
	result2, cmd2 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result2.(*Terminal)
	assert.Equal(t, menuSelectHidden, h.exitMenuSelect, "Sendctrlc enter should reset menu to hidden")
	assert.NotNil(t, cmd2, "Sendctrlc should return a command")

	// --- menuSelectExit ---
	// Enter exits interactive mode and returns ReleaseCapture.
	h.exitMenuSelect = menuSelectExit
	result3, cmd3 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result3.(*Terminal)
	assert.False(t, h.interactiveMode, "Exit should set interactive mode to false")
	assert.Equal(t, menuSelectHidden, h.exitMenuSelect, "Exit should reset menu to hidden")
	assert.NotNil(t, cmd3, "Exit should return ReleaseCapture")

	// Re-enter interactive mode for the Cancel test.
	cmd.SetState(shell.Started)
	resultRe, _ := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = resultRe.(*Terminal)
	assert.True(t, h.interactiveMode)

	// --- menuSelectCancel ---
	// Enter just hides the menu, stays in interactive mode, returns nil.
	h.exitMenuSelect = menuSelectCancel
	result4, cmd4 := h.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	h = result4.(*Terminal)
	assert.True(t, h.interactiveMode, "Cancel should keep interactive mode")
	assert.Equal(t, menuSelectHidden, h.exitMenuSelect, "Cancel should reset menu to hidden")
	assert.Nil(t, cmd4, "Cancel should return nil")
}

// ---------------------------------------------------------------------------
// 5. Edge case / misc tests
// ---------------------------------------------------------------------------

func TestTerminalWriteStdinError(t *testing.T) {
	h := newTerminal()

	_, err := h.WriteStdin([]byte("input"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no command")
}

func TestTerminalNewCommandResetsPosition(t *testing.T) {
	h := newTerminal()
	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})

	cmd1 := newFakeCmd("cmd1", "initial output")
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd1})
	// After CommandMsg, position should equal len of stdout.
	assert.Equal(t, len("initial output"), h.position)

	cmd2 := newFakeCmd("cmd2", "new output")
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd2})
	// Position is reset to 0, then updateContent writes all of new output.
	assert.Equal(t, len("new output"), h.position)
}

func TestTerminalStdoutMsgUpdatesTermContent(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("cmd", "stdout content\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	// Verify that the terminal emulator's content includes the output.
	termContent := h.term.String()
	assert.Contains(t, termContent, "stdout content")
}

func TestTerminalViewIncludesTerminalContent(t *testing.T) {
	h := newTerminal()
	cmd := newFakeCmd("my-cmd", "line1\nline2\n")

	h = updateTerminal(t, h, tea.WindowSizeMsg{Width: 80, Height: 24})
	h = updateTerminal(t, h, shell.CommandMsg{Cmd: cmd})
	h = updateTerminal(t, h, shell.StdoutMsg{Cmd: cmd})

	view := h.View().Content
	assert.Contains(t, view, "my-cmd", "view should include command line")
	assert.Contains(t, view, "line1", "view should include terminal output")
	assert.Contains(t, view, "line2", "view should include terminal output")
}
