package callbreak

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	TotalRounds  = 5
	TotalPlayers = 4
)

type GamePhase int

const (
	PhaseMenu GamePhase = iota
	PhasePlayerSelect
	PhaseBidding
	PhasePlaying
	PhaseTrickDone
	PhaseRoundResult
	PhaseGameOver
)

type AIMoveMsg struct{}
type TrickResultMsg struct{ Winner int }
type NextTrickMsg struct{}
type RoundEndMsg struct{}
type NextRoundMsg struct{}

// CallbreakState is the serializable game state for multiplayer sync via Firebase.
type CallbreakState struct {
	Phase         GamePhase  `json:"phase"`
	Round         int        `json:"round"`
	TrickNum      int        `json:"trickNum"`
	Hands         [4][]Card  `json:"hands"`
	Bids          [4]int     `json:"bids"`
	TricksWon     [4]int     `json:"tricksWon"`
	Scores        [4]float64 `json:"scores"`
	TableCards    [4]Card    `json:"tableCards"`
	TablePlayed   [4]bool    `json:"tablePlayed"`
	TrickLeader   int        `json:"trickLeader"`
	CurrentPlayer int        `json:"currentPlayer"`
	TrickWinner   int        `json:"trickWinner"`
	RoundScores   [4]float64 `json:"roundScores"`
	PlayerNames   [4]string  `json:"playerNames"`
	Message       string     `json:"message"`
	HumanPlayers  int        `json:"humanPlayers"`
	IsAI          [4]bool    `json:"isAI"`
}

type Model struct {
	Phase         GamePhase
	Round         int
	TrickNum      int
	Hands         [4][]Card
	Bids          [4]int
	TricksWon     [4]int
	Scores        [4]float64
	TableCards    [4]Card
	TablePlayed   [4]bool
	TrickLeader   int
	CurrentPlayer int
	SelectedCard  int
	ValidCards    []bool
	HumanBid      int
	Message       string
	TrickWinner   int
	RoundScores   [4]float64
	PlayerNames   [4]string
	Width         int
	Height        int

	// Menu / mode selection
	MenuSelection int     // 0 = AI, 1 = Create Room, 2 = Join Room
	HumanPlayers  int     // Number of human players (1=AI only, 2-4=multiplayer)
	IsAI          [4]bool // Which seats are AI

	// Multiplayer fields
	IsMultiplayer bool   // Whether this is a multiplayer game
	IsHost        bool   // Whether this client is the host
	MySeat        int    // Which seat (0-3) this player occupies
	RoomCode      string // Room code for multiplayer
}

