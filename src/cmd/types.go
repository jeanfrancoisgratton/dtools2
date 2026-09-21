// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 16:45
// Original filename: src/cmd/types.go

package cmd

import "dtools2/backend"

// Global flags used for option parsing by COBRA

var OutputJSON bool
var APIVersion string
var UseTLS bool
var TLSCACert string
var TLSCert string
var TLSKey string
var TLSSkipVerify bool

// Runtime selects which backend to talk to: "" (auto, docker/podman via
// REST) or "containerd". docker/podman need no explicit selection since
// they share the same REST dialect.
var Runtime string
var ContainerdSocket string
var ContainerdNamespace string
var AllNamespaces bool

// Resolved backend, shared by subcommands.
var activeBackend backend.Backend

// Auth / login-related flags.

var loginUsername string
var loginPassword string
var loginInsecure bool
var loginCACertPath string

// Image-related flags.

var imagePullRegistry string

// docker commit-like flags.

var commitAuthor string
var commitMessage string
var commitChanges []string

// buildVersion and buildDate are set via -ldflags -X at package-build time.
// Each __*/ builder reads its own already-authoritative version field
// (APKBUILD's pkgver, PKGBUILD's pkgver, control's Version minus the Debian
// revision, the spec's %{_version}); src/build.sh reads nxtools.json since it
// isn't tied to any one distro's packaging file. The fallbacks below are what
// you get from a plain `go build .` with no ldflags, e.g. local development.
var buildVersion = "dev"
var buildDate = "unknown"
