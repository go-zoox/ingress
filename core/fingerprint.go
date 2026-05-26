package core

import (
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns a short stable fingerprint of YAML/config bytes (first 8 bytes of SHA-256, hex).
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}
