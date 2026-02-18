// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/18
// Original filename: src/build/buildkit_session.go
//
// This file implements the minimal "BuildKit session" plumbing needed when the
// daemon uses the BuildKit backend for /build (Docker Engine 23+ default).
//
// The Docker daemon expects the client to open a long-lived upgraded connection
// to POST /session, and to reference that session on the build request through
// X-Docker-Expose-Session-* headers. Without that, builds can fail early with
// errors like "no active sessions".

package run_build

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"

	"dtools2/rest"

	"github.com/moby/buildkit/session"
)

func (c *bufferedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

func newBuildkitSession(ctx context.Context, client *rest.Client) (*buildkitSession, error) {
	sharedKey, err := randomHex(32)
	if err != nil {
		return nil, err
	}

	s, err := session.NewSession(ctx, sharedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init buildkit session: %w", err)
	}

	bs := &buildkitSession{
		ID:        s.ID(),
		SharedKey: sharedKey,
		sess:      s,
		ready:     make(chan error, 1),
	}

	go func() {
		_ = s.Run(ctx, bs.dialer(client))
	}()

	// Wait for the first dial attempt to complete so we don't start the build
	// before the daemon has an active session entry.
	if err := <-bs.ready; err != nil {
		_ = s.Close()
		return nil, err
	}

	return bs, nil
}

func (bs *buildkitSession) Close() {
	if bs == nil || bs.sess == nil {
		return
	}
	_ = bs.sess.Close()
}

func (bs *buildkitSession) dialer(client *rest.Client) session.Dialer {
	return func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
		h := http.Header{}
		h.Set("Upgrade", "h2c")
		h.Set("Connection", "Upgrade")

		for k, vv := range meta {
			for _, v := range vv {
				h.Add(k, v)
			}
		}

		hc, err := client.Hijack(ctx, http.MethodPost, "/session", nil, h, nil, true)
		if err != nil {
			select {
			case bs.ready <- fmt.Errorf("failed to create buildkit session: %w", err):
			default:
			}
			return nil, err
		}

		br := hc.Reader
		if br == nil {
			br = bufio.NewReader(hc.Conn)
		}

		select {
		case bs.ready <- nil:
		default:
		}

		return &bufferedConn{Conn: hc.Conn, r: br}, nil
	}
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
