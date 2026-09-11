// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Command docs regenerates the Markdown reference beside the schema: one
// README.md per package, plus the index that lists them.
//
// It is the second of the two targets built on model.Model (docs/generator.md).
// It reads the same pinned ontology sync does and never parses an emitted
// .proto back in -- a reference built by re-reading the output would agree with
// the output by construction, and so could not catch a generator that emitted
// the wrong thing.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/emit"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/internal/docs"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/load"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ordinals"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/spec"
)

const (
	pinPath    = "sync/spec.yaml"
	checkout   = "modules/digitalbuildings"
	ledgerPath = "sync/ordinals.yaml"
	// samplePath is the specimen docs/generator.md points at. It is written by
	// the generator rather than kept by hand so it cannot drift into
	// advertising a layout the generator stopped producing.
	samplePath   = "docs/sample-reference.md"
	sampleOf     = "protobuf/digitalbuildings/hvac/fan_coil_unit/v1/README.md"
	samplePrefix = "<!--\nA specimen of what `just docs` emits, copied verbatim from\n" +
		sampleOf + ". Kept here because docs/generator.md cites it and\n" +
		"because README.md files under protobuf/ are easy to miss.\nDO NOT EDIT.\n-->\n\n"
)

func main() {
	skipPin := flag.Bool("skip-pin-check", false, "read the checkout without verifying it matches the pin")
	flag.Parse()

	if err := run(*skipPin); err != nil {
		fmt.Fprintf(os.Stderr, "docs: %v\n", err)
		os.Exit(1)
	}
}

func run(skipPin bool) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	pin, err := spec.Load(filepath.Join(root, pinPath))
	if err != nil {
		return err
	}
	dir := filepath.Join(root, checkout)
	if !skipPin {
		if err := pin.Check(dir); err != nil {
			return fmt.Errorf("spec pin: %w", err)
		}
	}
	tree, err := load.Read(pin.Resources(dir))
	if err != nil {
		return err
	}
	onto, err := ontology.Parse(tree)
	if err != nil {
		return err
	}
	m, err := model.Build(onto)
	if err != nil {
		return err
	}

	// Read-only: Render never calls Assign, so this cannot move a slot.
	ledger, err := ordinals.Load(filepath.Join(root, ledgerPath))
	if err != nil {
		return err
	}
	// docs owns the README.md files and clears only those: sync owns the
	// .proto files beside them (sync/emit/tree.go).
	generated := emit.Root(root)
	if err := emit.Clear(generated, emit.IsReadme); err != nil {
		return err
	}
	pages := docs.New(onto, ledger, pin.Banner()).Render(m)
	for _, p := range pages {
		if err := write(filepath.Join(root, p.Path), p.Content); err != nil {
			return err
		}
	}
	if err := emit.Prune(generated); err != nil {
		return err
	}
	if err := sample(root, pages); err != nil {
		return err
	}
	fmt.Printf("wrote %d README files plus %s from ontology %s\n",
		len(pages), samplePath, pin.Banner())
	return nil
}

// sample copies one page to docs/sample-reference.md. FanCoilUnit is the
// specimen because it is the example CLAUDE.md uses throughout, and because it
// is large enough to show every section without being AirHandlingUnit.
func sample(root string, pages []docs.Page) error {
	for _, p := range pages {
		if p.Path == sampleOf {
			return write(filepath.Join(root, samplePath), samplePrefix+p.Content)
		}
	}
	return fmt.Errorf("no page at %s to copy to %s -- has the layout moved?", sampleOf, samplePath)
}

func write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// repoRoot walks up from the working directory looking for the pin, so the
// command works from anywhere in the tree.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, pinPath)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s in any parent of the working directory", pinPath)
		}
		dir = parent
	}
}
