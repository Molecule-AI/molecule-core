package main

import (
	"encoding/json"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

// WsEventMsg is sent into the bubbletea loop when a WebSocket event arrives.
type WsEventMsg struct {
	Event WSEvent
	Gen   int // connection generation that produced this event
}

// WsErrorMsg is sent when the WebSocket connection encounters an error.
type WsErrorMsg struct {
	Err error
	Gen int // connection generation that errored
}

// WsConnectedMsg is sent when the WebSocket connection is established.
type WsConnectedMsg struct {
	Conn *websocket.Conn
}

// wsReconnectTickMsg signals it's time to attempt a WebSocket reconnection.
type wsReconnectTickMsg struct{}

// connectWS builds a WebSocket URL from an HTTP base URL and dials.
func connectWS(baseURL string) (*websocket.Conn, error) {
	var wsURL string
	switch {
	case strings.HasPrefix(baseURL, "https://"):
		wsURL = "wss://" + baseURL[len("https://"):]
	case strings.HasPrefix(baseURL, "http://"):
		wsURL = "ws://" + baseURL[len("http://"):]
	default:
		wsURL = baseURL
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}
	// Preserve any existing base path (e.g. /api/v1 → /api/v1/ws).
	u.Path = path.Join(u.Path, "ws")

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// connectWSCmd returns a tea.Cmd that attempts to connect to the WebSocket.
func connectWSCmd(baseURL string) tea.Cmd {
	return func() tea.Msg {
		conn, err := connectWS(baseURL)
		if err != nil {
			return WsErrorMsg{Err: err}
		}
		return WsConnectedMsg{Conn: conn}
	}
}

// listenWS returns a tea.Cmd that blocks on a single WebSocket read.
// After receiving a message, it returns a WsEventMsg. The bubbletea loop
// should call listenWS again to continue reading.
func listenWS(conn *websocket.Conn, gen int) tea.Cmd {
	return func() tea.Msg {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return WsErrorMsg{Err: err, Gen: gen}
		}

		var evt WSEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			log.Printf("ws unmarshal error: %v", err)
			return WsEventMsg{Event: WSEvent{Event: "PARSE_ERROR", Timestamp: time.Now()}, Gen: gen}
		}

		return WsEventMsg{Event: evt, Gen: gen}
	}
}

// reconnectWSCmd waits briefly then signals that a reconnect should be attempted.
func reconnectWSCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return wsReconnectTickMsg{}
	})
}
