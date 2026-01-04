// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 00:03
// Original filename: src/build/build.go

package build

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"dtools2/auth"
	"dtools2/rest"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

// BuildImage emulates `docker build` using the daemon API: POST /build.
// This works for both Docker Engine and Podman service when using the compat API.
func BuildImage(client *rest.Client, contextDir string) error {
	ctx := rest.Context
	if ctx == nil {
		ctx = context.Background()
	}

	progressMode := strings.ToLower(strings.TrimSpace(Progress))
	if progressMode == "" {
		progressMode = "auto"
	}
	if progressMode != "auto" && progressMode != "plain" && progressMode != "tty" {
		return fmt.Errorf("invalid --progress %q (supported: auto|plain|tty)", Progress)
	}

	dfRel, err := dockerfileRelative(contextDir)
	if err != nil {
		return err
	}

	q := url.Values{}
	q.Set("dockerfile", dfRel)

	for _, t := range Tags {
		if strings.TrimSpace(t) == "" {
			continue
		}
		q.Add("t", t)
	}

	if Pull {
		q.Set("pull", "true")
	}
	if NoCache {
		q.Set("nocache", "true")
	}

	// docker default is rm=true
	if RemoveIntermediate {
		q.Set("rm", "true")
	} else {
		q.Set("rm", "false")
	}

	if ForceRemoveIntermediate {
		q.Set("forcerm", "true")
	}

	if Target != "" {
		q.Set("target", Target)
	}

	if Platform != "" {
		q.Set("platform", Platform)
	}

	if len(BuildArgs) > 0 {
		m, err := parseBuildArgs(BuildArgs)
		if err != nil {
			return err
		}
		b, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("failed to encode build args: %w", err)
		}
		q.Set("buildargs", string(b))
	}

	body, err := makeContextTarStream(ctx, contextDir, dfRel)
	if err != nil {
		return err
	}
	defer body.Close()

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")

	// If ~/.docker/config.json has auths, pass them in X-Registry-Config for private base images.
	if h, err := buildRegistryConfigHeader(); err == nil && h != "" {
		headers.Set("X-Registry-Config", h)
	}

	resp, err := client.Do(ctx, http.MethodPost, "/build", q, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Stream build output (Docker JSON message stream).
	termFd, autoIsTerm := term.GetFdInfo(os.Stdout)
	isTerm := autoIsTerm

	// `--progress=tty` is meaningful for BuildKit-style output.
	// If the daemon doesn't recommend BuildKit, we ignore the request and fall
	// back to the default (auto). If stdout isn't a TTY, we fall back to plain.
	if progressMode == "tty" {
		if !autoIsTerm {
			progressMode = "plain"
		} else {
			ok, derr := DaemonRecommendsBuildKit(ctx, client)
			if derr != nil {
				// If we can't detect, don't fail the build; just fall back.
				progressMode = "auto"
			} else if !ok {
				progressMode = "auto"
			}
		}
	}

	if progressMode == "plain" {
		isTerm = false
	}
	if progressMode == "tty" {
		isTerm = true
	}

	if derr := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, termFd, isTerm, nil); derr != nil {
		// Fallback: if daemon doesn't speak exact docker JSONMessage stream.
		// Still expose status code below.
		_, _ = ioCopyAll(os.Stdout, resp.Body)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("build failed: %s", resp.Status)
	}

	return nil
}

func dockerfileRelative(contextDir string) (string, error) {
	df := Dockerfile
	if strings.TrimSpace(df) == "" {
		df = "Dockerfile"
	}

	contextDirAbs, err := filepath.Abs(contextDir)
	if err != nil {
		return "", fmt.Errorf("invalid context dir %q: %w", contextDir, err)
	}

	dfPath := df
	if !filepath.IsAbs(dfPath) {
		// Pragmatic choice: interpret -f relative to the context directory.
		dfPath = filepath.Join(contextDirAbs, dfPath)
	}
	dfPath = filepath.Clean(dfPath)

	st, err := os.Stat(dfPath)
	if err != nil {
		return "", fmt.Errorf("cannot stat Dockerfile %q: %w", df, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("Dockerfile path %q is a directory", df)
	}

	rel, err := filepath.Rel(contextDirAbs, dfPath)
	if err != nil {
		return "", fmt.Errorf("cannot compute Dockerfile relative path: %w", err)
	}
	rel = filepath.ToSlash(rel)

	// Refuse Dockerfile outside context (subset behaviour).
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("Dockerfile %q must be inside the build context (%q)", df, contextDir)
	}

	return rel, nil
}

func parseBuildArgs(args []string) (map[string]*string, error) {
	out := make(map[string]*string)
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}

		if strings.Contains(a, "=") {
			parts := strings.SplitN(a, "=", 2)
			k := strings.TrimSpace(parts[0])
			v := parts[1]
			if k == "" {
				return nil, fmt.Errorf("invalid --build-arg %q", a)
			}
			vv := v
			out[k] = &vv
			continue
		}

		// KEY (no '='): docker CLI uses the client env var if present.
		k := a
		if k == "" {
			continue
		}
		if ev, ok := os.LookupEnv(k); ok {
			vv := ev
			out[k] = &vv
		} else {
			empty := ""
			out[k] = &empty
		}
	}
	return out, nil
}

func buildRegistryConfigHeader() (string, error) {
	cfg, _, err := auth.LoadDockerConfig()
	if err != nil {
		return "", err
	}
	if cfg == nil || len(cfg.Auths) == 0 {
		return "", nil
	}

	m := make(map[string]registryAuthConfig, len(cfg.Auths))
	for server, a := range cfg.Auths {
		s := strings.TrimSpace(server)
		if s == "" {
			continue
		}
		m[s] = registryAuthConfig{
			Username:      a.Username,
			Password:      a.Password,
			Auth:          a.Auth,
			Email:         a.Email,
			ServerAddress: s,
			IdentityToken: a.IdentityToken,
		}
	}

	if len(m) == 0 {
		return "", nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	// Docker expects URL-safe base64 for these auth headers.
	return base64.URLEncoding.EncodeToString(b), nil
}

// local helper to avoid importing io in multiple files
func ioCopyAll(dst *os.File, src interface{ Read([]byte) (int, error) }) (int64, error) {
	buf := make([]byte, 32*1024)
	var n int64
	for {
		r, err := src.Read(buf)
		if r > 0 {
			w, werr := dst.Write(buf[:r])
			n += int64(w)
			if werr != nil {
				return n, werr
			}
		}
		if err != nil {
			return n, err
		}
	}
}
