// images/types_inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/13

package images

// ImageInspect models the payload returned by:
//
//	GET /images/{name}/json
//
// Docker/Podman can add/remove fields depending on version/driver; this keeps the
// common top-level keys and stays permissive for nested structures.
type ImageInspect struct {
	ID          string   `json:"Id"`
	RepoTags    []string `json:"RepoTags,omitempty"`
	RepoDigests []string `json:"RepoDigests,omitempty"`

	Parent        string `json:"Parent,omitempty"`
	Comment       string `json:"Comment,omitempty"`
	Created       string `json:"Created,omitempty"`
	Container     string `json:"Container,omitempty"`
	DockerVersion string `json:"DockerVersion,omitempty"`
	Author        string `json:"Author,omitempty"`

	Architecture string `json:"Architecture,omitempty"`
	OS           string `json:"Os,omitempty"`
	OSVersion    string `json:"OsVersion,omitempty"`
	Variant      string `json:"Variant,omitempty"`

	Size        int64 `json:"Size,omitempty"`
	VirtualSize int64 `json:"VirtualSize,omitempty"`
	SharedSize  int64 `json:"SharedSize,omitempty"`

	Config          any `json:"Config,omitempty"`
	ContainerConfig any `json:"ContainerConfig,omitempty"`

	GraphDriver any `json:"GraphDriver,omitempty"`
	RootFS      any `json:"RootFS,omitempty"`
	Metadata    any `json:"Metadata,omitempty"`
}
