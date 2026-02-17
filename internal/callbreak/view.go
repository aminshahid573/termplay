package callbreak

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── Palette ─────────────────────────────────────────────────────────────────

var (
	clrBorder = lipgloss.Color("#30363d")
	clrSubtle = lipgloss.Color("#8b949e")
	clrGold   = lipgloss.Color("#e3b341")
	clrGreen  = lipgloss.Color("#3fb950")
	clrRed    = lipgloss.Color("#f85149")
	clrYellow = lipgloss.Color("#f0c862")
	clrWhite  = lipgloss.Color("#e6edf3")
	clrTitle  = lipgloss.Color("#58a6ff")

	suitColorMap = [4]lipgloss.Color{
		lipgloss.Color("#44AAFF"), // Clubs
		lipgloss.Color("#FFD700"), // Diamonds
		lipgloss.Color("#FF6B6B"), // Hearts
		lipgloss.Color("#50FA7B"), // Spades
	}
)

// ─── Style helpers ────────────────────────────────────────────────────────────

func fg(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c)
}
func bold(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

// joinRows safely joins two multi-line strings side by side.
// It pads each side to equal height to prevent black-fill artifacts
// that lipgloss.JoinHorizontal produces on height mismatches.
func joinRows(left, right string, gap int) string {
	ll := strings.Split(left, "\n")
	rl := strings.Split(right, "\n")
	h := len(ll)
	if len(rl) > h {
		h = len(rl)
	}
	lw := maxLineWidth(ll)
	rw := maxLineWidth(rl)
	for len(ll) < h {
		ll = append(ll, strings.Repeat(" ", lw))
	}
	for len(rl) < h {
		rl = append(rl, strings.Repeat(" ", rw))
	}
	sep := strings.Repeat(" ", gap)
	rows := make([]string, h)
	for i := 0; i < h; i++ {
		lpad := lw - lipgloss.Width(ll[i])
		if lpad < 0 {
			lpad = 0
		}
		rpad := rw - lipgloss.Width(rl[i])
		if rpad < 0 {
			rpad = 0
		}
		rows[i] = ll[i] + strings.Repeat(" ", lpad) + sep + rl[i] + strings.Repeat(" ", rpad)
	}
	return strings.Join(rows, "\n")
}

// joinRowsVCenter joins two multi-line strings side by side, vertically centering the shorter one.
func joinRowsVCenter(left, right string, gap int) string {
	ll := strings.Split(left, "\n")
	rl := strings.Split(right, "\n")
	lw := maxLineWidth(ll)
	rw := maxLineWidth(rl)
	h := len(ll)
	if len(rl) > h {
		h = len(rl)
	}

	// Pad shorter side with blank lines centered vertically
	padLines := func(lines []string, w, target int) []string {
		if len(lines) >= target {
			return lines
		}
		diff := target - len(lines)
		top := diff / 2
		bot := diff - top
		blank := strings.Repeat(" ", w)
		result := make([]string, 0, target)
		for i := 0; i < top; i++ {
			result = append(result, blank)
		}
		result = append(result, lines...)
		for i := 0; i < bot; i++ {
			result = append(result, blank)
		}
		return result
	}
	ll = padLines(ll, lw, h)
	rl = padLines(rl, rw, h)

	sep := strings.Repeat(" ", gap)
	rows := make([]string, h)
	for i := 0; i < h; i++ {
		lpad := lw - lipgloss.Width(ll[i])
		if lpad < 0 {
			lpad = 0
		}
		rpad := rw - lipgloss.Width(rl[i])
		if rpad < 0 {
			rpad = 0
		}
		rows[i] = ll[i] + strings.Repeat(" ", lpad) + sep + rl[i] + strings.Repeat(" ", rpad)
	}
	return strings.Join(rows, "\n")
}

func maxLineWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > w {
			w = lw
		}
	}
	return w
}

func padCenter(s string, w int) string {
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(s)
}

