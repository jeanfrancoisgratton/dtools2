// dtools2
// src/backend/restbackend/restbackend.go
//
// restbackend implements backend.Backend for docker and podman, which share
// a near-100%-compatible REST API. It is a thin adapter: every method
// forwards to the existing containers/images/networks/volumes/system/
// run_build/extras packages unchanged, so CLI behavior is byte-identical to
// before this abstraction existed.
package restbackend

import (
	"context"

	"dtools2/backend"
	"dtools2/rest"
)

type Backend struct {
	client *rest.Client
}

// New builds a rest.Client from cfg and negotiates the API version if one
// wasn't explicitly set, mirroring what cmd/root.go used to do directly.
func New(ctx context.Context, cfg rest.Config) (backend.Backend, error) {
	client, err := rest.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.APIVersion == "" {
		v, err := rest.NegotiateAPIVersion(ctx, client)
		if err != nil {
			return nil, err
		}
		client.SetAPIVersion(v)
	}

	return &Backend{client: client}, nil
}

func (b *Backend) Name() string { return "docker/podman" }
func (b *Backend) Close() error { return nil }

func (b *Backend) Containers() backend.ContainerService { return containerService{b.client} }
func (b *Backend) Images() backend.ImageService         { return imageService{b.client} }
func (b *Backend) System() backend.SystemService        { return systemService{b.client} }

func (b *Backend) Networks() (backend.NetworkService, bool) { return networkService{b.client}, true }
func (b *Backend) Volumes() (backend.VolumeService, bool)   { return volumeService{b.client}, true }
func (b *Backend) Runner() (backend.Runner, bool)           { return runner{b.client}, true }
func (b *Backend) Builder() (backend.Builder, bool)         { return builder{b.client}, true }
func (b *Backend) FileCopier() (backend.FileCopier, bool)   { return fileCopier{b.client}, true }
