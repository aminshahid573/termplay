package styles

import "github.com/charmbracelet/lipgloss"

// Define colors locally (private) so we can use them in styles
var (
	subtleColor    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	specialColor   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	errColor       = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F55385"}
	winColor       = lipgloss.AdaptiveColor{Light: "#00FF00", Dark: "#00FF00"}
)

var (
	// --- Base Text ---
	Base   = lipgloss.NewStyle().Foreground(colorText)
	Subtle = lipgloss.NewStyle().Foreground(colorSubtle)
	Muted  = lipgloss.NewStyle().Foreground(colorMuted)
	
	// --- Section / Headers ---
	SectionTitle = lipgloss.NewStyle().Foreground(colorText)
	SectionLine  = lipgloss.NewStyle().Foreground(colorBorder)
	ItemBlurred     = lipgloss.NewStyle().Padding(0, 1).Foreground(colorText)
	ItemFocused     = lipgloss.NewStyle().Padding(0, 1).Background(colorPurple).Foreground(colorBgDark)
	InfoTextBlurred = lipgloss.NewStyle().Foreground(colorSubtle)
	InfoTextFocused = lipgloss.NewStyle().Foreground(colorBgDark) // Dark text on purple bg


	// --- Game Board ---
	Title = lipgloss.NewStyle().
		Foreground(colorGreen).Bold(true).
		Background(lipgloss.Color("235")).
		Padding(0, 7).
		MarginBottom(1)

		ListContainer = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(0, 1).
		Width(60)

	// Search bar with NO border (just text style)
	SearchBar = lipgloss.NewStyle().
		Foreground(colorHighlight).
		Bold(true)

	Cell = lipgloss.NewStyle().
		Width(10).Height(5).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorMuted)

	CellSelected = Cell.Copy().
		BorderForeground(colorPurple).
		Background(lipgloss.Color("236"))

	CellWin = Cell.Copy().
		BorderForeground(colorGreen).
		Background(lipgloss.Color("22"))

	XStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	OStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	PopupBox = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#F25D94")).
		Padding(1, 2).
		Align(lipgloss.Center, lipgloss.Center)
)

var (
	colorPurple    = lipgloss.Color("#a1a9f5") // Charple
	colorText      = lipgloss.Color("#b8c5d6") // Ash
	colorMuted     = lipgloss.Color("#5f6f7f") // Squid
	colorSubtle    = lipgloss.Color("#a8a9a9") // Oyster
	colorBorder    = lipgloss.Color("#3d4d5c") // Charcoal
	colorHighlight = lipgloss.Color("#e3b7ff") // Dolly
	colorGreen     = lipgloss.Color("#76b639")
	colorBgDark    = lipgloss.Color("#000000")
)
var (
	// Text Styles (These have .Render methods)
	Highlight = lipgloss.NewStyle().Foreground(highlightColor)
	Special   = lipgloss.NewStyle().Foreground(specialColor)
	Err       = lipgloss.NewStyle().Foreground(errColor)
	Win       = lipgloss.NewStyle().Foreground(winColor)


	MenuItem     = lipgloss.NewStyle().PaddingLeft(2)
	MenuSelected = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(highlightColor).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(highlightColor)

	// Box / Board Styles
	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(0, 1)


)
