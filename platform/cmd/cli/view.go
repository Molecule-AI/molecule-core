package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// tableReservedRows is the number of terminal rows consumed by chrome around
// the agent table (title bar, detail pane, event mini-log, help bar, borders).
const tableReservedRows = 20

// eventsReservedRows is the number of terminal rows consumed by chrome in the
// full events view (title bar + help bar + borders).
const eventsReservedRows = 6

// View renders the entire TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	w := m.width

	// Chat mode takes over the full display
	if m.chatting {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.renderChatTitleBar(w),
			m.renderChatView(w),
			m.renderHelpBar(),
		)
	}

	var sections []string

	// Title bar + tabs
	sections = append(sections, m.renderTitleBar(w))

	// Main content based on active tab
	switch m.activeTab {
	case TabAgents:
		sections = append(sections, m.renderAgentsView(w))
	case TabEvents:
		sections = append(sections, m.renderEventsFullView(w))
	case TabHealth:
		sections = append(sections, m.renderHealthView(w))
	}

	// Help bar
	sections = append(sections, m.renderHelpBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderTitleBar(w int) string {
	// Tabs — derived from iota so adding a new Tab only requires updating model.go
	tabs := make([]string, 0, int(tabCount))
	for t := Tab(0); t < tabCount; t++ {
		if t == m.activeTab {
			tabs = append(tabs, activeTab.Render(t.String()))
		} else {
			tabs = append(tabs, inactiveTab.Render(t.String()))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Status summary
	online, degraded, offline, prov := m.statusCounts()
	parts := []string{}
	if online > 0 {
		parts = append(parts, fmt.Sprintf("%d online", online))
	}
	if degraded > 0 {
		parts = append(parts, fmt.Sprintf("%d degraded", degraded))
	}
	if offline > 0 {
		parts = append(parts, fmt.Sprintf("%d offline", offline))
	}
	if prov > 0 {
		parts = append(parts, fmt.Sprintf("%d prov", prov))
	}
	summary := summaryStyle.Render(strings.Join(parts, ", "))

	// Connection indicator (pre-computed styles)
	connStatus := wsDisconnected
	if m.wsReady {
		connStatus = wsConnected
	}

	title := titleStyle.Render(" molecli")

	// Layout: title + tabs on left, summary + ws on right
	left := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabBar)
	right := lipgloss.JoinHorizontal(lipgloss.Top, summary, "  ", connStatus)

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

	// Error message
	if m.errMsg != "" {
		bar += "\n" + errorStyle.Render("  "+m.errMsg)
	}

	return bar
}

func (m Model) renderAgentsView(w int) string {
	var sections []string

	sections = append(sections, m.renderAgentTable(w))

	switch {
	case m.spawning:
		sections = append(sections, m.renderSpawnForm(w))
	case m.editing:
		sections = append(sections, m.renderEditForm(w))
	default:
		sections = append(sections, m.renderDetailPane(w))
	}

	sections = append(sections, m.renderEventMiniLog(w))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderAgentTable(w int) string {
	filtered := m.filteredWorkspaces()

	// Header
	header := formatTableRow("  NAME", "STATUS", "TASKS", "ERR%", "UPTIME", "ID")
	lines := []string{headerStyle.Render(header)}

	// Compute available rows for the table
	maxRows := m.height - tableReservedRows
	if maxRows < 3 {
		maxRows = 3
	}

	// Scroll window: keep selected row visible
	startIdx := 0
	if m.selected >= maxRows {
		startIdx = m.selected - maxRows + 1
	}
	endIdx := startIdx + maxRows
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := startIdx; i < endIdx; i++ {
		ws := filtered[i]
		cursor := "  "
		if i == m.selected {
			cursor = "► "
		}

		name := truncate(ws.Name, 18)
		status := fmt.Sprintf("%s %-12s", statusDot(ws.Status), ws.Status)
		tasks := fmt.Sprintf("%d", ws.ActiveTasks)
		errPct := fmt.Sprintf("%.0f%%", ws.LastErrorRate*100)
		uptime := formatDuration(ws.UptimeSeconds)
		id := shortID(ws.ID)

		row := formatTableRow(cursor+name, status, tasks, errPct, uptime, id)

		if i == m.selected {
			lines = append(lines, selectedStyle.Render(row))
		} else {
			lines = append(lines, normalStyle.Render(row))
		}
	}

	if len(filtered) == 0 {
		lines = append(lines, normalStyle.Render("  (no agents)"))
	}

	// Filter indicator
	if m.filtering {
		lines = append(lines, filterStyle.Render(fmt.Sprintf("  filter: %s█", m.filter)))
	} else if m.filter != "" {
		lines = append(lines, filterStyle.Render(fmt.Sprintf("  filter: %s (esc to clear)", m.filter)))
	}

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(content)
}

func (m Model) renderDetailPane(w int) string {
	ws := m.selectedWorkspace()
	if ws == nil {
		return panelStyle.Width(w - 2).Render(detailLabel.Render("  No agent selected"))
	}

	// Confirm delete prompt
	if m.confirmDelete {
		prompt := fmt.Sprintf("  Delete %s (%s)? [y/N]", ws.Name, shortID(ws.ID))
		return panelStyle.Width(w - 2).Render(errorStyle.Render(prompt))
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("  %s %s",
		detailLabel.Render(ws.Name),
		detailValue.Render("("+shortID(ws.ID)+")")))

	// Status line
	statusLine := fmt.Sprintf("  Status: %s %s", statusDot(ws.Status), ws.Status)
	statusLine += fmt.Sprintf("  |  Tier: %d", ws.Tier)
	statusLine += fmt.Sprintf("  |  Tasks: %d", ws.ActiveTasks)
	statusLine += fmt.Sprintf("  |  Uptime: %s", formatDuration(ws.UptimeSeconds))
	lines = append(lines, detailValue.Render(statusLine))

	// URL
	if ws.URL != "" {
		lines = append(lines, detailValue.Render(fmt.Sprintf("  URL: %s", ws.URL)))
	}

	// Role
	if ws.Role != nil && *ws.Role != "" {
		lines = append(lines, detailValue.Render(fmt.Sprintf("  Role: %s", *ws.Role)))
	}

	// Parent
	if ws.ParentID != nil && *ws.ParentID != "" {
		lines = append(lines, detailValue.Render(fmt.Sprintf("  Parent: %s", shortID(*ws.ParentID))))
	}

	// Agent card fields
	card := ParseAgentCard(ws.AgentCard)
	if card != nil {
		if card.Description != "" {
			lines = append(lines, detailValue.Render(fmt.Sprintf("  Description: %s", truncate(card.Description, 80))))
		}
		if card.URL != "" {
			lines = append(lines, detailValue.Render(fmt.Sprintf("  Card URL: %s", card.URL)))
		}
		if len(card.Skills) > 0 {
			lines = append(lines, detailLabel.Render("  Skills:"))
			for _, s := range card.Skills {
				name := s.Name
				if name == "" {
					name = s.ID
				}
				lines = append(lines, detailValue.Render(fmt.Sprintf("    • %s (%s)", name, s.ID)))
			}
		}
	}

	// Error info
	if ws.LastErrorRate > 0 {
		lines = append(lines, errorStyle.Render(fmt.Sprintf("  Error Rate: %.0f%%  Last Error: %s",
			ws.LastErrorRate*100, truncate(ws.LastSampleError, 50))))
	} else {
		lines = append(lines, detailValue.Render("  Last Error: (none)"))
	}

	// Full workspace ID at bottom
	lines = append(lines, helpStyle.Render(fmt.Sprintf("  ID: %s", ws.ID)))

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(content)
}

func (m Model) renderEventMiniLog(w int) string {
	lines := eventLines(m.events, 5)
	if len(lines) == 0 {
		lines = append(lines, normalStyle.Render("  (no events)"))
	}
	return panelStyle.Width(w - 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderEventsFullView(w int) string {
	maxVisible := m.height - eventsReservedRows
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Build full list newest-first
	all := eventLines(m.events, len(m.events))

	// Clamp scroll offset
	maxScroll := len(all) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.eventScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := len(all) - scroll
	if end < 0 {
		end = 0
	}
	start := end - maxVisible
	if start < 0 {
		start = 0
	}

	lines := []string{headerStyle.Render(formatEventHeader())}
	lines = append(lines, all[start:end]...)
	if len(m.events) == 0 {
		lines = append(lines, normalStyle.Render("  (no events)"))
	}

	// Scroll indicator
	if len(all) > maxVisible {
		pct := 0
		if maxScroll > 0 {
			pct = 100 * (maxScroll - scroll) / maxScroll
		}
		lines = append(lines, helpStyle.Render(fmt.Sprintf("  ↑↓ scroll  %d/%d events  %d%%", end, len(all), pct)))
	}

	return panelStyle.Width(w - 2).Render(strings.Join(lines, "\n"))
}

// eventLines renders the most recent max events in reverse-chronological order.
func eventLines(events []WSEvent, max int) []string {
	start := len(events) - max
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, len(events)-start)
	for i := len(events) - 1; i >= start; i-- {
		evt := events[i]
		ts := eventTime.Render(evt.Timestamp.Local().Format("15:04:05"))
		evtStr := eventType.Render(fmt.Sprintf("%-25s", evt.Event))
		id := normalStyle.Render(shortID(evt.WorkspaceID))
		lines = append(lines, fmt.Sprintf("  %s %s %s", ts, evtStr, id))
	}
	return lines
}

func (m Model) renderHealthView(w int) string {
	online, degraded, offline, prov := m.statusCounts()
	total := len(m.workspaces)

	var lines []string
	lines = append(lines, "")
	lines = append(lines, detailLabel.Render("  Platform Health Overview"))
	lines = append(lines, "")

	// Summary
	lines = append(lines, detailValue.Render(fmt.Sprintf("  Total Workspaces: %d", total)))
	lines = append(lines, fmt.Sprintf("  %s Online:       %d", statusDot("online"), online))
	lines = append(lines, fmt.Sprintf("  %s Degraded:     %d", statusDot("degraded"), degraded))
	lines = append(lines, fmt.Sprintf("  %s Offline:      %d", statusDot("offline"), offline))
	lines = append(lines, fmt.Sprintf("  %s Provisioning: %d", statusDot("provisioning"), prov))
	lines = append(lines, "")

	// Health bar
	if total > 0 {
		barWidth := w - 10
		if barWidth > 60 {
			barWidth = 60
		}
		if barWidth < 10 {
			barWidth = 10
		}
		greenW := barWidth * online / total
		yellowW := barWidth * degraded / total
		redW := barWidth * offline / total
		grayW := barWidth - greenW - yellowW - redW

		bar := barOnline.Render(strings.Repeat("█", greenW)) +
			barDegraded.Render(strings.Repeat("█", yellowW)) +
			barOffline.Render(strings.Repeat("█", redW)) +
			barProv.Render(strings.Repeat("░", grayW))

		lines = append(lines, "  "+bar)
		lines = append(lines, "")
	}

	// WebSocket status
	if m.wsReady {
		lines = append(lines, detailValue.Render("  WebSocket: connected"))
	} else {
		lines = append(lines, errorStyle.Render("  WebSocket: disconnected"))
	}

	// Last refresh
	if m.lastRefresh != nil {
		ago := time.Since(*m.lastRefresh).Truncate(time.Second)
		lines = append(lines, detailValue.Render(fmt.Sprintf("  Last Refresh: %s ago", ago)))
	}

	// Degraded agents list
	var degradedAgents []WorkspaceInfo
	for _, ws := range m.workspaces {
		if ws.Status == "degraded" {
			degradedAgents = append(degradedAgents, ws)
		}
	}
	if len(degradedAgents) > 0 {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("  Degraded Agents:"))
		for _, ws := range degradedAgents {
			lines = append(lines, errorStyle.Render(fmt.Sprintf("    %s (%s) - err: %.0f%%",
				ws.Name, shortID(ws.ID), ws.LastErrorRate*100)))
		}
	}

	content := strings.Join(lines, "\n")
	return panelStyle.Width(w - 2).Render(content)
}

// renderForm renders a generic multi-step form panel.
func renderForm(w int, title string, labels []string, fields []string, step int) string {
	var lines []string
	lines = append(lines, detailLabel.Render("  "+title))
	lines = append(lines, "")
	for i, label := range labels {
		switch {
		case i == step:
			lines = append(lines, filterStyle.Render(fmt.Sprintf("  ► %s: %s█", label, fields[i])))
		case i < step:
			val := fields[i]
			if val == "" {
				val = "(skipped)"
			}
			lines = append(lines, detailValue.Render(fmt.Sprintf("    %s: %s", label, val)))
		default:
			lines = append(lines, normalStyle.Render(fmt.Sprintf("    %s:", label)))
		}
	}
	return panelStyle.Width(w - 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderSpawnForm(w int) string {
	return renderForm(w, "Spawn New Agent",
		[]string{"Name", "Role (optional)", "Tier (optional, default 1)"},
		[]string{m.spawnName, m.spawnRole, m.spawnTierStr},
		m.spawnStep)
}

func (m Model) renderEditForm(w int) string {
	ws := m.selectedWorkspace()
	title := "Edit Agent"
	if ws != nil {
		title = fmt.Sprintf("Edit Agent: %s", ws.Name)
	}
	return renderForm(w, title,
		[]string{"Name", "Role (optional)", "Tier"},
		[]string{m.editName, m.editRole, m.editTierStr},
		m.editStep)
}

func (m Model) renderChatTitleBar(w int) string {
	name := m.chatWorkspaceName
	if name == "" {
		name = shortID(m.chatWorkspaceID)
	}
	title := titleStyle.Render(" molecli chat")
	agentInfo := activeTab.Render(fmt.Sprintf(" %s ", name))
	urlInfo := detailValue.Render(fmt.Sprintf("  %s", m.chatURL))

	left := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", agentInfo, urlInfo)
	connStatus := wsDisconnected
	if m.wsReady {
		connStatus = wsConnected
	}
	gap := w - lipgloss.Width(left) - lipgloss.Width(connStatus) - 2
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + connStatus
}

// chatReservedRows is rows consumed by title bar + input area + help bar.
const chatReservedRows = 7

func (m Model) renderChatView(w int) string {
	maxVisible := m.height - chatReservedRows
	if maxVisible < 4 {
		maxVisible = 4
	}

	// Build display lines from chat history
	var allLines []string
	for _, msg := range m.chatHistory {
		prefix := ""
		style := detailValue
		switch msg.Role {
		case "you":
			prefix = "you   ▶ "
			style = filterStyle
		case "agent":
			prefix = "agent ◀ "
			style = normalStyle
		}
		// Word-wrap long messages
		text := msg.Text
		maxText := w - len(prefix) - 6
		if maxText < 20 {
			maxText = 20
		}
		// Simple line chunking by maxText runes
		runes := []rune(text)
		for len(runes) > 0 {
			chunk := maxText
			if chunk > len(runes) {
				chunk = len(runes)
			}
			allLines = append(allLines, style.Render("  "+prefix+string(runes[:chunk])))
			runes = runes[chunk:]
			prefix = strings.Repeat(" ", len(prefix)) // indent continuation lines
		}
		allLines = append(allLines, "") // blank line between messages
	}

	if m.chatWaiting {
		allLines = append(allLines, detailValue.Render("  agent ◀ ..."))
	}

	// Apply scroll offset
	scroll := m.chatScroll
	maxScroll := len(allLines) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	start := len(allLines) - maxVisible - scroll
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(allLines) {
		end = len(allLines)
	}

	var lines []string
	if len(allLines) == 0 {
		lines = append(lines, helpStyle.Render("  (no messages yet — type below and press Enter)"))
	} else {
		lines = append(lines, allLines[start:end]...)
	}

	// Divider + input
	divider := strings.Repeat("─", w-4)
	lines = append(lines, helpStyle.Render("  "+divider))

	cursor := "█"
	if m.chatWaiting {
		cursor = "⠿"
	}
	inputLine := fmt.Sprintf("  you ▶ %s%s", m.chatInput, cursor)
	lines = append(lines, filterStyle.Render(inputLine))

	return panelStyle.Width(w - 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHelpBar() string {
	switch {
	case m.chatting:
		keys := []struct{ key, desc string }{
			{"enter", "send"},
			{"↑↓", "scroll"},
			{"esc", "exit chat"},
			{"ctrl+c", "quit"},
		}
		var parts []string
		for _, k := range keys {
			parts = append(parts, helpKey.Render(k.key)+" "+helpStyle.Render(k.desc))
		}
		return helpStyle.Render("  ") + strings.Join(parts, "  ")

	case m.spawning:
		if m.spawnStep < spawnStepTier {
			return helpStyle.Render("  type value | enter next field | esc cancel")
		}
		return helpStyle.Render("  type tier | enter spawn agent | esc cancel")
	case m.editing:
		if m.editStep < spawnStepTier {
			return helpStyle.Render("  edit value | enter next field | esc cancel")
		}
		return helpStyle.Render("  edit tier | enter save changes | esc cancel")
	case m.filtering:
		return helpStyle.Render("  type to filter | enter confirm | esc cancel")
	case m.confirmDelete:
		return helpStyle.Render("  y confirm delete | any other key cancel")
	case m.activeTab == TabEvents:
		keys := []struct{ key, desc string }{
			{"↑↓/jk", "scroll"},
			{"Tab", "switch panel"},
			{"q", "quit"},
		}
		var parts []string
		for _, k := range keys {
			parts = append(parts, helpKey.Render(k.key)+" "+helpStyle.Render(k.desc))
		}
		return helpStyle.Render("  ") + strings.Join(parts, "  ")
	}

	keys := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"Tab", "switch panel"},
		{"enter", "chat"},
		{"n", "spawn"},
		{"e", "edit"},
		{"d", "delete"},
		{"r", "refresh"},
		{"/", "filter"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKey.Render(k.key)+" "+helpStyle.Render(k.desc))
	}
	return helpStyle.Render("  ") + strings.Join(parts, "  ")
}

// Helpers

func formatTableRow(name, status, tasks, errPct, uptime, id string) string {
	return fmt.Sprintf("%-20s %-14s %5s %5s %8s  %s", name, status, tasks, errPct, uptime, id)
}

func formatEventHeader() string {
	return fmt.Sprintf("  %-10s %-25s %s", "TIME", "EVENT", "WORKSPACE")
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	sec := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, min)
	}
	if min > 0 {
		return fmt.Sprintf("%dm%ds", min, sec)
	}
	return fmt.Sprintf("%ds", sec)
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// truncate safely truncates a string to maxLen runes, appending "..." if needed.
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-3]) + "..."
}
