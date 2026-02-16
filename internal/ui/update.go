package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"tictactoe-ssh/internal/db"
	"tictactoe-ssh/internal/game"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages
type roomUpdateMsg db.Room
type roomsFetchedMsg []db.Room
type errMsg error
type pollErrorMsg error

type roomCreatedMsg struct {
	code     string
	gameType string
}
type roomJoinedMsg struct {
	code     string
	side     string
	gameType string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// 1. Handle background polling (Highest Priority, Non-Blocking)
	if roomMsg, ok := msg.(roomUpdateMsg); ok {
		m.Game = db.Room(roomMsg)
		// Auto-transition from Lobby to Game
		if m.State == StateLobby && m.Game.PlayerO != "" {
			m.State = StateGame
		}
		// Room deleted?
		if m.Game.PlayerX == "" {
			m.Err = fmt.Errorf("Room closed by host")
			m.State = StateMenu
			m.RoomCode = ""
			m.Busy = false
			return m, nil
		}
		return m, pollCmd(m.RoomCode)
	}

	// 2. Handle Polling Errors
	if err, ok := msg.(pollErrorMsg); ok {
		m.Err = err
		// Retry polling after delay
		return m, pollCmd(m.RoomCode)
	}

	// 3. Handle Async DB Results
	switch msg := msg.(type) {
	case roomCreatedMsg:
		m.Busy = false
		m.RoomCode = msg.code
		m.MySide = "X"

		m.Cleanup.Mu.Lock()
		m.Cleanup.RoomCode = msg.code
		m.Cleanup.IsHost = true
		m.Cleanup.Mu.Unlock()

		if msg.gameType == "chess" {
			m.CursorR = 7 // White Pieces (Rank 1)
			m.CursorC = 4 // King File
		} else {
			m.CursorR = 1 // Middle of 3x3
			m.CursorC = 1
		}

		m.State = StateLobby
		return m, pollCmd(msg.code)

	case roomJoinedMsg:
		m.Busy = false
		m.RoomCode = msg.code
		m.MySide = msg.side

		m.Cleanup.Mu.Lock()
		m.Cleanup.RoomCode = msg.code
		m.Cleanup.IsHost = (msg.side == "X")
		m.Cleanup.Mu.Unlock()

		if msg.gameType == "chess" {
			// If Black ("O"), we want Rank 8 (Index 0)
			if msg.side == "O" {
				m.CursorR = 0
				m.CursorC = 4
			} else {
				// White or Spectator (Rank 1 -> Index 7)
				m.CursorR = 7
				m.CursorC = 4
			}
		} else {
			m.CursorR = 1
			m.CursorC = 1
		}

		m.State = StateGame
		return m, pollCmd(msg.code)

	case errMsg:
		m.Busy = false
		m.Err = msg
		// Stay in current state, allow retry
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Global Popup Handler
	if m.PopupActive {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.PopupType == PopupRestart {
				switch msg.String() {
				case "1":
					// Random
					next := "X"
					if m.Game.GameType == "chess" {
						next = "White"
						if rand.Intn(2) == 0 {
							next = "Black"
						}
					} else {
						if rand.Intn(2) == 0 {
							next = "O"
						}
					}
					m.PopupActive = false
					return m, func() tea.Msg {
						db.RestartGame(m.RoomCode, next)
						return nil
					}
				case "2":
					// Winner
					next := m.Game.Winner
					if next == "" {
						// If draw, Random
						if m.Game.GameType == "chess" {
							next = "White"
							if rand.Intn(2) == 0 {
								next = "Black"
							}
						} else {
							next = "X"
							if rand.Intn(2) == 0 {
								next = "O"
							}
						}
					} else {
						// Map winner to turn
						if m.Game.GameType == "chess" {
							// Winner is White/Black
							next = m.Game.Winner
						} else {
							// Winner is X/O
							next = m.Game.Winner
						}
					}
					m.PopupActive = false
					return m, func() tea.Msg {
						db.RestartGame(m.RoomCode, next)
						return nil
					}
				case "esc":
					m.PopupActive = false
				}
			} else {
				// Leave Popup
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
		}
		return m, nil
	}

	// State Machine
	switch m.State {
	case StateNameInput:
		m, cmd = updateName(m, msg)
	case StateGameSelect:
		m, cmd = updateGameSelect(m, msg)
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
				m.State = StateGameSelect // Transition to Game Select
				m.MenuIndex = 0           // Reset index
				return m, nil
			}
		}
	}
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

