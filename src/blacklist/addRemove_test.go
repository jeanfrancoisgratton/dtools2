// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/blacklist/addRemove_test.go

package blacklist

import (
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// isolate points HOME to a temp dir and silences output; returns the blacklist file path
func isolate(t *testing.T) string {
	t.Helper()
	dir := testutil.IsolateHome(t)
	quiet := rest.QuietOutput
	rest.QuietOutput = true
	t.Cleanup(func() { rest.QuietOutput = quiet })
	return filepath.Join(dir, BlacklistFile)
}

func readFile(t *testing.T, path string) ResourceBlacklist {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rb ResourceBlacklist
	if err := json.Unmarshal(b, &rb); err != nil {
		t.Fatal(err)
	}
	return rb
}

func TestLoadMissingAndEmpty(t *testing.T) {
	path := isolate(t)
	rb, err := Load()
	if err != nil || !reflect.DeepEqual(*rb, ResourceBlacklist{}) {
		t.Errorf("Load() without a file = %+v, %v; want an empty blacklist", rb, err)
	}
	os.WriteFile(path, []byte("  \n"), 0o600)
	if rb, err := Load(); err != nil || !reflect.DeepEqual(*rb, ResourceBlacklist{}) {
		t.Errorf("Load() of a blank file = %+v, %v; want an empty blacklist", rb, err)
	}
	os.WriteFile(path, []byte("{oops"), 0o600)
	if _, err := Load(); err == nil {
		t.Error("Load() of invalid JSON: want an error")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := isolate(t)
	rb := &ResourceBlacklist{Volumes: []string{"pgdata"}, Images: []string{"nginx:latest"}}
	if err := rb.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("blacklist file mode = %v; want 0600", fi.Mode().Perm())
	}
	got, err := Load()
	if err != nil || !reflect.DeepEqual(got, rb) {
		t.Errorf("Load() = %+v, %v; want %+v", got, err, rb)
	}
}

func TestAddRemove(t *testing.T) {
	isolate(t)
	rb := &ResourceBlacklist{}

	if changed, err := rb.Add("volume", "  pgdata  "); err != nil || !changed {
		t.Fatalf("Add = %v, %v; want changed", changed, err)
	}
	if changed, _ := rb.Add("volumes", "pgdata"); changed {
		t.Error("adding a duplicate must report no change")
	}
	if !reflect.DeepEqual(rb.Volumes, []string{"pgdata"}) {
		t.Errorf("Volumes = %v; want [pgdata] (trimmed, no duplicate)", rb.Volumes)
	}
	if _, err := rb.Add("volume", "   "); err == nil {
		t.Error("adding an empty name: want an error")
	}
	if _, err := rb.Add("secrets", "x"); err == nil {
		t.Error("adding to an unknown resource type: want an error")
	}

	if removed, err := rb.Remove("volume", "pgdata"); err != nil || !removed {
		t.Errorf("Remove = %v, %v; want removed", removed, err)
	}
	if removed, _ := rb.Remove("volume", "pgdata"); removed {
		t.Error("removing an absent name must report no removal")
	}
	if len(rb.Volumes) != 0 {
		t.Errorf("Volumes = %v; want empty", rb.Volumes)
	}
}

func TestAddImageNormalizesTag(t *testing.T) {
	isolate(t)
	for _, rt := range []string{"image", "images"} {
		rb := &ResourceBlacklist{}
		if _, err := rb.Add(rt, "nginx"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rb.Images, []string{"nginx:latest"}) {
			t.Errorf("Add(%q, nginx) stored %v; want [nginx:latest]", rt, rb.Images)
		}
	}
}

func TestAddResourceAndDeleteResource(t *testing.T) {
	path := isolate(t)
	if err := AddResource("volumes", []string{"a", "b"}); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	if got := readFile(t, path).Volumes; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("file Volumes = %v; want [a b]", got)
	}
	// Adding an existing name warns but succeeds
	_ = testutil.CaptureStdout(t, func() {
		if err := AddResource("volumes", []string{"a"}); err != nil {
			t.Errorf("AddResource(duplicate): %v", err)
		}
	})

	if removed, err := DeleteResource("volumes", []string{"a"}); err != nil || !removed {
		t.Errorf("DeleteResource = %v, %v; want removed", removed, err)
	}
	if got := readFile(t, path).Volumes; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("file Volumes = %v; want [b]", got)
	}
	if err := AddResource("bogus", []string{"x"}); err == nil {
		t.Error("AddResource to an unknown type: want an error")
	}
}

