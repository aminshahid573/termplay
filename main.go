package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"google.golang.org/api/option"
)

// --- Configuration ---

const (
	DBURL        = "https://YOUR-FIREBASE-DB-URL.firebasedatabase.app"
	CredPath     = "serviceAccount.json"
	SyncInterval = 200 * time.Millisecond
	Host         = "localhost"
	Port         = 23234
)

var dbClient *db.Client

// --- Styles ---

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	errColor  = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F55385"}
	winColor  = lipgloss.AdaptiveColor{Light: "#00FF00", Dark: "#00FF00"}
	loseColor = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF0000"}

	// Large Cell Style
	cellStyle = lipgloss.NewStyle().
			Width(11).Height(5). // Bigger cells
			Align(lipgloss.Center, lipgloss.Center).
			Border(lipgloss.DoubleBorder(), false, true, false, true).
			BorderForeground(subtle)

	cursorStyle = cellStyle.Copy().
			Background(lipgloss.Color("236")).
			BorderForeground(special)

	// Big Text for X and O
	xStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Blink(false) // Pink
	oStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Blink(false)  // Blue

	titleStyle = lipgloss.NewStyle().Foreground(special).Bold(true).Background(lipgloss.Color("235")).Padding(0, 1)
	inputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(highlight).Padding(0, 1)
)

// --- Firebase Data Structures ---

type RoomData struct {
	Board       [9]string `json:"board"`
	Turn        string    `json:"turn"` // "X" or "O"
	PlayerX     string    `json:"playerX"`
	PlayerO     string    `json:"playerO"`
	PlayerXName string    `json:"playerXName"`
	PlayerOName string    `json:"playerOName"`
	Winner      string    `json:"winner"`
	Status      string    `json:"status"` // "waiting", "playing", "finished"
}

// --- Bubble Tea Model ---

type sessionState int

const (
	stateNameInput sessionState = iota
	stateMenu
	stateInputCode
	stateLobby
	stateGame
)

type model struct {
	// Terminal Dims
	width  int
	height int

	// Local State
	state       sessionState
	textInput   textinput.Model
	err         error
	cursorR     int
	cursorC     int
	sessionID   string
	mySide      string // "X" or "O"
	myName      string
	roomCode    string
	quitting    bool
	rematchMenu int // 0 = Winner Starts, 1 = Random

	// Synced Game State
	game RoomData
}

// --- Init ---

func initialModel(sess ssh.Session) model {
	ti := textinput.New()
	ti.Placeholder = "Enter your name..."
	ti.CharLimit = 12
	ti.Focus()

	id := fmt.Sprintf("user_%d", time.Now().UnixNano())
	if sess != nil {
		id = sess.RemoteAddr().String()
	}

	return model{
		state:     stateNameInput,
		textInput: ti,
		sessionID: id,
		game: RoomData{
			Board: [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "},
		},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// --- Update ---

type dbUpdateMsg RoomData
type roomCreatedMsg string
type roomJoinedMsg string
type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg
		return m, nil
	}

	switch m.state {
	case stateNameInput:
		return m.updateNameInput(msg)
	case stateMenu:
		return m.updateMenu(msg)
	case stateInputCode:
		return m.updateCodeInput(msg)
	case stateLobby, stateGame:
		return m.updateGame(msg)
	}

	return m, cmd
}

// 1. Name Input
func (m model) updateNameInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if len(m.textInput.Value()) > 0 {
				m.myName = m.textInput.Value()
				m.state = stateMenu
				return m, nil
			}
		}
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// 2. Main Menu
func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			newCode := generateRoomCode()
			m.roomCode = newCode
			m.mySide = "X"
			return m, createRoomCmd(newCode, m.sessionID, m.myName)
		case "j":
			m.state = stateInputCode
			m.textInput.SetValue("")
			m.textInput.Placeholder = "4-Digit Code"
			m.textInput.CharLimit = 4
			return m, textinput.Blink
		case "q":
			m.quitting = true
			return m, tea.Quit
		}
	case roomCreatedMsg:
		m.state = stateLobby
		return m, pollGameCmd(m.roomCode)
	}
	return m, nil
}

