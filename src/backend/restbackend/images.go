// dtools2
// src/backend/restbackend/images.go

package restbackend

import (
	"dtools2/images"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type imageService struct{ client *rest.Client }

func (s imageService) Pull(ref string) error {
	return images.ImagePull(s.client, ref)
}

func (s imageService) Push(ref string) *ce.CustomError {
	return images.ImagePush(s.client, ref)
}

func (s imageService) List(displayOutput bool) *ce.CustomError {
	_, err := images.ImagesList(s.client, displayOutput)
	return err
}

func (s imageService) Tag(oldTag, newTag string) *ce.CustomError {
	return images.TagImage(s.client, oldTag, newTag)
}

func (s imageService) Remove(imageList []string) *ce.CustomError {
	return images.RemoveImage(s.client, imageList)
}

func (s imageService) Load(tarball string) *ce.CustomError {
	return images.ImageLoad(s.client, tarball)
}

func (s imageService) Save(imgs []string, outFile string) *ce.CustomError {
	return images.ImageSave(s.client, imgs, outFile)
}

func (s imageService) Commit(containerRef, repoTag, author, message string, changes []string) *ce.CustomError {
	return images.ImageCommit(s.client, containerRef, repoTag, author, message, changes)
}

func (s imageService) Inspect(imageRef string) *ce.CustomError {
	_, err := images.InspectImage(s.client, imageRef)
	return err
}