func NewModel() Model {
	return Model{
		Phase:         PhaseMenu,
		PlayerNames:   [4]string{"You", "West", "North", "East"},
		Message:       "Select game mode and press ENTER",
		MenuSelection: 0,
		HumanPlayers:  1,
		IsAI:          [4]bool{false, true, true, true},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case AIMoveMsg:
		return m.handleAIMove()

	case TrickResultMsg:
		m.TrickWinner = msg.Winner
		m.Phase = PhaseTrickDone
		m.Message = fmt.Sprintf("✨  %s wins the trick!", m.PlayerNames[msg.Winner])
		return m, tea.Tick(1100*time.Millisecond, func(t time.Time) tea.Msg {
			return NextTrickMsg{}
		})

	case NextTrickMsg:
		m.TricksWon[m.TrickWinner]++
		m.TrickLeader = m.TrickWinner
		m.TableCards = [4]Card{}
		m.TablePlayed = [4]bool{}
		m.TrickNum++
		if m.TrickNum > 13 {
			for p := 0; p < 4; p++ {
				m.RoundScores[p] = CalcScore(m.Bids[p], m.TricksWon[p])
			}
			m.Phase = PhaseRoundResult
			m.Message = m.buildRoundMessage()
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return RoundEndMsg{}
			})
		}
		m.CurrentPlayer = m.TrickLeader
		m.Phase = PhasePlaying
		m.Message = fmt.Sprintf("Trick %d of 13 — %s leads", m.TrickNum, m.PlayerNames[m.TrickLeader])
		if m.IsAI[m.CurrentPlayer] {
			return m, tea.Tick(550*time.Millisecond, func(t time.Time) tea.Msg {
				return AIMoveMsg{}
			})
		}
		m.updateValidCards()
		return m, nil

	case RoundEndMsg:
		for p := 0; p < 4; p++ {
			m.Scores[p] += m.RoundScores[p]
		}
		m.Round++
		if m.Round > TotalRounds {
			m.Phase = PhaseGameOver
			m.Message = m.buildGameOverMessage()
			return m, nil
		}
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return NextRoundMsg{}
		})

	case NextRoundMsg:
		m.initRound()
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	switch m.Phase {

	case PhaseMenu:
		switch key {
		case "up", "k":
			if m.MenuSelection > 0 {
				m.MenuSelection--
			}
		case "down", "j":
			if m.MenuSelection < 2 {
				m.MenuSelection++
			}
		case "enter", " ":
			if m.MenuSelection == 0 {
				// AI mode — go straight to game
				m.IsMultiplayer = false
				m.HumanPlayers = 1
				m.IsAI = [4]bool{false, true, true, true}
				m.StartGame()
			} else if m.MenuSelection == 1 {
				// Create Room — show player count selection
				m.Phase = PhasePlayerSelect
				m.HumanPlayers = 2
				m.Message = "Choose number of human players (↑/↓), then ENTER"
			} else {
				// Join Room — parent UI handles this transition
				m.Message = "Joining room..."
			}
		}

	case PhasePlayerSelect:
		switch key {
		case "up", "k":
			if m.HumanPlayers < 4 {
				m.HumanPlayers++
			}
		case "down", "j":
			if m.HumanPlayers > 2 {
				m.HumanPlayers--
			}
		case "enter", " ":
			// Set AI flags: seat 0 is always human, fill rest based on count
			m.IsMultiplayer = true
			m.IsAI = [4]bool{false, true, true, true}
			for i := 1; i < m.HumanPlayers && i < 4; i++ {
				m.IsAI[i] = false
			}
			m.Message = fmt.Sprintf("Waiting for %d opponent(s)...", m.HumanPlayers-1)
			// The parent UI will handle room creation and waiting.
			// We signal readiness by staying in PhasePlayerSelect with IsMultiplayer=true.
			// The parent (ui/update.go) detects this and transitions to lobby flow.
			return m, nil
		case "esc":
			m.Phase = PhaseMenu
			m.Message = "Select game mode and press ENTER"
		}

	case PhaseBidding:
		if m.IsAI[m.CurrentPlayer] {
			break
		}
		// Only handle if it's our seat (seat 0 in single-player, or our seat in multiplayer)
		if m.CurrentPlayer != m.MySeat {
			break
		}
		switch key {
		case "up", "k":
			if m.HumanBid < 13 {
				m.HumanBid++
			}
		case "down", "j":
			if m.HumanBid > 1 {
				m.HumanBid--
			}
		case "enter", " ":
			m.Bids[m.MySeat] = m.HumanBid
			// AI players bid
			for p := 0; p < 4; p++ {
				if m.IsAI[p] {
					m.Bids[p] = AIBid(m.Hands[p])
				}
			}
			m.initPlaying()
		}

	case PhasePlaying:
		if m.IsAI[m.CurrentPlayer] {
			break
		}
		if m.CurrentPlayer != m.MySeat {
			break
		}
		switch key {
		case "left", "h":
			m.moveSelection(-1)
		case "right", "l":
			m.moveSelection(1)
		case "enter", " ":
			return m.humanPlayCard()
		}

	case PhaseGameOver:
		if key == "r" || key == "enter" {
			nm := NewModel()
			return nm, nil
		}
	}
	return m, nil
}

func (m *Model) initRound() {
	deck := ShuffleDeck(NewDeck())
	for p := 0; p < 4; p++ {
		m.Hands[p] = SortHand(deck[p*13 : (p+1)*13])
		m.Bids[p] = 0
		m.TricksWon[p] = 0
		m.RoundScores[p] = 0
	}
	m.TrickNum = 1
	m.TrickLeader = 0
	m.CurrentPlayer = 0
	m.TableCards = [4]Card{}
	m.TablePlayed = [4]bool{}
	m.HumanBid = 2
	m.Phase = PhaseBidding
	m.Message = fmt.Sprintf("Round %d of %d  ⟩  Set your bid (↑/↓), then press ENTER", m.Round, TotalRounds)
}

func (m *Model) initPlaying() {
	m.TrickNum = 1
	m.TrickLeader = 0
	m.CurrentPlayer = 0
	m.TableCards = [4]Card{}
	m.TablePlayed = [4]bool{}
	m.Phase = PhasePlaying
	m.updateValidCards()
	m.Message = fmt.Sprintf("Round %d · Trick 1  ⟩  You lead! (←/→ select, ENTER play)", m.Round)
}

func (m *Model) updateValidCards() {
	leadCard := m.TableCards[m.TrickLeader]
	m.ValidCards = ValidPlays(m.Hands[m.MySeat], leadCard)
	if len(m.Hands[m.MySeat]) == 0 {
		return
	}
	if m.SelectedCard >= len(m.Hands[m.MySeat]) {
		m.SelectedCard = len(m.Hands[m.MySeat]) - 1
	}
	if !m.ValidCards[m.SelectedCard] {
		for i, v := range m.ValidCards {
			if v {
				m.SelectedCard = i
				break
			}
		}
	}
}

func (m *Model) moveSelection(dir int) {
	n := len(m.Hands[m.MySeat])
	if n == 0 {
		return
	}
	next := m.SelectedCard + dir
	for next >= 0 && next < n {
		if m.ValidCards[next] {
			m.SelectedCard = next
			return
		}
		next += dir
	}
}