// box wraps content in a rounded border with NO background (prevents black fill).
func box(content string, w int, borderClr lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderClr).
		Width(w).
		Padding(0, 1).
		Render(content)
}

// ─── Card drawing ─────────────────────────────────────────────────────────────

func buildCardLines(c Card, faceDown bool) []string {
	if faceDown {
		return []string{"┌───┐", "│▓▓▓│", "│▓▓▓│", "│▓▓▓│", "└───┘"}
	}
	if c.IsZero() {
		return []string{"┌───┐", "│   │", "│   │", "│   │", "└───┘"}
	}
	r := fmt.Sprintf("%-2s", c.Rank.String())
	s := c.Suit.Symbol()
	b := fmt.Sprintf("%2s", c.Rank.String())
	return []string{
		"┌───┐",
		"│" + r + " │",
		"│ " + s + " │",
		"│ " + b + "│",
		"└───┘",
	}
}

func renderCard(c Card, selected, valid bool) string {
	lines := buildCardLines(c, false)
	var clr lipgloss.Color
	if c.IsZero() {
		clr = clrBorder
	} else {
		clr = suitColorMap[c.Suit]
	}
	borderClr := clr
	textClr := clr
	if selected {
		borderClr = clrGold
	}
	if !valid {
		borderClr = clrBorder
		textClr = clrBorder
	}
	result := make([]string, 5)
	for i, l := range lines {
		if i == 0 || i == 4 {
			result[i] = fg(borderClr).Render(l)
		} else {
			result[i] = fg(textClr).Render(l)
		}
	}
	if selected {
		result[4] = fg(clrGold).Render("└─▲─┘")
	}
	return strings.Join(result, "\n")
}

func renderFaceDownCard() string {
	lines := buildCardLines(Card{}, true)
	result := make([]string, 5)
	for i, l := range lines {
		result[i] = fg(clrBorder).Render(l)
	}
	return strings.Join(result, "\n")
}

func renderEmptySlot() string {
	lines := buildCardLines(Card{}, false)
	result := make([]string, 5)
	for i, l := range lines {
		result[i] = fg(clrBorder).Faint(true).Render(l)
	}
	return strings.Join(result, "\n")
}

func renderTableCard(c Card) string {
	if c.IsZero() {
		return renderEmptySlot()
	}
	clr := suitColorMap[c.Suit]
	r := fmt.Sprintf("%-2s", c.Rank.String())
	s := c.Suit.Symbol()
	b := fmt.Sprintf("%2s", c.Rank.String())
	lines := []string{
		bold(clr).Render("╔═══╗"),
		fg(clr).Render("║" + r + " ║"),
		bold(clr).Render("║ " + s + " ║"),
		fg(clr).Render("║ " + b + "║"),
		bold(clr).Render("╚═══╝"),
	}
	return strings.Join(lines, "\n")
}

// ─── Hand rendering ───────────────────────────────────────────────────────────

func renderHandRow(hand []Card, selected int, valid []bool, active bool) string {
	if len(hand) == 0 {
		return fg(clrSubtle).Render("(no cards)")
	}
	const cardH = 5
	rows := make([][]string, cardH)
	for i, c := range hand {
		v := i >= len(valid) || valid[i]
		sel := active && i == selected
		card := renderCard(c, sel, v)
		lines := strings.Split(card, "\n")
		for r := 0; r < cardH; r++ {
			l := ""
			if r < len(lines) {
				l = lines[r]
			}
			rows[r] = append(rows[r], l)
		}
	}
	result := make([]string, cardH)
	for i, row := range rows {
		result[i] = strings.Join(row, " ")
	}
	return strings.Join(result, "\n")
}

func renderFaceDownRow(n, maxCards int) string {
	if n == 0 {
		return fg(clrSubtle).Render("(played out)")
	}
	show := n
	if show > maxCards {
		show = maxCards
	}
	const cardH = 5
	card := renderFaceDownCard()
	lines := strings.Split(card, "\n")
	rows := make([]string, cardH)
	for i := 0; i < show; i++ {
		for r := 0; r < cardH && r < len(lines); r++ {
			if i == 0 {
				rows[r] = lines[r]
			} else {
				rows[r] += " " + lines[r]
			}
		}
	}
	result := strings.Join(rows, "\n")
	if n > show {
		result += "\n" + fg(clrSubtle).Render(fmt.Sprintf("+%d more", n-show))
	}
	return result
}

