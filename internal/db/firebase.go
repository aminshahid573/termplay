package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"tictactoe-ssh/internal/config"
	"tictactoe-ssh/internal/game"

	"time"

	"firebase.google.com/go/v4"
	db "firebase.google.com/go/v4/db"
	"google.golang.org/api/option"
)

// Room is the clean, strict structure used by the Game UI
type Room struct {
	Code        string                `json:"code"`
	Board       [9]string             `json:"board"`
	Turn        string                `json:"turn"`
	PlayerX     string                `json:"playerX"`
	PlayerO     string                `json:"playerO"`
	PlayerXName string                `json:"playerXName"`
	PlayerOName string                `json:"playerOName"`
	IsPublic    bool                  `json:"isPublic"`
	Winner      string                `json:"winner"`
	WinningLine []int                 `json:"winningLine"`
	Status      string                `json:"status"`
	WinsX       int                   `json:"winsX"`
	WinsO       int                   `json:"winsO"`
	Spectators  map[string]string     `json:"spectators"`
	UpdatedAt   int64                 `json:"updatedAt"`
	GameType    string                `json:"gameType"`
	ChessBoard  [8][8]game.ChessPiece `json:"chessBoard"`
}

// rawRoom is a helper struct to safely read dirty data (mixed types) from Firebase
type rawRoom struct {
	Code        string                `json:"code"`
	Board       []interface{}         `json:"board"` // Loose type to prevent crashes
	Turn        string                `json:"turn"`
	PlayerX     string                `json:"playerX"`
	PlayerO     string                `json:"playerO"`
	PlayerXName string                `json:"playerXName"`
	PlayerOName string                `json:"playerOName"`
	IsPublic    bool                  `json:"isPublic"`
	Winner      string                `json:"winner"`
	WinningLine []int                 `json:"winningLine"`
	Status      string                `json:"status"`
	WinsX       int                   `json:"winsX"`
	WinsO       int                   `json:"winsO"`
	Spectators  map[string]string     `json:"spectators"`
	UpdatedAt   int64                 `json:"updatedAt"`
	GameType    string                `json:"gameType"`
	ChessBoard  [8][8]game.ChessPiece `json:"chessBoard"`
}

var client *db.Client

func Init() error {
	if config.DBURL == "" {
		return fmt.Errorf("FIREBASE_DB_URL environment variable is required")
	}

	var opts []option.ClientOption
	if config.CredPath != "" {
		if _, err := os.Stat(config.CredPath); err == nil {
			opts = append(opts, option.WithCredentialsFile(config.CredPath))
		}
	}

	cfg := &firebase.Config{DatabaseURL: config.DBURL}
	app, err := firebase.NewApp(context.Background(), cfg, opts...)
	if err != nil {
		return fmt.Errorf("error initializing app: %v", err)
	}
	client, err = app.Database(context.Background())
	if err != nil {
		return fmt.Errorf("error initializing db client: %v", err)
	}
	return nil
}

