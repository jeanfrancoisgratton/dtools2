// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 00:02
// Original filename: src/build/dockerignore.go

package build

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func newIgnoreMatcher(contextDir, dockerfileRel string) (*ignoreMatcher, error) {
	m := &ignoreMatcher{dockerfileRel: dockerfileRel}

	di := filepath.Join(contextDir, ".dockerignore")
	f, err := os.Open(di)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		neg := false
		if strings.HasPrefix(line, "!") {
			neg = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
			if line == "" {
				continue
			}
		}

		re, ok := dockerignorePatternToRegex(line)
		if !ok {
			continue
		}
		m.rules = append(m.rules, ignoreRule{neg: neg, regex: re})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// isIgnored applies dockerignore rules in order.
// This is a pragmatic implementation (not a byte-for-byte clone of Docker CLI),
// but supports: comments, blanks, !negation, *, ?, **, anchored (/...), and basename-only patterns.
func (m *ignoreMatcher) isIgnored(rel string) bool {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	if rel == "" {
		return false
	}

	// Always include build files.
	if rel == ".dockerignore" || (m.dockerfileRel != "" && rel == m.dockerfileRel) {
		return false
	}

	ignored := false
	for _, r := range m.rules {
		if r.regex.MatchString(rel) {
			if r.neg {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	return ignored
}

func dockerignorePatternToRegex(pat string) (*regexp.Regexp, bool) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil, false
	}

	anchored := strings.HasPrefix(pat, "/")
	if anchored {
		pat = strings.TrimPrefix(pat, "/")
	}

	pat = filepath.ToSlash(pat)
	dirPat := strings.HasSuffix(pat, "/")
	if dirPat {
		pat = strings.TrimSuffix(pat, "/")
	}

	// Basename-only pattern (no slash) matches anywhere.
	basenameOnly := !strings.Contains(pat, "/")

	glob := globToRegex(pat)

	var reStr string
	if basenameOnly {
		// Match any path segment; if it’s a directory match, also match everything under it.
		// Example: "node_modules" matches "node_modules" and "a/b/node_modules/..." etc.
		reStr = `(^|.*/)` + glob + `($|/.*$)`
	} else {
		if anchored {
			reStr = `^` + glob + `($|/.*$)`
		} else {
			reStr = `(^|.*/)` + glob + `($|/.*$)`
		}
	}

	// Directory pattern should match the directory itself and anything under it.
	// Our ($|/.*$) already covers that, so no special casing needed beyond accepting "foo/".
	_ = dirPat

	re, err := regexp.Compile(reStr)
	if err != nil {
		return nil, false
	}
	return re, true
}

func globToRegex(glob string) string {
	// Convert dockerignore-ish glob to regex:
	// ** -> .*
	// *  -> [^/]*    (no path separator)
	// ?  -> [^/]
	// escape regex specials.
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		// Handle **
		if glob[i] == '*' && i+1 < len(glob) && glob[i+1] == '*' {
			b.WriteString(`.*`)
			i++
			continue
		}
		switch glob[i] {
		case '*':
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(glob[i])
		default:
			b.WriteByte(glob[i])
		}
	}
	return b.String()
}
