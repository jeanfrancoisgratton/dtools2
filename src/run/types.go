// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 21:59
// Original filename: src/extras/types.go

package run

var RunDetach bool       // -d
var RunInteractive bool  // -i
var RunTTY bool          // -t
var RunRemove bool       // --rm
var RunName string       // --name
var RunUser string       // -u
var RunWorkdir string    // -w
var RunEnv []string      // -e
var RunPublish []string  // -p
var RunVolume []string   // -v
var RunMount []string    // --mount
var RunNetwork string    // --network
var RunEntrypoint string // --entrypoint
var RunHostname string   // --hostname

// Minimal structures for the Docker/Podman "docker run" flow.
//
// Endpoints:
//   - POST /containers/create
//   - POST /containers/{id}/start
//   - POST /containers/{id}/attach
//   - POST /containers/{id}/wait
//   - DELETE /containers/{id}

// ContainerCreateRequest is the JSON body for POST /containers/create.
// This is intentionally a *subset* of the full Docker API schema; flags not
// implemented by dtools2 are omitted.
type ContainerCreateRequest struct {
	Image string   `json:"Image"`
	Cmd   []string `json:"Cmd,omitempty"`

	Entrypoint []string `json:"Entrypoint,omitempty"`

	// IO
	AttachStdin  bool `json:"AttachStdin,omitempty"`
	AttachStdout bool `json:"AttachStdout,omitempty"`
	AttachStderr bool `json:"AttachStderr,omitempty"`
	OpenStdin    bool `json:"OpenStdin,omitempty"`
	StdinOnce    bool `json:"StdinOnce,omitempty"`
	Tty          bool `json:"Tty,omitempty"`

	// Process/user context
	User       string   `json:"User,omitempty"`
	Env        []string `json:"Env,omitempty"`
	WorkingDir string   `json:"WorkingDir,omitempty"`
	Hostname   string   `json:"Hostname,omitempty"`

	// Networking/ports
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`

	// Anonymous volumes ("-v /path") use this older field.
	Volumes map[string]struct{} `json:"Volumes,omitempty"`

	HostConfig *HostConfig `json:"HostConfig,omitempty"`
}

// HostConfig is a subset used by dtools2 for run.
type HostConfig struct {
	AutoRemove bool `json:"AutoRemove,omitempty"`
	// Prefer Mounts over Binds so we can support both bind mounts and named volumes.
	Mounts      []Mount `json:"Mounts,omitempty"`
	NetworkMode string  `json:"NetworkMode,omitempty"`

	PortBindings map[string][]PortBinding `json:"PortBindings,omitempty"`
}

// Mount is a minimal subset of the Docker API mount schema.
type Mount struct {
	Type     string `json:"Type"` // "bind", "volume", or "tmpfs"
	Source   string `json:"Source,omitempty"`
	Target   string `json:"Target"`
	ReadOnly bool   `json:"ReadOnly,omitempty"`
}

// PortBinding is the Docker API schema for published ports.
type PortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

// ContainerCreateResponse is returned by POST /containers/create.
type ContainerCreateResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

// ContainerWaitResponse is returned by POST /containers/{id}/wait.
type ContainerWaitResponse struct {
	StatusCode int `json:"StatusCode"`
	Error      *struct {
		Message string `json:"Message"`
	} `json:"Error,omitempty"`
}
