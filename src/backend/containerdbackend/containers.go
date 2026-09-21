// dtools2
// src/backend/containerdbackend/containers.go

package containerdbackend

import (
	"context"
	"fmt"
	"os"
	"syscall"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/errdefs"
	"github.com/jedib0t/go-pretty/v6/table"

	"dtools2/extras"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type containerService struct{ b *Backend }

func ctx() context.Context {
	if rest.Context != nil {
		return rest.Context
	}
	return context.Background()
}

// row is the per-container data used for both table and JSON display.
// Namespace is only populated (and only displayed) in --all-namespaces mode.
type row struct {
	ID        string `json:"Id"`
	Image     string `json:"Image"`
	Status    string `json:"Status"`
	Created   string `json:"Created"`
	Namespace string `json:"Namespace,omitempty"`
}

func (s containerService) taskStatus(c context.Context, cont containerd.Container) string {
	task, err := cont.Task(c, nil)
	if err != nil {
		return "created"
	}
	st, err := task.Status(c)
	if err != nil {
		return "unknown"
	}
	return string(st.Status)
}

func (s containerService) listRows() ([]row, *ce.CustomError) {
	c := ctx()
	nsList, err := s.b.namespacesToQuery(c)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list namespaces", Message: err.Error()}
	}

	var rows []row
	for _, ns := range nsList {
		nsCtx := c
		if s.b.allNamespaces {
			nsCtx = namespaces.WithNamespace(c, ns)
		}

		containers, err := s.b.client.Containers(nsCtx)
		if err != nil {
			return nil, &ce.CustomError{Title: "Unable to list containers in namespace " + ns, Message: err.Error()}
		}
		for _, cont := range containers {
			info, err := cont.Info(nsCtx)
			if err != nil {
				continue
			}
			rows = append(rows, row{
				ID:        cont.ID(),
				Image:     info.Image,
				Status:    s.taskStatus(nsCtx, cont),
				Created:   info.CreatedAt.Format("2006.01.02 15:04:05"),
				Namespace: ns,
			})
		}
	}
	return rows, nil
}