// ─── Player label ─────────────────────────────────────────────────────────────

func playerLabel(name string, bid, won int, active bool) string {
	info := ""
	if bid > 0 {
		info = fmt.Sprintf("  bid:%d  won:%d", bid, won)
	}
	label := name + info
	if active {
		return bold(clrGold).Render("▶ " + label)
	}
	return fg(clrSubtle).Render("  " + label)
}

// ─── Header / message bar ─────────────────────────────────────────────────────

func (m Model) renderHeader(title string) string {
	left := bold(clrGold).Render("♠ CALL BREAK")
	right := fg(clrSubtle).Render("Ctrl+C Quit")
	innerW := m.Width - lipgloss.Width(left) - lipgloss.Width(right) - 6
	if innerW < 0 {
		innerW = 0
	}
	mid := bold(clrWhite).Width(innerW).Align(lipgloss.Center).Render(title)
	return lipgloss.NewStyle().
		BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(clrBorder).
		Width(m.Width-2).Padding(0, 1).
		Render(left + "  " + mid + "  " + right)
}

func (m Model) renderMsgBar() string {
	return lipgloss.NewStyle().
		BorderTop(true).BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(clrBorder).
		Foreground(clrYellow).
		Width(m.Width-2).Padding(0, 2).
		Render(m.Message)
}

// ─── Side panel ──────────────────────────────────────────────────────────────

