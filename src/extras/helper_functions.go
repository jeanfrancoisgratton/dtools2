// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 14:37
// Original filename: src/images/helpers.go

package extras

import (
	"dtools2/env"
	"dtools2/rest"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	xterm "golang.org/x/term"
)

// SplitURI takes a RepoTag entry (e.g. "registry:5000/repo/img:tag")
// and returns (imageName, tag).
// If no tag exists, tag = "latest".
func SplitURI(ref string) (string, string) {
	// Find last colon. Tags are ALWAYS after the last colon,
	// except cases like registry:5000 without a tag.
	idx := strings.LastIndex(ref, ":")
	if idx == -1 {
		// No colon → no explicit tag
		return ref, "latest"
	}

	// Check if this colon belongs to a registry port, not a tag.
	// That happens if ":" appears before the last "/" in the path.
	slash := strings.LastIndex(ref, "/")
	if slash != -1 && idx < slash {
		// Example: "registry:5000/repo/image" → no tag
		return ref, "latest"
	}

	// Split into name and tag
	return ref[:idx], ref[idx+1:]
}

// GetDefaultRegistry : fetches the default registry from the JSON file
// An error here should not be fatal

func GetDefaultRegistry(regfile string) (string, *ce.CustomError) {
	var err *ce.CustomError
	var dre *env.RegistryEntry

	if dre, err = env.Load(regfile); err != nil {
		return "", err
	}
	return dre.RegistryName, nil
}

func CopyStdin(dst io.Writer, stop <-chan struct{}) {
	fd := int(os.Stdin.Fd())

	// If stdin isn't a terminal (pipe/file), a straight io.Copy is fine and won't hang.
	if !xterm.IsTerminal(fd) {
		_, _ = io.Copy(dst, os.Stdin)
		return
	}

	// For terminal stdin, avoid blocking forever on Read() when the remote side exits.
	// We set stdin non-blocking and poll.
	_ = syscall.SetNonblock(fd, true)
	defer func() { _ = syscall.SetNonblock(fd, false) }()

	buf := make([]byte, 32*1024)
	for {
		select {
		case <-stop:
			return
		case <-rest.Context.Done():
			return
		default:
		}

		n, err := os.Stdin.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
			continue
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			return
		}
		if errorsIsWouldBlock(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		return
	}
}

func StringsTrim(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
