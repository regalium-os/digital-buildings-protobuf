// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package load walks the ontology resource tree and returns its bytes,
// grouped by facet. Stage 2: it reads files and does not parse them, so that
// a change in the ontology's *layout* fails here rather than as a confusing
// parse error in stage 3.
package load

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The ontology's own file names, and the shapes it keeps them in. Every one
// is required: a missing facet is an ontology this generator does not
// recognise, not a facet to skip.
const (
	subfieldsFile   = "subfields/subfields.yaml"
	statesFile      = "states/states.yaml"
	unitsFile       = "units/units.yaml"
	connectionsFile = "connections/connections.yaml"
	telemetryFile   = "fields/telemetry_fields.yaml"
	metadataFile    = "fields/metadata_fields.yaml"
)

// entityDir is where a namespace keeps its entity types. The global namespace
// is the exception and keeps them directly under entity_types/.
const entityDir = "entity_types"

// globalNamespace is the ontology's own name for the namespace every other
// one may reference. It is a directory name here only by convention.
const globalNamespace = "GLOBAL"

// Source is one entity-type file: which namespace declared it, and what the
// ontology called the file. Both survive into diagnostics, because "a
// duplicate type" is only actionable if you know which two files declared it.
type Source struct {
	Namespace string // HVAC, LIGHTING, ... or GLOBAL
	File      string // ABSTRACT, GENERALTYPES, FCU, ...
	Path      string // path as read, for error messages
	Data      []byte
}

// Tree is the whole resource directory, read but not parsed.
type Tree struct {
	Subfields   []byte
	States      []byte
	Units       []byte
	Connections []byte
	Telemetry   []byte
	Metadata    []byte
	Entities    []Source
	Namespaces  []string // declared namespaces, excluding GLOBAL, sorted
}

// Read walks root, which is the ontology's resources directory.
func Read(root string) (*Tree, error) {
	t := &Tree{}
	for _, f := range []struct {
		name string
		dst  *[]byte
	}{
		{subfieldsFile, &t.Subfields},
		{statesFile, &t.States},
		{unitsFile, &t.Units},
		{connectionsFile, &t.Connections},
		{telemetryFile, &t.Telemetry},
		{metadataFile, &t.Metadata},
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.name)))
		if err != nil {
			return nil, fmt.Errorf("ontology facet %s: %w", f.name, err)
		}
		*f.dst = b
	}
	if err := t.readEntities(root); err != nil {
		return nil, err
	}
	if len(t.Entities) == 0 {
		return nil, fmt.Errorf("no entity type files under %s", root)
	}
	return t, nil
}

// readEntities collects every entity-type file. A namespace is a directory
// holding an entity_types/ subdirectory; the global namespace is
// entity_types/ at the top level.
func (t *Tree) readEntities(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read %s: %w", root, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == entityDir {
			if err := t.collect(root, globalNamespace, filepath.Join(root, name)); err != nil {
				return err
			}
			continue
		}
		dir := filepath.Join(root, name, entityDir)
		if _, err := os.Stat(dir); err != nil {
			continue // subfields/, states/, units/, connections/, fields/
		}
		if err := t.collect(root, name, dir); err != nil {
			return err
		}
		t.Namespaces = append(t.Namespaces, name)
	}
	sort.Strings(t.Namespaces)
	sort.Slice(t.Entities, func(i, j int) bool {
		if t.Entities[i].Namespace != t.Entities[j].Namespace {
			return t.Entities[i].Namespace < t.Entities[j].Namespace
		}
		return t.Entities[i].File < t.Entities[j].File
	})
	return nil
}

func (t *Tree) collect(root, namespace, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			rel = p
		}
		t.Entities = append(t.Entities, Source{
			Namespace: namespace,
			File:      strings.TrimSuffix(e.Name(), ".yaml"),
			Path:      filepath.ToSlash(rel),
			Data:      b,
		})
	}
	return nil
}
