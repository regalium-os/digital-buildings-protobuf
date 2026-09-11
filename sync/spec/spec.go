// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package spec reads sync/spec.yaml, the single ontology revision this
// schema is generated from, and checks it against the checkout under
// modules/. Stage 1: it imports nothing else in the generator.
//
// Rule 9: a stale pin puts a revision into every generated banner that the
// files were never generated from, and nothing else in the build can see that
// it is false.
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Pin is the pinned upstream revision.
type Pin struct {
	Repository string `yaml:"repository"`
	Commit     string `yaml:"commit"`
	Date       string `yaml:"date"`
	Path       string `yaml:"path"`
}

type file struct {
	Ontology Pin `yaml:"ontology"`
}

var (
	sha  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	date = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Load reads and validates the pin. A malformed pin is an error here rather
// than a confusing failure later: every banner in the tree quotes it.
func Load(path string) (Pin, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Pin{}, fmt.Errorf("read pin: %w", err)
	}
	var f file
	if err := yaml.Unmarshal(b, &f); err != nil {
		return Pin{}, fmt.Errorf("parse %s: %w", path, err)
	}
	p := f.Ontology
	switch {
	case p.Repository == "":
		return Pin{}, fmt.Errorf("%s: ontology.repository is empty", path)
	case !sha.MatchString(p.Commit):
		return Pin{}, fmt.Errorf("%s: ontology.commit %q is not a 40-character sha", path, p.Commit)
	case !date.MatchString(p.Date):
		return Pin{}, fmt.Errorf("%s: ontology.date %q is not YYYY-MM-DD", path, p.Date)
	case p.Path == "":
		return Pin{}, fmt.Errorf("%s: ontology.path is empty", path)
	}
	return p, nil
}

// Short is the abbreviated commit used in banners.
func (p Pin) Short() string { return p.Commit[:7] }

// Resources is the absolute path to the ontology resource tree.
func (p Pin) Resources(checkout string) string {
	return filepath.Join(checkout, filepath.FromSlash(p.Path))
}

// Banner is the provenance line every generated file carries.
func (p Pin) Banner() string {
	return fmt.Sprintf("revision %s (%s)", p.Date, p.Short())
}

// Check verifies the checkout is at the pinned revision, on the pinned date,
// with a clean tree. All three matter: a moved checkout, an edited pin and a
// dirty working tree each produce banners that name a revision the files were
// not generated from.
func (p Pin) Check(checkout string) error {
	head, err := git(checkout, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != p.Commit {
		return fmt.Errorf("%s is at %s, pin says %s\n"+
			"  fix: git -C %s checkout %s   (or update sync/spec.yaml)",
			checkout, head[:7], p.Short(), checkout, p.Commit)
	}
	got, err := git(checkout, "show", "-s", "--format=%cs", "HEAD")
	if err != nil {
		return err
	}
	if got != p.Date {
		return fmt.Errorf("%s commits on %s, pin says %s\n"+
			"  the date is the commit's, not the day the pin was edited",
			p.Short(), got, p.Date)
	}
	dirty, err := git(checkout, "status", "--porcelain")
	if err != nil {
		return err
	}
	if dirty != "" {
		return fmt.Errorf("%s has local modifications; generated banners would\n"+
			"name a revision that does not match what was read:\n%s", checkout, dirty)
	}
	if _, err := os.Stat(p.Resources(checkout)); err != nil {
		return fmt.Errorf("ontology.path %q missing under %s", p.Path, checkout)
	}
	return nil
}

func git(dir string, args ...string) (string, error) {
	out, err := run(dir, args...)
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(out), nil
}
