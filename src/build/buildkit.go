// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 01:14
// Original filename: src/build/buildkit.go

package build

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"dtools2/rest"
)

// BuilderVersionFromDaemon returns the daemon's recommended builder backend.
// Docker daemons may advertise this through the `Builder-Version` header on
// GET/HEAD /_ping. Values:
//
//	"1" = classic builder
//	"2" = BuildKit
//
// If the header is missing (older daemons), ok will be false.
//
// Note: this is a recommendation advertised by the daemon, not an absolute
// guarantee that a client must follow.
func BuilderVersionFromDaemon(ctx context.Context, client *rest.Client) (v string, ok bool, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Prefer HEAD, but be tolerant if a particular daemon doesn't implement it.
	v, ok, err = pingBuilderVersion(ctx, client, http.MethodHead)
	if err == nil {
		return v, ok, nil
	}

	// Fallback to GET.
	return pingBuilderVersion(ctx, client, http.MethodGet)
}

// DaemonRecommendsBuildKit is a convenience wrapper that returns true when the
// daemon advertises Builder-Version == "2".
func DaemonRecommendsBuildKit(ctx context.Context, client *rest.Client) (bool, error) {
	v, ok, err := BuilderVersionFromDaemon(ctx, client)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return strings.TrimSpace(v) == "2", nil
}

func pingBuilderVersion(ctx context.Context, client *rest.Client, method string) (v string, ok bool, err error) {
	if client == nil {
		return "", false, fmt.Errorf("nil rest client")
	}

	// /_ping is unversioned; some daemons accept it both with and without /vX.
	// Try with the current client version first (cheap), then retry unversioned.
	resp, err := client.Do(ctx, method, "/_ping", nil, nil, nil)
	if err == nil && resp != nil {
		status := resp.StatusCode
		bv := strings.TrimSpace(resp.Header.Get("Builder-Version"))
		_ = resp.Body.Close()
		if status < 400 && bv != "" {
			return bv, true, nil
		}
		// If status >= 400, or header missing, retry unversioned.
	}

	// Retry without a version prefix.
	orig := client.APIVersion()
	client.SetAPIVersion("")
	defer client.SetAPIVersion(orig)

	resp2, err2 := client.Do(ctx, method, "/_ping", nil, nil, nil)
	if err2 != nil {
		return "", false, err2
	}
	defer resp2.Body.Close()

	if resp2.StatusCode >= 400 {
		return "", false, fmt.Errorf("%s /_ping returned %s", method, resp2.Status)
	}

	bv := strings.TrimSpace(resp2.Header.Get("Builder-Version"))
	if bv == "" {
		return "", false, nil
	}
	return bv, true, nil
}
