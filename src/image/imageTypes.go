// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:10
// Original filename: src/image/imageTypes.go

package image

type pullMsg struct {
	ID             string         `json:"id"`
	Status         string         `json:"status"`
	Progress       string         `json:"progress"`
	ProgressDetail progressDetail `json:"progressDetail"`
	Error          string         `json:"error"`
	ErrorDetail    *struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}
