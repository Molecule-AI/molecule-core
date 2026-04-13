package main

import (
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Spawn form steps.
const (
	spawnStepName = 0
	spawnStepRole = 1
	spawnStepTier = 2
)

// Messages for async operations
type WorkspacesMsg struct {
	Workspaces []WorkspaceInfo
}

type EventsMsg struct {
	Events []EventInfo
}

type ErrMsg struct {
	Err error
}

type DeletedMsg struct {
	ID string
}

type CreatedMsg struct {
	ID     string
	Name   string
	Status string
}

type UpdatedMsg struct {
	ID string
}

type AgentDiscoveredForChatMsg struct {
	ID   string
	Name string
	URL  string
}

type ChatResponseMsg struct {
	Text string
}

type refreshTickMsg struct{}

// Init returns the initial commands: fetch data, connect WS, start tick.
// Events are bootstrapped once via HTTP; all subsequent events arrive over WS.
// Only workspaces are re-fetched on the periodic refresh tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchWorkspacesCmd(m.client),
		fetchEventsCmd(m.client),
		connectWSCmd(m.baseURL),
		tickCmd(),
	)
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case AgentDiscoveredForChatMsg:
		if msg.URL == "" {
			m.errMsg = "agent has no URL (is it online?)"
			return m, nil
		}
		m.chatting = true
		m.chatWorkspaceID = msg.ID
		m.chatWorkspaceName = msg.Name
		m.chatURL = msg.URL
		m.chatHistory = nil
		m.chatInput = ""
		m.chatWaiting = false
		m.chatScroll = 0
		return m, nil

	case ChatResponseMsg:
		m.chatWaiting = false
		if msg.Text != "" {
			m.chatHistory = append(m.chatHistory, ChatMsg{Role: "agent", Text: msg.Text})
		}
		m.chatScroll = 0 // snap to bottom on new reply
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case WorkspacesMsg:
		m.workspaces = msg.Workspaces
		sortWorkspaces(m.workspaces)
		m.clampSelected()
		now := time.Now()
		m.lastRefresh = &now
		m.errMsg = ""
		return m, nil

	case EventsMsg:
		// Convert EventInfo to WSEvent for display, deduplicating by ID
		for _, e := range msg.Events {
			if _, seen := m.eventIDs[e.ID]; seen {
				continue
			}
			m.eventIDs[e.ID] = struct{}{}
			wsID := ""
			if e.WorkspaceID != nil {
				wsID = *e.WorkspaceID
			}
			m.events = append(m.events, WSEvent{
				Event:       e.EventType,
				WorkspaceID: wsID,
				Timestamp:   e.CreatedAt,
				Payload:     e.Payload,
			})
		}
		trimEvents(&m.events, 200)
		pruneEventIDs(m.eventIDs, 500)
		return m, nil

	case WsConnectedMsg:
		// Close previous connection if any (prevents leak)
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		m.wsConn = msg.Conn
		m.wsReady = true
		m.wsGen++
		m.errMsg = ""
		return m, listenWS(msg.Conn, m.wsGen)

	case WsEventMsg:
		// Ignore events from stale connections
		if msg.Gen != m.wsGen {
			return m, nil
		}
		if msg.Event.Event != "PARSE_ERROR" {
			m.events = append(m.events, msg.Event)
			trimEvents(&m.events, 200)
			applyEvent(&m, msg.Event)
		}
		// Keep listening on current connection
		if m.wsConn != nil {
			return m, listenWS(m.wsConn, m.wsGen)
		}
		return m, nil

	case WsErrorMsg:
		// Ignore errors from stale connections
		if msg.Gen != m.wsGen {
			return m, nil
		}
		// Close the failed connection
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		m.wsReady = false
		m.wsConn = nil
		m.errMsg = "WS disconnected, reconnecting..."
		return m, reconnectWSCmd()

	case wsReconnectTickMsg:
		// Fired after reconnect delay — attempt a fresh connection
		return m, connectWSCmd(m.baseURL)

	case ErrMsg:
		m.errMsg = msg.Err.Error()
		return m, nil

	case DeletedMsg:
		// Remove from local list
		for i, w := range m.workspaces {
			if w.ID == msg.ID {
				m.workspaces = append(m.workspaces[:i], m.workspaces[i+1:]...)
				break
			}
		}
		m.clampSelected()
		m.confirmDelete = false
		return m, nil

	case CreatedMsg:
		m.errMsg = ""
		return m, fetchWorkspacesCmd(m.client)

	case UpdatedMsg:
		m.errMsg = ""
		return m, fetchWorkspacesCmd(m.client)

	case refreshTickMsg:
		return m, tea.Batch(
			fetchWorkspacesCmd(m.client),
			tickCmd(),
		)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If in chat mode, delegate all input there
	if m.chatting {
		return m.handleChatKey(msg)
	}

	// If in spawn form, handle multi-step input
	if m.spawning {
		return m.handleSpawnKey(msg)
	}

	// If in edit form, handle multi-step input
	if m.editing {
		return m.handleEditKey(msg)
	}

	// If in filter mode, handle text input
	if m.filtering {
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.wsConn != nil {
				m.wsConn.Close()
			}
			return m, tea.Quit
		case tea.KeyEnter, tea.KeyEscape:
			m.filtering = false
			m.clampSelected()
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.selected = 0
			}
			return m, nil
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.selected = 0
			return m, nil
		case tea.KeySpace:
			m.filter += " "
			m.selected = 0
			return m, nil
		default:
			return m, nil
		}
	}

	// If confirming delete
	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			ws := m.selectedWorkspace()
			if ws != nil {
				id := ws.ID
				m.confirmDelete = false
				return m, deleteWorkspaceCmd(m.client, id)
			}
			m.confirmDelete = false
			return m, nil
		default:
			m.confirmDelete = false
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		return m, tea.Quit

	case "tab":
		m.activeTab = (m.activeTab + 1) % tabCount
		return m, nil

	case "shift+tab":
		m.activeTab = (m.activeTab + tabCount - 1) % tabCount
		return m, nil

	case "up", "k":
		if m.activeTab == TabEvents {
			if m.eventScroll > 0 {
				m.eventScroll--
			}
		} else {
			if m.selected > 0 {
				m.selected--
			}
		}
		return m, nil

	case "down", "j":
		if m.activeTab == TabEvents {
			m.eventScroll++
		} else {
			filtered := m.filteredWorkspaces()
			if m.selected < len(filtered)-1 {
				m.selected++
			}
		}
		return m, nil

	case "enter":
		if m.activeTab == TabAgents {
			ws := m.selectedWorkspace()
			if ws != nil {
				return m, discoverForChatCmd(m.client, ws.ID, ws.Name)
			}
		}
		return m, nil

	case "n":
		m.spawning = true
		m.spawnStep = spawnStepName
		m.spawnName = ""
		m.spawnRole = ""
		m.spawnTierStr = ""
		return m, nil

	case "e":
		ws := m.selectedWorkspace()
		if ws != nil {
			m.editing = true
			m.editStep = spawnStepName
			m.editName = ws.Name
			m.editRole = ""
			if ws.Role != nil {
				m.editRole = *ws.Role
			}
			m.editTierStr = strconv.Itoa(ws.Tier)
		}
		return m, nil

	case "d":
		ws := m.selectedWorkspace()
		if ws != nil {
			m.confirmDelete = true
		}
		return m, nil

	case "r":
		return m, fetchWorkspacesCmd(m.client)

	case "/":
		m.filtering = true
		m.filter = ""
		m.selected = 0
		return m, nil

	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.clampSelected()
		}
		return m, nil
	}

	return m, nil
}

