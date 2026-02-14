package ui

import (
	"strings"
	"sync"
	"tictactoe-ssh/internal/db"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type SessionState int

const (
	StateNameInput SessionState = iota
	StateMenu
	StatePublicList
	StateCreateConfig
	StateInputCode
	StateLobby
	StateGame
)

const (
	PopupLeave = iota
	PopupRestart
)

type CleanupState struct {
	RoomCode  string
	IsHost    bool
	SessionID string
	Mu        sync.Mutex
}

type Model struct {
	Width, Height int
	SessionID     string
	Err           error

	Cleanup *CleanupState

	State       SessionState
	TextInput   textinput.Model
	MenuIndex   int
	PopupActive bool
	PopupType   int

	SearchInput     textinput.Model
	PublicRooms     []db.Room
	ListSelectedRow int

	IsPublicCreate bool

	MyName   string
	MySide   string
	RoomCode string

	CursorR int
	CursorC int

	Game db.Room
}

func InitialModel(s ssh.Session, cleanup *CleanupState) Model {
	// 1. Clean Name Input (Placeholder only)
	ti := textinput.New()
	ti.Placeholder = "Enter Name" // Shows when empty
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 12
	ti.Width = 20

	// 2. Search Input
	si := textinput.New()
	si.Placeholder = "Search rooms..."
	si.Prompt = "> "
	si.CharLimit = 20
	si.Width = 30

	id := "local"
	if s != nil {
		if key := s.PublicKey(); key != nil {
			id = gossh.FingerprintSHA256(key)
		} else {
			id = s.RemoteAddr().String()
		}
	}

	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "+", "-")
	id = strings.ReplaceAll(id, "=", "")
	id = strings.ReplaceAll(id, "[", "")
	id = strings.ReplaceAll(id, "]", "")

	cleanup.SessionID = id

	return Model{
		State:       StateNameInput,
		TextInput:   ti,
		SearchInput: si,
		SessionID:   id,
		Cleanup:     cleanup,
		MenuIndex:   0,
		CursorR:     1,
		CursorC:     1,
		Game:        db.Room{Board: [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "}},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