func (m Model) renderSidePanel(w int) string {
	var sb strings.Builder

	sb.WriteString(bold(clrGold).Render("SCORES") + "\n")
	sb.WriteString(fg(clrBorder).Render(strings.Repeat("─", w-4)) + "\n")
	for p := 0; p < 4; p++ {
		cursor := "  "
		if p == m.CurrentPlayer && m.Phase == PhasePlaying {
			cursor = fg(clrGreen).Render("▶ ")
		}
		sc := m.Scores[p]
		scStr := fmt.Sprintf("%+.1f", sc)
		scClr := clrSubtle
		if sc > 0 {
			scClr = clrGreen
		} else if sc < 0 {
			scClr = clrRed
		}
		sb.WriteString(cursor + fg(clrWhite).Render(fmt.Sprintf("%-8s", m.PlayerNames[p])) +
			fg(scClr).Render(scStr) + "\n")
	}

	sb.WriteString("\n" + bold(clrGold).Render("BID / WON") + "\n")
	sb.WriteString(fg(clrBorder).Render(strings.Repeat("─", w-4)) + "\n")
	for p := 0; p < 4; p++ {
		bid, won := m.Bids[p], m.TricksWon[p]
		s := fmt.Sprintf("%d/%d", won, bid)
		c := clrSubtle
		if bid > 0 {
			if won >= bid {
				c = clrGreen
			} else {
				c = clrYellow
			}
		}
		sb.WriteString("  " + fg(clrWhite).Render(fmt.Sprintf("%-8s", m.PlayerNames[p])) +
			fg(c).Render(s) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fg(clrSubtle).Render(fmt.Sprintf("  Round  %d/%d", m.Round, TotalRounds)) + "\n")
	sb.WriteString(fg(clrSubtle).Render(fmt.Sprintf("  Trick  %d/13", m.TrickNum)) + "\n")
	sb.WriteString("\n" + fg(clrSubtle).Render("  Trump: ") + bold(clrGreen).Render("♠ Spades"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(clrBorder).
		Width(w).Padding(0, 1).
		Render(sb.String())
}

// ─── Table (trick area) ───────────────────────────────────────────────────────

func (m Model) renderTableArea(w int) string {
	north := renderTableCard(m.TableCards[2])
	west := renderTableCard(m.TableCards[1])
	east := renderTableCard(m.TableCards[3])
	south := renderTableCard(m.TableCards[0])

	const cardW = 5
	const cardH = 5
	split := func(s string) []string { return strings.Split(s, "\n") }
	nl, wl, el, sl := split(north), split(west), split(east), split(south)

	centerLabel := []string{"     ", "  ┌─┐", "  │♠│", "  └─┘", "     "}

	padW := (w - cardW) / 2
	if padW < 0 {
		padW = 0
	}

	// North row — centered
	topRows := make([]string, len(nl))
	for i, l := range nl {
		topRows[i] = strings.Repeat(" ", padW) + l
	}

	// Mid row: west  centerLabel  east
	sideGap := (w - cardW - 7 - cardW - 4) / 2
	if sideGap < 0 {
		sideGap = 0
	}
	sp := strings.Repeat(" ", sideGap)
	midRows := make([]string, cardH)
	for i := 0; i < cardH; i++ {
		wLine, eLine, cLine := "", "", "       "
		if i < len(wl) {
			wLine = wl[i]
		}
		if i < len(el) {
			eLine = el[i]
		}
		if i < len(centerLabel) {
			cLine = centerLabel[i]
		}
		midRows[i] = sp + wLine + "  " + fg(clrSubtle).Render(cLine) + "  " + eLine
	}

	// South row — centered
	botRows := make([]string, len(sl))
	for i, l := range sl {
		botRows[i] = strings.Repeat(" ", padW) + l
	}

	all := append(topRows, append([]string{""}, append(midRows, append([]string{""}, botRows...)...)...)...)
	return strings.Join(all, "\n")
}

// ─── Main View dispatcher ─────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.Phase {
	case PhaseMenu:
		return m.viewMenu()
	case PhasePlayerSelect:
		return m.viewPlayerSelect()
	case PhaseBidding:
		return m.viewBidding()
	case PhasePlaying, PhaseTrickDone:
		return m.viewGame()
	case PhaseRoundResult:
		return m.viewRoundResult()
	case PhaseGameOver:
		return m.viewGameOver()
	}
	return ""
}

// ─── Game view ────────────────────────────────────────────────────────────────

func (m Model) viewGame() string {
	// Border takes 2 cols (left+right) + padding 2 (left+right) = 4 total chrome
	const borderChrome = 4
	const sideW = 24

	if m.Width < 40 {
		return "Terminal too narrow (min 40 cols)"
	}

	// Determine layout mode
	wide := m.Width >= 80

	var centerW int
	if wide {
		// side panel sits to the right: border(2) + gap(2) + sidePanel
		centerW = m.Width - sideW - 5
	} else {
		// stacked: full width minus border chrome
		centerW = m.Width - borderChrome
	}

	// Clamp centerW to avoid negative
	if centerW < 30 {
		centerW = 30
	}

	header := m.renderHeader(fmt.Sprintf(
		"Round %d/%d  ·  Trick %d/13  ·  Trump: %s",
		m.Round, TotalRounds, m.TrickNum, bold(clrGreen).Render("♠ Spades"),
	))

	// --- North: text label + card count (no card boxes) ---
	northLabel := playerLabel("NORTH · "+m.PlayerNames[2], m.Bids[2], m.TricksWon[2], m.CurrentPlayer == 2)
	northCount := fg(clrSubtle).Render(fmt.Sprintf("[%d cards]", len(m.Hands[2])))
	northSection := padCenter(northLabel+"\n"+northCount, centerW)

	// --- East / West: label + card count, vertically centered ---
	westLabel := playerLabel(m.PlayerNames[1], m.Bids[1], m.TricksWon[1], m.CurrentPlayer == 1)
	eastLabel := playerLabel(m.PlayerNames[3], m.Bids[3], m.TricksWon[3], m.CurrentPlayer == 3)
	westCount := fg(clrSubtle).Render(fmt.Sprintf("[%d cards]", len(m.Hands[1])))
	eastCount := fg(clrSubtle).Render(fmt.Sprintf("[%d cards]", len(m.Hands[3])))

	sideColW := 14

	tableW := centerW - sideColW*2 - 4
	if tableW < 20 {
		tableW = 20
	}

	westCol := lipgloss.NewStyle().Width(sideColW).Align(lipgloss.Center).Render(westLabel + "\n" + westCount)
	eastCol := lipgloss.NewStyle().Width(sideColW).Align(lipgloss.Center).Render(eastLabel + "\n" + eastCount)
	tableBlock := m.renderTableArea(tableW)

	// Join west | table | east with vertical centering, touching side edges
	midSection := joinRowsVCenter(
		joinRowsVCenter(westCol, tableBlock, 1),
		eastCol,
		1,
	)

	// --- South: your hand ---
	southLabel := playerLabel("YOU · "+m.PlayerNames[0], m.Bids[0], m.TricksWon[0], m.CurrentPlayer == 0)
	southHand := renderHandRow(m.Hands[0], m.SelectedCard, m.ValidCards, m.CurrentPlayer == 0)

	hint := ""
	if m.CurrentPlayer == 0 && m.Phase == PhasePlaying {
		hint = fg(clrSubtle).Render("  ← → select card     ENTER play")
	}

	centerContent := strings.Join([]string{
		northSection, "", midSection, "", southLabel, southHand, hint,
	}, "\n")

	centerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(clrBorder).
		Width(centerW).Padding(0, 1).
		Render(centerContent)

	var body string
	if wide {
		sidePanel := m.renderSidePanel(sideW)
		body = joinRows(centerBox, sidePanel, 2)
	} else {
		sidePanel := m.renderSidePanel(centerW)
		body = lipgloss.JoinVertical(lipgloss.Center, centerBox, sidePanel)
	}

	return strings.Join([]string{header, body, m.renderMsgBar()}, "\n")
}

// ─── Bidding view ─────────────────────────────────────────────────────────────

func (m Model) viewBidding() string {
	header := m.renderHeader(fmt.Sprintf("Round %d/%d  ·  BIDDING PHASE", m.Round, TotalRounds))

	sideW := 24
	// Account for border chrome (2) + gap (2) + side panel
	centerW := m.Width - sideW - 5
	if centerW < 30 {
		centerW = 30
	}

	var suitHint string
	for s := Spades; s >= Clubs; s-- {
		n := CountSuit(m.Hands[0], s)
		suitHint += fg(suitColorMap[s]).Render(fmt.Sprintf("%s:%d  ", SuitSymbols[s], n))
	}

	hand := renderHandRow(m.Hands[0], m.SelectedCard, make([]bool, len(m.Hands[0])), false)

	bidWidget := strings.Join([]string{
		padCenter(fg(clrSubtle).Render("[ ↑ ]"), 20),
		padCenter(bold(clrGold).Render(fmt.Sprintf("   %d   ", m.HumanBid)), 20),
		padCenter(fg(clrSubtle).Render("[ ↓ ]"), 20),
		"",
		padCenter(fg(clrGreen).Render("[ ENTER ] Confirm"), 20),
	}, "\n")
	bidBox := box(bold(clrGold).Render("YOUR BID\n\n")+bidWidget, 22, clrGold)

	handBox := box(
		bold(clrSubtle).Render("Your Hand\n")+suitHint+"\n\n"+hand,
		centerW, clrBorder,
	)

	main := joinRows(handBox, bidBox, 2)
	// If screen is narrow, stack vertically
	if centerW < 60 {
		main = lipgloss.JoinVertical(lipgloss.Center, handBox, bidBox)
	}

	var body string
	if m.Width >= 80 {
		body = joinRows(main, m.renderSidePanel(sideW), 2)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Center, main, m.renderSidePanel(m.Width-4))
	}

	return strings.Join([]string{header, body, m.renderMsgBar()}, "\n")
}

