// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 16:38
// Original filename: src/images/types.go

package images

import "io"

var ForceRemove = false
var RemoveBlacklisted = false

// PullOptions controls how an image is pulled.
type PullOptions struct {
	ImageTag string // e.g. "alpine:latest", "registry.example.com/ns/image:tag"
	Registry string // registry to use for auth header; if empty, no auth header is sent
}

type ImageSummary struct {
	ID          string            `json:"Id"`
	ParentID    string            `json:"ParentId.omitempty"`
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests.omitempty"`
	Created     int64             `json:"Created"`
	Size        int64             `json:"Size"`
	VirtualSize int64             `json:"VirtualSize"`
	SharedSize  int64             `json:"SharedSize"`
	Labels      map[string]string `json:"Labels.omitempty"`
	Containers  int               `json:"Containers"`
	RepoImgName string            `json:"RepoImgName"`
	ImgTag      string            `json:"ImgTag"`
}

type archiveCompression int

const (
	compNone archiveCompression = iota
	compGzip
	compBzip2
	compXz
)

type readerWithClose struct {
	io.Reader
	closeFn func() error
}

type writerWithClose struct {
	io.Writer
	closeFn func() error
}
