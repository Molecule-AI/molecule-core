package main

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"
)

// ---- formatDuration ----

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds int
		want    string
	}{
		{0, "0s"},
		{-5, "0s"},
		{1, "1s"},
		{59, "59s"},
		{60, "1m0s"},
		{61, "1m1s"},
		{3599, "59m59s"},
		{3600, "1h0m"},
		{7261, "2h1m"},
	}
	for _, c := range cases {
		got := formatDuration(c.seconds)
		if got != c.want {
			t.Errorf("formatDuration(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

// ---- truncate ----

func TestTruncate(t *testing.T) {
	cases := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},          // under limit
		{"hello", 5, "hello"},           // exact limit
		{"hello world", 8, "hello..."},  // over limit
		{"", 5, ""},                     // empty
		{"héllo wörld", 8, "héllo..."},  // multibyte UTF-8
		{"ab", 3, "ab"},                 // shorter than maxLen
	}
	for _, c := range cases {
		got := truncate(c.input, c.maxLen)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.input, c.maxLen, got, c.want)
		}
	}
}

// ---- shortID ----

func TestShortID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"abc", "abc"},                       // shorter than 8
		{"abcdefgh", "abcdefgh"},             // exactly 8
		{"abcdefgh-ijkl-mnop", "abcdefgh"},   // longer than 8
		{"", ""},                             // empty
	}
	for _, c := range cases {
		got := shortID(c.input)
		if got != c.want {
			t.Errorf("shortID(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ---- parsePayloadMap ----

func TestParsePayloadMap(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		if parsePayloadMap(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})
	t.Run("empty input", func(t *testing.T) {
		if parsePayloadMap([]byte{}) != nil {
			t.Error("expected nil for empty input")
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		if parsePayloadMap([]byte(`{not json}`)) != nil {
			t.Error("expected nil for malformed JSON")
		}
	})
	t.Run("empty object", func(t *testing.T) {
		m := parsePayloadMap([]byte(`{}`))
		if m == nil {
			t.Error("expected non-nil for empty object")
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %v", m)
		}
	})
	t.Run("valid keys", func(t *testing.T) {
		m := parsePayloadMap([]byte(`{"error_rate": 0.8, "sample_error": "timeout"}`))
		if m == nil {
			t.Fatal("expected non-nil map")
		}
		if v, ok := m["error_rate"].(float64); !ok {
			t.Fatalf("error_rate is not float64: %T", m["error_rate"])
		} else if v != 0.8 {
			t.Errorf("wrong error_rate: %v", v)
		}
		if v, ok := m["sample_error"].(string); !ok {
			t.Fatalf("sample_error is not string: %T", m["sample_error"])
		} else if v != "timeout" {
			t.Errorf("wrong sample_error: %v", v)
		}
	})
}

// ---- extractPayloadString ----

func TestExtractPayloadString(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		got := extractPayloadString([]byte(`{"other": "val"}`), "name")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
	t.Run("wrong type", func(t *testing.T) {
		got := extractPayloadString([]byte(`{"name": 42}`), "name")
		if got != "" {
			t.Errorf("expected empty string for non-string value, got %q", got)
		}
	})
	t.Run("valid", func(t *testing.T) {
		got := extractPayloadString([]byte(`{"name": "echo-agent"}`), "name")
		if got != "echo-agent" {
			t.Errorf("expected %q, got %q", "echo-agent", got)
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		got := extractPayloadString([]byte(`not json`), "name")
		if got != "" {
			t.Errorf("expected empty string for malformed JSON, got %q", got)
		}
	})
}

// ---- extractPayloadRaw ----

func TestExtractPayloadRaw(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		got := extractPayloadRaw([]byte(`{"other": {}}`), "agent_card")
		if got != nil {
			t.Errorf("expected nil for missing key, got %v", got)
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		got := extractPayloadRaw([]byte(`not json`), "agent_card")
		if got != nil {
			t.Errorf("expected nil for malformed JSON, got %v", got)
		}
	})
	t.Run("valid nested object", func(t *testing.T) {
		payload := []byte(`{"agent_card": {"name": "Echo", "skills": []}}`)
		got := extractPayloadRaw(payload, "agent_card")
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		var card AgentCardInfo
		if err := json.Unmarshal(got, &card); err != nil {
			t.Fatalf("failed to unmarshal extracted raw: %v", err)
		}
		if card.Name != "Echo" {
			t.Errorf("expected name %q, got %q", "Echo", card.Name)
		}
	})
}

// ---- pruneEventIDs ----

func TestPruneEventIDs(t *testing.T) {
	t.Run("below threshold — no prune", func(t *testing.T) {
		ids := map[string]struct{}{"a": {}, "b": {}}
		pruneEventIDs(ids, 5)
		if len(ids) != 2 {
			t.Errorf("expected 2 entries, got %d", len(ids))
		}
	})
	t.Run("at threshold — no prune", func(t *testing.T) {
		ids := map[string]struct{}{"a": {}, "b": {}, "c": {}}
		pruneEventIDs(ids, 3)
		if len(ids) != 3 {
			t.Errorf("expected 3 entries, got %d", len(ids))
		}
	})
	t.Run("above threshold — clears map", func(t *testing.T) {
		ids := map[string]struct{}{"a": {}, "b": {}, "c": {}, "d": {}}
		pruneEventIDs(ids, 3)
		if len(ids) != 0 {
			t.Errorf("expected 0 entries after prune, got %d", len(ids))
		}
	})
}

// ---- trimEvents ----

func TestTrimEvents(t *testing.T) {
	makeEvents := func(n int) []WSEvent {
		evts := make([]WSEvent, n)
		for i := range evts {
			evts[i] = WSEvent{Event: "E", Timestamp: time.Now()}
		}
		return evts
	}

	t.Run("under limit — unchanged", func(t *testing.T) {
		evts := makeEvents(3)
		trimEvents(&evts, 5)
		if len(evts) != 3 {
			t.Errorf("expected 3, got %d", len(evts))
		}
	})
	t.Run("at limit — unchanged", func(t *testing.T) {
		evts := makeEvents(5)
		trimEvents(&evts, 5)
		if len(evts) != 5 {
			t.Errorf("expected 5, got %d", len(evts))
		}
	})
	t.Run("over limit — trimmed to max", func(t *testing.T) {
		evts := makeEvents(8)
		trimEvents(&evts, 5)
		if len(evts) != 5 {
			t.Errorf("expected 5, got %d", len(evts))
		}
	})
	t.Run("keeps most recent", func(t *testing.T) {
		evts := []WSEvent{
			{Event: "old1"}, {Event: "old2"}, {Event: "keep1"},
			{Event: "keep2"}, {Event: "keep3"},
		}
		trimEvents(&evts, 3)
		if evts[0].Event != "keep1" || evts[2].Event != "keep3" {
			t.Errorf("expected last 3 events, got %v", evts)
		}
	})
	t.Run("new backing array after trim", func(t *testing.T) {
		original := makeEvents(10)
		ptr := &original[9] // pointer to last element before trim
		trimEvents(&original, 5)
		// After trim the slice should be a fresh copy, so ptr should not
		// be within the new backing array.
		if len(original) > 0 && ptr == &original[4] {
			t.Error("trimEvents should produce a new backing array")
		}
	})
}

// ---- filteredWorkspaces ----

func TestFilteredWorkspaces(t *testing.T) {
	workspaces := []WorkspaceInfo{
		{ID: "1", Name: "Echo Agent"},
		{ID: "2", Name: "Summarizer"},
		{ID: "3", Name: "echo bot"},
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		m := Model{workspaces: workspaces}
		got := m.filteredWorkspaces()
		if len(got) != 3 {
			t.Errorf("expected 3, got %d", len(got))
		}
	})
	t.Run("no match returns empty", func(t *testing.T) {
		m := Model{workspaces: workspaces, filter: "zzz"}
		got := m.filteredWorkspaces()
		if len(got) != 0 {
			t.Errorf("expected 0, got %d", len(got))
		}
	})
	t.Run("case-insensitive partial match", func(t *testing.T) {
		m := Model{workspaces: workspaces, filter: "echo"}
		got := m.filteredWorkspaces()
		if len(got) != 2 {
			t.Errorf("expected 2 matches for 'echo', got %d", len(got))
		}
	})
	t.Run("exact match", func(t *testing.T) {
		m := Model{workspaces: workspaces, filter: "Summarizer"}
		got := m.filteredWorkspaces()
		if len(got) != 1 || got[0].ID != "2" {
			t.Errorf("expected only Summarizer, got %v", got)
		}
	})
}

// ---- applyEvent ----

func makeModel() Model {
	return Model{
		workspaces: []WorkspaceInfo{
			{ID: "ws-1", Name: "Alpha", Status: "online"},
			{ID: "ws-2", Name: "Beta", Status: "provisioning"},
		},
		eventIDs: make(map[string]struct{}),
	}
}

func findWorkspace(m Model, id string) *WorkspaceInfo {
	for i := range m.workspaces {
		if m.workspaces[i].ID == id {
			return &m.workspaces[i]
		}
	}
	return nil
}

func TestApplyEvent_Provisioning(t *testing.T) {
	m := makeModel()
	payload, _ := json.Marshal(map[string]any{"name": "Gamma", "tier": 1})

	t.Run("adds new workspace", func(t *testing.T) {
		applyEvent(&m, WSEvent{Event: "WORKSPACE_PROVISIONING", WorkspaceID: "ws-3", Payload: payload})
		ws := findWorkspace(m, "ws-3")
		if ws == nil {
			t.Fatal("ws-3 not found after WORKSPACE_PROVISIONING")
		}
		if ws.Status != "provisioning" || ws.Name != "Gamma" {
			t.Errorf("unexpected workspace: %+v", ws)
		}
	})
	t.Run("idempotent for existing workspace", func(t *testing.T) {
		before := len(m.workspaces)
		applyEvent(&m, WSEvent{Event: "WORKSPACE_PROVISIONING", WorkspaceID: "ws-3", Payload: payload})
		if len(m.workspaces) != before {
			t.Errorf("duplicate workspace added: expected %d, got %d", before, len(m.workspaces))
		}
	})
}

func TestApplyEvent_Online(t *testing.T) {
	m := makeModel()

	t.Run("updates existing workspace status", func(t *testing.T) {
		applyEvent(&m, WSEvent{Event: "WORKSPACE_ONLINE", WorkspaceID: "ws-2"})
		ws := findWorkspace(m, "ws-2")
		if ws == nil || ws.Status != "online" {
			t.Errorf("expected online status, got %v", ws)
		}
	})
	t.Run("adds unknown workspace", func(t *testing.T) {
		applyEvent(&m, WSEvent{Event: "WORKSPACE_ONLINE", WorkspaceID: "ws-99"})
		ws := findWorkspace(m, "ws-99")
		if ws == nil || ws.Status != "online" {
			t.Errorf("expected ws-99 to be added with online status")
		}
	})
}

func TestApplyEvent_Degraded(t *testing.T) {
	m := makeModel()
	payload, _ := json.Marshal(map[string]any{"error_rate": 0.75, "sample_error": "timeout"})

	applyEvent(&m, WSEvent{Event: "WORKSPACE_DEGRADED", WorkspaceID: "ws-1", Payload: payload})
	ws := findWorkspace(m, "ws-1")
	if ws == nil {
		t.Fatal("ws-1 not found")
	}
	if ws.Status != "degraded" {
		t.Errorf("expected degraded, got %q", ws.Status)
	}
	if ws.LastErrorRate != 0.75 {
		t.Errorf("expected error_rate 0.75, got %v", ws.LastErrorRate)
	}
	if ws.LastSampleError != "timeout" {
		t.Errorf("expected sample_error 'timeout', got %q", ws.LastSampleError)
	}
}

func TestApplyEvent_Offline(t *testing.T) {
	m := makeModel()
	applyEvent(&m, WSEvent{Event: "WORKSPACE_OFFLINE", WorkspaceID: "ws-1"})
	ws := findWorkspace(m, "ws-1")
	if ws == nil || ws.Status != "offline" {
		t.Errorf("expected offline status, got %v", ws)
	}
}

func TestApplyEvent_Removed(t *testing.T) {
	m := makeModel()
	before := len(m.workspaces)
	applyEvent(&m, WSEvent{Event: "WORKSPACE_REMOVED", WorkspaceID: "ws-1"})
	if len(m.workspaces) != before-1 {
		t.Errorf("expected %d workspaces after remove, got %d", before-1, len(m.workspaces))
	}
	if findWorkspace(m, "ws-1") != nil {
		t.Error("ws-1 still present after WORKSPACE_REMOVED")
	}
}

func TestApplyEvent_AgentCardUpdated(t *testing.T) {
	m := makeModel()
	card := json.RawMessage(`{"name":"Alpha","skills":[{"id":"echo","name":"Echo"}]}`)
	payload, _ := json.Marshal(map[string]json.RawMessage{"agent_card": card})

	applyEvent(&m, WSEvent{Event: "AGENT_CARD_UPDATED", WorkspaceID: "ws-1", Payload: payload})
	ws := findWorkspace(m, "ws-1")
	if ws == nil {
		t.Fatal("ws-1 not found")
	}
	parsed := ParseAgentCard(ws.AgentCard)
	if parsed == nil || parsed.Name != "Alpha" {
		t.Errorf("unexpected agent card: %v", parsed)
	}
	if len(parsed.Skills) != 1 || parsed.Skills[0].ID != "echo" {
		t.Errorf("unexpected skills: %v", parsed.Skills)
	}
}

// ---- ParseAgentCard ----

func TestParseAgentCard(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		if ParseAgentCard(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})
	t.Run("empty input", func(t *testing.T) {
		if ParseAgentCard(json.RawMessage{}) != nil {
			t.Error("expected nil for empty input")
		}
	})
	t.Run("JSON null", func(t *testing.T) {
		if ParseAgentCard(json.RawMessage("null")) != nil {
			t.Error("expected nil for JSON null")
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		if ParseAgentCard(json.RawMessage("{bad}")) != nil {
			t.Error("expected nil for malformed JSON")
		}
	})
	t.Run("valid with skills", func(t *testing.T) {
		raw := json.RawMessage(`{"name":"Echo","skills":[{"id":"s1","name":"Skill One"}]}`)
		card := ParseAgentCard(raw)
		if card == nil {
			t.Fatal("expected non-nil card")
		}
		if card.Name != "Echo" {
			t.Errorf("expected name %q, got %q", "Echo", card.Name)
		}
		if len(card.Skills) != 1 || card.Skills[0].ID != "s1" {
			t.Errorf("unexpected skills: %v", card.Skills)
		}
	})
	t.Run("valid empty skills", func(t *testing.T) {
		raw := json.RawMessage(`{"name":"Bare"}`)
		card := ParseAgentCard(raw)
		if card == nil || card.Name != "Bare" {
			t.Errorf("unexpected card: %v", card)
		}
		if len(card.Skills) != 0 {
			t.Errorf("expected no skills, got %v", card.Skills)
		}
	})
}

// ---- deleteURL ----

func TestDeleteURL(t *testing.T) {
	cases := []struct {
		base string
		id   string
		want string
	}{
		{"http://localhost:8080", "ws-abc", "http://localhost:8080/workspaces/ws-abc"},
		{"http://localhost:8080/", "ws-abc", "http://localhost:8080/workspaces/ws-abc"},
		{"http://host/api/v1", "x", "http://host/api/v1/workspaces/x"},
	}
	for _, c := range cases {
		got, err := deleteURL(c.base, c.id)
		if err != nil {
			t.Errorf("deleteURL(%q, %q) error: %v", c.base, c.id, err)
			continue
		}
		if got != c.want {
			t.Errorf("deleteURL(%q, %q) = %q, want %q", c.base, c.id, got, c.want)
		}
	}
}

// ---- sortWorkspaces ----

func TestSortWorkspaces(t *testing.T) {
	ws := []WorkspaceInfo{
		{ID: "3", Name: "Zebra"},
		{ID: "1", Name: "Alpha"},
		{ID: "2", Name: "Mango"},
	}
	sortWorkspaces(ws)
	names := make([]string, len(ws))
	for i, w := range ws {
		names[i] = w.Name
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("sortWorkspaces did not sort by name: %v", names)
	}
	if ws[0].Name != "Alpha" || ws[2].Name != "Zebra" {
		t.Errorf("wrong order: %v", names)
	}
}

// ---- statusCounts ----

func TestStatusCounts(t *testing.T) {
	m := Model{workspaces: []WorkspaceInfo{
		{Status: "online"},
		{Status: "online"},
		{Status: "degraded"},
		{Status: "offline"},
		{Status: "provisioning"},
		{Status: "unknown"}, // should not be counted in any bucket
	}}
	online, degraded, offline, prov := m.statusCounts()
	if online != 2 {
		t.Errorf("expected 2 online, got %d", online)
	}
	if degraded != 1 {
		t.Errorf("expected 1 degraded, got %d", degraded)
	}
	if offline != 1 {
		t.Errorf("expected 1 offline, got %d", offline)
	}
	if prov != 1 {
		t.Errorf("expected 1 provisioning, got %d", prov)
	}
}

// ---- eventLines ----

func TestEventLines(t *testing.T) {
	makeEvts := func(labels ...string) []WSEvent {
		evts := make([]WSEvent, len(labels))
		for i, l := range labels {
			evts[i] = WSEvent{Event: l, Timestamp: time.Now()}
		}
		return evts
	}

	t.Run("empty slice returns empty", func(t *testing.T) {
		if got := eventLines(nil, 5); len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
	t.Run("returns at most max lines", func(t *testing.T) {
		evts := makeEvts("a", "b", "c", "d", "e", "f")
		got := eventLines(evts, 3)
		if len(got) != 3 {
			t.Errorf("expected 3 lines, got %d", len(got))
		}
	})
	t.Run("returns all when fewer than max", func(t *testing.T) {
		evts := makeEvts("a", "b")
		got := eventLines(evts, 10)
		if len(got) != 2 {
			t.Errorf("expected 2 lines, got %d", len(got))
		}
	})
	t.Run("most recent event appears first", func(t *testing.T) {
		evts := makeEvts("oldest", "middle", "newest")
		got := eventLines(evts, 3)
		// reverse-chronological: newest first
		if len(got) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(got))
		}
		// Each line contains the event name; newest should appear in got[0]
		if !strings.Contains(got[0], "newest") {
			t.Errorf("expected newest event first, got %q", got[0])
		}
		if !strings.Contains(got[2], "oldest") {
			t.Errorf("expected oldest event last, got %q", got[2])
		}
	})
}

// ---- clampSelected ----

func TestClampSelected(t *testing.T) {
	workspaces := []WorkspaceInfo{
		{ID: "1", Name: "A"},
		{ID: "2", Name: "B"},
		{ID: "3", Name: "C"},
	}

	t.Run("within bounds — unchanged", func(t *testing.T) {
		m := Model{workspaces: workspaces, selected: 1}
		m.clampSelected()
		if m.selected != 1 {
			t.Errorf("expected 1, got %d", m.selected)
		}
	})
	t.Run("above max — clamped to last", func(t *testing.T) {
		m := Model{workspaces: workspaces, selected: 10}
		m.clampSelected()
		if m.selected != 2 {
			t.Errorf("expected 2, got %d", m.selected)
		}
	})
	t.Run("empty list — clamped to 0", func(t *testing.T) {
		m := Model{selected: 5}
		m.clampSelected()
		if m.selected != 0 {
			t.Errorf("expected 0, got %d", m.selected)
		}
	})
	t.Run("filter reduces list — clamped to filtered length", func(t *testing.T) {
		m := Model{workspaces: workspaces, filter: "A", selected: 2}
		m.clampSelected() // only "A" matches, so max valid index is 0
		if m.selected != 0 {
			t.Errorf("expected 0, got %d", m.selected)
		}
	})
}

// ---- NewModel ----

func TestNewModel(t *testing.T) {
	m := NewModel("http://localhost:8080")
	if m.baseURL != "http://localhost:8080" {
		t.Errorf("unexpected baseURL: %q", m.baseURL)
	}
	if m.client == nil {
		t.Error("expected non-nil client")
	}
	if m.eventIDs == nil {
		t.Error("expected non-nil eventIDs map")
	}
	if m.wsGen != 1 {
		t.Errorf("expected wsGen=1, got %d", m.wsGen)
	}
	if m.workspaces != nil {
		t.Errorf("expected nil workspaces slice, got %v", m.workspaces)
	}
}
