package utils

import "io"

// DrainAndClose closes the given ReadCloser.
func DrainAndClose(rc io.ReadCloser) error {
	if rc == nil {
		return nil
	}
	// Drain to let the transport reuse the connection.
	_, _ = io.Copy(io.Discard, rc)
	return rc.Close()
}
