package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// Blob fetches indexer blobs (current + previous) for a height.
func (c *HTTPClient) Blob(ctx context.Context, height uint64) (*fsm.IndexerBlobs, error) {
	var resp fsm.IndexerBlobs
	if err := c.doProtobuf(ctx, http.MethodPost, indexerBlobsPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// doProtobuf makes an HTTP request with JSON payload and binary protobuf response.
func (c *HTTPClient) doProtobuf(ctx context.Context, method, path string, payload any, out any) error {
	if len(c.endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	var lastErr error
	for i := 0; i < len(c.endpoints); i++ {
		ep := c.endpoints[i%len(c.endpoints)]
		if c.isOpen(ep) {
			continue
		}

		c.acquire()

		var body *bytes.Reader
		if payload != nil {
			b, mErr := json.Marshal(payload)
			if mErr != nil {
				return mErr
			}
			body = bytes.NewReader(b)
		} else {
			body = bytes.NewReader(nil)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, ep+path, body)
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			c.noteFailure(ep)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server %d", resp.StatusCode)
			c.noteFailure(ep)
			resp.Body.Close()
			continue
		}
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}

		if out != nil {
			// Read raw body for debugging
			rawBody, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				lastErr = err
				continue
			}

			slog.Debug("rpc", "path", path, "len", len(rawBody))

			if err := lib.Unmarshal(rawBody, out); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("protobuf unmarshal: %w (body len: %d)", err, len(rawBody))
				continue
			}
		}

		resp.Body.Close()
		return nil
	}

	return lastErr
}
