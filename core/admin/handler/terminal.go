package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	wsconn "github.com/go-zoox/websocket/conn"
	"github.com/go-zoox/zoox"
)

type terminalControlMsg struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

type terminalSessionMsg struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Reattach bool   `json:"reattach"`
}

func sessionIDFromConn(conn wsconn.Conn) string {
	if conn == nil || conn.Request() == nil || conn.Request().URL == nil {
		return ""
	}
	return strings.TrimSpace(conn.Request().URL.Query().Get("session"))
}

func writeTerminalSessionMeta(conn wsconn.Conn, id string, reattach bool) error {
	payload, err := json.Marshal(terminalSessionMsg{
		Type:     "session",
		ID:       id,
		Reattach: reattach,
	})
	if err != nil {
		return err
	}
	return conn.WriteTextMessage(payload)
}

func handleTerminalMessage(conn wsconn.Conn, typ int, message []byte) error {
	raw := conn.Get("terminal_session_id")
	id, _ := raw.(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	sess := defaultTerminalManager.get(id)
	if sess == nil {
		return nil
	}

	if typ == wsconn.TextMessage {
		var msg terminalControlMsg
		if json.Unmarshal(message, &msg) == nil && msg.Type != "" {
			switch msg.Type {
			case "resize":
				if msg.Rows > 0 && msg.Cols > 0 {
					sess.resize(msg.Rows, msg.Cols)
				}
				return nil
			case "close":
				defaultTerminalManager.destroy(id)
				return nil
			}
		}
	}

	return sess.write(message)
}

func detachTerminalSession(conn wsconn.Conn) {
	raw := conn.Get("terminal_session_id")
	id, _ := raw.(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	if sess := defaultTerminalManager.get(id); sess != nil {
		sess.detach(conn)
	}
}

// MountTerminal registers GET /api/v1/terminal/ws (WebSocket upgrade).
func MountTerminal(g *zoox.RouterGroup) error {
	server, err := g.WebSocket("/terminal/ws")
	if err != nil {
		return err
	}

	server.OnConnect(func(conn wsconn.Conn) error {
		requestedID := sessionIDFromConn(conn)
		sess, reattach, err := defaultTerminalManager.open(requestedID)
		if err != nil {
			_ = conn.WriteTextMessage([]byte("\r\n\x1b[31mfailed to start shell: " + err.Error() + "\x1b[0m\r\n"))
			return err
		}

		if err := writeTerminalSessionMeta(conn, sess.id, reattach); err != nil {
			return err
		}

		sess.bind(conn)
		if !sess.isClosed() {
			_ = conn.Set("terminal_session_id", sess.id)
			return nil
		}
		return fmt.Errorf("terminal session %s is closed", sess.id)
	})

	server.OnMessage(func(conn wsconn.Conn, typ int, message []byte) error {
		return handleTerminalMessage(conn, typ, message)
	})

	server.OnClose(func(conn wsconn.Conn, _ int, _ string) error {
		detachTerminalSession(conn)
		return nil
	})

	return nil
}
