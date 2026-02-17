package chess

import (
	"crypto/sha256"
	"fmt"
)

type Piece struct {
	Type     string `json:"type"` // "K", "Q", "R", "B", "N", "P" or ""
	IsWhite  bool   `json:"isWhite"`
	HasMoved bool   `json:"hasMoved"` // For castling
}

func (p Piece) IsEmpty() bool {
	return p.Type == ""
}

type Pos struct {
	Row, Col int
}

// Move represents a full move details
type Move struct {
	From, To      Pos
	Piece         Piece
	Captured      Piece
	IsEnPassant   bool
	IsCastling    bool
	IsPromotion   bool
	PromotionType string // "Q", "R", "B", "N"
}

// GameState holds all info needed for full rules
type GameState struct {
	Board           [8][8]Piece    `json:"board"`
	Turn            string         `json:"turn"`            // "White" or "Black"
	EnPassantTarget *Pos           `json:"enPassantTarget"` // Square behind pawn moved 2 steps
	HalfMoveClock   int            `json:"halfMoveClock"`   // For 50-move rule
	FullMoveNumber  int            `json:"fullMoveNumber"`
	History         map[string]int `json:"history"` // For 3-fold repetition
	Status          string         `json:"status"`  // "playing", "checkmate", "stalemate", "draw"
	Winner          string         `json:"winner"`  // "White", "Black", "Draw", ""
}

// StartingBoard: rows 0-7 map to ranks 8-1
var StartingBoard = [8][8]Piece{
	{{"R", false, false}, {"N", false, false}, {"B", false, false}, {"Q", false, false}, {"K", false, false}, {"B", false, false}, {"N", false, false}, {"R", false, false}},
	{{"P", false, false}, {"P", false, false}, {"P", false, false}, {"P", false, false}, {"P", false, false}, {"P", false, false}, {"P", false, false}, {"P", false, false}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{"P", true, false}, {"P", true, false}, {"P", true, false}, {"P", true, false}, {"P", true, false}, {"P", true, false}, {"P", true, false}, {"P", true, false}},
	{{"R", true, false}, {"N", true, false}, {"B", true, false}, {"Q", true, false}, {"K", true, false}, {"B", true, false}, {"N", true, false}, {"R", true, false}},
}

func NewGame() GameState {
	return GameState{
		Board:           StartingBoard,
		Turn:            "White",
		EnPassantTarget: nil,
		HalfMoveClock:   0,
		FullMoveNumber:  1,
		History:         make(map[string]int),
		Status:          "playing",
	}
}

func inBounds(r, c int) bool {
	return r >= 0 && r < 8 && c >= 0 && c < 8
}