// 3. Join Code Input
func (m model) updateCodeInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			code := strings.ToUpper(m.textInput.Value())
			if len(code) > 0 {
				return m, joinRoomCmd(code, m.sessionID, m.myName)
			}
		}
		if msg.Type == tea.KeyEsc {
			m.state = stateMenu
			return m, nil
		}
	case roomJoinedMsg:
		m.roomCode = string(msg)
		m.mySide = "O"
		m.state = stateGame
		return m, pollGameCmd(m.roomCode)
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// 4. Game Logic
func (m model) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dbUpdateMsg:
		prevWinner := m.game.Winner
		m.game = RoomData(msg)

		// Transition waiting -> playing
		if m.state == stateLobby && m.game.Status == "playing" {
			m.state = stateGame
		}
		
		// If game was reset (Winner cleared), clear error and reset cursor
		if prevWinner != "" && m.game.Winner == "" && m.game.Status == "playing" {
			m.err = nil
			m.cursorR = 1
			m.cursorC = 1
		}
		
		return m, pollGameCmd(m.roomCode)

	case tea.KeyMsg:
		// HOST REMATCH MENU
		if m.game.Status == "finished" && m.mySide == "X" {
			switch msg.String() {
			case "left", "h":
				m.rematchMenu = 0
			case "right", "l":
				m.rematchMenu = 1
			case "enter", " ":
				// Trigger Rematch
				rule := "winner"
				if m.rematchMenu == 1 { rule = "random" }
				return m, triggerRematchCmd(m.roomCode, rule, m.game.Winner)
			case "q":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// CLIENT WAIT (Allow Quit)
		if m.game.Status == "finished" && m.mySide == "O" {
			if msg.String() == "q" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		
		if m.game.Status != "playing" {
			return m, nil
		}

		// GAMEPLAY
		switch msg.String() {
		case "up", "k":
			if m.cursorR > 0 { m.cursorR-- }
		case "down", "j":
			if m.cursorR < 2 { m.cursorR++ }
		case "left", "h":
			if m.cursorC > 0 { m.cursorC-- }
		case "right", "l":
			if m.cursorC < 2 { m.cursorC++ }
		case " ", "enter":
			index := m.cursorR*3 + m.cursorC
			if m.game.Turn == m.mySide && m.game.Board[index] == " " {
				return m, makeMoveCmd(m.roomCode, m.game, index, m.mySide)
			}
		}
	}
	return m, nil
}

// --- View ---

func (m model) View() string {
	if m.quitting { return "Bye!\n" }

	// Use a StringBuilder to construct the content
	var doc strings.Builder

	// Title
	doc.WriteString(titleStyle.Render("ULTIMATE TIC-TAC-TOE SSH"))
	doc.WriteString("\n\n")

	if m.err != nil {
		doc.WriteString(lipgloss.NewStyle().Foreground(errColor).Render(fmt.Sprintf("ERROR: %v", m.err)) + "\n\n")
	}

	// Route based on state
	switch m.state {
	case stateNameInput:
		doc.WriteString("What is your name?\n\n")
		doc.WriteString(m.textInput.View())
		doc.WriteString("\n\n(Press Enter)")

	case stateMenu:
		doc.WriteString(fmt.Sprintf("Hello, %s!\n\n", m.myName))
		doc.WriteString(" [ N ] Create New Room\n")
		doc.WriteString(" [ J ] Join Room\n")
		doc.WriteString(" [ Q ] Quit")

	case stateInputCode:
		doc.WriteString("Enter 4-Digit Room Code:\n\n")
		doc.WriteString(m.textInput.View())
		doc.WriteString("\n\n(Esc to cancel)")

	case stateLobby:
		doc.WriteString("Room Created!\n\n")
		doc.WriteString(fmt.Sprintf("CODE: %s\n\n", lipgloss.NewStyle().Background(highlight).Foreground(lipgloss.Color("255")).Bold(true).Padding(0, 1).Render(m.roomCode)))
		doc.WriteString("Waiting for opponent to join...")

	case stateGame:
		// --- Header: Players ---
		pX := m.game.PlayerXName
		if pX == "" { pX = "Player X" }
		pO := m.game.PlayerOName
		if pO == "" { pO = "Player O" }
		
		// Highlight current turn in header
		headerX := xStyle.Render(pX)
		headerO := oStyle.Render(pO)
		if m.game.Turn == "X" && m.game.Status == "playing" { headerX = lipgloss.NewStyle().Underline(true).Inherit(xStyle).Render(pX) }
		if m.game.Turn == "O" && m.game.Status == "playing" { headerO = lipgloss.NewStyle().Underline(true).Inherit(oStyle).Render(pO) }

		doc.WriteString(fmt.Sprintf("%s  vs  %s\n\n", headerX, headerO))

		// --- The Board ---
		var rows []string
		for r := 0; r < 3; r++ {
			var cols []string
			for c := 0; c < 3; c++ {
				idx := r*3 + c
				val := m.game.Board[idx]
				
				// ASCII Art for X and O
				styledVal := ""
				if val == "X" { 
					styledVal = xStyle.Render("X") // Use simple char but big font
				} else if val == "O" {
					styledVal = oStyle.Render("O")
				}

				// Styling
				currentStyle := cellStyle
				if m.game.Status == "playing" && m.game.Turn == m.mySide {
					if r == m.cursorR && c == m.cursorC {
						currentStyle = cursorStyle
					}
				}
				cols = append(cols, currentStyle.Render(styledVal))
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
		}
		
		// Assemble Board
		boardView := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(highlight).Render(
			lipgloss.JoinVertical(lipgloss.Left, rows...),
		)
		doc.WriteString(boardView + "\n\n")

		// --- Footer / Status ---
		if m.game.Status == "finished" {
			if m.game.Winner == m.mySide {
				doc.WriteString(lipgloss.NewStyle().Foreground(winColor).Bold(true).Render("YOU WIN!") + "\n\n")
			} else if m.game.Winner == "" {
				doc.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render("DRAW GAME") + "\n\n")
			} else {
				doc.WriteString(lipgloss.NewStyle().Foreground(loseColor).Bold(true).Render("YOU LOSE!") + "\n\n")
			}

			// Rematch Menu for Host
			if m.mySide == "X" {
				doc.WriteString("Select next first player:\n")
				opt1 := "[ Winner Starts ]"
				opt2 := "[ Random Start ]"
				
				activeOpt := lipgloss.NewStyle().Foreground(special).Bold(true)
				if m.rematchMenu == 0 { opt1 = activeOpt.Render(opt1) }
				if m.rematchMenu == 1 { opt2 = activeOpt.Render(opt2) }
				
				doc.WriteString(fmt.Sprintf("%s   %s", opt1, opt2))
			} else {
				doc.WriteString("Waiting for host to restart...")
			}
		} else {
			if m.game.Turn == m.mySide {
				doc.WriteString(lipgloss.NewStyle().Background(special).Foreground(lipgloss.Color("235")).Bold(true).Padding(0, 1).Render(" YOUR TURN "))
			} else {
				doc.WriteString("Opponent is thinking...")
			}
		}
	}

	// CENTER THE WHOLE CONTENT
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, doc.String())
}

