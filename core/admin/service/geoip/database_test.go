package geoip

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDatabaseCheck_empty(t *testing.T) {
	ok, reason, msg := DatabaseCheck("")
	if ok || reason != ReasonUnset || msg != "" {
		t.Fatalf("ok=%v reason=%q msg=%q", ok, reason, msg)
	}
}

func TestDatabaseCheck_notFound(t *testing.T) {
	ok, reason, _ := DatabaseCheck(filepath.Join(t.TempDir(), "missing.mmdb"))
	if ok || reason != ReasonNotFound {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestDatabaseCheck_directory(t *testing.T) {
	dir := t.TempDir()
	ok, reason, _ := DatabaseCheck(dir)
	if ok || reason != ReasonInvalid {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestDatabaseCheck_readableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GeoLite2-City.mmdb")
	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, reason, msg := DatabaseCheck(path)
	if !ok || reason != "" || msg != "" {
		t.Fatalf("ok=%v reason=%q msg=%q", ok, reason, msg)
	}
}

func TestDatabaseCheck_notReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission model")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "GeoLite2-City.mmdb")
	if err := os.WriteFile(path, []byte("fake"), 0o000); err != nil {
		t.Fatal(err)
	}
	ok, reason, _ := DatabaseCheck(path)
	if ok || reason != ReasonPermissionDenied {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestInit_missingDatabaseDoesNotEnable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.mmdb")
	s, err := Init(Config{Database: path})
	if err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if st.Enabled || st.Loaded || st.Reason != ReasonNotFound {
		t.Fatalf("status=%+v", st)
	}
	if s.reader != nil {
		t.Fatal("expected no reader")
	}
}

func TestInit_unreadableDatabaseDoesNotEnable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission model")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "GeoLite2-City.mmdb")
	if err := os.WriteFile(path, []byte("fake"), 0o000); err != nil {
		t.Fatal(err)
	}
	s, err := Init(Config{Database: path})
	if err != nil {
		t.Fatal(err)
	}
	st := s.Status()
	if st.Enabled || st.Loaded || st.Reason != ReasonPermissionDenied {
		t.Fatalf("status=%+v", st)
	}
}