func (s containerService) List(displayOutput bool) *ce.CustomError {
	rows, cerr := s.listRows()
	if cerr != nil {
		return cerr
	}
	if !displayOutput {
		return nil
	}

	if extras.OutputJSON {
		b, cerr := extras.MarshalJSON(rows)
		if cerr != nil {
			return cerr
		}
		return extras.PrintJSONBytes(b)
	}
	if rest.QuietOutput {
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if s.b.allNamespaces {
		t.AppendHeader(table.Row{"Namespace", "Container ID", "Image", "Status", "Created"})
	} else {
		t.AppendHeader(table.Row{"Container ID", "Image", "Status", "Created"})
	}
	if len(rows) == 0 {
		if s.b.allNamespaces {
			t.AppendRow(table.Row{"", "", "", "", ""})
		} else {
			t.AppendRow(table.Row{"", "", "", ""})
		}
	}
	for _, r := range rows {
		if s.b.allNamespaces {
			t.AppendRow(table.Row{r.Namespace, r.ID, r.Image, r.Status, r.Created})
		} else {
			t.AppendRow(table.Row{r.ID, r.Image, r.Status, r.Created})
		}
	}
	t.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	t.Render()
	return nil
}

func (s containerService) Info(container string) *ce.CustomError {
	return s.Inspect(container)
}

func (s containerService) Inspect(id string) *ce.CustomError {
	c := ctx()
	cont, err := s.b.client.LoadContainer(c, id)
	if err != nil {
		return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
	}
	info, err := cont.Info(c)
	if err != nil {
		return &ce.CustomError{Title: "Unable to inspect container " + id, Message: err.Error()}
	}

	out := map[string]any{
		"Id":          info.ID,
		"Image":       info.Image,
		"Status":      s.taskStatus(c, cont),
		"CreatedAt":   info.CreatedAt,
		"Snapshotter": info.Snapshotter,
		"Labels":      info.Labels,
	}

	if extras.OutputJSON || rest.QuietOutput {
		b, cerr := extras.MarshalJSON(out)
		if cerr != nil {
			return cerr
		}
		return extras.PrintJSONBytes(b)
	}

	for _, k := range []string{"Id", "Image", "Status", "CreatedAt", "Snapshotter", "Labels"} {
		fmt.Printf("%-12s%v\n", k+":", out[k])
	}
	return nil
}

// loadTask returns the running task for id, or creates a new one (detached,
// no stdio) from the container's stored spec if none exists yet.
func (s containerService) loadOrCreateTask(c context.Context, cont containerd.Container) (containerd.Task, error) {
	task, err := cont.Task(c, nil)
	if err == nil {
		return task, nil
	}
	if !errdefs.IsNotFound(err) {
		return nil, err
	}
	return cont.NewTask(c, cio.NullIO)
}

func (s containerService) startOne(c context.Context, id string) *ce.CustomError {
	cont, err := s.b.client.LoadContainer(c, id)
	if err != nil {
		return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
	}
	task, err := s.loadOrCreateTask(c, cont)
	if err != nil {
		return &ce.CustomError{Title: "Unable to create task for " + id, Message: err.Error()}
	}
	if err := task.Start(c); err != nil {
		return &ce.CustomError{Title: "Unable to start container " + id, Message: err.Error()}
	}
	return nil
}

func (s containerService) Start(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		if cerr := s.startOne(c, id); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) StartAll() *ce.CustomError {
	c := ctx()
	rows, cerr := s.listRows()
	if cerr != nil {
		return cerr
	}
	for _, r := range rows {
		if r.Status == "running" {
			continue
		}
		if cerr := s.startOne(c, r.ID); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) signalOne(c context.Context, id string, sig syscall.Signal) *ce.CustomError {
	cont, err := s.b.client.LoadContainer(c, id)
	if err != nil {
		return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
	}
	task, err := cont.Task(c, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil // no running task, nothing to signal
		}
		return &ce.CustomError{Title: "Unable to load task for " + id, Message: err.Error()}
	}
	if err := task.Kill(c, sig); err != nil {
		return &ce.CustomError{Title: "Unable to signal container " + id, Message: err.Error()}
	}
	return nil
}

func (s containerService) Stop(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		if cerr := s.signalOne(c, id, syscall.SIGTERM); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) StopAll() *ce.CustomError {
	c := ctx()
	rows, cerr := s.listRows()
	if cerr != nil {
		return cerr
	}
	for _, r := range rows {
		if r.Status != "running" {
			continue
		}
		if cerr := s.signalOne(c, r.ID, syscall.SIGTERM); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) Kill(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		if cerr := s.signalOne(c, id, syscall.SIGKILL); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) KillAll() *ce.CustomError {
	c := ctx()
	rows, cerr := s.listRows()
	if cerr != nil {
		return cerr
	}
	for _, r := range rows {
		if r.Status != "running" {
			continue
		}
		if cerr := s.signalOne(c, r.ID, syscall.SIGKILL); cerr != nil {
			return cerr
		}
	}
	return nil
}

func (s containerService) Restart(names []string) *ce.CustomError {
	if cerr := s.Stop(names); cerr != nil {
		return cerr
	}
	return s.Start(names)
}

func (s containerService) RestartAll() *ce.CustomError {
	if cerr := s.StopAll(); cerr != nil {
		return cerr
	}
	return s.StartAll()
}

func (s containerService) Pause(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		cont, err := s.b.client.LoadContainer(c, id)
		if err != nil {
			return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
		}
		task, err := cont.Task(c, nil)
		if err != nil {
			return &ce.CustomError{Title: "Unable to load task for " + id, Message: err.Error()}
		}
		if err := task.Pause(c); err != nil {
			return &ce.CustomError{Title: "Unable to pause container " + id, Message: err.Error()}
		}
	}
	return nil
}

func (s containerService) Unpause(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		cont, err := s.b.client.LoadContainer(c, id)
		if err != nil {
			return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
		}
		task, err := cont.Task(c, nil)
		if err != nil {
			return &ce.CustomError{Title: "Unable to load task for " + id, Message: err.Error()}
		}
		if err := task.Resume(c); err != nil {
			return &ce.CustomError{Title: "Unable to unpause container " + id, Message: err.Error()}
		}
	}
	return nil
}

func (s containerService) Remove(names []string) *ce.CustomError {
	c := ctx()
	for _, id := range names {
		cont, err := s.b.client.LoadContainer(c, id)
		if err != nil {
			return &ce.CustomError{Title: "Unable to load container " + id, Message: err.Error()}
		}
		if task, err := cont.Task(c, nil); err == nil {
			_, _ = task.Delete(c, containerd.WithProcessKill)
		}
		if err := cont.Delete(c, containerd.WithSnapshotCleanup); err != nil {
			return &ce.CustomError{Title: "Unable to remove container " + id, Message: err.Error()}
		}
	}
	return nil
}

// Rename is not supported: a containerd container's ID is its immutable
// identity, there is no separate mutable name to change.
func (s containerService) Rename(oldName, newName string) *ce.CustomError {
	return &ce.CustomError{Title: "Not supported on containerd", Message: notSupported("container rename").Error()}
}
