// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/19 22:52
// Original filename: src/auth/types.go

package auth

import "time"

// WhoAmIResult describes the stored auth for a registry.
type WhoAmIResult struct {
	Registry     string // normalized registry key used in config.json
	Mode         string // "basic", "token", "missing", "helper", "unknown"
	Username     string // for Mode == "basic"
	TokenPreview string // for Mode == "token" (non-sensitive short prefix)
}

// No explicit auths entry: see if a helper is configured
// (we can't query the helper here; just surface that one is likely in use).
type helperCfg struct {
	CredsStore  string            `json:"credsStore"`
	CredHelpers map[string]string `json:"credHelpers"`
}

// Mode is the authentication mode that succeeded.
type Mode string

const (
	ModeNone   Mode = "none"   // /v2/ returned 200 OK (no auth required)
	ModeBasic  Mode = "basic"  // Basic user:pass succeeded
	ModeBearer Mode = "bearer" // Bearer token flow succeeded
)

// LoginOptions controls how CentralizedLogin behaves.
type LoginOptions struct {
	Registry string // host[:port] or full URL (e.g., "myreg:3281" or "https://myreg:3281")
	Username string // may be empty if you expect an anonymous Bearer flow (rare)
	Password string // password or PAT; can be empty with some setups

	// If Registry has no scheme, default to HTTPS unless AllowHTTP is true.
	AllowHTTP bool

	// TLS knobs (applied to both registry and token realm HTTP clients)
	CAFile             string
	ClientCertFile     string
	ClientKeyFile      string
	InsecureSkipVerify bool

	// Network timeouts
	Timeout time.Duration

	// If true, do NOT attempt Basic fallback when Bearer fetch fails.
	DisableBasicFallback bool
}