// --- DB Commands ---

func createRoomCmd(code, playerID, name string) tea.Cmd {
	return func() tea.Msg {
		ref := dbClient.NewRef("rooms/" + code)
		data := RoomData{
			Board:       [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "},
			Turn:        "X",
			PlayerX:     playerID,
			PlayerXName: name,
			Status:      "waiting",
		}
		if err := ref.Set(context.Background(), data); err != nil {
			return errMsg(err)
		}
		return roomCreatedMsg(code)
	}
}

func joinRoomCmd(code, playerID, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ref := dbClient.NewRef("rooms/" + code)
		
		var room RoomData
		if err := ref.Get(ctx, &room); err != nil {
			return errMsg(fmt.Errorf("room invalid"))
		}
		
		if room.PlayerO != "" {
			return errMsg(fmt.Errorf("room is full"))
		}

		updates := map[string]interface{}{
			"playerO":     playerID,
			"playerOName": name,
			"status":      "playing",
		}
		if err := ref.Update(ctx, updates); err != nil {
			return errMsg(err)
		}

		return roomJoinedMsg(code)
	}
}

func pollGameCmd(code string) tea.Cmd {
	return tea.Tick(SyncInterval, func(t time.Time) tea.Msg {
		var room RoomData
		if err := dbClient.NewRef("rooms/"+code).Get(context.Background(), &room); err != nil {
			return errMsg(err)
		}
		return dbUpdateMsg(room)
	})
}

