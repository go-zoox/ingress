package handler

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

func defaultShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/bash"
}

const (
	terminalReplayMaxBytes = 128 << 10
)

type terminalManager struct {
	mu          sync.Mutex
	sessions    map[string]*managedTerminalSession
	detachGrace time.Duration
}

func newTerminalManager(grace time.Duration) *terminalManager {
	return &terminalManager{
		sessions:    make(map[string]*managedTerminalSession),
		detachGrace: grace,
	}
}

var defaultTerminalManager = newTerminalManager(60 * time.Second)

type terminalConn interface {
	WriteBinaryMessage(msg []byte) error
	Close() error
}

type managedTerminalSession struct {
	id        string
	ptmx      *os.File
	cmd       *exec.Cmd
	mgr       *terminalManager
	mu        sync.Mutex
	closed    bool
	conn      terminalConn
	replay    []byte
	idleTimer *time.Timer
}

func (m *terminalManager) get(id string) *managedTerminalSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *terminalManager) open(requestedID string) (*managedTerminalSession, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if requestedID != "" {
		if sess := m.sessions[requestedID]; sess != nil && !sess.isClosed() {
			return sess, true, nil
		}
	}

	sess, err := m.createLocked()
	if err != nil {
		return nil, false, err
	}
	return sess, false, nil
}

func (m *terminalManager) createLocked() (*managedTerminalSession, error) {
	cmd := exec.Command(defaultShell())
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	sess := &managedTerminalSession{
		id:   uuid.NewString(),
		ptmx: ptmx,
		cmd:  cmd,
		mgr:  m,
	}
	m.sessions[sess.id] = sess
	sess.startPump()
	return sess, nil
}

func (s *managedTerminalSession) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *managedTerminalSession) bind(conn terminalConn) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	old := s.conn
	s.conn = conn
	replay := append([]byte(nil), s.replay...)
	s.mu.Unlock()

	if old != nil && old != conn {
		_ = old.Close()
	}
	if len(replay) > 0 {
		_ = conn.WriteBinaryMessage(replay)
	}
}

func (s *managedTerminalSession) detach(conn terminalConn) {
	s.mu.Lock()
	if s.closed || s.conn != conn {
		s.mu.Unlock()
		return
	}
	s.conn = nil
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	grace := s.mgr.detachGrace
	s.idleTimer = time.AfterFunc(grace, func() {
		s.mgr.destroy(s.id)
	})
	s.mu.Unlock()
}

func (m *terminalManager) destroy(id string) {
	m.mu.Lock()
	sess := m.sessions[id]
	if sess != nil {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if sess != nil {
		sess.close()
	}
}

func (s *managedTerminalSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	if s.ptmx != nil {
		_ = s.ptmx.Close()
		s.ptmx = nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

func (s *managedTerminalSession) write(message []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return nil
	}
	_, err := s.ptmx.Write(message)
	return err
}

func (s *managedTerminalSession) resize(rows, cols uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil || rows == 0 || cols == 0 {
		return
	}
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (s *managedTerminalSession) deliverOutput(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.replay = appendReplay(s.replay, data, terminalReplayMaxBytes)
	if s.conn != nil {
		_ = s.conn.WriteBinaryMessage(data)
	}
}

func appendReplay(buf, data []byte, max int) []byte {
	if len(data) >= max {
		return append([]byte(nil), data[len(data)-max:]...)
	}
	out := append(buf, data...)
	if len(out) > max {
		return append([]byte(nil), out[len(out)-max:]...)
	}
	return out
}

func (s *managedTerminalSession) startPump() {
	go func() {
		buf := make([]byte, 4096)
		for {
			s.mu.Lock()
			if s.closed || s.ptmx == nil {
				s.mu.Unlock()
				return
			}
			ptmx := s.ptmx
			s.mu.Unlock()

			n, err := ptmx.Read(buf)
			if n > 0 {
				s.deliverOutput(buf[:n])
			}
			if err != nil {
				s.mgr.destroy(s.id)
				return
			}
		}
	}()
}