// Commands

// handleSpawnKey processes key input for the multi-step spawn form.
func (m Model) handleSpawnKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		return m, tea.Quit

	case tea.KeyEscape:
		m.spawning = false
		return m, nil

	case tea.KeyBackspace:
		switch m.spawnStep {
		case spawnStepName:
			if r := []rune(m.spawnName); len(r) > 0 {
				m.spawnName = string(r[:len(r)-1])
			}
		case spawnStepRole:
			if r := []rune(m.spawnRole); len(r) > 0 {
				m.spawnRole = string(r[:len(r)-1])
			}
		case spawnStepTier:
			if r := []rune(m.spawnTierStr); len(r) > 0 {
				m.spawnTierStr = string(r[:len(r)-1])
			}
		}
		return m, nil

	case tea.KeyEnter:
		// Validate name on first step
		if m.spawnStep == spawnStepName && m.spawnName == "" {
			return m, nil // require a name
		}
		if m.spawnStep < spawnStepTier {
			m.spawnStep++
			return m, nil
		}
		// Final step — submit
		tier := 1
		if n, err := strconv.Atoi(m.spawnTierStr); err == nil && n > 0 {
			tier = n
		}
		req := CreateWorkspaceRequest{
			Name: m.spawnName,
			Role: m.spawnRole,
			Tier: tier,
		}
		m.spawning = false
		return m, createWorkspaceCmd(m.client, req)

	case tea.KeyRunes:
		switch m.spawnStep {
		case spawnStepName:
			m.spawnName += string(msg.Runes)
		case spawnStepRole:
			m.spawnRole += string(msg.Runes)
		case spawnStepTier:
			// Only allow digits
			for _, r := range msg.Runes {
				if r >= '0' && r <= '9' {
					m.spawnTierStr += string(r)
				}
			}
		}
		return m, nil

	case tea.KeySpace:
		switch m.spawnStep {
		case spawnStepName:
			m.spawnName += " "
		case spawnStepRole:
			m.spawnRole += " "
		}
		return m, nil
	}

	return m, nil
}

