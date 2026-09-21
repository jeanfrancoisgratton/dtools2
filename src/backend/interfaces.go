// dtools2
// src/backend/interfaces.go
//
// Backend abstracts the container runtime dtools talks to (docker/podman via
// their shared REST API, or containerd via its local gRPC API). Method
// signatures mirror the existing containers/images/networks/volumes/system
// package functions (minus the *rest.Client receiver): every cmd/*.go call
// site already discards the data those functions return and only checks the
// error, since display happens as a side effect inside the call. So the
// interface does the same — no separate "neutral" domain-object layer is
// needed; each backend keeps its own internal data shapes private.
//
// Networks, Volumes, Runner (interactive run/attach/exec/logs), Builder and
// FileCopier are capability-gated: containerd has no native network or named
// volume store, no build primitive, and a different exec/attach/cp mechanism
// entirely, so a containerd Backend returns ok=false for these rather than a
// half-working implementation.
package backend

import (
	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type Backend interface {
	// Name identifies the backend for display purposes, e.g. "docker/podman" or "containerd".
	Name() string
	Close() error

	Containers() ContainerService
	Images() ImageService
	System() SystemService

	Networks() (NetworkService, bool)
	Volumes() (VolumeService, bool)
	Runner() (Runner, bool)
	Builder() (Builder, bool)
	FileCopier() (FileCopier, bool)
}

type ContainerService interface {
	List(displayOutput bool) *ce.CustomError
	Info(container string) *ce.CustomError
	Inspect(id string) *ce.CustomError
	Remove(containerList []string) *ce.CustomError
	Pause(containers []string) *ce.CustomError
	Unpause(containers []string) *ce.CustomError
	Start(containers []string) *ce.CustomError
	StartAll() *ce.CustomError
	Stop(containers []string) *ce.CustomError
	StopAll() *ce.CustomError
	Rename(oldName, newName string) *ce.CustomError
	Kill(containers []string) *ce.CustomError
	KillAll() *ce.CustomError
	Restart(containers []string) *ce.CustomError
	RestartAll() *ce.CustomError
}

type ImageService interface {
	Pull(ref string) error
	Push(ref string) *ce.CustomError
	List(displayOutput bool) *ce.CustomError
	Tag(oldTag, newTag string) *ce.CustomError
	Remove(imageList []string) *ce.CustomError
	Load(tarball string) *ce.CustomError
	Save(images []string, outFile string) *ce.CustomError
	Commit(containerRef, repoTag, author, message string, changes []string) *ce.CustomError
	Inspect(imageRef string) *ce.CustomError
}

type NetworkService interface {
	List(displayOutput bool) *ce.CustomError
	Add(networkName string) *ce.CustomError
	Remove(netList []string) *ce.CustomError
	Attach(network, container string) *ce.CustomError
	Detach(network, container string) *ce.CustomError
	Inspect(networkID string) *ce.CustomError
}

type VolumeService interface {
	List(displayOutput bool) *ce.CustomError
	Remove(volList []string) *ce.CustomError
	Prune() *ce.CustomError
	Create(volumeName string) *ce.CustomError
	Inspect(volumeName string) *ce.CustomError
}

type SystemService interface {
	// RmContainers removes all exited/created (non-running) containers.
	RmContainers() *ce.CustomError
	// Clean removes unused images, volumes and networks.
	Clean() *ce.CustomError
	Info() *ce.CustomError
}

// Runner covers interactive/attached workflows: `dtools run`, `attach`,
// `exec`, `logs`. Only the docker/podman backend implements this.
type Runner interface {
	Run(image string, cmd []string) (exitCode int, containerID string, err *ce.CustomError)
	Attach(id string) *ce.CustomError
	Exec(container string, command []string) (exitCode int, err *ce.CustomError)
	Logs(container string) *ce.CustomError
}

// Builder covers `dtools build`. Only the docker/podman backend implements
// this — containerd has no build primitive of its own.
type Builder interface {
	Build(contextDir string) error
}

// FileCopier covers `dtools cp`. Only the docker/podman backend implements
// this — containerd would need a different, snapshot-mount-based mechanism.
type FileCopier interface {
	Copy(source, destination string) *ce.CustomError
}
