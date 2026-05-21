package core

import (
	"os"
	"path/filepath"
	"testing"

	zcfg "github.com/go-zoox/zoox/config"
)

func TestLoggingNormalize_enableDefaultTransports(t *testing.T) {
	enabled := true
	l := &Logging{Enable: &enabled, Level: "warn"}
	l.applyFileDefaults()
	if len(l.Transports) != 1 {
		t.Fatalf("transports: got %d want 1", len(l.Transports))
	}
	if l.Transports[0].Path != DefaultAccessLogPath {
		t.Fatalf("access: got %q", l.Transports[0].Path)
	}
	if l.Transports[0].Levels["error"] != DefaultErrorLogPath {
		t.Fatalf("error: got %q", l.Transports[0].Levels["error"])
	}
}

func TestLoggingNormalize_disabledClearsTransports(t *testing.T) {
	disabled := false
	l := &Logging{
		Enable:     &disabled,
		Transports: DefaultFileTransport(),
	}
	l.applyFileDefaults()
	if len(l.Transports) != 0 {
		t.Fatalf("transports: got %d want 0", len(l.Transports))
	}
}

func TestLoggingNormalize_createsLogDir(t *testing.T) {
	root := t.TempDir()
	access := filepath.Join(root, "nested", "access.log")
	errorLog := filepath.Join(root, "nested", "error.log")
	l := &Logging{
		Transports: []zcfg.Transport{{
			Type: "file",
			Path: access,
			Levels: map[string]string{"error": errorLog},
		}},
	}
	if err := l.Normalize(); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "nested")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("dir %q: err=%v", dir, err)
	}
}
