// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/env/add_remove_load_test.go

package env

import (
	"os"
	"path/filepath"
	"testing"
)

// withTempRegConfig points the package-level RegConfigFile at a throwaway file
// for the duration of a test and restores it afterwards.
func withTempRegConfig(t *testing.T) string {
	t.Helper()
	prev := RegConfigFile
	path := filepath.Join(t.TempDir(), "defaultRegistry.json")
	RegConfigFile = path
	t.Cleanup(func() { RegConfigFile = prev })
	return path
}

func TestAddRegAndLoadRoundTrip(t *testing.T) {
	path := withTempRegConfig(t)

	re := RegistryEntry{
		RegistryName:  "registry.example.com:5000",
		Comments:      "test registry",
		Username:      "alice",
		EncodedPasswd: "c2VjcmV0",
	}
	if err := re.AddReg(); err != nil {
		t.Fatalf("AddReg failed: %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected registry file to exist: %v", statErr)
	}

	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if *loaded != re {
		t.Errorf("round trip mismatch: got %+v; want %+v", *loaded, re)
	}
}

func TestLoadAppendsJSONSuffix(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "myreg")
	re := RegistryEntry{RegistryName: "r"}
	prev := RegConfigFile
	RegConfigFile = base + ".json"
	t.Cleanup(func() { RegConfigFile = prev })

	if err := re.AddReg(); err != nil {
		t.Fatalf("AddReg failed: %v", err)
	}

	// Load with a path missing the .json suffix must still find the file.
	loaded, err := Load(base)
	if err != nil {
		t.Fatalf("Load(%q) failed: %v", base, err)
	}
	if loaded.RegistryName != "r" {
		t.Errorf("RegistryName = %q; want %q", loaded.RegistryName, "r")
	}
}

func TestLoadMissingFile(t *testing.T) {
	withTempRegConfig(t) // points at a non-existent file
	if _, err := Load(""); err == nil {
		t.Errorf("Load of a missing file should return an error")
	}
}

func TestRemoveRegDeletesExistingFile(t *testing.T) {
	path := withTempRegConfig(t)

	seed := RegistryEntry{RegistryName: "to-be-removed"}
	if err := seed.AddReg(); err != nil {
		t.Fatalf("seed AddReg failed: %v", err)
	}

	// When the file exists, RemoveReg simply deletes it.
	empty := RegistryEntry{}
	if err := empty.RemoveReg(); err != nil {
		t.Fatalf("RemoveReg failed: %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("expected registry file to be gone, stat err = %v", statErr)
	}
}

func TestRemoveRegRecreatesWhenAbsent(t *testing.T) {
	path := withTempRegConfig(t) // file does not exist yet

	// When the file is already absent, RemoveReg writes back an empty entry.
	empty := RegistryEntry{}
	if err := empty.RemoveReg(); err != nil {
		t.Fatalf("RemoveReg failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after RemoveReg failed: %v", err)
	}
	if loaded.RegistryName != "" {
		t.Errorf("expected empty registry entry, got %q", loaded.RegistryName)
	}
}
