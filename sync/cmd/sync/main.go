// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Command sync regenerates protobuf/ from the pinned ontology revision.
// Nothing under protobuf/ is written by hand (rule 2).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/emit"
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
)

func main() {
	var (
		survey  = flag.Bool("survey", false, "report what was parsed and every AIP naming trap, then exit")
		skipPin = flag.Bool("skip-pin-check", false, "read the checkout without verifying it matches the pin")
	)
	flag.Parse()

	if err := run(*survey, *skipPin); err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
}

func run(survey, skipPin bool) error {
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
	if survey {
		return report(os.Stdout, pin, onto, m)
	}
	ledger, err := ordinals.Load(filepath.Join(root, ledgerPath))
	if err != nil {
		return err
	}
	written, err := emit.New(onto, ledger, pin.Banner()).Run(root, m)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %d files across %d packages from ontology %s\n",
		written, len(emit.Packages(m)), pin.Banner())
	return nil
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