func TestIsResourceBlackListed(t *testing.T) {
	path := isolate(t)
	os.WriteFile(path, []byte(`{"Volumes":["pgdata"],"Containers":["db"]}`), 0o600)

	cases := []struct {
		rt, name string
		want     bool
	}{
		{"volumes", "pgdata", true},
		{"volume", "pgdata", true},
		{"volumes", "other", false},
		{"containers", "db", true},
		{"networks", "db", false},
	}
	for _, c := range cases {
		got, err := IsResourceBlackListed(c.rt, c.name)
		if err != nil || got != c.want {
			t.Errorf("IsResourceBlackListed(%s, %s) = %v, %v; want %v", c.rt, c.name, got, err, c.want)
		}
	}
	if _, err := IsResourceBlackListed("bogus", "x"); err == nil {
		t.Error("an unknown resource type: want an error")
	}
}

// A resource whose name holds uppercase letters must be protectable
func TestIsResourceBlackListedMixedCase(t *testing.T) {
	isolate(t)
	if err := AddResource("volumes", []string{"MyData"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := IsResourceBlackListed("volumes", "MyData"); !got {
		t.Error("volume MyData was blacklisted, but IsResourceBlackListed(volumes, MyData) is false")
	}
}

// `blacklist add image nginx` stores nginx:latest; `rmi nginx` must still be blocked
func TestIsResourceBlackListedImageWithoutTag(t *testing.T) {
	isolate(t)
	if err := AddResource("image", []string{"nginx"}); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"nginx", "nginx:latest"} {
		if got, _ := IsResourceBlackListed("images", ref); !got {
			t.Errorf("image nginx was blacklisted, but IsResourceBlackListed(images, %s) is false", ref)
		}
	}
}

func TestListFunctions(t *testing.T) {
	path := isolate(t)
	os.WriteFile(path, []byte(`{"Volumes":["pgdata"],"Images":["nginx:latest"]}`), 0o600)

	rb, _ := Load()
	all := rb.ListAll()
	if !reflect.DeepEqual(all["volumes"], []string{"pgdata"}) || len(all["networks"]) != 0 {
		t.Errorf("ListAll = %v", all)
	}
	all["volumes"][0] = "mutated"
	if rb.Volumes[0] != "pgdata" {
		t.Error("ListAll must return copies")
	}
	if l, err := rb.List("image"); err != nil || !reflect.DeepEqual(l, []string{"nginx:latest"}) {
		t.Errorf("List(image) = %v, %v", l, err)
	}
	if _, err := rb.List("bogus"); err == nil {
		t.Error("List(unknown type): want an error")
	}

	out := testutil.CaptureStdout(t, func() {
		if err := ListAllFromFile(); err != nil {
			t.Errorf("ListAllFromFile: %v", err)
		}
	})
	if !strings.Contains(out, "pgdata") || !strings.Contains(out, "nginx:latest") {
		t.Errorf("ListAllFromFile output %q: want both entries", out)
	}

	out = testutil.CaptureStdout(t, func() {
		if err := ListFromFile("volumes"); err != nil {
			t.Errorf("ListFromFile: %v", err)
		}
	})
	if !strings.Contains(out, "pgdata") || strings.Contains(out, "nginx") {
		t.Errorf("ListFromFile(volumes) output %q: want only the volume", out)
	}
	if err := ListFromFile("bogus"); err == nil {
		t.Error("ListFromFile(unknown type): want an error")
	}
}
