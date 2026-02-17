package ui

import (
	"fmt"
	"github.com/aminshahid573/termplay/internal/chess"
	"github.com/aminshahid573/termplay/internal/db"
	"github.com/aminshahid573/termplay/internal/styles"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

func (m Model) View() string {
	// Global Popup
	if m.PopupActive {
		var box string
		if m.PopupType == PopupRestart {
			msg := "Who should start next?"
			content := lipgloss.JoinVertical(lipgloss.Center,
				styles.Title.Render("RESTART GAME"),
				"\n"+msg+"\n",
				lipgloss.JoinHorizontal(lipgloss.Center,
					styles.ItemFocused.Render("[1] Random"),
					"  ",
					styles.ItemFocused.Render("[2] Winner Starts"),
				),
				"\n",
				styles.Subtle.Render("[Esc] Cancel"),
			)
			box = styles.PopupBox.Render(content)
		} else {
			// Default to Leave Popup
			msg := "Are you sure you want to leave?\n(Room will be deleted if you are Host)"
			box = styles.PopupBox.Render(
				fmt.Sprintf("%s\n\n[Y] Yes    [N] No", msg),
			)
		}
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
	}

	var content string
	var helpText string

	switch m.State {
	case StateNameInput:
		// Clean Name Input
		content = lipgloss.JoinVertical(lipgloss.Center,
			"\n",
			styles.Title.Render("WELCOME"),
			"\n\n",
			m.TextInput.View(),
			"\n",
		)
		helpText = "Enter: Confirm • Ctrl+C: Quit"

	case StateMenu:
		opts := []string{"Create Room", "Join with Code", "Public Rooms", "Quit"}
		var renderedOpts []string
		for i, opt := range opts {
			if i == m.MenuIndex {
				renderedOpts = append(renderedOpts, styles.ItemFocused.Render(" "+opt+" "))
			} else {
				renderedOpts = append(renderedOpts, styles.ItemBlurred.Render(" "+opt+" "))
			}
		}
		list := lipgloss.JoinVertical(lipgloss.Left, renderedOpts...)
		content = lipgloss.JoinVertical(lipgloss.Center,
			styles.Title.Render("MAIN MENU"),
			list,
		)
		helpText = "↑/↓: Navigate • Enter: Select"

	case StateCreateConfig:
		pubLabel := "  Public"
		privLabel := "  Private"
		var pubRendered, privRendered string
		if m.IsPublicCreate {
			pubRendered = styles.ItemFocused.Render("● " + pubLabel)
			privRendered = styles.ItemBlurred.Render("○ " + privLabel)
		} else {
			pubRendered = styles.ItemBlurred.Render("○ " + pubLabel)
			privRendered = styles.ItemFocused.Render("● " + privLabel)
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			styles.Title.Render("ROOM SETTINGS"),
			"Select Visibility:",
			"\n",
			lipgloss.JoinVertical(lipgloss.Left, pubRendered, privRendered),
			"\n",
		)
		if m.Err != nil {
			content = lipgloss.JoinVertical(lipgloss.Center, content, styles.Err.Render(m.Err.Error()))
		}
		helpText = "↑/↓: Change • Enter: Create • Esc: Back"

	case StateInputCode:
		errView := ""
		if m.Err != nil {
			errView = styles.Base.Foreground(lipgloss.Color("#F25D94")).Render("\n" + m.Err.Error())
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			styles.Title.Render("JOIN ROOM"),
			styles.ListContainer.Width(30).Render( // Re-use container for consistent look
				m.TextInput.View(),
			),
			errView,
		)
		helpText = "Enter: Join • Esc: Back"

	case StatePublicList:
		content = renderPublicList(m)
		// Add error display if fetch failed
		if m.Err != nil {
			errText := styles.Base.Foreground(lipgloss.Color("#F25D94")).Render(fmt.Sprintf("\nError: %v", m.Err))
			content = lipgloss.JoinVertical(lipgloss.Center, content, errText)
		}
		helpText = "↑/↓: Navigate • Enter: Join • Type: Filter • Esc: Back"

	case StateLobby:
		code := styles.Base.Foreground(lipgloss.Color("#e3b7ff")).Bold(true).Render(m.RoomCode)
		content = lipgloss.JoinVertical(lipgloss.Center,
			styles.Title.Render("LOBBY"),
			fmt.Sprintf("CODE: %s", code),
			"\nWaiting for opponent...",
			styles.Subtle.Render("Share this code with your friend"),
		)
		helpText = "Esc: Leave Room"

	case StateGameSelect:
		content = renderGameSelect(m)
		helpText = "↑/↓: Navigate • Enter: Select"

	case StateSnakeGame:
		// Snake handles its own rendering; we just center it
		m.Snake.TermW = m.Width
		m.Snake.TermH = m.Height
		snakeView := m.Snake.View()
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, snakeView)
		}
		return snakeView

	case StateGame:
		content = renderGame(m)
		if m.Game.GameType == "chess" {
			helpText = "arrows/hjkl move • enter/space select • esc deselect • f font • q quit"
		} else {
			helpText = "Arrows: Move • Space: Place • R: Restart • Q: Quit"
		}
	}

	// Combine Content + Help Footer
	finalView := lipgloss.JoinVertical(lipgloss.Center,
		content,
		"\n",
		styles.Subtle.Render(helpText),
	)

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, finalView)
}

