// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/29 02:26
// Original filename: src/registry/types.go

package registry

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

type tokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client

	creds CredentialsProvider

	mu         sync.Mutex
	tokenCache map[string]cachedToken // key => token
}

type cachedToken struct {
	token  string
	expiry time.Time
}

type Option func(*Client) error

type CredentialsProvider func(registryHost string) (username, password string, ok bool)

type bearerChallenge struct {
	Realm   string
	Service string
	Scope   string
}
