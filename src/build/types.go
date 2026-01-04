// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 00:00
// Original filename: src/build/buildFlags.go

package build

import "regexp"

// CLI-bound flags (wired in cmd/root.go).
//

// Dockerfile is the path (in the context dir) to the Dockerfile. Wired to -f / --file
var Dockerfile string = "Dockerfile"

// Tags is a repeatable list of image tags. Wired to -t / --tag
var Tags []string

// BuildArgs is a repeatable list of KEY=VALUE (or KEY) build arguments. Wired to --build-arg
var BuildArgs []string

// NoCache disables build cache. Wired to: --no-cache
var NoCache bool

// Pull attempts to pull newer base images.  Wired to: --pull
var Pull bool

// RemoveIntermediate removes intermediate containers after successful build. Wired to: --rm
var RemoveIntermediate bool = true

// ForceRemoveIntermediate always removes intermediate containers, even on failure. Wired to: --force-rm
var ForceRemoveIntermediate bool

// Target is an optional multi-stage target. Wired to: --target
var Target string

// Platform is an optional platform (BuildKit / compatible daemons). Wired to: --platform
var Platform string

// Other vars and structs
type registryAuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

type ignoreRule struct {
	neg   bool
	regex *regexp.Regexp
}

type ignoreMatcher struct {
	rules         []ignoreRule
	dockerfileRel string
}
