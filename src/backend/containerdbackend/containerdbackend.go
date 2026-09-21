// dtools2
// src/backend/containerdbackend/containerdbackend.go
//
// containerdbackend implements backend.Backend against a local containerd
// daemon via its gRPC client (github.com/containerd/containerd/v2/client).
// containerd is local-only (no -H equivalent) and has no native network or
// named-volume store, no build primitive, and no docker-compatible
// create/attach flow — those capabilities are gated off (see
// backend.Backend.Networks/Volumes/Runner/Builder/FileCopier).
//
// v1 scope: List/Inspect/lifecycle for containers that already exist in the
// configured namespace, and image list/pull/push/tag/remove/inspect.
// Containers cannot yet be *created* through dtools on this backend (that's
// part of the Runner capability, which containerd doesn't implement here).
package containerdbackend

import (
	"context"
	"fmt"
	"time"

	containerd "github.com/containerd/containerd/v2/client"

	"dtools2/backend"
)

const (
	DefaultSocket    = "/run/containerd/containerd.sock"
	DefaultNamespace = "default"

	// connectTimeout bounds how long New() waits for containerd to answer,
	// regardless of the caller's context: the gRPC client connects lazily,
	// so without an explicit deadline here a dead/missing socket hangs
	// forever instead of failing fast.
	connectTimeout = 5 * time.Second
)

type Backend struct {
	client        *containerd.Client
	namespace     string
	allNamespaces bool
}

// New connects to containerd. namespace is the single namespace used for
// every operation except listing; if allNamespaces is true, List operations
// (only — not lifecycle or prune operations, which stay scoped to namespace)
// enumerate every namespace on the daemon instead, similar to `kubectl get
// -A`.
func New(ctx context.Context, socket, namespace string, allNamespaces bool) (backend.Backend, error) {
	if socket == "" {
		socket = DefaultSocket
	}
	if namespace == "" {
		namespace = DefaultNamespace
	}

	client, err := containerd.New(socket, containerd.WithDefaultNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("connect to containerd at %s: %w", socket, err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if _, err := client.IsServing(probeCtx); err != nil {
		client.Close()
		return nil, fmt.Errorf("containerd at %s is not responding: %w", socket, err)
	}

	return &Backend{client: client, namespace: namespace, allNamespaces: allNamespaces}, nil
}

// namespacesToQuery returns the namespace(s) a List operation should cover.
func (b *Backend) namespacesToQuery(c context.Context) ([]string, error) {
	if !b.allNamespaces {
		return []string{b.namespace}, nil
	}
	return b.client.NamespaceService().List(c)
}

func (b *Backend) Name() string { return "containerd" }
func (b *Backend) Close() error { return b.client.Close() }

func (b *Backend) Containers() backend.ContainerService {
	return containerService{b: b}
}
func (b *Backend) Images() backend.ImageService {
	return imageService{b: b}
}
func (b *Backend) System() backend.SystemService {
	return systemService{b: b}
}

// containerd has no native network/volume store, build primitive, or
// docker-compatible interactive run/attach/exec/cp mechanism.
func (b *Backend) Networks() (backend.NetworkService, bool) { return nil, false }
func (b *Backend) Volumes() (backend.VolumeService, bool)   { return nil, false }
func (b *Backend) Runner() (backend.Runner, bool)           { return nil, false }
func (b *Backend) Builder() (backend.Builder, bool)         { return nil, false }
func (b *Backend) FileCopier() (backend.FileCopier, bool)   { return nil, false }

func notSupported(op string) error {
	return fmt.Errorf("%s is not supported on the containerd backend", op)
}