// Helper to convert raw data to clean Room
func sanitizeRoom(code string, raw rawRoom) Room {
	clean := Room{
		Code:        code,
		Turn:        raw.Turn,
		PlayerX:     raw.PlayerX,
		PlayerO:     raw.PlayerO,
		PlayerXName: raw.PlayerXName,
		PlayerOName: raw.PlayerOName,
		IsPublic:    raw.IsPublic,
		Winner:      raw.Winner,
		WinningLine: raw.WinningLine,
		Status:      raw.Status,
		WinsX:       raw.WinsX,
		WinsO:       raw.WinsO,
		Spectators:  raw.Spectators,
		GameType:    raw.GameType,
		ChessBoard:  raw.ChessBoard,
	}

	if clean.GameType == "" {
		clean.GameType = "tictactoe"
	}

	if clean.Spectators == nil {
		clean.Spectators = make(map[string]string)
	}

	// Fix Code if missing in body
	if clean.Code == "" {
		clean.Code = code
	}

	// Safely convert Board
	clean.Board = [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "} // Default empty
	for i, val := range raw.Board {
		if i >= 9 {
			break
		}
		// Type assertion to handle strings vs numbers
		switch v := val.(type) {
		case string:
			clean.Board[i] = v
		case float64: // JSON numbers come as float64
			clean.Board[i] = fmt.Sprintf("%.0f", v) // Convert 0 -> "0"
		case int:
			clean.Board[i] = fmt.Sprintf("%d", v)
		default:
			clean.Board[i] = " "
		}
	}
	return clean
}

func CreateRoom(code, pid, name string, public bool, gameType string) error {
	ref := client.NewRef("rooms/" + code)

	// Check collision
	var raw rawRoom
	if err := ref.Get(context.Background(), &raw); err == nil {
		if raw.PlayerX != "" {
			return fmt.Errorf("room code taken")
		}
	}

	r := Room{
		Code:        code,
		PlayerX:     pid,
		PlayerXName: name,
		IsPublic:    public,
		Status:      "waiting",
		Spectators:  make(map[string]string),
		UpdatedAt:   time.Now().Unix(),
		GameType:    gameType,
	}

	if gameType == "chess" {
		r.ChessBoard = game.StartingChessBoard
		r.Turn = "White"
	} else {
		r.Board = [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "}
		r.Turn = "X"
	}

	log.Printf("Creating Room: %s (%s)", code, gameType)
	return ref.Set(context.Background(), r)
}

func GetRoom(code string) (*Room, error) {
	ref := client.NewRef("rooms/" + code)
	// Fetch as Raw first to avoid crashing on bad data
	var raw rawRoom
	if err := ref.Get(context.Background(), &raw); err != nil {
		return nil, err
	}
	if raw.PlayerX == "" {
		return nil, fmt.Errorf("room does not exist")
	}

	clean := sanitizeRoom(code, raw)
	return &clean, nil
}

func JoinRoom(code, pid, name string) error {
	ctx := context.Background()

	// Transaction needs strict type mapping, so if the room is corrupted,
	// this might still fail unless we handle it inside.
	// For simplicity, we assume GetRoom checks passed.
	fn := func(tn db.TransactionNode) (interface{}, error) {
		var raw rawRoom
		if err := tn.Unmarshal(&raw); err != nil {
			return nil, err
		}
		if raw.PlayerX == "" {
			return nil, fmt.Errorf("room not found")
		}

		// Check if Host is rejoining
		if raw.PlayerX == pid {
			raw.PlayerXName = name
			raw.UpdatedAt = time.Now().Unix()
			return raw, nil
		}

		if raw.PlayerO != "" && raw.PlayerO != pid {
			// Room full -> Join as Spectator
			if raw.Spectators == nil {
				raw.Spectators = make(map[string]string)
			}
			raw.Spectators[pid] = name
			return raw, nil
		}

		// Update fields
		raw.PlayerO = pid
		raw.PlayerOName = name
		raw.Status = "playing"
		return raw, nil
	}
	return client.NewRef("rooms/"+code).Transaction(ctx, fn)
}

func LeaveRoom(code, pid string, isHost bool) error {
	ctx := context.Background()
	ref := client.NewRef("rooms/" + code)

	if isHost {
		// Host leaves -> Delete room
		return ref.Delete(ctx)
	}

	// Not host. Check if PlayerO or Spectator
	// We need to fetch current state to know role, but here we passed isHost.
	// Actually we should fetch first to be safe, or just try to remove from both.
	// Firebase update paths:
	// If O: update playerO=""
	// If Spectator: delete spectators/pid

	// Let's use transaction to be safe and atomic
	fn := func(tn db.TransactionNode) (interface{}, error) {
		var raw rawRoom
		if err := tn.Unmarshal(&raw); err != nil {
			return nil, err
		}

		if raw.PlayerO == pid {
			raw.PlayerO = ""
			raw.PlayerOName = ""
			raw.Status = "waiting"
		} else {
			if raw.Spectators != nil {
				delete(raw.Spectators, pid)
			}
		}
		return raw, nil
	}
	return ref.Transaction(ctx, fn)
}

func UpdateMove(code, pid string, idx int, r Room) error {
	// Game Logic
	r.Board[idx] = r.Turn
	winner, line := game.CheckWinner(r.Board)

	if winner != "" {
		r.Winner = winner
		r.WinningLine = line
		r.Status = "finished"
		if winner == "X" {
			r.WinsX++
		} else {
			r.WinsO++
		}
	} else if game.CheckDraw(r.Board) {
		r.Status = "finished"
	} else {
		if r.Turn == "X" {
			r.Turn = "O"
		} else {
			r.Turn = "X"
		}
	}

	// When saving back, we save strict Room, effectively "fixing" the data
	return client.NewRef("rooms/"+code).Set(context.Background(), r)
}

func UpdateChessState(code string, board [8][8]game.ChessPiece, turn string) error {
	ref := client.NewRef("rooms/" + code)
	fn := func(tn db.TransactionNode) (interface{}, error) {
		var r Room
		if err := tn.Unmarshal(&r); err != nil {
			return nil, err
		}
		r.ChessBoard = board
		r.Turn = turn
		r.UpdatedAt = time.Now().Unix()
		return r, nil
	}
	return ref.Transaction(context.Background(), fn)
}

func RestartGame(code string, nextTurn string) error {
	ctx := context.Background()
	ref := client.NewRef("rooms/" + code)
	fn := func(tn db.TransactionNode) (interface{}, error) {
		var r Room
		if err := tn.Unmarshal(&r); err != nil {
			return nil, err
		}

		if r.GameType == "chess" {
			r.ChessBoard = game.StartingChessBoard
			// Map X/O to White/Black if needed, or rely on caller
			if nextTurn == "X" {
				nextTurn = "White"
			}
			if nextTurn == "O" {
				nextTurn = "Black"
			}
			r.Turn = nextTurn
		} else {
			r.Board = [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "}
			r.Turn = nextTurn
		}

		r.Winner = ""
		r.WinningLine = nil
		r.Status = "playing"
		return r, nil
	}
	return ref.Transaction(ctx, fn)
}

func GetPublicRooms() ([]Room, error) {
	ref := client.NewRef("rooms")

	// 1. Fetch as map of RawRooms (tolerant to bad data)
	var rawMap map[string]rawRoom
	if err := ref.Get(context.Background(), &rawMap); err != nil {
		log.Printf("Error fetching public rooms: %v", err)
		return nil, err
	}

	var list []Room
	for code, raw := range rawMap {
		// 2. Filter Public
		if raw.IsPublic {
			// 3. Sanitize (Fix types)
			clean := sanitizeRoom(code, raw)
			list = append(list, clean)
		}
	}

	// 4. Sort
	sort.Slice(list, func(i, j int) bool {
		return list[i].Code < list[j].Code
	})

	return list, nil
}

// CleanZombies removes rooms that haven't been updated in 1 hour
func CleanZombies() {
	ref := client.NewRef("rooms")
	var rawMap map[string]rawRoom
	if err := ref.Get(context.Background(), &rawMap); err != nil {
		log.Printf("Janitor: Error fetching rooms: %v", err)
		return
	}

	now := time.Now().Unix()
	limit := int64(3600) // 1 hour

	for code, r := range rawMap {
		if now-r.UpdatedAt > limit {
			log.Printf("Janitor: Deleting zombie room %s (Last active: %ds ago)", code, now-r.UpdatedAt)
			ref.Child(code).Delete(context.Background())
		}
	}
}
