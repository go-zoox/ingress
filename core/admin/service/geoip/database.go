package geoip

import (
	"fmt"
	"os"
	"strings"
)

const (
	ReasonUnset            = "unset"
	ReasonNotFound         = "not_found"
	ReasonPermissionDenied = "permission_denied"
	ReasonInvalid          = "invalid"
	ReasonOpenFailed       = "open_failed"
)

// DatabaseCheck validates that path exists and is readable before opening MaxMind.
// ok=false means GeoIP must not be used; reason and message describe why.
func DatabaseCheck(path string) (ok bool, reason, message string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, ReasonUnset, ""
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, ReasonNotFound, fmt.Sprintf("geoip database not found: %s", path)
		}
		if os.IsPermission(err) {
			return false, ReasonPermissionDenied, fmt.Sprintf("geoip database not accessible (permission denied): %s", path)
		}
		return false, ReasonOpenFailed, fmt.Sprintf("geoip database unavailable: %s (%v)", path, err)
	}
	if fi.IsDir() {
		return false, ReasonInvalid, fmt.Sprintf("geoip database path is a directory: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsPermission(err) {
			return false, ReasonPermissionDenied, fmt.Sprintf("geoip database not readable (permission denied): %s", path)
		}
		return false, ReasonOpenFailed, fmt.Sprintf("geoip database not readable: %s (%v)", path, err)
	}
	_ = f.Close()
	return true, "", ""
}

// DatabaseExists reports whether the path stat succeeds (file may still be unreadable).
func DatabaseExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi != nil && !fi.IsDir()
}

func openFailureReason(err error) string {
	if err == nil {
		return ReasonOpenFailed
	}
	if os.IsPermission(err) {
		return ReasonPermissionDenied
	}
	return ReasonOpenFailed
}

func openFailureMessage(path string, err error) string {
	if os.IsPermission(err) {
		return fmt.Sprintf("geoip open failed (permission denied): %s", path)
	}
	return fmt.Sprintf("geoip open failed: %s (%v)", path, err)
}
