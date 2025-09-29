// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:51
// Original filename: src/image/imageHelpers.go

package image

import (
	"context"
	"dtools2/auth"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func renderProgress(ctx context.Context, r io.Reader) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("done")
				return nil
			}
			return err
		}
		if e, ok := obj["error"].(string); ok && e != "" {
			return fmt.Errorf("daemon: %s", e)
		}
		if ed, ok := obj["errorDetail"].(map[string]any); ok {
			if msg, _ := ed["message"].(string); msg != "" {
				return fmt.Errorf("daemon: %s", msg)
			}
		}

		id, _ := obj["id"].(string)
		status, _ := obj["status"].(string)

		if id == "" {
			if status != "" {
				fmt.Println(status)
			}
			continue
		}

		if pd, ok := obj["progressDetail"].(map[string]any); ok {
			cur := numToI64(pd["current"])
			tot := numToI64(pd["total"])
			if tot > 0 {
				pct := int(float64(cur) / float64(tot) * 100.0)
				fmt.Printf("%s: %s %d%% (%d/%d)\n", id, status, pct, cur, tot)
				continue
			}
		}
		if prog, _ := obj["progress"].(string); prog != "" {
			fmt.Printf("%s: %s %s\n", id, status, prog)
			continue
		}
		if status != "" {
			fmt.Printf("%s: %s\n", id, status)
		}
	}
}

func numToI64(v any) int64 {
	switch n := v.(type) {
	case json.Number:
		i, _ := n.Int64()
		return i
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// inferRegistry returns the registry host used for auth lookup.
// docker.io default for single-component names like "alpine" or "library/alpine".
func inferRegistry(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return auth.NormalizeRegistry("")
	}
	parts := strings.Split(ref, "/")
	if len(parts) == 1 {
		return auth.NormalizeRegistry("") // docker hub
	}
	first := parts[0]
	if first == "localhost" || strings.Contains(first, ".") || strings.Contains(first, ":") {
		return auth.NormalizeRegistry(first)
	}
	// e.g. "library/alpine" still docker hub
	return auth.NormalizeRegistry("")
}

func stripRegistry(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) == 1 {
		return ref
	}
	first := parts[0]
	if first == "localhost" || strings.Contains(first, ".") || strings.Contains(first, ":") {
		return strings.Join(parts[1:], "/")
	}
	return ref
}

func splitNameTag(ref string) (name, tag string) {
	// digest check
	if i := strings.Index(ref, "@"); i >= 0 {
		return ref[:i], "" // by digest; tag unused
	}
	// last colon as tag separator when after last slash
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		return ref[:lastColon], ref[lastColon+1:]
	}
	return ref, ""
}
