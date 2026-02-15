package game

type ChessPiece struct {
	Type    string `json:"type"` // "K", "Q", "R", "B", "N", "P" or ""
	IsWhite bool   `json:"isWhite"`
}

func (p ChessPiece) IsEmpty() bool {
	return p.Type == ""
}

type Pos struct {
	Row, Col int
}

// Starting position: rows 0-7 map to ranks 8-1
var StartingChessBoard = [8][8]ChessPiece{
	{{"R", false}, {"N", false}, {"B", false}, {"Q", false}, {"K", false}, {"B", false}, {"N", false}, {"R", false}},
	{{"P", false}, {"P", false}, {"P", false}, {"P", false}, {"P", false}, {"P", false}, {"P", false}, {"P", false}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{}, {}, {}, {}, {}, {}, {}, {}},
	{{"P", true}, {"P", true}, {"P", true}, {"P", true}, {"P", true}, {"P", true}, {"P", true}, {"P", true}},
	{{"R", true}, {"N", true}, {"B", true}, {"Q", true}, {"K", true}, {"B", true}, {"N", true}, {"R", true}},
}

func inBounds(r, c int) bool {
	return r >= 0 && r < 8 && c >= 0 && c < 8
}

func GetValidChessMoves(board [8][8]ChessPiece, r, c int) map[Pos]bool {
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
			return true // keep sliding
		}
		// occupied square
		if target.IsWhite != isWhite {
			moves[Pos{nr, nc}] = true // can capture enemy
		}
		return false // stop sliding (blocked by any piece)
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
		// Pawns move forward relative to their color
		dir := -1 // white moves up (decreasing row)
		startRow := 6
		if !isWhite {
			dir = 1 // black moves down
			startRow = 1
		}
		// Forward 1
		nr := r + dir
		if inBounds(nr, c) && board[nr][c].IsEmpty() {
			moves[Pos{nr, c}] = true
			// Forward 2 from starting position
			nr2 := r + dir*2
			if r == startRow && inBounds(nr2, c) && board[nr2][c].IsEmpty() {
				moves[Pos{nr2, c}] = true
			}
		}
		// Diagonal captures
		for _, dc := range []int{-1, 1} {
			nc := c + dc
			if inBounds(nr, nc) && !board[nr][nc].IsEmpty() && board[nr][nc].IsWhite != isWhite {
				moves[Pos{nr, nc}] = true
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
		// Queen = Rook + Bishop
		slide(0, 1)
		slide(0, -1)
		slide(1, 0)
		slide(-1, 0)
		slide(1, 1)
		slide(1, -1)
		slide(-1, 1)
		slide(-1, -1)

	case "K":
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				if dr == 0 && dc == 0 {
					continue
				}
				tryAdd(r+dr, c+dc)
			}
		}

	case "N":
		knightMoves := [][2]int{
			{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
			{1, -2}, {1, 2}, {2, -1}, {2, 1},
		}
		for _, km := range knightMoves {
			tryAdd(r+km[0], c+km[1])
		}
	}

	return moves
}
