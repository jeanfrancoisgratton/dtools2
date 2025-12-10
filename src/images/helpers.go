// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/20 00:06
// Original filename: src/images/helpers.go

package images

import (
	"fmt"
	"math"
	"strings"
)

// splitRepoTag splits an image reference into repository and tag.
// It treats the last ':' after the last '/' as the tag separator.
// If no such ':' exists, tag is empty.
func splitRepoTag(ref string) (repo, tag string) {
	// Strip any @digest part.
	if i := strings.Index(ref, "@"); i != -1 {
		ref = ref[:i]
	}

	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")

	if colon > slash {
		// repo:tag
		return ref[:colon], ref[colon+1:]
	}

	// no tag
	return ref, ""
}

// registryFromImageRef attempts to extract the registry host from an image
// reference. It follows Docker's rough heuristic:
//
//   - Take the first path component (before the first '/').
//   - If it contains a '.' or ':', or is "localhost", treat it as a registry.
//   - Otherwise, assume no explicit registry (Docker Hub style) and return "".
func registryFromImageRef(ref string) string {
	// Strip any @digest part.
	if i := strings.Index(ref, "@"); i != -1 {
		ref = ref[:i]
	}

	slash := strings.Index(ref, "/")
	if slash == -1 {
		return ""
	}

	first := ref[:slash]
	if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
		return first
	}

	return ""
}

// splitURI takes a RepoTag entry (e.g. "registry:5000/repo/img:tag")
// and returns (imageName, tag).
// If no tag exists, tag = "latest".
func splitURI(ref string) (string, string) {
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

func formatSize(sz int64) string {
	numSize := (float64)(sz) / 1000.0 / 1000.0 // this will give us the size in MB
	if (int)(math.Log10(float64(numSize))) > 2 {
		return fmt.Sprintf("%.3f GB", numSize/1000.0)
	} else {
		return fmt.Sprintf("%.3f MB", numSize)
	}
}
