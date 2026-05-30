package handler

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/creack/pty"
	wsconn "github.com/go-zoox/websocket/conn"
	"github.com/go-zoox/zoox"
)

type terminalSession struct {
	ptmx   *os.File
	cmd    *exec.Cmd
	mu     sync.Mutex
	closed bool
}

type terminalResizeMsg struct {
	Type string `json:"type"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func defaultShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/bash"
}

func (s *terminalSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.ptmx != nil {
		_ = s.ptmx.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

func (s *terminalSession) write(message []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return nil
	}
	_, err := s.ptmx.Write(message)
	return err
}

func (s *terminalSession) resize(rows, cols uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil || rows == 0 || cols == 0 {
		return
	}
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func startTerminalSession(conn wsconn.Conn) (*terminalSession, error) {
	cmd := exec.Command(defaultShell())
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	sess := &terminalSession{ptmx: ptmx, cmd: cmd}
	conn.Set("terminal_session", sess)

	go pumpPTYOutput(conn, sess)
	return sess, nil
}

func pumpPTYOutput(conn wsconn.Conn, sess *terminalSession) {
	buf := make([]byte, 4096)
	for {
		sess.mu.Lock()
		if sess.closed || sess.ptmx == nil {
			sess.mu.Unlock()
			return
		}
		ptmx := sess.ptmx
		sess.mu.Unlock()

		n, err := ptmx.Read(buf)
		if n > 0 {
			_ = conn.WriteBinaryMessage(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func handleTerminalMessage(conn wsconn.Conn, typ int, message []byte) error {
	raw := conn.Get("terminal_session")
	sess, _ := raw.(*terminalSession)
	if sess == nil {
		return nil
	}

	if typ == wsconn.TextMessage {
		var msg terminalResizeMsg
		if json.Unmarshal(message, &msg) == nil && msg.Type == "resize" {
			sess.resize(msg.Rows, msg.Cols)
			return nil
		}
	}

	return sess.write(message)
}

func closeTerminalSession(conn wsconn.Conn) {
	raw := conn.Get("terminal_session")
	sess, _ := raw.(*terminalSession)
	if sess != nil {
		sess.close()
	}
}

// MountTerminal registers GET /api/v1/terminal/ws (WebSocket upgrade).
func MountTerminal(g *zoox.RouterGroup) error {
	server, err := g.WebSocket("/terminal/ws")
	if err != nil {
		return err
	}

	server.OnConnect(func(conn wsconn.Conn) error {
		if _, err := startTerminalSession(conn); err != nil {
			_ = conn.WriteTextMessage([]byte("\r\n\x1b[31mfailed to start shell: " + err.Error() + "\x1b[0m\r\n"))
			return err
		}
		return nil
	})

	server.OnMessage(func(conn wsconn.Conn, typ int, message []byte) error {
		return handleTerminalMessage(conn, typ, message)
	})

	server.OnClose(func(conn wsconn.Conn, _ int, _ string) error {
		closeTerminalSession(conn)
		return nil
	})

	return nil
}
