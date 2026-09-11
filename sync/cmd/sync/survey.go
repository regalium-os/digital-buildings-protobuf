// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/spec"
)

// report prints what the generator parsed. Rule 15: run this after a spec bump
// before anything else -- it is where a removed field or a renamed entity type
// shows up as a number that moved.
func report(w io.Writer, pin spec.Pin, o *ontology.Ontology, m *model.Model) error {
	fmt.Fprintf(w, "ontology %s\n\n", pin.Banner())

	fmt.Fprintln(w, "facets")
	fmt.Fprintf(w, "  namespaces        %5d  %s\n", len(o.Namespaces), strings.Join(o.Namespaces, " "))
	fmt.Fprintf(w, "  subfields         %5d  in %d categories\n", len(o.Subfields), len(o.Categories))
	fmt.Fprintf(w, "  states            %5d\n", len(o.States))
	fmt.Fprintf(w, "  unit kinds        %5d  (%d aliases), %d unit names\n",
		len(o.Units), countAliases(o), countUnits(o))
	fmt.Fprintf(w, "  connections       %5d\n", len(o.Connections))

	var telemetry, metadata int
	byKind := map[ontology.Kind]int{}
	byPoint := map[string]int{}
	dimensionless := 0
	for _, f := range o.Fields {
		if f.Dimensionless() {
			dimensionless++
		}
		if f.Origin == ontology.Telemetry {
			telemetry++
		} else {
			metadata++
		}
		byKind[f.Kind]++
		byPoint[f.PointType]++
	}
	fmt.Fprintf(w, "  field literals    %5d  (%d telemetry, %d metadata)\n",
		len(o.Fields), telemetry, metadata)
	fmt.Fprintf(w, "    numeric         %5d\n", byKind[ontology.Numeric])
	fmt.Fprintf(w, "    multistate      %5d  in %d distinct state sets\n",
		byKind[ontology.Multistate], len(stateSets(o)))
	fmt.Fprintf(w, "    bare            %5d\n", byKind[ontology.Bare])
	fmt.Fprintf(w, "    dimensionless   %5d  numeric with no measurement subfield: integers\n", dimensionless)

	var abstract, canonical int
	for _, e := range o.Types {
		if e.IsAbstract {
			abstract++
		}
		if e.IsCanonical {
			canonical++
		}
	}
	fmt.Fprintf(w, "  entity types      %5d  (%d abstract, %d canonical)\n",
		len(o.Types), abstract, canonical)

	fmt.Fprintln(w, "\npoint types (rule 8: command and setpoint are writable, the rest OUTPUT_ONLY)")
	for _, p := range sortedByCount(byPoint) {
		fmt.Fprintf(w, "  %-16s %5d\n", p.name, p.n)
	}

	reportResources(w, m)

	fmt.Fprintln(w, "\nstate sets (rule 13: one shared enum per distinct set)")
	sets := stateSets(o)
	for _, s := range sortedByCount(sets) {
		fmt.Fprintf(w, "  %5d  %s\n", s.n, s.name)
	}
	return nil
}

func countAliases(o *ontology.Ontology) int {
	n := 0
	for _, k := range o.Units {
		if k.AliasOf != "" {
			n++
		}
	}
	return n
}

func countUnits(o *ontology.Ontology) int {
	seen := map[string]bool{}
	for _, k := range o.Units {
		for _, u := range k.Units {
			seen[u.Name] = true
		}
	}
	return len(seen)
}

// stateSets counts how many fields share each distinct set of states.
func stateSets(o *ontology.Ontology) map[string]int {
	out := map[string]int{}
	for _, f := range o.Fields {
		if f.Kind != ontology.Multistate {
			continue
		}
		s := append([]string(nil), f.States...)
		sort.Strings(s)
		out[strings.Join(s, ",")]++
	}
	return out
}

type counted struct {
	name string
	n    int
}

func sortedByCount(m map[string]int) []counted {
	out := make([]counted, 0, len(m))
	for k, v := range m {
		out = append(out, counted{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].name < out[j].name
	})
	return out
}

// reportResources prints rule 7's cut: what became a resource, how many
// variants it absorbed, and how wide the union got.
func reportResources(w io.Writer, m *model.Model) {
	var equip, other int
	byNS := map[string]int{}
	for _, r := range m.Resources {
		if r.Kind == model.NonEquipment {
			other++
		} else {
			equip++
		}
		byNS[r.Namespace]++
	}
	fmt.Fprintf(w, "\nresources %d (%d equipment, %d not) + %d ontology facets\n",
		len(m.Resources), equip, other, len(m.Facets))
	fmt.Fprint(w, "  by namespace: ")
	for _, c := range sortedByCount(byNS) {
		fmt.Fprintf(w, "%s=%d ", c.name, c.n)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %-34s %8s %8s %8s\n", "package", "variants", "fields", "enum'd")
	rs := append([]*model.Resource(nil), m.Resources...)
	sort.Slice(rs, func(i, j int) bool { return len(rs[i].Fields) > len(rs[j].Fields) })
	for _, r := range rs {
		enum := 0
		for _, f := range r.Fields {
			if f.Enumerated {
				enum++
			}
		}
		fmt.Fprintf(w, "  %-34s %8d %8d %8d\n",
			r.Namespace+"/"+r.Segment+" "+r.Message, len(r.Variants), len(r.Fields), enum)
	}
}
