// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 22:19
// Original filename: src/extras/pull.go

package run

import (
	"dtools2/images"
	"dtools2/rest"
	"io"
	"os"
)

// pullImageViaDaemon pulls an image through the daemon (same endpoint as `docker pull`).
//
// The actual pull implementation lives in images/pull.go; this wrapper exists so the
// run package can:
//   - keep its existing call sites unchanged
//   - control output (quiet mode) without baking rest.QuietOutput into images
func pullImageViaDaemon(client *rest.Client, ref string) error {
	var out io.Writer = os.Stdout
	if rest.QuietOutput {
		out = io.Discard
	}
	return images.PullRef(client, ref, out)
}