func makeMoveCmd(code string, current RoomData, index int, player string) tea.Cmd {
	return func() tea.Msg {
		board := current.Board
		board[index] = player
		
		winner := checkWinner(board)
		status := "playing"
		if winner != "" || checkDraw(board) {
			status = "finished"
		}

		nextTurn := "O"
		if player == "O" { nextTurn = "X" }

		updates := map[string]interface{}{
			"board":  board,
			"turn":   nextTurn,
			"winner": winner,
			"status": status,
		}

		dbClient.NewRef("rooms/"+code).Update(context.Background(), updates)
		return nil
	}
}

func triggerRematchCmd(code, rule, prevWinner string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ref := dbClient.NewRef("rooms/" + code)

		fn := func(tn db.TransactionNode) (interface{}, error) {
			var r RoomData
			if err := tn.Unmarshal(&r); err != nil { return nil, err }
			
			// 1. Reset Board
			r.Board = [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "}
			
			// 2. Determine Turn
			newTurn := "X"
			if rule == "random" {
				if rand.Intn(2) == 0 { newTurn = "O" }
			} else if rule == "winner" {
				if prevWinner == "O" { newTurn = "O" }
				// If prevWinner was "" (Draw), defaults to X, or keep random? default X.
			}
			
			r.Turn = newTurn
			r.Winner = "" // CRITICAL: Must be empty string
			r.Status = "playing"
			
			return r, nil
		}

		if err := ref.Transaction(ctx, fn); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

// --- Helpers ---

func generateRoomCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func checkWinner(b [9]string) string {
	wins := [][]int{
		{0,1,2},{3,4,5},{6,7,8},
		{0,3,6},{1,4,7},{2,5,8},
		{0,4,8},{2,4,6},
	}
	for _, w := range wins {
		if b[w[0]] != " " && b[w[0]] == b[w[1]] && b[w[1]] == b[w[2]] {
			return b[w[0]]
		}
	}
	return ""
}

func checkDraw(b [9]string) bool {
	for _, v := range b { if v == " " { return false } }
	return true
}

// --- Server Main ---

func main() {
	// Firebase
	opt := option.WithCredentialsFile(CredPath)
	config := &firebase.Config{DatabaseURL: DBURL}
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil { log.Fatal("Firebase Init Error", "err", err) }

	dbClient, err = app.Database(context.Background())
	if err != nil { log.Fatal("DB Init Error", "err", err) }

	// SSH Server
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", Host, Port)),
		wish.WithHostKeyPath("ssh_host_key"), // Fixes the key changed warning
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			logging.Middleware(),
			activeterm.Middleware(),
		),
	)
	if err != nil { log.Error("Server Error", "err", err) }

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting Game Server", "host", Host, "port", Port)

	go func() {
		if err = s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			log.Error("Listen Error", "err", err)
			done <- nil
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil { log.Error("Shutdown Error", "err", err) }
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return initialModel(s), []tea.ProgramOption{tea.WithAltScreen()}
}