// --- List Rendering Logic ---

func renderPublicList(m Model) string {
	// Filter logic
	var openRooms, fullRooms []db.Room
	filter := strings.ToUpper(m.SearchInput.Value())

	for _, r := range m.PublicRooms {
		// Show ALL by default (filter == ""), or match filter
		if filter == "" || strings.Contains(r.Code, filter) || strings.Contains(strings.ToUpper(r.PlayerXName), filter) {
			if r.PlayerO == "" {
				openRooms = append(openRooms, r)
			} else {
				fullRooms = append(fullRooms, r)
			}
		}
	}

	// Container is 70 wide; border uses 2, padding(0,1) uses 2, so inner = 66
	listWidth := 66

	var listContent []string

	// 1. Search Bar (Borderless inside the box)
	searchView := m.SearchInput.View()
	listContent = append(listContent, searchView)
	listContent = append(listContent, "") // Spacer

	// 2. Open Rooms Section
	listContent = append(listContent, renderSectionHeader(" Open Rooms ", listWidth, "✓ Joinable"))
	if len(openRooms) == 0 {
		listContent = append(listContent, styles.Subtle.Render("  No open rooms found"))
	} else {
		for i, r := range openRooms {
			isSelected := (i == m.ListSelectedRow)
			listContent = append(listContent, renderRoomItem(r, isSelected, listWidth))
		}
	}
	listContent = append(listContent, "")

	// 3. Full Rooms Section
	listContent = append(listContent, renderSectionHeader(" Full Rooms ", listWidth, "Spectate"))
	if len(fullRooms) == 0 {
		listContent = append(listContent, styles.Subtle.Render("  No full rooms"))
	} else {
		for i, r := range fullRooms {
			isSelected := (i+len(openRooms) == m.ListSelectedRow)
			listContent = append(listContent, renderRoomItem(r, isSelected, listWidth))
		}
	}

	// Wrap everything in the Bordered Container
	inner := lipgloss.JoinVertical(lipgloss.Left, listContent...)

	return lipgloss.JoinVertical(lipgloss.Center,
		styles.Title.Render("PUBLIC ROOMS"),
		styles.ListContainer.Render(inner),
	)
}

func renderSectionHeader(text string, width int, info string) string {
	char := "─"
	infoRendered := ""
	if info != "" {
		infoRendered = " " + styles.Subtle.Render(info)
	}

	titleRendered := styles.SectionTitle.Render(text)

	remaining := width - lipgloss.Width(titleRendered) - lipgloss.Width(infoRendered)
	if remaining < 0 {
		remaining = 0
	}

	line := styles.SectionLine.Render(strings.Repeat(char, remaining))
	return titleRendered + " " + line + infoRendered
}

