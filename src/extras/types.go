// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/15 18:23
// Original filename: src/extras/types.go

package extras

var Debug bool
var LogTimestamps bool // -t
var LogFollow bool     // -f
var LogTail int        // -n (lines); -1 means "all"
var OutputJSON bool    // render output in JSON
var OutputFile = ""

// Structures for the Docker/Podman exec API.
//
// Endpoints:
//   - POST /containers/{id}/exec
//   - POST /exec/{id}/start
//   - GET  /exec/{id}/json
//   - POST /exec/{id}/resize

// ExecCreateRequest matches the JSON body for POST /containers/{id}/exec.
type ExecCreateRequest struct {
	AttachStdin  bool     `json:"AttachStdin,omitempty"`
	AttachStdout bool     `json:"AttachStdout,omitempty"`
	AttachStderr bool     `json:"AttachStderr,omitempty"`
	Tty          bool     `json:"Tty,omitempty"`
	Cmd          []string `json:"Cmd"`

	// Optional.
	User       string   `json:"User,omitempty"`
	Env        []string `json:"Env,omitempty"`
	WorkingDir string   `json:"WorkingDir,omitempty"`
	Privileged bool     `json:"Privileged,omitempty"`
}

// ExecCreateResponse matches the JSON returned by POST /containers/{id}/exec.
type ExecCreateResponse struct {
	ID string `json:"Id"`
}

// ExecStartRequest matches the JSON body for POST /exec/{id}/start.
type ExecStartRequest struct {
	Detach bool `json:"Detach"`
	Tty    bool `json:"Tty"`
}

// ExecInspectResponse matches (partially) the JSON returned by GET /exec/{id}/json.
type ExecInspectResponse struct {
	Running  bool `json:"Running"`
	ExitCode int  `json:"ExitCode"`
}
