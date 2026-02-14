package db

import (
	"context"
	"fmt"
	"log"
	"sort"
	"tictactoe-ssh/internal/config"
	"tictactoe-ssh/internal/game"

	"firebase.google.com/go/v4"
	db "firebase.google.com/go/v4/db"
	"google.golang.org/api/option"
)

// Room is the clean, strict structure used by the Game UI
type Room struct {
	Code        string    `json:"code"`
	Board       [9]string `json:"board"`
	Turn        string    `json:"turn"`
	PlayerX     string    `json:"playerX"`
	PlayerO     string    `json:"playerO"`
	PlayerXName string    `json:"playerXName"`
	PlayerOName string    `json:"playerOName"`
	IsPublic    bool      `json:"isPublic"`
	Winner      string    `json:"winner"`
	WinningLine []int     `json:"winningLine"`
	Status      string    `json:"status"`
	WinsX       int       `json:"winsX"`
	WinsO       int       `json:"winsO"`
}

// rawRoom is a helper struct to safely read dirty data (mixed types) from Firebase
type rawRoom struct {
	Code        string        `json:"code"`
	Board       []interface{} `json:"board"` // Loose type to prevent crashes
	Turn        string        `json:"turn"`
	PlayerX     string        `json:"playerX"`
	PlayerO     string        `json:"playerO"`
	PlayerXName string        `json:"playerXName"`
	PlayerOName string        `json:"playerOName"`
	IsPublic    bool          `json:"isPublic"`
	Winner      string        `json:"winner"`
	WinningLine []int         `json:"winningLine"`
	Status      string        `json:"status"`
	WinsX       int           `json:"winsX"`
	WinsO       int           `json:"winsO"`
}

var client *db.Client

func Init() error {
	opt := option.WithCredentialsFile(config.CredPath)
	cfg := &firebase.Config{DatabaseURL: config.DBURL}
	app, err := firebase.NewApp(context.Background(), cfg, opt)
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

func CreateRoom(code, pid, name string, public bool) error {
	ref := client.NewRef("rooms/" + code)
	r := Room{
		Code:        code,
		Board:       [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "},
		Turn:        "X",
		PlayerX:     pid,
		PlayerXName: name,
		IsPublic:    public,
		Status:      "waiting",
	}
	log.Printf("Creating Room: %s", code)
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
		if raw.PlayerO != "" && raw.PlayerO != pid {
			return nil, fmt.Errorf("room is full")
		}

		// Update fields
		raw.PlayerO = pid
		raw.PlayerOName = name
		raw.Status = "playing"
		return raw, nil
	}
	return client.NewRef("rooms/" + code).Transaction(ctx, fn)
}

func LeaveRoom(code, pid string, isHost bool) error {
	ctx := context.Background()
	ref := client.NewRef("rooms/" + code)

	if isHost {
		return ref.Delete(ctx)
	} else {
		updates := map[string]interface{}{
			"playerO": "",
			"status":  "waiting",
		}
		return ref.Update(ctx, updates)
	}
}

func UpdateMove(code, pid string, idx int, r Room) error {
	// Game Logic
	r.Board[idx] = r.Turn
	winner, line := game.CheckWinner(r.Board)
	
	if winner != "" {
		r.Winner = winner
		r.WinningLine = line
		r.Status = "finished"
		if winner == "X" { r.WinsX++ } else { r.WinsO++ }
	} else if game.CheckDraw(r.Board) {
		r.Status = "finished"
	} else {
		if r.Turn == "X" { r.Turn = "O" } else { r.Turn = "X" }
	}

	// When saving back, we save strict Room, effectively "fixing" the data
	return client.NewRef("rooms/" + code).Set(context.Background(), r)
}

func RestartGame(code string) error {
	ctx := context.Background()
	ref := client.NewRef("rooms/" + code)
	fn := func(tn db.TransactionNode) (interface{}, error) {
		var r Room
		if err := tn.Unmarshal(&r); err != nil { return nil, err }
		r.Board = [9]string{" ", " ", " ", " ", " ", " ", " ", " ", " "}
		r.Winner = ""
		r.WinningLine = nil
		r.Status = "playing"
		r.Turn = "X"
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