func renderRoomItem(r db.Room, focused bool, width int) string {
	name := fmt.Sprintf("%s's Room", r.PlayerXName)
	code := r.Code

	style := styles.ItemBlurred
	infoStyle := styles.InfoTextBlurred

	if focused {
		style = styles.ItemFocused
		infoStyle = styles.InfoTextFocused
	}

	rightText := fmt.Sprintf(" %s ", code)
	rightRendered := infoStyle.Render(rightText)
	rightWidth := lipgloss.Width(rightRendered)

	availableWidth := width - rightWidth - 2
	name = truncate.StringWithTail(name, uint(availableWidth), "...")

	nameWidth := lipgloss.Width(name)
	gap := strings.Repeat(" ", max(0, width-nameWidth-rightWidth))

	return style.Render(name + gap + rightRendered)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func renderGameSelect(m Model) string {
	opts := []string{"Tic Tac Toe", "Chess", "Snake"}
	var renderedOpts []string
	for i, opt := range opts {
		if i == m.MenuIndex {
			renderedOpts = append(renderedOpts, styles.ItemFocused.Render(" "+opt+" "))
		} else {
			renderedOpts = append(renderedOpts, styles.ItemBlurred.Render(" "+opt+" "))
		}
	}
	list := lipgloss.JoinVertical(lipgloss.Left, renderedOpts...)
	return lipgloss.JoinVertical(lipgloss.Center,
		styles.Title.Render("SELECT GAME"),
		list,
	)
}

func renderGame(m Model) string {
	if m.Game.GameType == "chess" {
		return renderChessGame(m)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		fmt.Sprintf("%s (Wins: %d)", m.Game.PlayerXName, m.Game.WinsX),
		"  VS  ",
		fmt.Sprintf("%s (Wins: %d)", m.Game.PlayerOName, m.Game.WinsO),
	)

	var rows []string
	for r := 0; r < 3; r++ {
		var cols []string
		for c := 0; c < 3; c++ {
			idx := r*3 + c
			val := m.Game.Board[idx]
			style := styles.Cell
			if m.Game.GameType == "tictactoe" {
				// No change, tictactoe is default
			}

			isWinCell := false
			for _, wIdx := range m.Game.WinningLine {
				if idx == wIdx {
					isWinCell = true
				}
			}
			if isWinCell {
				style = styles.CellWin
			}

			if m.Game.Status == "playing" && m.Game.Turn == m.MySide {
				if r == m.CursorR && c == m.CursorC {
					style = styles.CellSelected
				}
			}

			content := " "
			if val == "X" {
				content = styles.XStyle.Render("X")
			}
			if val == "O" {
				content = styles.OStyle.Render("O")
			}
			cols = append(cols, style.Render(content))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	}
	board := lipgloss.JoinVertical(lipgloss.Center, rows...)

	status := ""
	if m.Game.Status == "waiting" {
		status = "Opponent disconnected. Waiting..."
	} else if m.Game.Status == "finished" {
		res := "DRAW"
		if m.Game.Winner != "" {
			res = m.Game.Winner + " WINS!"
		}
		status = fmt.Sprintf("%s", res)
	} else {
		turn := m.Game.Turn
		status = fmt.Sprintf("Turn: %s", turn)
		if m.MySide == "Spectator" {
			status = fmt.Sprintf("[SPECTATING] Turn: %s", turn)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		styles.Title.Render("TICTACTOE"),
		header,
		"\n",
		board,
		"\n",
		status,
	)
}

func renderChessGame(m Model) string {
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		fmt.Sprintf("%s (White)", m.Game.PlayerXName),
		"  VS  ",
		fmt.Sprintf("%s (Black)", m.Game.PlayerOName),
	)

	sqW, sqH := computeChessSquareSize(m.Width, m.Height)

	isFlipped := (m.MySide == "O")

	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	ranks := []string{"8", "7", "6", "5", "4", "3", "2", "1"}

	if isFlipped {
		files = []string{"h", "g", "f", "e", "d", "c", "b", "a"}
		ranks = []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	}

	var rankLabels []string
	var rankLabelsR []string
	var gridRows []string

	for r, rank := range ranks {
		rankLabel := lipgloss.NewStyle().
			Foreground(styles.ChessLabel).
			Bold(true).
			Width(2).
			Height(sqH).
			Align(lipgloss.Right, lipgloss.Center).
			PaddingRight(1).
			Render(rank)
		rankLabels = append(rankLabels, rankLabel)

		rankLabelR := lipgloss.NewStyle().
			Foreground(styles.ChessLabel).
			Bold(true).
			Width(2).
			Height(sqH).
			Align(lipgloss.Left, lipgloss.Center).
			PaddingLeft(1).
			Render(rank)
		rankLabelsR = append(rankLabelsR, rankLabelR)

		var rowCells []string
		for c := range files {
			isLight := (r+c)%2 == 0
			bg := styles.ChessDarkSquare
			if isLight {
				bg = styles.ChessLightSquare
			}

			br, bc := r, c
			if isFlipped {
				br = 7 - r
				bc = 7 - c
			}

			piece := m.Game.ChessState.Board[br][bc]
			fg := styles.ChessBlackPiece
			if piece.IsWhite {
				fg = styles.ChessWhitePiece
			}

			isCursor := (m.CursorR == br && m.CursorC == bc)
			isSelected := (m.ChessSelected && m.ChessSelRow == br && m.ChessSelCol == bc)
			isValidMove := m.ChessValidMoves[chess.Pos{Row: br, Col: bc}]
			isCapture := isValidMove && !m.Game.ChessState.Board[br][bc].IsEmpty()

			if isSelected {
				bg = styles.ChessSelected
			} else if isCapture {
				bg = styles.ChessCapture
			} else if isValidMove {
				bg = styles.ChessHighlight
			}

			if isCursor && !isSelected {
				bg = lipgloss.Color("#FFD700")
			}

			cell := lipgloss.NewStyle().
				Background(bg).
				Foreground(fg).
				Bold(true).
				Width(sqW).
				Height(sqH).
				Align(lipgloss.Center, lipgloss.Center).
				Render(chessPieceSymbol(piece, m.UseNerdFont))

			rowCells = append(rowCells, cell)
		}
		gridRows = append(gridRows, lipgloss.JoinHorizontal(lipgloss.Top, rowCells...))
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, gridRows...)

	// Frame the grid
	gridWithBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.ChessBorder).
		Render(grid)

	// Join ranks
	leftRanks := lipgloss.JoinVertical(lipgloss.Top, rankLabels...)
	rightRanks := lipgloss.JoinVertical(lipgloss.Top, rankLabelsR...)

	// Add margin to ranks to account for top border
	leftRanks = lipgloss.NewStyle().MarginTop(1).Render(leftRanks)
	rightRanks = lipgloss.NewStyle().MarginTop(1).Render(rightRanks)

	boardArea := lipgloss.JoinHorizontal(lipgloss.Top, leftRanks, gridWithBorder, rightRanks)

	// File labels
	var fileLabels []string
	for _, f := range files {
		fl := lipgloss.NewStyle().
			Foreground(styles.ChessLabel).
			Bold(true).
			Width(sqW).
			Align(lipgloss.Center).
			Render(f)
		fileLabels = append(fileLabels, fl)
	}

	// PaddingLeft = RankWidth (3) + Border (1) = 4
	fileLabelRow := lipgloss.NewStyle().
		PaddingLeft(4).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, fileLabels...))
	fileLabelRowTop := lipgloss.NewStyle().
		PaddingLeft(4).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, fileLabels...))

	// Status
	var statusText string
	var statusColor lipgloss.Color = lipgloss.Color("#CCCCCC")
	isBold := false

	if m.Game.Status == "waiting" {
		statusText = "Opponent disconnected. Waiting..."
	} else if m.Game.Status == "finished" {
		isBold = true
		statusColor = styles.ChessCapture
		if m.Game.Winner == "Draw" {
			statusText = "STALEMATE - DRAW!"
		} else if m.Game.Winner != "" {
			statusText = "CHECKMATE! " + strings.ToUpper(m.Game.Winner) + " WINS!"
		} else {
			statusText = "GAME OVER"
		}
	} else {
		isMyTurn := (m.MySide == "X" && m.Game.Turn == "White") || (m.MySide == "O" && m.Game.Turn == "Black")
		inCheck := chess.IsInCheck(m.Game.ChessState.Board, m.Game.Turn == "White")

		if inCheck {
			isBold = true
			statusColor = styles.ChessCapture
			statusText = "CHECK! "
		}

		if m.MySide == "Spectator" {
			statusText += "[SPECTATING]"
		} else if isMyTurn {
			statusText += "Your turn"
		} else {
			opponentName := m.Game.PlayerOName
			if m.MySide == "O" {
				opponentName = m.Game.PlayerXName
			}
			statusText += opponentName + "'s turn"
		}
	}

	status := lipgloss.NewStyle().
		Foreground(statusColor).
		Bold(isBold).
		Render(statusText)

	content := lipgloss.JoinVertical(lipgloss.Center,
		styles.Title.Render("CHESS"),
		header,
		"",
		fileLabelRowTop,
		"",
		boardArea,
		"",
		fileLabelRow,
		"",
		status,
	)

	blockBorder := lipgloss.Border{
		Top:         "▄",
		Bottom:      "▀",
		Left:        "▐",
		Right:       "▌",
		TopLeft:     "▗",
		TopRight:    "▖",
		BottomLeft:  "▝",
		BottomRight: "▘",
	}

	bordered := lipgloss.NewStyle().
		Border(blockBorder).
		BorderForeground(styles.ChessBorder).
		Padding(1, 2).
		Render(content)

	return bordered
}

