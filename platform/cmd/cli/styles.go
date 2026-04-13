package main

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorOnline    = lipgloss.Color("#00FF00")
	colorDegraded  = lipgloss.Color("#FFAA00")
	colorOffline   = lipgloss.Color("#FF4444")
	colorProvision = lipgloss.Color("#888888")
	colorAccent    = lipgloss.Color("#7D56F4")
	colorDim       = lipgloss.Color("#666666")
	colorWhite     = lipgloss.Color("#FFFFFF")
	colorBorder    = lipgloss.Color("#444444")
	colorNormal    = lipgloss.Color("#CCCCCC")

	colorTabBg = lipgloss.Color("#555555")

	// Tab styles
	activeTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorAccent).
			Padding(0, 2)

	inactiveTab = lipgloss.NewStyle().
			Foreground(colorDim).
			Background(colorTabBg).
			Padding(0, 2)

	// Panel borders
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	// Status summary in title bar
	summaryStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// Table header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Underline(true)

	// Selected row
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	// Normal row
	normalStyle = lipgloss.NewStyle().
			Foreground(colorNormal)

	// Detail pane label
	detailLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	// Detail pane value
	detailValue = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Event timestamp
	eventTime = lipgloss.NewStyle().
			Foreground(colorDim)

	// Event type
	eventType = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	// Help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	helpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWhite)

	// Filter input
	filterStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	// Error message
	errorStyle = lipgloss.NewStyle().
			Foreground(colorOffline)

	// Pre-computed status dot styles (avoid allocating on every render)
	dotOnline   = lipgloss.NewStyle().Foreground(colorOnline).Render("●")
	dotDegraded = lipgloss.NewStyle().Foreground(colorDegraded).Render("●")
	dotOffline  = lipgloss.NewStyle().Foreground(colorOffline).Render("●")
	dotProv     = lipgloss.NewStyle().Foreground(colorProvision).Render("○")
	dotDefault  = lipgloss.NewStyle().Foreground(colorDim).Render("○")

	// Pre-computed health bar segment styles
	barOnline   = lipgloss.NewStyle().Foreground(colorOnline)
	barDegraded = lipgloss.NewStyle().Foreground(colorDegraded)
	barOffline  = lipgloss.NewStyle().Foreground(colorOffline)
	barProv     = lipgloss.NewStyle().Foreground(colorProvision)

	// Pre-computed WS indicator styles
	wsConnected    = lipgloss.NewStyle().Foreground(colorOnline).Render("● WS")
	wsDisconnected = lipgloss.NewStyle().Foreground(colorOffline).Render("○ WS")
)

func statusDot(status string) string {
	switch status {
	case "online":
		return dotOnline
	case "degraded":
		return dotDegraded
	case "offline":
		return dotOffline
	case "provisioning":
		return dotProv
	default:
		return dotDefault
	}
}