// handleEditKey processes key input for the multi-step edit form.
func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		return m, tea.Quit

	case tea.KeyEscape:
		m.editing = false
		return m, nil

	case tea.KeyBackspace:
		switch m.editStep {
		case spawnStepName:
			if r := []rune(m.editName); len(r) > 0 {
				m.editName = string(r[:len(r)-1])
			}
		case spawnStepRole:
			if r := []rune(m.editRole); len(r) > 0 {
				m.editRole = string(r[:len(r)-1])
			}
		case spawnStepTier:
			if r := []rune(m.editTierStr); len(r) > 0 {
				m.editTierStr = string(r[:len(r)-1])
			}
		}
		return m, nil

	case tea.KeyEnter:
		if m.editStep == spawnStepName && m.editName == "" {
			return m, nil // name is required
		}
		if m.editStep < spawnStepTier {
			m.editStep++
			return m, nil
		}
		// Submit — build patch from edited values
		ws := m.selectedWorkspace()
		if ws == nil {
			m.editing = false
			return m, nil
		}
		req := UpdateWorkspaceRequest{}
		if m.editName != ws.Name {
			req.Name = &m.editName
		}
		currentRole := ""
		if ws.Role != nil {
			currentRole = *ws.Role
		}
		if m.editRole != currentRole {
			req.Role = &m.editRole
		}
		tier := ws.Tier
		if n, err := strconv.Atoi(m.editTierStr); err == nil && n > 0 {
			tier = n
		}
		if tier != ws.Tier {
			req.Tier = &tier
		}
		id := ws.ID
		m.editing = false
		return m, updateWorkspaceCmd(m.client, id, req)

	case tea.KeyRunes:
		switch m.editStep {
		case spawnStepName:
			m.editName += string(msg.Runes)
		case spawnStepRole:
			m.editRole += string(msg.Runes)
		case spawnStepTier:
			for _, r := range msg.Runes {
				if r >= '0' && r <= '9' {
					m.editTierStr += string(r)
				}
			}
		}
		return m, nil

	case tea.KeySpace:
		switch m.editStep {
		case spawnStepName:
			m.editName += " "
		case spawnStepRole:
			m.editRole += " "
		}
		return m, nil
	}

	return m, nil
}

// handleChatKey processes key input while in A2A chat mode.
func (m Model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.wsConn != nil {
			m.wsConn.Close()
		}
		return m, tea.Quit

	case tea.KeyEscape:
		m.chatting = false
		return m, nil

	case tea.KeyEnter:
		input := m.chatInput
		if input == "" || m.chatWaiting {
			return m, nil
		}
		m.chatHistory = append(m.chatHistory, ChatMsg{Role: "you", Text: input})
		m.chatInput = ""
		m.chatWaiting = true
		m.chatScroll = 0
		return m, sendChatCmd(m.chatURL, input)

	case tea.KeyBackspace:
		if r := []rune(m.chatInput); len(r) > 0 {
			m.chatInput = string(r[:len(r)-1])
		}
		return m, nil

	case tea.KeyUp:
		m.chatScroll++
		return m, nil

	case tea.KeyDown:
		if m.chatScroll > 0 {
			m.chatScroll--
		}
		return m, nil

	case tea.KeyRunes:
		if !m.chatWaiting {
			m.chatInput += string(msg.Runes)
		}
		return m, nil

	case tea.KeySpace:
		if !m.chatWaiting {
			m.chatInput += " "
		}
		return m, nil
	}

	return m, nil
}

func discoverForChatCmd(client *PlatformClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		disc, err := client.DiscoverWorkspace(id, "")
		if err != nil {
			return ErrMsg{Err: err}
		}
		return AgentDiscoveredForChatMsg{ID: disc.ID, Name: name, URL: disc.URL}
	}
}

func sendChatCmd(agentURL, text string) tea.Cmd {
	return func() tea.Msg {
		a2a := newA2AClient(agentURL)
		reply, err := a2a.SendTask(text)
		if err != nil {
			return ChatResponseMsg{Text: "[error] " + err.Error()}
		}
		return ChatResponseMsg{Text: reply}
	}
}

func updateWorkspaceCmd(client *PlatformClient, id string, req UpdateWorkspaceRequest) tea.Cmd {
	return func() tea.Msg {
		if err := client.UpdateWorkspace(id, req); err != nil {
			return ErrMsg{Err: err}
		}
		return UpdatedMsg{ID: id}
	}
}

