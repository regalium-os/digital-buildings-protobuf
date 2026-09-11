// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Two generators write into protobuf/digitalbuildings: sync puts the .proto
// files there and docs puts the README.md files beside them. Each owns its own
// extension and must clear only that.
//
// This used to be one os.RemoveAll of the whole directory, which was correct
// while sync was the only writer and silently wrong the moment docs existed:
// `just sync` deleted 113 README files it had not written and could not
// replace, and the staleness only surfaced as a CI diff.

// Clear deletes every file under dir whose name matches, so a package the
// ontology has dropped does not survive as an import nothing provides. Empty
// directories left behind are pruned, which is what actually removes a dropped
// package once both generators have run.
func Clear(dir string, match func(name string) bool) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && match(d.Name()) {
			return os.Remove(p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("clear %s: %w", dir, err)
	}
	return nil
}

// Prune removes directories under dir that hold nothing, deepest first, so a
// package emptied by Clear leaves no husk. dir itself is kept.
func Prune(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	var dirs []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != dir {
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("prune %s: %w", dir, err)
	}
	// Deepest first, so a directory whose only content was empty directories
	// is itself empty by the time it is considered.
	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], string(os.PathSeparator)) >
			strings.Count(dirs[j], string(os.PathSeparator))
	})
	for _, p := range dirs {
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsProto and IsReadme are the two ownership predicates.
func IsProto(name string) bool  { return strings.HasSuffix(name, ".proto") }
func IsReadme(name string) bool { return name == "README.md" }

// Root is the generated tree both commands write into.
func Root(repo string) string { return filepath.Join(repo, "protobuf", "digitalbuildings") }
