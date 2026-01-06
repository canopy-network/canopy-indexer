package transform

import "encoding/hex"

// BytesToHex converts a byte slice to a hex string.
// Returns empty string if byte slice is empty.
func BytesToHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hex.EncodeToString(b)
}

// bytesToHex is an internal alias for BytesToHex.
func bytesToHex(b []byte) string {
	return BytesToHex(b)
}
