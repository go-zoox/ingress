package handler

import (
	"bytes"
	"testing"
	"time"
)

type fakeTerminalConn struct {
	writes [][]byte
}

func (f *fakeTerminalConn) WriteBinaryMessage(msg []byte) error {
	f.writes = append(f.writes, append([]byte(nil), msg...))
	return nil
}

func (f *fakeTerminalConn) Close() error { return nil }

func TestAppendReplay_CapsSize(t *testing.T) {
	t.Parallel()
	buf := appendReplay(nil, bytes.Repeat([]byte("a"), 100), 50)
	if len(buf) != 50 {
		t.Fatalf("len=%d want 50", len(buf))
	}
	buf = appendReplay(buf, bytes.Repeat([]byte("b"), 100), 50)
	if len(buf) != 50 || buf[0] != 'b' {
		t.Fatalf("expected trailing b fill, got %q", buf)
	}
}

func TestTerminalManager_DetachAndReattachWithinGrace(t *testing.T) {
	mgr := newTerminalManager(200 * time.Millisecond)
	sess, reattach, err := mgr.open("")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if reattach {
		t.Fatal("expected new session")
	}

	conn1 := &fakeTerminalConn{}
	sess.bind(conn1)
	sess.deliverOutput([]byte("hello"))

	conn2 := &fakeTerminalConn{}
	sess.detach(conn1)
	if len(conn1.writes) != 1 {
		t.Fatalf("conn1 writes=%d", len(conn1.writes))
	}

	sess.deliverOutput([]byte("while-detached"))
	sess.bind(conn2)

	if len(conn2.writes) != 1 || string(conn2.writes[0]) != "hellowhile-detached" {
		t.Fatalf("replay=%q", conn2.writes)
	}

	got, reattach, err := mgr.open(sess.id)
	if err != nil || !reattach || got.id != sess.id {
		t.Fatalf("reattach got id=%q reattach=%v err=%v", got.id, reattach, err)
	}

	sess.close()
	mgr.destroy(sess.id)
}

func TestTerminalManager_DestroyAfterGrace(t *testing.T) {
	mgr := newTerminalManager(30 * time.Millisecond)
	sess, _, err := mgr.open("")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	id := sess.id

	conn := &fakeTerminalConn{}
	sess.bind(conn)
	sess.detach(conn)

	time.Sleep(80 * time.Millisecond)

	if mgr.get(id) != nil {
		t.Fatal("session should be destroyed after grace")
	}
}
