package ui

import (
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

type Model struct {
	Width, Height int
	SessionID     string
	Err           error

	State       SessionState
	TextInput   textinput.Model
	MenuIndex   int
	PopupActive bool

	SearchInput     textinput.Model
	PublicRooms     []db.Room
	ListSelectedRow int
	
	IsPublicCreate bool

	MyName   string
	MySide   string
	RoomCode string
	
	CursorR  int 
	CursorC  int

	Game     db.Room
}

func InitialModel(s ssh.Session) Model {
	// 1. Clean Name Input (Placeholder only)
	ti := textinput.New()
	ti.Placeholder = "Enter Name" // Shows when empty
	ti.Focus()
	ti.CharLimit = 12

	// 2. Search Input
	si := textinput.New()
	si.Placeholder = "Search rooms..."
	si.CharLimit = 20

	id := "local"
	if s != nil {
		if key := s.PublicKey(); key != nil {
			id = gossh.FingerprintSHA256(key)
		} else {
			id = s.RemoteAddr().String()
		}
	}

	return Model{
		State:       StateNameInput,
		TextInput:   ti,
		SearchInput: si,
		SessionID:   id,
		MenuIndex:   0,
		CursorR:     1, 
		CursorC:     1,
		Game:        db.Room{Board: [9]string{" "," "," "," "," "," "," "," "," "}},
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