// --- 1.5 Game Selection Logic ---
func updateGameSelect(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.MenuIndex > 0 {
				m.MenuIndex--
			}
		case "down", "j":
			if m.MenuIndex < 1 { // 0: TicTacToe, 1: Chess
				m.MenuIndex++
			}
		case "enter":
			if m.MenuIndex == 0 {
				m.SelectedGame = "tictactoe"
			} else {
				m.SelectedGame = "chess"
			}
			m.State = StateMenu
			m.MenuIndex = 0
			return m, nil
		}
	}
	return m, nil
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
			if m.Busy {
				return m, nil
			}
			m.Busy = true
			code := generateCode()
			// Use SelectedGame
			gameType := m.SelectedGame
			if gameType == "" {
				gameType = "tictactoe"
			} // Fallback
			return m, createRoomCmd(code, m.SessionID, m.MyName, m.IsPublicCreate, gameType)
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
			if m.Busy {
				return m, nil
			}
			m.Busy = true
			code := strings.ToUpper(m.TextInput.Value())
			return m, joinRoomCmd(code, m.SessionID, m.MyName)
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
		if m.Err != nil {
			m.Err = nil
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.State = StateMenu
		case "up", "shift+tab":
			if m.ListSelectedRow > 0 {
				m.ListSelectedRow--
			}
		case "down", "tab":
			list := getSortedList()
			if m.ListSelectedRow < len(list)-1 {
				m.ListSelectedRow++
			}
		case "enter":
			list := getSortedList()
			if len(list) > 0 && m.ListSelectedRow < len(list) {
				if m.Busy {
					return m, nil
				}
				sel := list[m.ListSelectedRow]
				m.Busy = true
				return m, joinRoomCmd(sel.Code, m.SessionID, m.MyName)
			}
		}
	}
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	return m, cmd
}

func updateGame(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			m.PopupActive = true
			m.PopupType = PopupLeave
			return m, nil
		}
		if m.Game.Status == "finished" {
			if msg.String() == "r" {
				if m.MySide == "Spectator" {
					return m, nil
				}
				m.PopupActive = true
				m.PopupType = PopupRestart
				return m, nil
			}
			return m, nil
		}
		if m.Game.Status == "waiting" {
			return m, nil
		}

		if m.Game.GameType == "chess" {
			// Handle Chess Input
			return updateChessInput(m, msg)
		} else {
			// Handle TicTacToe Input
			switch msg.String() {
			case "up", "k":
				if m.CursorR > 0 {
					m.CursorR--
				}
			case "down", "j":
				if m.CursorR < 2 {
					m.CursorR++
				}
			case "left", "h":
				if m.CursorC > 0 {
					m.CursorC--
				}
			case "right", "l":
				if m.CursorC < 2 {
					m.CursorC++
				}
			case " ", "enter":
				if m.MySide == "Spectator" {
					return m, nil
				}
				idx := m.CursorR*3 + m.CursorC
				if m.Game.Turn == m.MySide && m.Game.Board[idx] == " " {
					return m, func() tea.Msg {
						db.UpdateMove(m.RoomCode, m.SessionID, idx, m.Game)
						return nil
					}
				}
			}
		}
	}
	return m, nil
}

