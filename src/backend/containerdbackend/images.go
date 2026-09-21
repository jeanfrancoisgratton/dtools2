// dtools2
// src/backend/containerdbackend/images.go

package containerdbackend

import (
	"fmt"
	"os"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/jedib0t/go-pretty/v6/table"

	"dtools2/extras"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type imageService struct{ b *Backend }

func (s imageService) Pull(ref string) error {
	c := ctx()
	img, err := s.b.client.Pull(c, ref, containerd.WithPullUnpack)
	if err != nil {
		return fmt.Errorf("unable to pull %s: %w", ref, err)
	}
	if !rest.QuietOutput {
		fmt.Printf("Pulled %s (%s)\n", img.Name(), img.Target().Digest)
	}
	return nil
}

func (s imageService) Push(ref string) *ce.CustomError {
	c := ctx()
	img, err := s.b.client.GetImage(c, ref)
	if err != nil {
		return &ce.CustomError{Title: "Unable to find image " + ref, Message: err.Error()}
	}
	if err := s.b.client.Push(c, ref, img.Target()); err != nil {
		return &ce.CustomError{Title: "Unable to push " + ref, Message: err.Error()}
	}
	return nil
}

type imgRow struct {
	Name      string `json:"Name"`
	Digest    string `json:"Digest"`
	Size      int64  `json:"Size"`
	Namespace string `json:"Namespace,omitempty"`
}

func (s imageService) listRows() ([]imgRow, *ce.CustomError) {
	c := ctx()
	nsList, err := s.b.namespacesToQuery(c)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list namespaces", Message: err.Error()}
	}

	var rows []imgRow
	for _, ns := range nsList {
		nsCtx := c
		if s.b.allNamespaces {
			nsCtx = namespaces.WithNamespace(c, ns)
		}

		imgs, err := s.b.client.ListImages(nsCtx)
		if err != nil {
			return nil, &ce.CustomError{Title: "Unable to list images in namespace " + ns, Message: err.Error()}
		}
		for _, img := range imgs {
			size, _ := img.Size(nsCtx)
			rows = append(rows, imgRow{Name: img.Name(), Digest: img.Target().Digest.String(), Size: size, Namespace: ns})
		}
	}
	return rows, nil
}

func (s imageService) List(displayOutput bool) *ce.CustomError {
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
		t.AppendHeader(table.Row{"Namespace", "Name", "Digest", "Size"})
	} else {
		t.AppendHeader(table.Row{"Name", "Digest", "Size"})
	}
	if len(rows) == 0 {
		if s.b.allNamespaces {
			t.AppendRow(table.Row{"", "", "", ""})
		} else {
			t.AppendRow(table.Row{"", "", ""})
		}
	}
	for _, r := range rows {
		if s.b.allNamespaces {
			t.AppendRow(table.Row{r.Namespace, r.Name, r.Digest, r.Size})
		} else {
			t.AppendRow(table.Row{r.Name, r.Digest, r.Size})
		}
	}
	t.SetStyle(table.StyleColoredYellowWhiteOnBlack)
	t.Render()
	return nil
}

// Tag creates a new image record with the given name pointing at the same
// content as oldTag — containerd has no separate "tag" operation, image
// names are just labels on the same content-addressed target.
func (s imageService) Tag(oldTag, newTag string) *ce.CustomError {
	c := ctx()
	img, err := s.b.client.ImageService().Get(c, oldTag)
	if err != nil {
		return &ce.CustomError{Title: "Unable to find image " + oldTag, Message: err.Error()}
	}
	img.Name = newTag
	if _, err := s.b.client.ImageService().Create(c, img); err != nil {
		return &ce.CustomError{Title: "Unable to tag " + oldTag + " as " + newTag, Message: err.Error()}
	}
	return nil
}

func (s imageService) Remove(imageList []string) *ce.CustomError {
	c := ctx()
	for _, ref := range imageList {
		if err := s.b.client.ImageService().Delete(c, ref); err != nil {
			return &ce.CustomError{Title: "Unable to remove image " + ref, Message: err.Error()}
		}
	}
	return nil
}

func (s imageService) Inspect(imageRef string) *ce.CustomError {
	c := ctx()
	img, err := s.b.client.GetImage(c, imageRef)
	if err != nil {
		return &ce.CustomError{Title: "Unable to find image " + imageRef, Message: err.Error()}
	}
	size, _ := img.Size(c)
	out := map[string]any{
		"Name":   img.Name(),
		"Digest": img.Target().Digest.String(),
		"Size":   size,
		"Labels": img.Labels(),
	}
	if extras.OutputJSON || rest.QuietOutput {
		b, cerr := extras.MarshalJSON(out)
		if cerr != nil {
			return cerr
		}
		return extras.PrintJSONBytes(b)
	}
	for _, k := range []string{"Name", "Digest", "Size", "Labels"} {
		fmt.Printf("%-10s%v\n", k+":", out[k])
	}
	return nil
}

func (s imageService) Load(tarball string) *ce.CustomError {
	return &ce.CustomError{Title: "Not supported on containerd", Message: notSupported("image load (docker archive format)").Error()}
}

func (s imageService) Save(imgs []string, outFile string) *ce.CustomError {
	return &ce.CustomError{Title: "Not supported on containerd", Message: notSupported("image save (docker archive format)").Error()}
}

func (s imageService) Commit(containerRef, repoTag, author, message string, changes []string) *ce.CustomError {
	return &ce.CustomError{Title: "Not supported on containerd", Message: notSupported("container commit").Error()}
}
