// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/20 00:06
// Original filename: src/images/helpers.go

package images

import "strings"

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