// IsInCheck checks if the given color's King is under attack
func IsInCheck(board [8][8]Piece, isWhite bool) bool {
	// 1. Find King
	var kingPos Pos
	found := false
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := board[r][c]
			if p.Type == "K" && p.IsWhite == isWhite {
				kingPos = Pos{r, c}
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return false // Should not happen in valid game
	}

	// 2. Check for attacks from opponent pieces
	opponentWhite := !isWhite

	// Check lines (Rook/Queen)
	dirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, d := range dirs {
		for i := 1; i < 8; i++ {
			r, c := kingPos.Row+d[0]*i, kingPos.Col+d[1]*i
			if !inBounds(r, c) {
				break
			}
			p := board[r][c]
			if !p.IsEmpty() {
				if p.IsWhite == opponentWhite && (p.Type == "R" || p.Type == "Q") {
					return true
				}
				break // Blocked
			}
		}
	}

	// Check diagonals (Bishop/Queen)
	diag := [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for _, d := range diag {
		for i := 1; i < 8; i++ {
			r, c := kingPos.Row+d[0]*i, kingPos.Col+d[1]*i
			if !inBounds(r, c) {
				break
			}
			p := board[r][c]
			if !p.IsEmpty() {
				if p.IsWhite == opponentWhite && (p.Type == "B" || p.Type == "Q") {
					return true
				}
				break // Blocked
			}
		}
	}

	// Check Knights
	knights := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	for _, d := range knights {
		r, c := kingPos.Row+d[0], kingPos.Col+d[1]
		if inBounds(r, c) {
			p := board[r][c]
			if p.IsWhite == opponentWhite && p.Type == "N" {
				return true
			}
		}
	}

	// Check Pawns
	// If I am White King, Black pawns attack from (r-1, c-1) or (r-1, c+1)
	// Because Black pawn at (r-1) moves to (r)

	attackRow := kingPos.Row - 1 // Look "up" for Black pawns
	if opponentWhite {
		attackRow = kingPos.Row + 1 // Look "down" for White pawns
	}
	// The king is at kingPos. We check if there's a pawn at (r - dir, c +/- 1)
	// Actually easier: look from King's perspective.
	// If I am White King, Black pawns attack from (r-1, c+/-1)
	// Wait, Black pawns are at low index (row 1), move to high index.
	// White pawns are at high index (row 6), move to low index.

	// Let's use the standard "attacked by pawn" logic
	// If I am White King (at r,c), I am attacked by Black Pawn if Black Pawn is at (r-1, c-1) or (r-1, c+1)
	// Because Black pawn at (r-1) moves to (r)

	if inBounds(attackRow, kingPos.Col-1) {
		p := board[attackRow][kingPos.Col-1]
		if p.IsWhite == opponentWhite && p.Type == "P" {
			return true
		}
	}
	if inBounds(attackRow, kingPos.Col+1) {
		p := board[attackRow][kingPos.Col+1]
		if p.IsWhite == opponentWhite && p.Type == "P" {
			return true
		}
	}

	// Check King (adjacency)
	kingMoves := [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	for _, d := range kingMoves {
		r, c := kingPos.Row+d[0], kingPos.Col+d[1]
		if inBounds(r, c) {
			p := board[r][c]
			if p.IsWhite == opponentWhite && p.Type == "K" {
				return true
			}
		}
	}

	return false
}

// GetPseudoLegalMoves generates moves based on piece movement rules only
func GetPseudoLegalMoves(board [8][8]Piece, r, c int, enPassantTarget *Pos) map[Pos]bool {
	moves := make(map[Pos]bool)
	piece := board[r][c]
	if piece.IsEmpty() {
		return moves
	}

	isWhite := piece.IsWhite

	tryAdd := func(nr, nc int) bool {
		if !inBounds(nr, nc) {
			return false
		}
		target := board[nr][nc]
		if target.IsEmpty() {
			moves[Pos{nr, nc}] = true
			return true
		}
		if target.IsWhite != isWhite {
			moves[Pos{nr, nc}] = true
		}
		return false
	}

	slide := func(dr, dc int) {
		for i := 1; i < 8; i++ {
			if !tryAdd(r+dr*i, c+dc*i) {
				break
			}
		}
	}

	switch piece.Type {
	case "P":
		dir := -1
		startRow := 6
		if !isWhite {
			dir = 1
			startRow = 1
		}
		// Forward 1
		if inBounds(r+dir, c) && board[r+dir][c].IsEmpty() {
			moves[Pos{r + dir, c}] = true
			// Forward 2
			if r == startRow && inBounds(r+dir*2, c) && board[r+dir*2][c].IsEmpty() {
				moves[Pos{r + dir*2, c}] = true
			}
		}
		// Capture
		for _, dc := range []int{-1, 1} {
			nr, nc := r+dir, c+dc
			if inBounds(nr, nc) {
				target := board[nr][nc]
				if !target.IsEmpty() && target.IsWhite != isWhite {
					moves[Pos{nr, nc}] = true
				}
				// En Passant
				if enPassantTarget != nil && enPassantTarget.Row == nr && enPassantTarget.Col == nc {
					moves[Pos{nr, nc}] = true
				}
			}
		}

	case "R":
		slide(0, 1)
		slide(0, -1)
		slide(1, 0)
		slide(-1, 0)
	case "B":
		slide(1, 1)
		slide(1, -1)
		slide(-1, 1)
		slide(-1, -1)
	case "Q":
		slide(0, 1)
		slide(0, -1)
		slide(1, 0)
		slide(-1, 0)
		slide(1, 1)
		slide(1, -1)
		slide(-1, 1)
		slide(-1, -1)
	case "N":
		knightMoves := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
		for _, km := range knightMoves {
			tryAdd(r+km[0], c+km[1])
		}
	case "K":
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr == 0 && dc == 0 {
					continue
				}
				tryAdd(r+dr, c+dc)
			}
		}
	}
	return moves
}