// updateChessInput handles chess specific keys
func updateChessInput(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	// Spectators cannot move
	if m.MySide == "Spectator" {
		return m, nil
	}

	// Turn Check: "White" vs "Black"
	// MySide is "X" (Host) or "O" (Guest).
	// Host is White, Guest is Black.
	isMyTurn := false
	if m.MySide == "X" && m.Game.Turn == "White" {
		isMyTurn = true
	}
	if m.MySide == "O" && m.Game.Turn == "Black" {
		isMyTurn = true
	}

	if !isMyTurn {
		// Just navigation is allowed
	}

	isFlipped := (m.MySide == "O")

	switch msg.String() {
	case "up", "k":
		if isFlipped {
			if m.CursorR < 7 {
				m.CursorR++
			}
		} else {
			if m.CursorR > 0 {
				m.CursorR--
			}
		}
	case "down", "j":
		if isFlipped {
			if m.CursorR > 0 {
				m.CursorR--
			}
		} else {
			if m.CursorR < 7 {
				m.CursorR++
			}
		}
	case "left", "h":
		if isFlipped {
			if m.CursorC < 7 {
				m.CursorC++
			}
		} else {
			if m.CursorC > 0 {
				m.CursorC--
			}
		}
	case "right", "l":
		if isFlipped {
			if m.CursorC > 0 {
				m.CursorC--
			}
		} else {
			if m.CursorC < 7 {
				m.CursorC++
			}
		}
	case "f":
		m.UseNerdFont = !m.UseNerdFont
		return m, nil

	case "enter", " ":
		if !isMyTurn {
			return m, nil
		}

		// Chess Move Logic
		// Using m.Chess* fields
		// Map CursorR/C to internal board indices?
		// UI board is 0..7. m.Game.ChessBoard is [8][8].
		// Indices match.

		// Logic adapted from CONTEXT.md
		if m.ChessSelected {
			// Moving to target?
			// Check if (m.CursorR, m.CursorC) is in ValidMoves
			// Pos struct in game package
			// m.ChessValidMoves is map[game.Pos]bool

			// If clicking same piece -> deselect
			if m.CursorR == m.ChessSelRow && m.CursorC == m.ChessSelCol {
				m.ChessSelected = false
				m.ChessValidMoves = make(map[game.Pos]bool)
				return m, nil
			}

			// If valid move
			// But wait, m.ChessValidMoves is local state?
			// It should be calculated when selecting.
			// Is it persisted? No, local UI state.
			// OK.

			if m.ChessValidMoves[game.Pos{Row: m.CursorR, Col: m.CursorC}] {
				// Execute Move
				// Update Board locally then send to DB
				newBoard := m.Game.ChessBoard
				// Move piece
				newBoard[m.CursorR][m.CursorC] = newBoard[m.ChessSelRow][m.ChessSelCol]
				newBoard[m.ChessSelRow][m.ChessSelCol] = game.ChessPiece{} // Empty

				// Clear selection
				m.ChessSelected = false
				m.ChessValidMoves = make(map[game.Pos]bool)

				// Switch Turn
				nextTurn := "Black"
				if m.Game.Turn == "Black" {
					nextTurn = "White"
				}

				return m, func() tea.Msg {
					db.UpdateChessState(m.RoomCode, newBoard, nextTurn)
					return nil
				}
			}

			// If clicking another friendly piece -> select that instead
			p := m.Game.ChessBoard[m.CursorR][m.CursorC]
			if !p.IsEmpty() {
				// Check color
				isWhite := p.IsWhite
				myColorWhite := (m.MySide == "X")
				if isWhite == myColorWhite {
					// Select this one
					m.ChessSelected = true
					m.ChessSelRow = m.CursorR
					m.ChessSelCol = m.CursorC
					// Calc moves
					m.ChessValidMoves = game.GetValidChessMoves(m.Game.ChessBoard, m.CursorR, m.CursorC)
					return m, nil
				}
			}

			// Clicked empty/invalid -> deselect
			m.ChessSelected = false
			m.ChessValidMoves = make(map[game.Pos]bool)

		} else {
			// Selecting
			p := m.Game.ChessBoard[m.CursorR][m.CursorC]
			if !p.IsEmpty() {
				// Check color
				isWhite := p.IsWhite
				myColorWhite := (m.MySide == "X")
				if isWhite == myColorWhite {
					m.ChessSelected = true
					m.ChessSelRow = m.CursorR
					m.ChessSelCol = m.CursorC
					m.ChessValidMoves = game.GetValidChessMoves(m.Game.ChessBoard, m.CursorR, m.CursorC)
				}
			}
		}
	}
	return m, nil
}

func pollCmd(code string) tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		r, err := db.GetRoom(code)
		if err != nil {
			if err.Error() == "room does not exist" {
				return roomUpdateMsg{}
			}
			return pollErrorMsg(err)
		}
		if r == nil {
			return roomUpdateMsg{}
		}
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

func createRoomCmd(code, pid, name string, public bool, gameType string) tea.Cmd {
	return func() tea.Msg {
		if err := db.CreateRoom(code, pid, name, public, gameType); err != nil {
			return errMsg(err)
		}
		return roomCreatedMsg{code: code, gameType: gameType}
	}
}

func joinRoomCmd(code, pid, name string) tea.Cmd {
	return func() tea.Msg {
		if err := db.JoinRoom(code, pid, name); err != nil {
			return errMsg(err)
		}
		// Determine role async
		r, _ := db.GetRoom(code)
		side := "O"
		gameType := "tictactoe"
		if r != nil {
			gameType = r.GameType
			if r.PlayerX == pid {
				side = "X"
			} else if r.PlayerO == pid {
				side = "O"
			} else {
				side = "Spectator"
			}
		}
		return roomJoinedMsg{code: code, side: side, gameType: gameType}
	}
}

func generateCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
