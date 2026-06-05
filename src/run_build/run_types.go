// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 21:59
// Original filename: src/extras/types.go

package run_build

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
var RunUlimits []string  // --ulimit
var RunMemory string     // --memory
var RunCPUs float64      // --cpus
var RunCPUShares int64   // --cpu-shares
var RunRestart string    // --restart
var RunPrivileged bool   // --privileged
var RunCapAdd []string   // --cap-add
var RunCapDrop []string  // --cap-drop
var RunReadOnly bool     // --read-only
var RunShmSize string    // --shm-size
var RunPidsLimit int64   // --pids-limit

// Minimal structures for the Docker/Podman "docker run_build" flow.
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

// HostConfig is a subset used by dtools2 for run_build.
type HostConfig struct {
	AutoRemove bool `json:"AutoRemove,omitempty"`
	// Prefer Mounts to Binds so we can support both bind mounts and named volumes.
	Mounts         []Mount                  `json:"Mounts,omitempty"`
	NetworkMode    string                   `json:"NetworkMode,omitempty"`
	PortBindings   map[string][]PortBinding `json:"PortBindings,omitempty"`
	Ulimits        []UlimitFlag             `json:"Ulimits,omitempty"`
	Memory         int64                    `json:"Memory,omitempty"`
	NanoCPUs       int64                    `json:"NanoCpus,omitempty"`
	CpuShares      int64                    `json:"CpuShares,omitempty"`
	RestartPolicy  RestartPolicy            `json:"RestartPolicy,omitempty"`
	Privileged     bool                     `json:"Privileged,omitempty"`
	CapAdd         []string                 `json:"CapAdd,omitempty"`
	CapDrop        []string                 `json:"CapDrop,omitempty"`
	ReadonlyRootfs bool                     `json:"ReadonlyRootfs,omitempty"`
	ShmSize        int64                    `json:"ShmSize,omitempty"`
	PidsLimit      int64                    `json:"PidsLimit,omitempty"`
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
