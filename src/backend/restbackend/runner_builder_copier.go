// dtools2
// src/backend/restbackend/runner_builder_copier.go
//
// Runner, Builder and FileCopier are only implemented by the docker/podman
// backend: containerd has no build primitive, and a fundamentally different
// exec/attach and file-copy mechanism (see backend.Runner/Builder/FileCopier
// doc comments).
package restbackend

import (
	"dtools2/extras"
	"dtools2/rest"
	"dtools2/run_build"
	"dtools2/system"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type runner struct{ client *rest.Client }

func (r runner) Run(image string, cmd []string) (int, string, *ce.CustomError) {
	return run_build.RunContainer(r.client, image, cmd)
}

func (r runner) Attach(id string) *ce.CustomError {
	_, err := run_build.AttachContainer(r.client, id)
	return err
}

func (r runner) Exec(container string, command []string) (int, *ce.CustomError) {
	return extras.Run(r.client, container, command)
}

func (r runner) Logs(container string) *ce.CustomError {
	return extras.Logs(r.client, container)
}

type builder struct{ client *rest.Client }

func (b builder) Build(contextDir string) error {
	return run_build.BuildImage(b.client, contextDir)
}

type fileCopier struct{ client *rest.Client }

func (f fileCopier) Copy(source, destination string) *ce.CustomError {
	return system.CopyFile(f.client, source, destination)
}