// GetLegalMoves filters pseudo-legal moves for king safety and adds castling
func GetLegalMoves(state GameState, r, c int) map[Pos]bool {
	piece := state.Board[r][c]
	if piece.IsEmpty() || (piece.IsWhite && state.Turn != "White") || (!piece.IsWhite && state.Turn != "Black") {
		return make(map[Pos]bool) // Not your turn or empty
	}

	pseudo := GetPseudoLegalMoves(state.Board, r, c, state.EnPassantTarget)
	legal := make(map[Pos]bool)

	// Filter moves that leave King in check
	for dest := range pseudo {
		// Simulate move
		tempBoard := state.Board

		// Handle regular move/capture
		tempBoard[dest.Row][dest.Col] = piece
		tempBoard[r][c] = Piece{}

		// Handle En Passant capture removal
		if piece.Type == "P" && state.EnPassantTarget != nil && dest.Row == state.EnPassantTarget.Row && dest.Col == state.EnPassantTarget.Col {
			// Captured pawn is at [r][dest.Col]
			tempBoard[r][dest.Col] = Piece{}
		}

		if !IsInCheck(tempBoard, piece.IsWhite) {
			legal[dest] = true
		}
	}

	// Add Castling (Special case: logic usually not in pseudo moves)
	if piece.Type == "K" && !piece.HasMoved && !IsInCheck(state.Board, piece.IsWhite) {
		row := 7
		if !piece.IsWhite {
			row = 0
		}

		// Kingside (G file)
		// Check Rook at H file
		rookK := state.Board[row][7]
		if rookK.Type == "R" && !rookK.HasMoved {
			// Check path clear F, G
			if state.Board[row][5].IsEmpty() && state.Board[row][6].IsEmpty() {
				// Check squares not attacked (F, G)
				// We already checked King (E) not in check
				// Check F
				tempBoardF := state.Board
				tempBoardF[row][5] = piece
				tempBoardF[r][c] = Piece{}
				if !IsInCheck(tempBoardF, piece.IsWhite) {
					// Check G (Destination)
					tempBoardG := state.Board
					tempBoardG[row][6] = piece
					tempBoardG[r][c] = Piece{}
					if !IsInCheck(tempBoardG, piece.IsWhite) {
						legal[Pos{row, 6}] = true
					}
				}
			}
		}

		// Queenside (C file)
		// Check Rook at A file
		rookQ := state.Board[row][0]
		if rookQ.Type == "R" && !rookQ.HasMoved {
			// Check path clear B, C, D
			if state.Board[row][1].IsEmpty() && state.Board[row][2].IsEmpty() && state.Board[row][3].IsEmpty() {
				// Check squares not attacked (C, D) - King moves E -> C, passes through D
				// Check D
				tempBoardD := state.Board
				tempBoardD[row][3] = piece
				tempBoardD[r][c] = Piece{}
				if !IsInCheck(tempBoardD, piece.IsWhite) {
					// Check C (Destination)
					tempBoardC := state.Board
					tempBoardC[row][2] = piece
					tempBoardC[r][c] = Piece{}
					if !IsInCheck(tempBoardC, piece.IsWhite) {
						legal[Pos{row, 2}] = true
					}
				}
			}
		}
	}

	return legal
}