// ─── Menu view ────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	w := m.Width
	if w < 40 {
		w = 40
	}

	banner := strings.Join([]string{
		bold(clrGold).Render("╔════════════════════════════════╗"),
		bold(clrGold).Render("║  ♠  ") + bold(clrWhite).Render("  C A L L   B R E A K  ") + bold(clrGold).Render("  ♠  ║"),
		bold(clrGold).Render("╚════════════════════════════════╝"),
	}, "\n")

	suits := fg(suitColorMap[3]).Render("♠") + "  " +
		fg(suitColorMap[2]).Render("♥") + "  " +
		fg(suitColorMap[1]).Render("♦") + "  " +
		fg(suitColorMap[0]).Render("♣")

	rules := []string{
		"♠ Spades are always trump",
		"Follow lead suit if you can",
		"Bid tricks you expect to win",
		"Win ≥ bid  →  bid + 0.1×extras",
		"Win < bid  →  −bid points",
		"Play 5 rounds · highest score wins",
	}
	rLines := make([]string, len(rules))
	for i, r := range rules {
		rLines[i] = "  " + fg(clrSubtle).Render(r)
	}
	rulesBox := box(
		bold(clrGold).Render("RULES\n")+
			fg(clrBorder).Render(strings.Repeat("─", 30))+"\n"+
			strings.Join(rLines, "\n"),
		34, clrBorder)

	keys := strings.Join([]string{
		bold(clrWhite).Render("← →  ") + fg(clrSubtle).Render(" select card"),
		bold(clrWhite).Render("↑ ↓  ") + fg(clrSubtle).Render(" adjust bid"),
		bold(clrWhite).Render("ENTER") + fg(clrSubtle).Render(" confirm / play"),
		bold(clrWhite).Render("Q    ") + fg(clrSubtle).Render(" quit"),
	}, "\n")
	keysBox := box(
		bold(clrGold).Render("CONTROLS\n")+
			fg(clrBorder).Render(strings.Repeat("─", 20))+"\n"+keys,
		26, clrBorder)

	mid := joinRows(rulesBox, keysBox, 2)

	// Mode selection
	modeItems := []string{"Play vs AI", "Create Room", "Join Room"}
	var modeRendered []string
	for i, item := range modeItems {
		if i == m.MenuSelection {
			modeRendered = append(modeRendered, bold(clrGold).Render("▶ "+item))
		} else {
			modeRendered = append(modeRendered, fg(clrSubtle).Render("  "+item))
		}
	}
	modeSection := bold(clrTitle).Render("GAME MODE") + "\n" + strings.Join(modeRendered, "\n")

	start := "\n" + bold(clrGreen).Render("[ ENTER ]  Start Game") +
		"    " + fg(clrSubtle).Render("[ Q ]  Quit")

	content := strings.Join([]string{
		"", banner, "", padCenter(suits, 36), "", mid, "", modeSection, start, "",
	}, "\n")

	return padCenter(content, w)
}

