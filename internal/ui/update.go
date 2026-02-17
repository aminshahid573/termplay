package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aminshahid573/termplay/internal/callbreak"
	"github.com/aminshahid573/termplay/internal/chess"
	"github.com/aminshahid573/termplay/internal/db"
	"github.com/aminshahid573/termplay/internal/snake"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
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

type cbRoomCreatedMsg struct {
	code string
}

type cbRoomJoinedMsg struct {
	code string
	seat int
}

type cbRoomUpdateMsg db.CBRoom

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// 1. Handle background polling (Highest Priority, Non-Blocking)
	if roomMsg, ok := msg.(roomUpdateMsg); ok {
		m.Game = db.Room(roomMsg)
		// Auto-transition from Lobby to Game
		if m.State == StateLobby && m.Game.PlayerO != "" {
			if m.Game.GameType == "callbreak" {
				// This branch should not be hit for new Callbreak implementation
				// as we use CBRoom struct now.
			} else {
				m.State = StateGame
			}
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

	// Handle Callbreak Room Updates
	if cbMsg, ok := msg.(cbRoomUpdateMsg); ok {
		m.CBRoom = db.CBRoom(cbMsg)

		// If Room deleted/closed
		if m.CBRoom.HostID == "" {
			m.Err = fmt.Errorf("Room closed by host")
			m.State = StateMenu
			m.RoomCode = ""
			m.Busy = false
			return m, nil
		}

		// Transition Logic
		if m.State == StateLobby {
			// If we have enough players
			if len(m.CBRoom.Players) >= m.CBRoom.PlayerCount {
				m.Callbreak.IsMultiplayer = true

				// Configure AI seats
				for i := 0; i < 4; i++ {
					if i < m.CBRoom.PlayerCount {
						m.Callbreak.IsAI[i] = false
					} else {
						m.Callbreak.IsAI[i] = true
						m.Callbreak.PlayerNames[i] = fmt.Sprintf("Bot%d", i)
					}
				}

				// Map players to seats using PlayerSeats from DB
				for pid, seatIdx := range m.CBRoom.PlayerSeats {
					if seatIdx >= 0 && seatIdx < 4 {
						name := m.CBRoom.Players[pid]
						m.Callbreak.PlayerNames[seatIdx] = name
						if pid == m.SessionID {
							m.Callbreak.MySeat = seatIdx
							m.Callbreak.IsHost = (seatIdx == 0)
						}
					}
				}

				m.Callbreak.StartGame() // Reset/Init game state locally
				m.State = StateCallbreak
			}
		}

		if m.State == StateCallbreak {
			// Sync State ONLY if not Host
			// Host is the source of truth for game state.
			if !m.Callbreak.IsHost {
				cb := m.Callbreak
				cb.ApplyState(m.CBRoom.GameState)
				m.Callbreak = cb
			}

			// Rebuild names from DB state
			for pid, seatIdx := range m.CBRoom.PlayerSeats {
				if seatIdx >= 0 && seatIdx < 4 {
					m.Callbreak.PlayerNames[seatIdx] = m.CBRoom.Players[pid]
				}
			}
		}

		return m, pollCBRoomCmd(m.RoomCode)
	}

	// 2. Handle Polling Errors
	if err, ok := msg.(pollErrorMsg); ok {
		m.Err = err
		// Retry polling after delay
		if m.SelectedGame == "callbreak" || m.CBRoom.Code != "" {
			return m, pollCBRoomCmd(m.RoomCode)
		}
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

	case cbRoomCreatedMsg:
		m.Busy = false
		m.RoomCode = msg.code
		m.MySide = "Host" // Callbreak Host
		m.SelectedGame = "callbreak"

		m.Cleanup.Mu.Lock()
		m.Cleanup.RoomCode = msg.code
		m.Cleanup.IsHost = true
		m.Cleanup.Mu.Unlock()

		m.Callbreak = callbreak.NewModel()
		m.Callbreak.IsMultiplayer = true
		m.Callbreak.IsHost = true
		m.Callbreak.MySeat = 0
		m.Callbreak.Width = m.Width
		m.Callbreak.Height = m.Height

		m.State = StateLobby
		return m, pollCBRoomCmd(msg.code)

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

	case cbRoomJoinedMsg:
		m.Busy = false
		m.RoomCode = msg.code
		m.SelectedGame = "callbreak"
		// m.MySide? Callbreak doesn't use X/O.

		m.Cleanup.Mu.Lock()
		m.Cleanup.RoomCode = msg.code
		m.Cleanup.IsHost = false
		m.Cleanup.Mu.Unlock()

		m.Callbreak = callbreak.NewModel()
		m.Callbreak.IsMultiplayer = true
		m.Callbreak.IsHost = false
		m.Callbreak.MySeat = msg.seat
		m.Callbreak.Width = m.Width
		m.Callbreak.Height = m.Height

		m.State = StateLobby // Wait in lobby until full
		return m, pollCBRoomCmd(msg.code)

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
		m.Snake.TermW = msg.Width
		m.Snake.TermH = msg.Height
		m.Callbreak.Width = msg.Width
		m.Callbreak.Height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Handle snake game ticks and input
	if m.State == StateSnakeGame {
		switch msg := msg.(type) {
		case snake.TickMsg:
			m.Snake, cmd = m.Snake.Update(msg)
			if m.Snake.WantsQuit {
				m.Snake.WantsQuit = false
				m.State = StateGameSelect
				m.MenuIndex = 0
				return m, nil
			}
			return m, cmd
		case tea.KeyMsg:
			m.Snake, cmd = m.Snake.Update(msg)
			if m.Snake.WantsQuit {
				m.Snake.WantsQuit = false
				m.State = StateGameSelect
				m.MenuIndex = 0
				return m, nil
			}
			return m, cmd
		}
		return m, nil
	}

	// Handle Callbreak game ticks and input
	if m.State == StateCallbreak {
		switch msg := msg.(type) {
		case callbreak.AIMoveMsg, callbreak.TrickResultMsg, callbreak.NextTrickMsg, callbreak.RoundEndMsg, callbreak.NextRoundMsg, tea.KeyMsg:
			if k, ok := msg.(tea.KeyMsg); ok && k.String() == "q" {
				// Quit Logic?
				m.State = StateGameSelect
				m.MenuIndex = 0
				return m, nil
			}

			// Update Callbreak Model
			m.Callbreak, cmd = m.Callbreak.Update(msg)

			// If Host, Push State to Firebase
			if m.Callbreak.IsHost && m.RoomCode != "" {
				// We should probably debounce this or only do it on state changes
				// For now, simple approach:
				db.UpdateCBGameState(m.RoomCode, m.Callbreak.ToState())
			}

			// Handle Menu Selection (Join Room)
			if m.Callbreak.Phase == callbreak.PhaseMenu {
				if keyMsg, ok := msg.(tea.KeyMsg); ok {
					if keyMsg.String() == "enter" || keyMsg.String() == " " {
						if m.Callbreak.MenuSelection == 2 {
							// Join Room Selected -> Go to Code Input
							m.State = StateInputCode
							m.TextInput.Placeholder = "4-Digit Code"
							m.TextInput.SetValue("")
							m.TextInput.Focus()
							return m, textinput.Blink
						}
					}
				}
			}

			// If multiplayer was selected and player count confirmed (Local AI game setup)
			if m.Callbreak.IsMultiplayer && m.Callbreak.Phase == callbreak.PhasePlayerSelect {
				// This block is legacy from local multiplayer setup?
				// No, local "IsMultiplayer" flag usage in callbreak package might be confusing.
				// In new flow, we set IsMultiplayer=true for online too.
				// But PhasePlayerSelect is for choosing 2/3/4 players locally.
				// For Online, we skip this or use it to config room?
				// In `updateGameSelect`, we set `m.State = StateCallbreak`.
				// The Callbreak model starts at `PhasePlayerSelect` by default.
				// If we want to create an ONLINE room, we intercept before this?
				// Actually, `updateGameSelect` sets `m.State = StateCallbreak`.
				// The user sees the Callbreak Menu (Play vs AI, Multiplayer).
				// If they select Multiplayer there, `m.Callbreak` sets `IsMultiplayer=true`.
				// We need to catch that transition to redirect to `StateCreateConfig`.

				m.SelectedGame = "callbreak"
				m.State = StateMenu // Go to Main Menu (Create/Join)
				m.MenuIndex = 0
				return m, nil
			}

			return m, cmd
		}
		return m, nil
	}

	// Global Popup Handler
	if m.PopupActive {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.PopupType == PopupRestart {
				// ... Restart logic (omit for brevity if not changing) ...
				// We need to include restart logic or it will be lost
				switch msg.String() {
				case "1":
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
					next := m.Game.Winner
					if next == "" {
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
					isHost := (m.MySide == "X")
					if m.RoomCode != "" {
						if m.SelectedGame == "callbreak" {
							db.LeaveCBRoom(m.RoomCode, m.SessionID)
						} else {
							db.LeaveRoom(m.RoomCode, m.SessionID, isHost)
						}
					}
					m.PopupActive = false
					m.State = StateMenu
					m.Err = nil
					m.RoomCode = ""
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
			if m.MenuIndex < 3 { // 0: TicTacToe, 1: Chess, 2: Snake, 3: Callbreak
				m.MenuIndex++
			}
		case "enter":
			switch m.MenuIndex {
			case 0:
				m.SelectedGame = "tictactoe"
				m.State = StateMenu
				m.MenuIndex = 0
			case 1:
				m.SelectedGame = "chess"
				m.State = StateMenu
				m.MenuIndex = 0
			case 2:
				// Snake is single-player — go directly to snake game
				m.Snake = snake.InitialModel()
				m.Snake.TermW = m.Width
				m.Snake.TermH = m.Height
				m.State = StateSnakeGame
				return m, snake.TickCmd()
			case 3:
				// Callbreak — show callbreak menu (AI or Multiplayer)
				m.Callbreak = callbreak.NewModel()
				m.Callbreak.Width = m.Width
				m.Callbreak.Height = m.Height
				m.State = StateCallbreak
				return m, nil
			}
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
			}

			if gameType == "callbreak" {
				return m, createCBRoomCmd(code, m.SessionID, m.MyName, m.Callbreak.HumanPlayers)
			}

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

// --- Public List Logic ---
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
		if msg.String() == "q" {
			m.PopupActive = true
			m.PopupType = PopupLeave
			return m, nil
		}
		if msg.String() == "esc" {
			if m.Game.GameType == "chess" && m.ChessSelected {
				m.ChessSelected = false
				m.ChessValidMoves = make(map[chess.Pos]bool)
				return m, nil
			}
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

	// Turn enforcement only on Enter/Space
	if (msg.String() == "enter" || msg.String() == " ") && !isMyTurn {
		return m, nil
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
		log.Info("Key pressed", "key", msg.String())
		log.Info("Turn check", "mySide", m.MySide, "turn", m.Game.Turn, "isMyTurn", isMyTurn)
		if !isMyTurn {
			return m, nil
		}

		// Chess Move Logic
		if m.ChessSelected {
			// If clicking same piece -> deselect
			if m.CursorR == m.ChessSelRow && m.CursorC == m.ChessSelCol {
				m.ChessSelected = false
				m.ChessValidMoves = make(map[chess.Pos]bool)
				return m, nil
			}

			// If valid move
			if m.ChessValidMoves[chess.Pos{Row: m.CursorR, Col: m.CursorC}] {
				log.Info("Executing move", "from", m.ChessSelRow, m.ChessSelCol, "to", m.CursorR, m.CursorC)
				// Execute Move
				newState := chess.ApplyMove(m.Game.ChessState, chess.Pos{Row: m.ChessSelRow, Col: m.ChessSelCol}, chess.Pos{Row: m.CursorR, Col: m.CursorC}, "Q")

				// Clear selection
				m.ChessSelected = false
				m.ChessValidMoves = make(map[chess.Pos]bool)

				return m, func() tea.Msg {
					err := db.UpdateChessState(m.RoomCode, newState)
					if err != nil {
						log.Error("UpdateChessState failed", "err", err)
						return errMsg(fmt.Errorf("move failed: %v", err))
					}
					return nil
				}
			} else {
				log.Info("Invalid move attempted", "target", m.CursorR, m.CursorC)
			}

			// If clicking another friendly piece -> select that instead
			p := m.Game.ChessState.Board[m.CursorR][m.CursorC]
			if !p.IsEmpty() {
				// Check color
				isWhite := p.IsWhite
				myColorWhite := (m.MySide == "X")
				if isWhite == myColorWhite {
					log.Info("Switching selection", "to", m.CursorR, m.CursorC)
					// Select this one
					m.ChessSelected = true
					m.ChessSelRow = m.CursorR
					m.ChessSelCol = m.CursorC
					// Calc moves
					m.ChessValidMoves = chess.GetLegalMoves(m.Game.ChessState, m.CursorR, m.CursorC)
					log.Info("Legal moves calculated", "count", len(m.ChessValidMoves))
					return m, nil
				}
			}

			// Clicked empty/invalid -> deselect
			m.ChessSelected = false
			m.ChessValidMoves = make(map[chess.Pos]bool)

		} else {
			// Selecting
			p := m.Game.ChessState.Board[m.CursorR][m.CursorC]
			if !p.IsEmpty() {
				// Check color
				isWhite := p.IsWhite
				myColorWhite := (m.MySide == "X")
				if isWhite == myColorWhite {
					log.Info("Selected piece", "row", m.CursorR, "col", m.CursorC)
					m.ChessSelected = true
					m.ChessSelRow = m.CursorR
					m.ChessSelCol = m.CursorC
					m.ChessValidMoves = chess.GetLegalMoves(m.Game.ChessState, m.CursorR, m.CursorC)
					log.Info("Legal moves calculated", "count", len(m.ChessValidMoves))
				} else {
					log.Info("Clicked opponent piece", "isWhite", isWhite, "myColorWhite", myColorWhite)
				}
			} else {
				log.Info("Clicked empty square")
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
		// Try Regular Room First
		if err := db.JoinRoom(code, pid, name); err == nil {
			// Success
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

		// If failed, try Callbreak Room
		if seat, err := db.JoinCBRoom(code, pid, name); err == nil {
			return cbRoomJoinedMsg{code: code, seat: seat}
		} else {
			// Return original error or generic
			return errMsg(fmt.Errorf("room not found or full"))
		}
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

// --- Callbreak Specific Commands ---

func createCBRoomCmd(code, hostID, hostName string, playerCount int) tea.Cmd {
	return func() tea.Msg {
		if err := db.CreateCBRoom(code, hostID, hostName, playerCount); err != nil {
			return errMsg(err)
		}
		return cbRoomCreatedMsg{code: code}
	}
}

func joinCBRoomCmd(code, playerID, playerName string) tea.Cmd {
	return func() tea.Msg {
		seat, err := db.JoinCBRoom(code, playerID, playerName)
		if err != nil {
			return errMsg(err)
		}
		return cbRoomJoinedMsg{code: code, seat: seat}
	}
}

func pollCBRoomCmd(code string) tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		r, err := db.GetCBRoom(code)
		if err != nil {
			if err.Error() == "room not found" {
				// Room might have been closed
				return cbRoomUpdateMsg{HostID: ""}
			}
			return pollErrorMsg(err)
		}
		if r == nil {
			return cbRoomUpdateMsg{HostID: ""}
		}
		return cbRoomUpdateMsg(*r)
	})
}
