package history

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/Melkor333/oils-readline/shell"
)

var (
	ErrEndOFHistory   = errors.New("End of history Reached")
	ErrBeginOFHistory = errors.New("Beginning of history Reached")
	ErrNotFound       = errors.New("Entry not found")
)

// TODO: Uncomment once it's not in the main module anymore
//var _ TaggedMsg = RequestHistoryEntryMsg{}
//var _ TargetedMsg = HistoryEntryMsg{}

type RequestHistoryEntryMsg struct {
	Index int
	Id    uint64
}

func (msg RequestHistoryEntryMsg) Tag(id uint64) tea.Msg {
	msg.Id = id
	return msg
}

type HistoryEntryMsg struct {
	Cmd   shell.Command
	Index int
	Total int
	Id    uint64
}

func (msg HistoryEntryMsg) TargetWidget() uint64 { return msg.Id }

type History struct {
	cc      []shell.Command
	current int
}

// Do not use `Update` because the signature is different!
func (h *History) Dispatch(msg tea.Msg) (tea.Msg, tea.Cmd) {
	switch msg := msg.(type) {
	case RequestHistoryEntryMsg:
		// Get latest History entry
		if msg.Index < 0 {
			cmd, err := h.Last()
			if err != nil {
				return nil, nil
			}
			index, err := h.GetIndexOf(cmd)
			if err != nil {
				return nil, nil
			}
			return HistoryEntryMsg{
				Cmd:   cmd,
				Index: index,
				Total: h.Count(),
				Id:    msg.Id,
			}, nil

			// Get History entry at index
		} else {
			cmd, err := h.AtIndex(msg.Index)
			if err != nil {
				return nil, nil
			}
			return HistoryEntryMsg{
				Cmd:   cmd,
				Index: msg.Index,
				Total: h.Count(),
				Id:    msg.Id,
			}, nil
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			// Only reset the
			h.SetCurrent(len(h.cc) - 1)
			return nil, nil
		case "ctrl+l":
			// TODO: Error handling!
			cmd, err := h.Next()
			if err != nil {
				cmd, err = h.AtIndex(0)
				if err != nil {
					return nil, nil
				}
			}
			// TODO: Always return a ShellHistoryEntry, but with `Id` == -1
			return shell.CommandMsg{cmd}, nil
		case "ctrl+h":
			cmd, err := h.Prev()
			if err != nil {
				cmd, err = h.Last()
				if err != nil {
					return nil, nil
				}
			}
			// TODO: Always return a ShellHistoryEntry, but with `Id` == -1
			return shell.CommandMsg{cmd}, nil
		}
	}
	return msg, nil
}

func (h *History) Add(c shell.Command) error {
	h.cc = append(h.cc, c)
	return nil
}

func (h *History) Next() (shell.Command, error) {
	if h.current >= len(h.cc)-1 {
		return nil, ErrEndOFHistory
	}
	h.current++
	return h.cc[h.current], nil
}

func (h *History) GetIndexOf(c shell.Command) (int, error) {
	for i, cmd := range h.cc {
		if cmd == c {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%w; command %v", ErrNotFound, c)
}

func (h *History) AtIndex(i int) (shell.Command, error) {
	if l := len(h.cc); i >= l {
		return nil, fmt.Errorf("%w, index %v out of range %v", ErrNotFound, i, l)
	}
	return h.cc[i], nil
}

func (h *History) SetCurrent(i int) error {
	if l := len(h.cc); i >= l || i < 0 {
		return fmt.Errorf("%w, index %v out of range %v", ErrNotFound, i, l)
	}
	h.current = i
	return nil
}

func (h *History) Prev() (shell.Command, error) {
	if h.current <= 0 {
		return nil, ErrBeginOFHistory
	}
	h.current--
	return h.cc[h.current], nil
}

func (h *History) Last() (shell.Command, error) {
	if len(h.cc) == 0 {
		return nil, ErrNotFound
	}
	return h.cc[len(h.cc)-1], nil
}
func (h *History) Count() int {
	return len(h.cc)
}
