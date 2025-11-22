// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 16:38
// Original filename: src/images/types.go

package images

// PullOptions controls how an image is pulled.
type PullOptions struct {
	ImageTag string // e.g. "alpine:latest", "registry.example.com/ns/image:tag"
	Registry string // registry to use for auth header; if empty, no auth header is sent
}
