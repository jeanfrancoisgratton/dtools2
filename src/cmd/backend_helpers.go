// dtools2
// src/cmd/backend_helpers.go

package cmd

import "fmt"

// notSupportedByBackend reports that the current backend doesn't implement a
// capability (containerd has no networks/volumes/run/build/cp of its own).
func notSupportedByBackend(op string) {
	fmt.Printf("%q is not supported by the %q backend\n", op, activeBackend.Name())
}
