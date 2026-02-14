package ui

import (
	"fmt"
	"strings"
	"tictactoe-ssh/internal/styles"
	"tictactoe-ssh/internal/db"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

func (m Model) View() string {
	// Global Popup
	if m.PopupActive {
		msg := "Are you sure you want to leave?\n(Room will be deleted if you are Host)"
		box := styles.PopupBox.Render(
			fmt.Sprintf("%s\n\n[Y] Yes    [N] No", msg),
		)
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
			// The > arrow + the input (which has placeholder)
			styles.SearchBar.Render("> ") + m.TextInput.View(), 
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
		pub := "[ ] Public"
		priv := "[x] Private"
		if m.IsPublicCreate {
			pub = "[x] Public"
			priv = "[ ] Private"
		}
		content = lipgloss.JoinVertical(lipgloss.Center,
			styles.Title.Render("ROOM SETTINGS"),
			"Select Visibility:",
			"\n",
			fmt.Sprintf("%s\n%s", pub, priv),
			"\n",
		)
		helpText = "↑/↓: Change • Enter: Create • Esc: Back"

	case StateInputCode:
		errView := ""
		if m.Err != nil { errView = styles.Base.Foreground(lipgloss.Color("#F25D94")).Render("\n" + m.Err.Error()) }
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

	case StateGame:
		content = renderGame(m)
		// Game help is rendered inside renderGame to be closer to board, 
		// but we can add global help too if needed.
		helpText = "Arrows: Move • Space: Place • R: Restart • Q: Quit"
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

	// Calculate container width from style
	listWidth := 58 // slightly less than container width (60)
	
	var listContent []string

	// 1. Search Bar (Borderless inside the box)
	// We add the ">" prefix manually
	searchView := styles.SearchBar.Render("> ") + m.SearchInput.View()
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
	listContent = append(listContent, renderSectionHeader(" Full Rooms ", listWidth, "Spectate (Soon)"))
	if len(fullRooms) == 0 {
		listContent = append(listContent, styles.Subtle.Render("  No full rooms"))
	} else {
		for i, r := range fullRooms {
			isSelected := (i + len(openRooms) == m.ListSelectedRow)
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
	if info != "" { infoRendered = " " + styles.Subtle.Render(info) }

	titleRendered := styles.SectionTitle.Render(text)
	
	remaining := width - lipgloss.Width(titleRendered) - lipgloss.Width(infoRendered)
	if remaining < 0 { remaining = 0 }
	
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
	gap := strings.Repeat(" ", max(0, width - nameWidth - rightWidth))

	return style.Render(name + gap + rightRendered)
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func renderGame(m Model) string {
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
			
			isWinCell := false
			for _, wIdx := range m.Game.WinningLine {
				if idx == wIdx { isWinCell = true }
			}
			if isWinCell { style = styles.CellWin }
			
			if m.Game.Status == "playing" && m.Game.Turn == m.MySide {
				if r == m.CursorR && c == m.CursorC {
					style = styles.CellSelected
				}
			}
			
			content := " "
			if val == "X" { content = styles.XStyle.Render("X") }
			if val == "O" { content = styles.OStyle.Render("O") }
			cols = append(cols, style.Render(content))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cols...))
	}
	board := lipgloss.JoinVertical(lipgloss.Left, rows...)

	status := ""
	if m.Game.Status == "waiting" {
		status = "Opponent disconnected. Waiting..."
	} else if m.Game.Status == "finished" {
		res := "DRAW"
		if m.Game.Winner != "" { res = m.Game.Winner + " WINS!" }
		status = fmt.Sprintf("%s", res)
	} else {
		turn := m.Game.Turn
		status = fmt.Sprintf("Turn: %s", turn)
	}
	
	return lipgloss.JoinVertical(lipgloss.Center, 
		styles.Title.Render("TICTACTOE"), 
		header, 
		"\n",
		styles.ListContainer.BorderForeground(styles.Muted.GetForeground()).Padding(0).Render(board), 
		"\n",
		status,
	)
}