func (m Model) humanPlayCard() (Model, tea.Cmd) {
	if m.IsAI[m.CurrentPlayer] || len(m.Hands[m.MySeat]) == 0 {
		return m, nil
	}
	if m.CurrentPlayer != m.MySeat {
		return m, nil
	}
	if !m.ValidCards[m.SelectedCard] {
		m.Message = "⚠  Must follow suit! Choose a highlighted card."
		return m, nil
	}
	return m.playCard(m.MySeat, m.SelectedCard)
}

func (m Model) handleAIMove() (Model, tea.Cmd) {
	p := m.CurrentPlayer
	if !m.IsAI[p] {
		// It's a human player's turn
		m.updateValidCards()
		m.Message = fmt.Sprintf("Trick %d · %s's turn  ⟩  (←/→ select, ENTER to play)", m.TrickNum, m.PlayerNames[p])
		return m, nil
	}
	idx := AIPlay(m.Hands[p], m.TableCards, m.TablePlayed, m.TrickLeader)
	return m.playCard(p, idx)
}

func (m Model) playCard(player, cardIdx int) (Model, tea.Cmd) {
	hand := m.Hands[player]
	card := hand[cardIdx]
	m.Hands[player] = append(hand[:cardIdx:cardIdx], hand[cardIdx+1:]...)
	m.TableCards[player] = card
	m.TablePlayed[player] = true

	allPlayed := m.TablePlayed[0] && m.TablePlayed[1] && m.TablePlayed[2] && m.TablePlayed[3]
	if allPlayed {
		winner := FindTrickWinner(m.TableCards, m.TablePlayed, m.TrickLeader)
		return m, tea.Tick(700*time.Millisecond, func(t time.Time) tea.Msg {
			return TrickResultMsg{Winner: winner}
		})
	}

	m.CurrentPlayer = (player + 1) % 4
	if !m.IsAI[m.CurrentPlayer] && m.CurrentPlayer == m.MySeat {
		m.updateValidCards()
		m.Message = fmt.Sprintf("Trick %d · Your turn  ⟩  (←/→ select, ENTER to play)", m.TrickNum)
		return m, nil
	}
	// Next player is AI or a remote human — schedule AI move or wait for sync
	if m.IsAI[m.CurrentPlayer] {
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return AIMoveMsg{}
		})
	}
	// Remote human in multiplayer — host will sync state
	return m, nil
}

func (m Model) buildRoundMessage() string {
	return fmt.Sprintf("Round %d complete! Results updating...", m.Round)
}

func (m Model) buildGameOverMessage() string {
	winner := 0
	for p := 1; p < 4; p++ {
		if m.Scores[p] > m.Scores[winner] {
			winner = p
		}
	}
	result := "You lose"
	if winner == m.MySeat {
		result = "YOU WIN!"
	}
	return fmt.Sprintf("Game Over! %s  (%s scored %.1f) — [R] Restart", result, m.PlayerNames[winner], m.Scores[winner])
}

func (m *Model) StartGame() {
	m.Round = 1
	m.Scores = [4]float64{}
	m.MySeat = 0 // Default: seat 0 for single-player
	m.initRound()
}

func (m Model) IsGameOver() bool {
	return m.Phase == PhaseGameOver
}

// ToState exports the current model state for Firebase sync.
func (m Model) ToState() CallbreakState {
	return CallbreakState{
		Phase:         m.Phase,
		Round:         m.Round,
		TrickNum:      m.TrickNum,
		Hands:         m.Hands,
		Bids:          m.Bids,
		TricksWon:     m.TricksWon,
		Scores:        m.Scores,
		TableCards:    m.TableCards,
		TablePlayed:   m.TablePlayed,
		TrickLeader:   m.TrickLeader,
		CurrentPlayer: m.CurrentPlayer,
		TrickWinner:   m.TrickWinner,
		RoundScores:   m.RoundScores,
		PlayerNames:   m.PlayerNames,
		Message:       m.Message,
		HumanPlayers:  m.HumanPlayers,
		IsAI:          m.IsAI,
	}
}

// ApplyState updates the model from a Firebase-synced state (for guests).
func (m *Model) ApplyState(s CallbreakState) {
	m.Phase = s.Phase
	m.Round = s.Round
	m.TrickNum = s.TrickNum
	m.Hands = s.Hands
	m.Bids = s.Bids
	m.TricksWon = s.TricksWon
	m.Scores = s.Scores
	m.TableCards = s.TableCards
	m.TablePlayed = s.TablePlayed
	m.TrickLeader = s.TrickLeader
	m.CurrentPlayer = s.CurrentPlayer
	m.TrickWinner = s.TrickWinner
	m.RoundScores = s.RoundScores
	m.PlayerNames = s.PlayerNames
	m.Message = s.Message
	m.HumanPlayers = s.HumanPlayers
	m.IsAI = s.IsAI
}