// ─── Player Select view ───────────────────────────────────────────────────────

func (m Model) viewPlayerSelect() string {
	w := m.Width
	if w < 40 {
		w = 40
	}

	header := m.renderHeader("MULTIPLAYER SETUP")

	title := bold(clrGold).Render("Number of Human Players")
	desc := fg(clrSubtle).Render("Remaining seats will be filled by AI")

	// Show player count selector (2-4)
	countStr := bold(clrGold).Render(fmt.Sprintf("   %d   ", m.HumanPlayers))
	selector := strings.Join([]string{
		padCenter(fg(clrSubtle).Render("[ ↑ ]"), 20),
		padCenter(countStr, 20),
		padCenter(fg(clrSubtle).Render("[ ↓ ]"), 20),
	}, "\n")

	// Show seat assignment preview
	var seats []string
	seatNames := [4]string{"South (You)", "West", "North", "East"}
	for i := 0; i < 4; i++ {
		isAI := i >= m.HumanPlayers
		label := seatNames[i]
		if isAI {
			label += " " + fg(clrSubtle).Render("[AI]")
			seats = append(seats, fg(clrSubtle).Render("  "+label))
		} else {
			label += " " + fg(clrGreen).Render("[Human]")
			seats = append(seats, fg(clrWhite).Render("  "+label))
		}
	}
	seatPreview := strings.Join(seats, "\n")

	confirm := "\n" + bold(clrGreen).Render("[ ENTER ]  Create Room") +
		"    " + fg(clrSubtle).Render("[ ESC ]  Back")

	content := strings.Join([]string{
		"", title, desc, "", selector, "", seatPreview, confirm, "",
	}, "\n")

	body := padCenter(content, w-4)

	return strings.Join([]string{header, body, m.renderMsgBar()}, "\n")
}

