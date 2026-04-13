package main

import (
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Tab represents the active view tab.
type Tab int

const (
	TabAgents Tab = iota
	TabEvents
	TabHealth
	tabCount // sentinel: always equals the number of tabs
)

func (t Tab) String() string {
	switch t {
	case TabAgents:
		return "Agents"
	case TabEvents:
		return "Events"
	case TabHealth:
		return "Health"
	default:
		return "?"
	}
}

// Model is the top-level bubbletea model.
type Model struct {
	// Data
	workspaces []WorkspaceInfo
	events     []WSEvent // Real-time events (from WS + initial fetch)
	eventIDs   map[string]struct{} // Deduplicates HTTP-fetched EventInfo by ID; WS events (no ID) bypass this

	// UI state
	activeTab Tab
	selected  int    // Index in filtered workspace list
	filter    string // Name filter text
	filtering bool   // Whether filter input is active

	// Connection state.
	// wsConn is a pointer to mutable shared state — intentional, since bubbletea
	// copies Model by value but the pointer keeps the connection shared between
	// the model and the background listenWS goroutine.
	baseURL string
	client  *PlatformClient
	wsConn  *websocket.Conn
	wsReady bool
	wsGen   int // connection generation; incremented on each new WS connection

	// Dimensions
	width  int
	height int

	// Status
	lastRefresh *time.Time // nil until first successful fetch
	errMsg      string

	// Confirm delete
	confirmDelete bool

	// Spawn form (multi-step)
	spawning     bool
	spawnStep    int // 0=name, 1=role, 2=tier
	spawnName    string
	spawnRole    string
	spawnTierStr string

	// Edit form (multi-step, pre-filled from selected workspace)
	editing     bool
	editStep    int // 0=name, 1=role, 2=tier
	editName    string
	editRole    string
	editTierStr string

	// Event log scroll offset (Events tab)
	eventScroll int

	// Chat mode (A2A)
	chatting          bool
	chatWorkspaceID   string
	chatWorkspaceName string
	chatURL           string
	chatHistory       []ChatMsg
	chatInput         string
	chatWaiting       bool
	chatScroll        int // how many lines scrolled up from bottom
}

// ChatMsg is a single turn in a chat session.
type ChatMsg struct {
	Role string // "you" or "agent"
	Text string
}

// NewModel creates the initial model.
func NewModel(baseURL string) Model {
	return Model{
		baseURL:  baseURL,
		client:   NewPlatformClient(baseURL),
		eventIDs: make(map[string]struct{}),
		wsGen:    1, // start at 1 so Gen==0 is never valid — no special case needed
	}
}

// filteredWorkspaces returns workspaces matching the current filter.
func (m Model) filteredWorkspaces() []WorkspaceInfo {
	if m.filter == "" {
		return m.workspaces
	}
	f := strings.ToLower(m.filter)
	var result []WorkspaceInfo
	for _, w := range m.workspaces {
		if strings.Contains(strings.ToLower(w.Name), f) {
			result = append(result, w)
		}
	}
	return result
}

// selectedWorkspace returns the currently selected workspace, or nil.
func (m Model) selectedWorkspace() *WorkspaceInfo {
	filtered := m.filteredWorkspaces()
	if m.selected < 0 || m.selected >= len(filtered) {
		return nil
	}
	ws := filtered[m.selected]
	return &ws
}

// statusCounts returns counts by status.
func (m Model) statusCounts() (online, degraded, offline, provisioning int) {
	for _, w := range m.workspaces {
		switch w.Status {
		case "online":
			online++
		case "degraded":
			degraded++
		case "offline":
			offline++
		case "provisioning":
			provisioning++
		}
	}
	return
}

// clampSelected ensures selected index is within bounds.
func (m *Model) clampSelected() {
	filtered := m.filteredWorkspaces()
	if m.selected >= len(filtered) {
		m.selected = len(filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}