func createWorkspaceCmd(client *PlatformClient, req CreateWorkspaceRequest) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.CreateWorkspace(req)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return CreatedMsg{ID: resp.ID, Name: req.Name, Status: resp.Status}
	}
}

func fetchWorkspacesCmd(client *PlatformClient) tea.Cmd {
	return func() tea.Msg {
		workspaces, err := client.FetchWorkspaces()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return WorkspacesMsg{Workspaces: workspaces}
	}
}

func fetchEventsCmd(client *PlatformClient) tea.Cmd {
	return func() tea.Msg {
		events, err := client.FetchEvents()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return EventsMsg{Events: events}
	}
}

func deleteWorkspaceCmd(client *PlatformClient, id string) tea.Cmd {
	return func() tea.Msg {
		if err := client.DeleteWorkspace(id); err != nil {
			return ErrMsg{Err: err}
		}
		return DeletedMsg{ID: id}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(_ time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

// applyEvent updates workspace list in-place from a WebSocket event.
func applyEvent(m *Model, evt WSEvent) {
	switch evt.Event {
	case "WORKSPACE_PROVISIONING":
		// Add new workspace if not present
		for _, w := range m.workspaces {
			if w.ID == evt.WorkspaceID {
				return
			}
		}
		m.workspaces = append(m.workspaces, WorkspaceInfo{
			ID:     evt.WorkspaceID,
			Status: "provisioning",
			Name:   extractPayloadString(evt.Payload, "name"),
		})
		sortWorkspaces(m.workspaces)

	case "WORKSPACE_ONLINE":
		for i := range m.workspaces {
			if m.workspaces[i].ID == evt.WorkspaceID {
				m.workspaces[i].Status = "online"
				return
			}
		}
		// Workspace not in list yet — add it
		m.workspaces = append(m.workspaces, WorkspaceInfo{
			ID:     evt.WorkspaceID,
			Status: "online",
		})
		sortWorkspaces(m.workspaces)

	case "WORKSPACE_DEGRADED":
		for i := range m.workspaces {
			if m.workspaces[i].ID == evt.WorkspaceID {
				m.workspaces[i].Status = "degraded"
				// Update error rate and sample error from payload (single parse)
				if p := parsePayloadMap(evt.Payload); p != nil {
					if f, ok := p["error_rate"].(float64); ok && f > 0 {
						m.workspaces[i].LastErrorRate = f
					}
					if s, ok := p["sample_error"].(string); ok && s != "" {
						m.workspaces[i].LastSampleError = s
					}
				}
				return
			}
		}

	case "WORKSPACE_OFFLINE":
		for i := range m.workspaces {
			if m.workspaces[i].ID == evt.WorkspaceID {
				m.workspaces[i].Status = "offline"
				return
			}
		}

	case "WORKSPACE_REMOVED":
		for i, w := range m.workspaces {
			if w.ID == evt.WorkspaceID {
				m.workspaces = append(m.workspaces[:i], m.workspaces[i+1:]...)
				m.clampSelected()
				return
			}
		}

	case "AGENT_CARD_UPDATED":
		for i := range m.workspaces {
			if m.workspaces[i].ID == evt.WorkspaceID {
				card := extractPayloadRaw(evt.Payload, "agent_card")
				if card != nil {
					m.workspaces[i].AgentCard = card
				}
				return
			}
		}
	}
}

func sortWorkspaces(ws []WorkspaceInfo) {
	sort.Slice(ws, func(i, j int) bool {
		return ws[i].Name < ws[j].Name
	})
}

func trimEvents(events *[]WSEvent, max int) {
	if len(*events) > max {
		keep := (*events)[len(*events)-max:]
		// Copy into a new slice so the old backing array can be GC'd.
		trimmed := make([]WSEvent, len(keep))
		copy(trimmed, keep)
		*events = trimmed
	}
}

// pruneEventIDs caps the dedup map to avoid unbounded growth over long sessions.
func pruneEventIDs(ids map[string]struct{}, maxSize int) {
	if len(ids) <= maxSize {
		return
	}
	// Clear the whole map — duplicates from already-trimmed events are harmless
	for k := range ids {
		delete(ids, k)
	}
}

// parsePayloadMap unmarshals a JSON payload into a generic map once,
// so callers can read multiple keys without re-parsing.
func parsePayloadMap(payload []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil
	}
	return m
}

func extractPayloadString(payload []byte, key string) string {
	p := parsePayloadMap(payload)
	if p == nil {
		return ""
	}
	if s, ok := p[key].(string); ok {
		return s
	}
	return ""
}

func extractPayloadRaw(payload []byte, key string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil
	}
	if v, ok := m[key]; ok {
		return v
	}
	return nil
}

// deleteURL safely constructs the delete endpoint URL.
func deleteURL(baseURL, id string) (string, error) {
	return url.JoinPath(baseURL, "workspaces", id)
}
