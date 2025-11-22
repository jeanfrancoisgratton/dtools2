// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:12
// Original filename: src/rest/apiversion.go

// dtools2
// rest/version.go

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// versionInfo matches the JSON returned by /version.
type versionInfo struct {
	ApiVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	Version       string `json:"Version"`
}

// NegotiateAPIVersion queries /version without a version prefix and returns ApiVersion.
// Caller is expected to call client.SetAPIVersion() with the returned value.
func NegotiateAPIVersion(ctx context.Context, client *Client) (string, error) {
	resp, err := client.Do(ctx, http.MethodGet, "/version", nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("GET /version failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET /version returned %s", resp.Status)
	}

	var info versionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("decoding /version response failed: %w", err)
	}

	if info.ApiVersion == "" {
		return "", fmt.Errorf("daemon did not return ApiVersion")
	}

	return info.ApiVersion, nil
}