func computeChessSquareSize(termWidth, termHeight int) (sqW, sqH int) {
	availW := termWidth - 8
	availH := termHeight - 14 // Increased buffer for UI elements

	if availW < 16 || availH < 8 {
		return 2, 1
	}

	maxFromW := availW / (8 * 2)
	maxFromH := availH / 8

	cellUnit := maxFromW
	if maxFromH < cellUnit {
		cellUnit = maxFromH
	}
	if cellUnit > 3 {
		cellUnit = 3
	}
	if cellUnit < 1 {
		cellUnit = 1
	}

	return cellUnit * 2, cellUnit
}

// Nerd Font chess icons (md-chess_* from Material Design Icons)
const (
	nfKing   = "\U000F0857" // nf-md-chess_king
	nfQueen  = "\U000F085A" // nf-md-chess_queen
	nfRook   = "\U000F085B" // nf-md-chess_rook
	nfBishop = "\U000F085C" // nf-md-chess_bishop
	nfKnight = "\U000F0858" // nf-md-chess_knight
	nfPawn   = "\U000F0859" // nf-md-chess_pawn
)

// Unicode fallback chess symbols (distinct sets)
const (
	// White
	ucWhiteKing   = "♔"
	ucWhiteQueen  = "♕"
	ucWhiteRook   = "♖"
	ucWhiteBishop = "♗"
	ucWhiteKnight = "♘"
	ucWhitePawn   = "♙"

	// Black
	ucBlackKing   = "♚"
	ucBlackQueen  = "♛"
	ucBlackRook   = "♜"
	ucBlackBishop = "♝"
	ucBlackKnight = "♞"
	ucBlackPawn   = "♟"
)

func chessPieceSymbol(p chess.Piece, useNerd bool) string {
	if p.IsEmpty() {
		return ""
	}

	if useNerd {
		switch p.Type {
		case "K":
			return nfKing
		case "Q":
			return nfQueen
		case "R":
			return nfRook
		case "B":
			return nfBishop
		case "N":
			return nfKnight
		case "P":
			return nfPawn
		}
	}

	// Unicode fallback
	if p.IsWhite {
		switch p.Type {
		case "K":
			return ucWhiteKing
		case "Q":
			return ucWhiteQueen
		case "R":
			return ucWhiteRook
		case "B":
			return ucWhiteBishop
		case "N":
			return ucWhiteKnight
		case "P":
			return ucWhitePawn
		}
	} else {
		switch p.Type {
		case "K":
			return ucBlackKing
		case "Q":
			return ucBlackQueen
		case "R":
			return ucBlackRook
		case "B":
			return ucBlackBishop
		case "N":
			return ucBlackKnight
		case "P":
			return ucBlackPawn
		}
	}
	return ""
}
