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

func TestLoggingPrepare_adminDefaultsWhenLoggingUnset(t *testing.T) {
	root := t.TempDir()
	configFile := filepath.Join(root, "ingress.yaml")
	if err := os.WriteFile(configFile, []byte("version: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := &Logging{}
	admin := Admin{Enabled: true}
	if err := l.Prepare(admin, configFile); err != nil {
		t.Fatal(err)
	}
	access, errorLog := l.FileLogPaths()
	wantAccess := filepath.Join(root, "access.log")
	wantError := filepath.Join(root, "error.log")
	if access != wantAccess {
		t.Fatalf("access: got %q want %q", access, wantAccess)
	}
	if errorLog != wantError {
		t.Fatalf("error: got %q want %q", errorLog, wantError)
	}
	if l.Enable == nil || !*l.Enable {
		t.Fatal("expected logging enabled by default when admin is on")
	}
}

func TestLoggingPrepare_adminKeepsLoggingTransports(t *testing.T) {
	root := t.TempDir()
	disabled := false
	loggingAccess := filepath.Join(root, "access.log")
	loggingError := filepath.Join(root, "error.log")
	l := &Logging{
		Enable: &disabled,
		Transports: []zcfg.Transport{{
			Type: "file",
			Path: loggingAccess,
			Levels: map[string]string{"error": loggingError},
		}},
	}
	admin := Admin{Enabled: true}
	if err := l.Prepare(admin, ""); err != nil {
		t.Fatal(err)
	}
	access, errorLog := l.FileLogPaths()
	if access != loggingAccess || errorLog != loggingError {
		t.Fatalf("logging should win: access=%q error=%q", access, errorLog)
	}
	if l.Enable == nil || *l.Enable {
		t.Fatal("expected explicit logging.enable=false to remain")
	}
}

func TestLoggingPrepare_adminRespectsLoggingDisabled(t *testing.T) {
	disabled := false
	l := &Logging{Enable: &disabled}
	admin := Admin{Enabled: true}
	if err := l.Prepare(admin, ""); err != nil {
		t.Fatal(err)
	}
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