// ApplyMove updates the game state with the move
func ApplyMove(state GameState, from, to Pos, promotionType string) GameState {
	piece := state.Board[from.Row][from.Col]
	target := state.Board[to.Row][to.Col]

	// 1. Move Piece
	state.Board[to.Row][to.Col] = piece
	state.Board[from.Row][from.Col] = Piece{}
	state.Board[to.Row][to.Col].HasMoved = true // Mark moved

	// 2. Handle Castling
	if piece.Type == "K" && (to.Col-from.Col == 2 || from.Col-to.Col == 2) {
		row := from.Row
		if to.Col == 6 { // Kingside
			rook := state.Board[row][7]
			state.Board[row][5] = rook
			state.Board[row][5].HasMoved = true
			state.Board[row][7] = Piece{}
		} else if to.Col == 2 { // Queenside
			rook := state.Board[row][0]
			state.Board[row][3] = rook
			state.Board[row][3].HasMoved = true
			state.Board[row][0] = Piece{}
		}
	}

	// 3. Handle En Passant Capture
	if piece.Type == "P" && state.EnPassantTarget != nil && to.Row == state.EnPassantTarget.Row && to.Col == state.EnPassantTarget.Col {
		// Remove captured pawn
		capRow := from.Row // Same row as starting pawn
		state.Board[capRow][to.Col] = Piece{}
	}

	// 4. Update En Passant Target for next turn
	state.EnPassantTarget = nil
	if piece.Type == "P" && (to.Row-from.Row == 2 || from.Row-to.Row == 2) {
		midRow := (from.Row + to.Row) / 2
		state.EnPassantTarget = &Pos{midRow, from.Col}
	}

	// 5. Handle Promotion
	if piece.Type == "P" && (to.Row == 0 || to.Row == 7) {
		// Default to Queen if not specified
		pType := "Q"
		if promotionType != "" {
			pType = promotionType
		}
		state.Board[to.Row][to.Col].Type = pType
	}

	// 6. Update Clocks
	if piece.Type == "P" || !target.IsEmpty() {
		state.HalfMoveClock = 0
	} else {
		state.HalfMoveClock++
	}

	if state.Turn == "Black" {
		state.FullMoveNumber++
		state.Turn = "White"
	} else {
		state.Turn = "Black"
	}

	// 7. Check Game End Conditions
	state.Status = "playing"

	// Has legal moves?
	hasMoves := false
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			if state.Board[r][c].IsWhite == (state.Turn == "White") {
				moves := GetLegalMoves(state, r, c)
				if len(moves) > 0 {
					hasMoves = true
					break
				}
			}
		}
		if hasMoves {
			break
		}
	}

	if !hasMoves {
		if IsInCheck(state.Board, state.Turn == "White") {
			state.Status = "finished"
			state.Winner = "Black"
			if state.Turn == "Black" {
				state.Winner = "White"
			}
		} else {
			state.Status = "finished"
			state.Winner = "Draw" // Stalemate
		}
	}

	// 50 Move Rule
	if state.HalfMoveClock >= 100 {
		state.Status = "finished"
		state.Winner = "Draw"
	}

	// Insufficient Material
	if IsInsufficientMaterial(state.Board) {
		state.Status = "finished"
		state.Winner = "Draw"
	}

	// 3-fold repetition
	raw := fmt.Sprintf("%v-%s-%v", state.Board, state.Turn, state.EnPassantTarget)
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
	if state.History == nil {
		state.History = make(map[string]int)
	}
	state.History[key]++
	if state.History[key] >= 3 {
		state.Status = "finished"
		state.Winner = "Draw"
	}

	return state
}

func IsInsufficientMaterial(board [8][8]Piece) bool {
	// Count pieces
	whitePieces := []string{}
	blackPieces := []string{}

	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			p := board[r][c]
			if !p.IsEmpty() {
				if p.Type == "P" || p.Type == "Q" || p.Type == "R" {
					return false // Major pieces or pawns = sufficient
				}
				if p.IsWhite {
					whitePieces = append(whitePieces, p.Type)
				} else {
					blackPieces = append(blackPieces, p.Type)
				}
			}
		}
	}

	// K vs K
	if len(whitePieces) == 1 && len(blackPieces) == 1 {
		return true
	}

	// K+N vs K or K+B vs K
	if (len(whitePieces) == 2 && len(blackPieces) == 1) || (len(whitePieces) == 1 && len(blackPieces) == 2) {
		return true
	}

	// K+N vs K+N (technically a draw often, but not forced. Let's stick to FIDE basics)
	// FIDE: Draw if no series of legal moves can lead to mate.
	// Common simplify: K vs K, K+B vs K, K+N vs K.

	return false
}