// ─── Round Result view ────────────────────────────────────────────────────────

func (m Model) viewRoundResult() string {
	w := m.Width
	header := m.renderHeader(fmt.Sprintf("Round %d Complete!", m.Round))

	var sb strings.Builder
	sb.WriteString(bold(clrGold).Render(
		fmt.Sprintf("%-10s %5s %5s %8s %8s\n", "Player", "Bid", "Won", "Delta", "Total")))
	sb.WriteString(fg(clrBorder).Render(strings.Repeat("─", 42)) + "\n")
	for p := 0; p < 4; p++ {
		d := m.RoundScores[p]
		dStr := fmt.Sprintf("%+.1f", d)
		dClr := clrGreen
		if d < 0 {
			dClr = clrRed
		}
		total := m.Scores[p] + d
		tClr := clrWhite
		if total < 0 {
			tClr = clrRed
		} else if total > 0 {
			tClr = clrGreen
		}
		marker := "  "
		if p == 0 {
			marker = fg(clrTitle).Render("▶ ")
		}
		sb.WriteString(fmt.Sprintf("%s%-10s %5d %5d %s %s\n",
			marker, m.PlayerNames[p], m.Bids[p], m.TricksWon[p],
			fg(dClr).Render(fmt.Sprintf("%8s", dStr)),
			fg(tClr).Render(fmt.Sprintf("%8.1f", total))))
	}
	sb.WriteString("\n" + fg(clrSubtle).Render("Next round starting shortly…"))

	return strings.Join([]string{
		header, "",
		padCenter(box(sb.String(), 50, clrBorder), w),
		"",
	}, "\n")
}

// ─── Game Over view ───────────────────────────────────────────────────────────

func (m Model) viewGameOver() string {
	w := m.Width
	header := m.renderHeader("GAME OVER")

	winner := 0
	for p := 1; p < 4; p++ {
		if m.Scores[p] > m.Scores[winner] {
			winner = p
		}
	}

	resultLine := ""
	if winner == 0 {
		resultLine = bold(clrGold).Render("YOU WIN!  Well played!")
	} else {
		resultLine = fg(clrRed).Render(m.PlayerNames[winner] + " wins this time!")
	}

	var sb strings.Builder
	sb.WriteString(bold(clrGold).Render(fmt.Sprintf("%-14s %8s\n", "Player", "Score")))
	sb.WriteString(fg(clrBorder).Render(strings.Repeat("─", 26)) + "\n")
	for p := 0; p < 4; p++ {
		crown := "  "
		if p == winner {
			crown = fg(clrGold).Render("*  ")
		}
		sc := m.Scores[p]
		c := clrWhite
		if sc > 0 {
			c = clrGreen
		} else if sc < 0 {
			c = clrRed
		}
		sb.WriteString(crown + fg(clrWhite).Render(fmt.Sprintf("%-14s", m.PlayerNames[p])) +
			fg(c).Render(fmt.Sprintf("%8.1f", sc)) + "\n")
	}

	return strings.Join([]string{
		header, "",
		padCenter(resultLine, w),
		"",
		padCenter(box(sb.String(), 36, clrBorder), w),
		"",
		padCenter(fg(clrSubtle).Render("[ R / ENTER ] Restart    [ Q ] Quit"), w),
	}, "\n")
}
