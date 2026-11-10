// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/backend/containerdbackend/containerdbackend_test.go

// These tests cover what does not need a live containerd daemon (the socket is usually
// root-only): connection failure, capability gating and unsupported operations.

package containerdbackend

import (
	"context"
	"dtools2/rest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFailsFastOnMissingSocket(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "nope.sock")
	start := time.Now()
	b, err := New(context.Background(), sock, "", false)
	if err == nil {
		b.Close()
		t.Fatal("New on a missing socket: want an error")
	}
	if !strings.Contains(err.Error(), sock) {
		t.Errorf("error %q: want it to name the socket", err)
	}
	if d := time.Since(start); d > connectTimeout+2*time.Second {
		t.Errorf("New took %v; it must fail within the %v connect timeout", d, connectTimeout)
	}
}

func TestCapabilities(t *testing.T) {
	b := &Backend{namespace: DefaultNamespace}
	if b.Name() != "containerd" {
		t.Errorf("Name() = %q", b.Name())
	}
	if _, ok := b.Networks(); ok {
		t.Error("containerd has no network store")
	}
	if _, ok := b.Volumes(); ok {
		t.Error("containerd has no volume store")
	}
	if _, ok := b.Runner(); ok {
		t.Error("containerd has no Runner")
	}
	if _, ok := b.Builder(); ok {
		t.Error("containerd has no Builder")
	}
	if _, ok := b.FileCopier(); ok {
		t.Error("containerd has no FileCopier")
	}
	if b.Containers() == nil || b.Images() == nil || b.System() == nil {
		t.Error("core services must always be available")
	}
}

func TestNamespacesToQuerySingle(t *testing.T) {
	b := &Backend{namespace: "k8s.io"}
	ns, err := b.namespacesToQuery(context.Background())
	if err != nil || len(ns) != 1 || ns[0] != "k8s.io" {
		t.Errorf("namespacesToQuery = %v, %v; want only the configured namespace", ns, err)
	}
}

func TestUnsupportedOperations(t *testing.T) {
	b := &Backend{}
	cs, is := b.Containers(), b.Images()
	cases := map[string]func() error{
		"container rename": func() error { return cs.Rename("a", "b") },
		"container port":   func() error { return cs.Port("a", "") },
		"image load":       func() error { return is.Load("x.tar") },
		"image save":       func() error { return is.Save([]string{"a"}, "x.tar") },
		"container commit": func() error { return is.Commit("c", "r:t", "", "", nil) },
	}
	for op, fn := range cases {
		err := fn()
		if err == nil || !strings.Contains(err.Error(), "not supported on the containerd backend") {
			t.Errorf("%s = %v; want a 'not supported' error", op, err)
		}
	}
}

func TestCtxFallsBackToBackground(t *testing.T) {
	saved := rest.Context
	t.Cleanup(func() { rest.Context = saved })

	rest.Context = nil
	if ctx() == nil {
		t.Error("ctx() must never return nil")
	}
	type key struct{}
	rest.Context = context.WithValue(context.Background(), key{}, "v")
	if ctx().Value(key{}) != "v" {
		t.Error("ctx() must return rest.Context when set")
	}
}
