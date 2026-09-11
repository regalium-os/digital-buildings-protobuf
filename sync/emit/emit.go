// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package emit writes the .proto tree. Stage 10, the last one: everything it
// needs has been decided by the time it runs.
//
// Rule 2: nothing under protobuf/ is written by hand, and every file carries a
// DO NOT EDIT banner naming the ontology revision it came from.
package emit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ordinals"
)

// Emitter renders a model.
type Emitter struct {
	onto   *ontology.Ontology
	ledger *ordinals.Ledger
	banner string
	// stateNames memoises the enum name for a state set, so the same set gets
	// the same name in every package that copies it (rule 4).
	stateNames map[string]string
}

// New starts an emitter.
func New(o *ontology.Ontology, l *ordinals.Ledger, banner string) *Emitter {
	return &Emitter{onto: o, ledger: l, banner: banner, stateNames: map[string]string{}}
}

// Run writes every .proto under root, removing what the previous run left.
//
// The removal matters: the generator writes over what it produces but would
// never delete what it no longer produces, so a package for a general type the
// ontology dropped would survive every later run -- and then fail the build
// with an import nothing provides, long after the change that caused it.
//
// It clears .proto files rather than the whole directory, because docs owns
// the README.md files beside them (sync/emit/tree.go).
func (e *Emitter) Run(root string, m *model.Model) (int, error) {
	dir := Root(root)
	if err := Clear(dir, IsProto); err != nil {
		return 0, err
	}
	files := []*File{annotations(e.banner)}
	for _, r := range m.Resources {
		files = append(files, e.resourceFiles(r)...)
	}
	written := 0
	for _, f := range files {
		if f == nil {
			continue
		}
		if err := write(root, f); err != nil {
			return 0, err
		}
		written++
	}
	// Retire numbers whose field the ontology no longer has, once per message
	// and only after every Assign has run.
	for _, r := range m.Resources {
		present := map[string]bool{}
		for _, fl := range r.Fields {
			present[fl.Name] = true
		}
		e.ledger.Retire(r.Package+"."+r.Message, present)
	}
	if err := Prune(dir); err != nil {
		return 0, err
	}
	return written, e.ledger.Save()
}

// resourceFiles is one package: the message, its enums, the shared value types
// it references, and its service.
func (e *Emitter) resourceFiles(r *model.Resource) []*File {
	files := []*File{
		e.message(r),
		e.connection(r),
		e.link(r),
		e.translation(r),
		e.service(r),
		e.messages(r),
	}
	if len(r.Variants) > 0 {
		files = append(files, e.typeEnum(r))
	}
	files = append(files, e.stateEnums(r)...)
	return files
}

func write(root string, f *File) error {
	p := filepath.Join(root, filepath.FromSlash(f.Path))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir for %s: %w", f.Path, err)
	}
	if err := os.WriteFile(p, []byte(f.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", f.Path, err)
	}
	return nil
}

// Count reports how many files a model would produce, for the survey.
func Packages(m *model.Model) []string {
	out := []string{annotationsPkg}
	for _, r := range m.Resources {
		out = append(out, r.Package)
	}
	sort.Strings(out)
	return out
}
