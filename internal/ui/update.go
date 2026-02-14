package ui

import (
	"math/rand"
	"strings"
	"time"

	"tictactoe-ssh/internal/db"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages
type roomUpdateMsg db.Room
type roomsFetchedMsg []db.Room
type errMsg error

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			// If we are in a game and hosting, we might want to clean up, 
			// but usually Wish handles the connection drop. 
			// Explicit quit here is fine.
			return m, tea.Quit
		}
	}

	// Global Popup Handler (Are you sure you want to leave?)
	if m.PopupActive {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "enter":
				// Confirm Leave
				isHost := (m.MySide == "X")
				if m.RoomCode != "" {
					db.LeaveRoom(m.RoomCode, m.SessionID, isHost)
				}
				m.PopupActive = false
				m.State = StateMenu
				m.Err = nil
				m.RoomCode = "" // Clear room code on exit
				return m, nil
			case "n", "esc":
				m.PopupActive = false
			}
		}
		return m, nil
	}

	// State Machine
	switch m.State {
	case StateNameInput:
		m, cmd = updateName(m, msg)
	case StateMenu:
		m, cmd = updateMenu(m, msg)
	case StateCreateConfig:
		m, cmd = updateCreateConfig(m, msg)
	case StateInputCode:
		m, cmd = updateCodeInput(m, msg)
	case StatePublicList:
		m, cmd = updatePublicList(m, msg)
	case StateLobby, StateGame:
		m, cmd = updateGame(m, msg)
	}

	return m, cmd
}

// --- 1. Name Input Logic ---
func updateName(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			val := strings.TrimSpace(m.TextInput.Value())
			if len(val) > 0 {
				m.MyName = val
				m.State = StateMenu
				return m, nil
			}
		}
	}
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

// --- 2. Main Menu Logic ---
func updateMenu(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.MenuIndex > 0 {
				m.MenuIndex--
			}
		case "down", "j":
			if m.MenuIndex < 3 {
				m.MenuIndex++
			}
		case "enter":
			if m.MenuIndex == 0 { // Create Room
				m.State = StateCreateConfig
				m.IsPublicCreate = false // default to private
			} else if m.MenuIndex == 1 { // Join via Code
				m.State = StateInputCode
				m.TextInput.Placeholder = "4-Digit Code"
				m.TextInput.SetValue("")
				m.TextInput.Focus()
				return m, textinput.Blink
			} else if m.MenuIndex == 2 { // Public Rooms List
				m.State = StatePublicList
				m.SearchInput.Focus()
				m.ListSelectedRow = 0 // Reset selection to top
				return m, fetchPublicRoomsCmd()
			} else { // Quit
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// --- 3. Create Room Configuration ---
func updateCreateConfig(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "down", "k", "j":
			m.IsPublicCreate = !m.IsPublicCreate
		case "enter":
			code := generateCode()
			m.RoomCode = code
			m.MySide = "X"
			// Create room in DB
			if err := db.CreateRoom(code, m.SessionID, m.MyName, m.IsPublicCreate); err != nil {
				m.Err = err
				return m, nil
			}
			m.State = StateLobby
			return m, pollCmd(code)
		case "esc":
			m.State = StateMenu
		}
	}
	return m, nil
}

// --- 4. Manual Code Input ---
func updateCodeInput(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			code := strings.ToUpper(m.TextInput.Value())
			if err := db.JoinRoom(code, m.SessionID, m.MyName); err != nil {
				m.Err = err // Display error in view
			} else {
				m.RoomCode = code
				m.MySide = "O"
				m.State = StateGame
				return m, pollCmd(code)
			}
		}
		if msg.Type == tea.KeyEsc {
			m.State = StateMenu
			m.Err = nil
		}
	}
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

// --- Public List Logic (Fixed Error Handling) ---
func updatePublicList(m Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	getSortedList := func() []db.Room {
		var open, full []db.Room
		filter := strings.ToUpper(m.SearchInput.Value())
		
		for _, r := range m.PublicRooms {
			// Show all if filter empty, otherwise match
			if filter == "" || strings.Contains(r.Code, filter) || strings.Contains(strings.ToUpper(r.PlayerXName), filter) {
				if r.PlayerO == "" {
					open = append(open, r)
				} else {
					full = append(full, r)
				}
			}
		}
		return append(open, full...)
	}

	switch msg := msg.(type) {
	case roomsFetchedMsg:
		m.PublicRooms = []db.Room(msg)
		// Clear previous errors if fetch succeeded
		if m.Err != nil {
			// Check if error was related to fetching? 
			// For simplicity, just clear it so UI looks clean
			m.Err = nil
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.State = StateMenu
		case "up", "shift+tab":
			if m.ListSelectedRow > 0 { m.ListSelectedRow-- }
		case "down", "tab":
			list := getSortedList()
			if m.ListSelectedRow < len(list)-1 { m.ListSelectedRow++ }
		case "enter":
			list := getSortedList()
			if len(list) > 0 && m.ListSelectedRow < len(list) {
				sel := list[m.ListSelectedRow]
				if sel.PlayerO == "" {
					if err := db.JoinRoom(sel.Code, m.SessionID, m.MyName); err != nil {
						m.Err = err
					} else {
						m.RoomCode = sel.Code
						m.MySide = "O"
						m.State = StateGame
						return m, pollCmd(sel.Code)
					}
				}
			}
		}
	}
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	return m, cmd
}

func updateGame(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case roomUpdateMsg:
		m.Game = db.Room(msg)
		if m.State == StateLobby && m.Game.PlayerO != "" {
			m.State = StateGame
		}
		if m.Game.PlayerX == "" {
			m.Err = nil
			m.State = StateMenu
			m.RoomCode = ""
			return m, nil
		}
		return m, pollCmd(m.RoomCode)

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			m.PopupActive = true
			return m, nil
		}
		if m.Game.Status == "finished" {
			if msg.String() == "r" {
				return m, func() tea.Msg {
					db.RestartGame(m.RoomCode)
					return nil
				}
			}
			return m, nil
		}
		if m.Game.Status == "waiting" { return m, nil }

		switch msg.String() {
		case "up", "k":
			if m.CursorR > 0 { m.CursorR-- }
		case "down", "j":
			if m.CursorR < 2 { m.CursorR++ }
		case "left", "h":
			if m.CursorC > 0 { m.CursorC-- }
		case "right", "l":
			if m.CursorC < 2 { m.CursorC++ }
		case " ", "enter":
			idx := m.CursorR*3 + m.CursorC
			if m.Game.Turn == m.MySide && m.Game.Board[idx] == " " {
				return m, func() tea.Msg {
					db.UpdateMove(m.RoomCode, m.SessionID, idx, m.Game)
					return nil
				}
			}
		}
	}
	return m, nil
}

func pollCmd(code string) tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		r, err := db.GetRoom(code)
		if err != nil || r == nil { return roomUpdateMsg{} }
		return roomUpdateMsg(*r)
	})
}

// Updated Fetch Command
func fetchPublicRoomsCmd() tea.Cmd {
	return func() tea.Msg {
		rooms, err := db.GetPublicRooms()
		if err != nil {
			return errMsg(err)
		}
		return roomsFetchedMsg(rooms)
	}
}

func generateCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b { b[i] = chars[rand.Intn(len(chars))] }
	return string(b)
}
