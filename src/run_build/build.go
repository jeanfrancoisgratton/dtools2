// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 00:03
// Original filename: src/build/build.go

package run_build

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	if Compress {
		// Matches docker CLI behaviour: client gzips the context *and* sets this.
		q.Set("compress", "1")
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

	useBuildKit, buildKitForced := decideBuildKit(ctx, client, progressMode)
	if useBuildKit {
		q.Set("version", "2")
	} else {
		q.Set("version", "1")
	}

	body, err := makeContextTarStream(ctx, contextDir, dfRel, Compress)
	if err != nil {
		return err
	}
	defer body.Close()

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")
	if Compress {
		// Many daemons infer gzip from /build?compress=1, but advertising it makes
		// behaviour consistent across Docker and Podman.
		headers.Set("Content-Encoding", "gzip")
	}

	// If ~/.docker/config.json has auths, pass them in X-Registry-Config for private base images.
	if h, err := buildRegistryConfigHeader(); err == nil && h != "" {
		headers.Set("X-Registry-Config", h)
	}

	// BuildKit session (required by modern Docker for buildkit backend).
	var bks *buildkitSession
	if useBuildKit {
		bs, err := newBuildkitSession(ctx, client)
		if err != nil {
			if buildKitForced {
				return err
			}
			// Auto mode: fall back to legacy builder.
			q.Set("version", "1")
			useBuildKit = false
		} else {
			bks = bs
			headers.Set(dockerSessionHeaderID, bks.ID)
			headers.Set(dockerSessionHeaderSharedKey, bks.SharedKey)
			headers.Set(dockerSessionHeaderName, "dtools2")
			defer bks.Close()
		}
	}

	resp, err := client.Do(ctx, http.MethodPost, "/build", q, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	termFd, autoIsTerm := term.GetFdInfo(os.Stdout)
	isTerm := autoIsTerm

	// If stdout isn't a TTY, tty mode degrades to plain.
	if progressMode == "tty" && !autoIsTerm {
		progressMode = "plain"
	}

	if progressMode == "plain" {
		isTerm = false
	}
	if progressMode == "tty" {
		isTerm = true
	}

	// Podman sometimes returns plain text instead of docker's JSONMessage stream.
	br := bufio.NewReader(resp.Body)
	if streamLooksLikeJSON(br) {
		if derr := jsonmessage.DisplayJSONMessagesStream(br, os.Stdout, termFd, isTerm, nil); derr != nil {
			return derr
		}
	} else {
		_, cErr := io.Copy(os.Stdout, br)
		if cErr != nil {
			return cErr
		}
	}

	// If the daemon used the JSON message stream, failures are reported in-stream.
	if resp.StatusCode >= 400 {
		return fmt.Errorf("build failed: %s", resp.Status)
	}

	return nil
}

func decideBuildKit(ctx context.Context, client *rest.Client, progressMode string) (useBuildKit bool, forced bool) {
	// Force-on: DOCKER_BUILDKIT=1
	// Force-off: DOCKER_BUILDKIT=0
	if v, ok := os.LookupEnv("DOCKER_BUILDKIT"); ok {
		forced = true
		v = strings.TrimSpace(v)
		if v == "0" {
			return false, true
		}
		// Any other value behaves like on (don't silently fall back).
		return true, true
	}

	// `--progress=tty` is a UI preference. Prefer BuildKit if the daemon
	// recommends it, but do not force it (fallback to v1 if session setup
	// fails).
	if progressMode == "tty" {
		ok, err := DaemonRecommendsBuildKit(ctx, client)
		if err != nil {
			// Unknown: try BuildKit first, allow fallback.
			return true, false
		}
		return ok, false
	}

	ok, err := DaemonRecommendsBuildKit(ctx, client)
	if err != nil {
		// If we can't detect, try BuildKit first (and fall back automatically).
		return true, false
	}
	return ok, false
}

func streamLooksLikeJSON(br *bufio.Reader) bool {
	peek, err := br.Peek(512)
	if err != nil && len(peek) == 0 {
		return false
	}
	for _, b := range peek {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
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
