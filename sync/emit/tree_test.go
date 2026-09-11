// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule 16: protobuf/ has two roots. sync owns protobuf/digitalbuildings/ and
// wipes it every run; protobuf/extensions/ is hand-written and must survive.
//
// These are the tests for a failure that has already happened once in a
// smaller form. The emitter began with a single os.RemoveAll of the whole
// generated tree, which was correct while sync was the only writer and
// silently wrong the moment cmd/docs existed: `just sync` deleted 113
// README.md files it had not written and could not replace. The same mistake
// made one directory higher would delete the extensions root, and nothing in
// the build would say so -- `just ci` would pass, because a missing hand-
// written package is not a lint error, only an absence.
//
// So the scoping is asserted rather than commented.

// TestRootIsScopedToDigitalBuildings pins the one function that decides what
// the generator is allowed to destroy.
func TestRootIsScopedToDigitalBuildings(t *testing.T) {
	got := Root("/repo")
	want := filepath.Join("/repo", "protobuf", "digitalbuildings")
	if got != want {
		t.Fatalf("Root(%q) = %q, want %q\n\nWidening this to protobuf/ would "+
			"make every sync delete the extensions root.", "/repo", got, want)
	}
	if filepath.Base(got) != "digitalbuildings" {
		t.Fatalf("Root must end in digitalbuildings, got %q", got)
	}
}

// TestClearLeavesTheExtensionsRootAlone is the regression proper: a full
// generator pass over a tree that also contains hand-written packages.
func TestClearLeavesTheExtensionsRootAlone(t *testing.T) {
	repo := t.TempDir()

	generated := filepath.Join(repo, "protobuf", "digitalbuildings", "hvac", "fan_coil_unit", "v1")
	handWritten := filepath.Join(repo, "protobuf", "extensions", "points", "v1")
	writeFile(t, filepath.Join(generated, "fan_coil_unit.proto"), "generated")
	writeFile(t, filepath.Join(generated, "README.md"), "generated reference")
	writeFile(t, filepath.Join(handWritten, "point_set.proto"), "hand-written")
	writeFile(t, filepath.Join(handWritten, "README.md"), "hand-written notes")

	// What `just sync` does.
	if err := Clear(Root(repo), IsProto); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if err := Prune(Root(repo)); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// Everything under extensions/ survives, untouched.
	assertContent(t, filepath.Join(handWritten, "point_set.proto"), "hand-written")
	assertContent(t, filepath.Join(handWritten, "README.md"), "hand-written notes")

	// The generated proto is gone; the generated README is not sync's to
	// delete -- cmd/docs owns that extension.
	assertGone(t, filepath.Join(generated, "fan_coil_unit.proto"))
	assertContent(t, filepath.Join(generated, "README.md"), "generated reference")
}

// TestClearOwnsOneExtensionEach checks the split that replaced the original
// RemoveAll: sync clears .proto, docs clears README.md, neither touches the
// other's files.
func TestClearOwnsOneExtensionEach(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "protobuf", "digitalbuildings", "hvac", "fan", "v1")
	writeFile(t, filepath.Join(dir, "fan.proto"), "schema")
	writeFile(t, filepath.Join(dir, "README.md"), "reference")

	if err := Clear(Root(repo), IsReadme); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	assertGone(t, filepath.Join(dir, "README.md"))
	assertContent(t, filepath.Join(dir, "fan.proto"), "schema")
}

// TestPruneRemovesEmptyPackagesButKeepsTheRoot covers the case rule 9 of
// docs/spec.md warns about: a general type losing its last canonical variant
// removes a package, and an empty directory left behind is still a published
// import path.
func TestPruneRemovesEmptyPackagesButKeepsTheRoot(t *testing.T) {
	repo := t.TempDir()
	root := Root(repo)
	dropped := filepath.Join(root, "hvac", "retired_type", "v1")
	kept := filepath.Join(root, "hvac", "fan", "v1")
	writeFile(t, filepath.Join(dropped, "retired_type.proto"), "gone next run")
	writeFile(t, filepath.Join(kept, "fan.proto"), "still here")

	if err := Clear(root, IsProto); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if err := Prune(root); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// Both packages emptied, so both directories go -- including the hvac/
	// parent that now holds nothing.
	assertGone(t, dropped)
	assertGone(t, filepath.Join(root, "hvac"))

	// The root itself stays, so the next write has somewhere to go.
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("Prune removed the root itself: %v", err)
	}
}

// TestClearOnMissingTreeIsNotAnError covers a first run, where nothing has
// been generated yet.
func TestClearOnMissingTreeIsNotAnError(t *testing.T) {
	repo := t.TempDir()
	if err := Clear(Root(repo), IsProto); err != nil {
		t.Fatalf("Clear on a missing tree: %v", err)
	}
	if err := Prune(Root(repo)); err != nil {
		t.Fatalf("Prune on a missing tree: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s should have survived: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s should have been removed", path)
	}
}
