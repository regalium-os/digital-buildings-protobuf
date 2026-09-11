// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Command erd explains the pinned ontology as an entity-relationship model:
// what the primary objects are, how a namespace narrows to a resource, and
// which relationships hold between them.
//
// It reads modules/digitalbuildings and nothing else. In particular it never
// looks at protobuf/ -- a view built by re-reading the emitted schema would
// agree with the schema by construction, and so could not show that the
// schema says something the ontology does not (docs/generator.md).
//
// The four levels the views share, and the ontology term behind each:
//
//	namespace     ELECTRICAL          the ontology's own thirteen, plus global
//	general type  ATS                 declared in GENERALTYPES.yaml
//	canonical     ATS_SRC1_SRC2       is_canonical: true, a variant of ATS
//	abstract      SS, IOBM, PWM       is_abstract: true, a functional group
//
// Rule 7 cuts between the second and third: the general type is the resource
// (the message), the canonical types are enum values on it, and the abstract
// groups are the mixins whose fields both inherit. That cut is the single
// thing this command exists to make visible.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/load"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/spec"
)

const (
	pinPath  = "sync/spec.yaml"
	checkout = "modules/digitalbuildings"
)

func main() {
	var (
		view = flag.String("view", "map",
			"map, objects, or mermaid")
		fields = flag.Int("fields", 12,
			"standard fields to show per resource (0 for none, -1 for all)")
		skipPin = flag.Bool("skip-pin-check", false,
			"read the checkout without verifying it matches the pin")
	)
	flag.Usage = usage
	flag.Parse()

	if err := run(*view, flag.Arg(0), *fields, *skipPin); err != nil {
		fmt.Fprintf(os.Stderr, "erd: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "usage: erd [flags] [scope]")
	fmt.Fprintln(out, `
Scope narrows the output, and is one of:

    (empty)                      every namespace, resources collapsed
    electrical                   one namespace, resources and variants
    electrical/generator         one resource, in full

Views:

    -view map        the namespace -> resource -> variant breakdown
    -view objects    the primary objects and how they relate
    -view mermaid    the same relationships as a Mermaid erDiagram

Flags:`)
	flag.PrintDefaults()
}

// scope is the part of the model a run is about. Both fields empty means the
// whole thing.
type scope struct {
	namespace string // hvac, electrical, global, ...
	segment   string // fan_coil_unit, generator, ... empty for a whole namespace
}

func (s scope) empty() bool { return s.namespace == "" }

func (s scope) String() string {
	switch {
	case s.empty():
		return "every namespace"
	case s.segment == "":
		return s.namespace
	}
	return s.namespace + "/" + s.segment
}

// selects reports whether a resource is in scope.
func (s scope) selects(r *model.Resource) bool {
	switch {
	case s.empty():
		return true
	case r.Namespace != s.namespace:
		return false
	case s.segment == "":
		return true
	}
	return r.Segment == s.segment
}

func run(view, arg string, fields int, skipPin bool) error {
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
	sc, err := resolve(m, arg)
	if err != nil {
		return err
	}

	fmt.Printf("ontology %s — %s\n\n", pin.Banner(), sc)
	switch view {
	case "map":
		return drawMap(os.Stdout, onto, m, sc, fields)
	case "objects":
		return drawObjects(os.Stdout, onto, m, sc)
	case "mermaid":
		return drawMermaid(os.Stdout, onto, m, sc, fields)
	}
	return fmt.Errorf("unknown view %q: want map, objects or mermaid", view)
}

// resolve turns the positional argument into a scope, and fails with the
// candidates rather than with "not found" -- the whole point of the command is
// that the reader does not yet know what is in there (rule 15).
func resolve(m *model.Model, arg string) (scope, error) {
	if arg == "" {
		return scope{}, nil
	}
	ns, seg, _ := strings.Cut(strings.ToLower(arg), "/")
	s := scope{namespace: ns, segment: seg}
	for _, r := range m.Resources {
		if s.selects(r) {
			return s, nil
		}
	}
	if seg == "" {
		return s, fmt.Errorf("no namespace %q; try one of: %s",
			ns, strings.Join(namespaces(m), " "))
	}
	var segs []string
	for _, r := range m.Resources {
		if r.Namespace == ns {
			segs = append(segs, r.Segment)
		}
	}
	if len(segs) == 0 {
		return s, fmt.Errorf("no namespace %q; try one of: %s",
			ns, strings.Join(namespaces(m), " "))
	}
	return s, fmt.Errorf("no resource %q in %s; try one of: %s",
		seg, ns, strings.Join(segs, " "))
}

// namespaces lists the package-form namespaces that carry a resource, which
// is the ontology's thirteen plus global (rule 6).
func namespaces(m *model.Model) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range m.Resources {
		if !seen[r.Namespace] {
			seen[r.Namespace] = true
			out = append(out, r.Namespace)
		}
	}
	sort.Strings(out)
	return out
}

// inScope is every resource the scope selects, in package order.
func inScope(m *model.Model, s scope) []*model.Resource {
	var out []*model.Resource
	for _, r := range m.Resources {
		if s.selects(r) {
			out = append(out, r)
		}
	}
	return out
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
			return "", fmt.Errorf(
				"no %s in any parent of the working directory", pinPath)
		}
		dir = parent
	}
}
