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
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"dtools2/rest"

	"github.com/moby/buildkit/session"
)

// When doing an h2c upgrade, HTTP/1.1 requires the HTTP2-Settings header.
// This value encodes a minimal SETTINGS payload (ENABLE_PUSH=0).
// It is base64url (raw, no padding) of: 00 02 00 00 00 00
const defaultHTTP2Settings = "AAIAAAAA"

func (c *bufferedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

func newBuildkitSession(ctx context.Context, client *rest.Client) (*buildkitSession, error) {
	sharedKey, err := randomSharedKey(32)
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

		// Meta comes from buildkit/session and may already include the proper
		// h2c upgrade headers. Do not overwrite; merge.
		for k, vv := range meta {
			for _, v := range vv {
				h.Add(k, v)
			}
		}

		// Docker's BuildKit backend requires the session to be registered under
		// the same UUID that will be referenced by the /build request.
		if h.Get(dockerSessionHeaderID) == "" {
			h.Set(dockerSessionHeaderID, bs.ID)
		}
		if h.Get(dockerSessionHeaderSharedKey) == "" {
			h.Set(dockerSessionHeaderSharedKey, bs.SharedKey)
		}
		if h.Get(dockerSessionHeaderName) == "" {
			h.Set(dockerSessionHeaderName, "dtools2")
		}

		// Ensure the request performs an h2c upgrade. Some meta sets this, but
		// do not assume it.
		up := strings.TrimSpace(h.Get("Upgrade"))
		if up == "" {
			if strings.TrimSpace(proto) != "" {
				up = proto
			} else {
				up = "h2c"
			}
			h.Set("Upgrade", up)
		}

		// For h2c upgrades, HTTP2-Settings is required by RFC 7540.
		// If absent, provide a minimal, known-good value.
		if strings.EqualFold(up, "h2c") {
			if strings.TrimSpace(h.Get("HTTP2-Settings")) == "" {
				h.Set("HTTP2-Settings", defaultHTTP2Settings)
			}
			connHdr := strings.TrimSpace(h.Get("Connection"))
			if connHdr == "" {
				h.Set("Connection", "Upgrade, HTTP2-Settings")
			} else {
				// Ensure both tokens are present.
				if !headerTokenContains(connHdr, "upgrade") {
					connHdr = connHdr + ", Upgrade"
				}
				if !headerTokenContains(connHdr, "http2-settings") {
					connHdr = connHdr + ", HTTP2-Settings"
				}
				h.Set("Connection", connHdr)
			}
		} else {
			// Non-h2c: at least keep Upgrade semantics.
			if strings.TrimSpace(h.Get("Connection")) == "" {
				h.Set("Connection", "Upgrade")
			}
		}

		hc, err := hijackSession(ctx, client, h)
		if err != nil {
			select {
			case bs.ready <- fmt.Errorf("failed to create buildkit session: %w", err):
			default:
			}
			return nil, err
		}

		// BuildKit sessions *must* be an upgrade (101). If we accept a 200 here,
		// the build will later fail with "no active sessions" because nothing was
		// registered.
		if hc.Code != http.StatusSwitchingProtocols {
			msg, _ := readSmall(hc.Reader, 8*1024)
			_ = hc.Conn.Close()
			err := fmt.Errorf("/session did not upgrade (HTTP %d): %s", hc.Code, strings.TrimSpace(msg))
			select {
			case bs.ready <- err:
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

func hijackSession(ctx context.Context, client *rest.Client, headers http.Header) (*rest.HijackedConn, error) {
	if client == nil {
		return nil, errors.New("nil rest client")
	}

	// Try versioned first.
	hc, err := client.Hijack(ctx, http.MethodPost, "/session", nil, headers, nil, true)
	if err == nil {
		return hc, nil
	}

	// Retry unversioned (some daemons accept /session only without /vX).
	orig := client.APIVersion()
	client.SetAPIVersion("")
	defer client.SetAPIVersion(orig)

	return client.Hijack(ctx, http.MethodPost, "/session", nil, headers, nil, true)
}

func headerTokenContains(v, tokenLower string) bool {
	for _, part := range strings.Split(v, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == tokenLower {
			return true
		}
	}
	return false
}

func readSmall(r *bufio.Reader, n int64) (string, error) {
	if r == nil {
		return "", nil
	}
	b, err := r.Peek(int(n))
	if err == nil {
		return string(b), nil
	}
	// If Peek fails (e.g. EOF), return whatever we got.
	if len(b) > 0 {
		return string(b), nil
	}
	return "", err
}

func randomSharedKey(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	// Docker expects the shared key as base64url (raw, no padding) on the wire.
	return base64.RawURLEncoding.EncodeToString(b), nil
}
