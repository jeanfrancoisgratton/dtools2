// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/21 13:37
// Original filename: src/auth/centralizedLogin.go

package auth

import (
	"context"
	"dtools2/rest"
	"fmt"
	"strings"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// CentralizedLogin does the whole dance and stores the winning credential in
// ~/.docker/config.json. It returns (mode, normalizedRegistryKey, error).
func CentralizedLogin(ctx context.Context, opts LoginOptions) (Mode, string, *ce.CustomError) {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if strings.TrimSpace(opts.Registry) == "" {
		return "", "", &ce.CustomError{Code: 901, Title: "Error logging in", Message: "Registry is required"}
	}

	// Decide scheme if none provided.
	regURL := opts.Registry
	if !strings.Contains(regURL, "://") {
		if opts.AllowHTTP {
			regURL = "http://" + regURL
		} else {
			regURL = "https://" + regURL
		}
	}
	// This is the key used in config.json (canonicalize docker.io aliases inside NormalizeRegistry).
	regKey := NormalizeRegistry(regURL)

	// Build a plain *http.Client for registry requests (/v2/ probe, Basic retry).
	regHTTP, err := rest.NewHTTPClient(rest.TLSOptions{
		CAFile:             opts.CAFile,
		ClientCertFile:     opts.ClientCertFile,
		ClientKeyFile:      opts.ClientKeyFile,
		InsecureSkipVerify: opts.InsecureSkipVerify,
	}, opts.Timeout)
	if err != nil {
		return "", "", &ce.CustomError{Code: 902, Title: "Error connecting to the registry", Message: err.Error()}
	}

	// 1) Probe /v2/ to learn auth scheme.
	scheme, params, perr := ProbeAuthScheme(ctx, regHTTP, regURL)
	if perr != nil {
		return "", "", &ce.CustomError{Code: 903, Title: "Error fetching probe information", Message: perr.Error()}
	}

	switch strings.ToLower(scheme) {
	case "none":
		// Open registry — store user:pass only if provided; otherwise do nothing.
		if opts.Username != "" {
			if err := WriteDockerConfigAuth(regKey, opts.Username, opts.Password); err != nil {
				return "", "", err
			}
		}
		return ModeNone, regKey, nil

	case "basic":
		// Re-ping /v2/ with Basic
		if err := tryBasic(ctx, regHTTP, regURL, opts.Username, opts.Password); err != nil {
			return "", "", &ce.CustomError{Code: 904, Title: "Error attempting basic auth method", Message: err.Error()}
		}
		if err := WriteDockerConfigAuth(regKey, opts.Username, opts.Password); err != nil {
			return "", "", &ce.CustomError{Code: 904, Title: "Error writing the config file", Message: err.Error()}
		}
		return ModeBasic, regKey, nil

	case "bearer":
		// Build an http.Client for the token realm (can be a different host).
		tokenHTTP, err := rest.NewHTTPClient(rest.TLSOptions{
			CAFile:             opts.CAFile,
			ClientCertFile:     opts.ClientCertFile,
			ClientKeyFile:      opts.ClientKeyFile,
			InsecureSkipVerify: opts.InsecureSkipVerify,
		}, opts.Timeout)
		if err != nil {
			return "", "", &ce.CustomError{Code: 905, Title: "Error building the http client", Message: err.Error()}
		}

		// Try to fetch a token from the realm.
		token, _, _, terr := FetchBearerToken(ctx, tokenHTTP, params, opts.Username, opts.Password)
		if terr == nil && token != "" {
			if err := WriteDockerConfigToken(regKey, token); err != nil {
				return "", "", &ce.CustomError{Code: 906, Title: "Error writing the token in the config file", Message: err.Error()}
			}
			return ModeBearer, regKey, nil
		}

		// Optional Basic fallback (some registries advertise Bearer but accept Basic).
		if !opts.DisableBasicFallback {
			if be := tryBasic(ctx, regHTTP, regURL, opts.Username, opts.Password); be == nil {
				if err := WriteDockerConfigAuth(regKey, opts.Username, opts.Password); err != nil {
					return "", "", &ce.CustomError{Code: 906, Title: "Error writing the token in the config file", Message: err.Error()}
				}
				return ModeBasic, regKey, nil
			}
		}
		return "", "", &ce.CustomError{Code: 907, Title: "Bearer token fetch failed", Message: terr.Error()}

	default:
		return "", "", &ce.CustomError{Code: 908, Title: "Error fetching auth scheme", Message: fmt.Sprintf("Scheme %s is not supported", scheme)}
	}
}

// NOT USED YET
// Optional: tiny helper for callers that only need skip-verify quickly.
func CentralizedLoginInsecure(ctx context.Context, reg, user, pass string) (Mode, string, error) {
	return CentralizedLogin(ctx, LoginOptions{
		Registry:           reg,
		Username:           user,
		Password:           pass,
		AllowHTTP:          false,
		InsecureSkipVerify: true,
	})
}

// NOT USED YET
// Optional: when you want to force HTTP explicitly (lab only).
func CentralizedLoginHTTP(ctx context.Context, reg, user, pass string) (Mode, string, error) {
	// If caller passes a full URL with http:// it will be honored anyway.
	return CentralizedLogin(ctx, LoginOptions{
		Registry:  reg,
		Username:  user,
		Password:  pass,
		AllowHTTP: true,
	})
}
