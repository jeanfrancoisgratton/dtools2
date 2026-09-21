// dtools2
// src/backend/containerdbackend/system.go

package containerdbackend

import (
	"fmt"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type systemService struct {
	b *Backend
}

func (s systemService) Info() *ce.CustomError {
	c := ctx()
	v, err := s.b.client.Version(c)
	if err != nil {
		return &ce.CustomError{Title: "Unable to fetch containerd version", Message: err.Error()}
	}
	fmt.Printf("%-14s%s\n", "Backend:", "containerd")
	fmt.Printf("%-14s%s\n", "Namespace:", s.b.namespace)
	fmt.Printf("%-14s%s\n", "Version:", v.Version)
	fmt.Printf("%-14s%s\n", "Revision:", v.Revision)
	return nil
}

// RmContainers removes all non-running containers (mirrors docker/podman's
// `system rms`). containerd containers only carry a status via their task,
// so "not running" means "has no task, or its task isn't running". Scoped to
// the single configured namespace, regardless of --all-namespaces: sweeping
// every namespace on the daemon from a "clean up after myself" command would
// be a surprising, destructive blast radius.
func (s systemService) RmContainers() *ce.CustomError {
	svc := containerService{b: &Backend{client: s.b.client, namespace: s.b.namespace}}
	rows, cerr := svc.listRows()
	if cerr != nil {
		return cerr
	}
	var toRemove []string
	for _, r := range rows {
		if r.Status == "running" || r.Status == "paused" {
			continue
		}
		toRemove = append(toRemove, r.ID)
	}
	if len(toRemove) == 0 {
		return nil
	}
	return svc.Remove(toRemove)
}

// Clean removes unused images. containerd has no native volume/network
// store to prune, so this only covers images that aren't referenced by any
// container's Image field.
func (s systemService) Clean() *ce.CustomError {
	c := ctx()
	containers, err := s.b.client.Containers(c)
	if err != nil {
		return &ce.CustomError{Title: "Unable to list containers", Message: err.Error()}
	}
	inUse := make(map[string]bool, len(containers))
	for _, cont := range containers {
		if info, err := cont.Info(c); err == nil {
			inUse[info.Image] = true
		}
	}

	imgs, err := s.b.client.ListImages(c)
	if err != nil {
		return &ce.CustomError{Title: "Unable to list images", Message: err.Error()}
	}
	for _, img := range imgs {
		if inUse[img.Name()] {
			continue
		}
		if err := s.b.client.ImageService().Delete(c, img.Name()); err != nil {
			return &ce.CustomError{Title: "Unable to remove image " + img.Name(), Message: err.Error()}
		}
	}
	return nil
}
