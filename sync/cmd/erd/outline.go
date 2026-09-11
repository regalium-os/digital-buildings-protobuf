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
)

// The map view: namespace -> general type -> canonical variant, at whatever
// depth the scope asks for. The three levels are the ontology's own, and the
// column headings say which is which -- the thing a reader most needs is to
// stop confusing a general type with a canonical one.

func drawMap(w io.Writer, o *ontology.Ontology, m *model.Model,
	s scope, fields int) error {
	switch {
	case s.empty():
		drawNamespaces(w, m)
	case s.segment == "":
		drawNamespace(w, o, m, s)
	default:
		for _, r := range inScope(m, s) {
			drawResource(w, o, r, fields)
		}
	}
	return nil
}

// drawNamespaces is level one: the thirteen namespaces plus global, each with
// what it carries. Resources are collapsed to one line because 112 of them
// expanded is a census, not a map -- `erd electrical` is the next step.
func drawNamespaces(w io.Writer, m *model.Model) {
	byNS := map[string][]*model.Resource{}
	for _, r := range m.Resources {
		byNS[r.Namespace] = append(byNS[r.Namespace], r)
	}
	names := make([]string, 0, len(byNS))
	for ns := range byNS {
		names = append(names, ns)
	}
	sort.Slice(names, func(i, j int) bool {
		if a, b := len(byNS[names[i]]), len(byNS[names[j]]); a != b {
			return a > b
		}
		return names[i] < names[j]
	})

	fmt.Fprintf(w, "%d namespaces, %d resources, %d canonical variants\n\n",
		len(names), len(m.Resources), variantCount(m.Resources))
	fmt.Fprintf(w, "  %-18s %9s %9s %9s   %s\n",
		"namespace", "resources", "variants", "fields", "largest resource")
	for _, ns := range names {
		rs := byNS[ns]
		fmt.Fprintf(w, "  %-18s %9d %9d %9d   %s\n",
			ns, len(rs), variantCount(rs), fieldCount(rs), largest(rs))
	}
	fmt.Fprintf(w, "\n  %-18s %9d %9d %9d\n",
		"", len(m.Resources), variantCount(m.Resources),
		fieldCount(m.Resources))
	fmt.Fprintf(w, "\nplus %d read-only ontology facets: %s\n",
		len(m.Facets), strings.Join(m.Facets, " "))
	fmt.Fprintln(w, "\nnext: erd <namespace>, e.g. erd electrical")
}

// drawNamespace is level two: one namespace's general types, each with the
// canonical types that specialise it.
func drawNamespace(w io.Writer, o *ontology.Ontology, m *model.Model, s scope) {
	rs := inScope(m, s)
	fmt.Fprintf(w, "%s — %d resources, %d canonical variants\n",
		s.namespace, len(rs), variantCount(rs))
	if o.IsSpaceNamespace(strings.ToUpper(s.namespace)) {
		fmt.Fprintln(w, "\nThis namespace declares no GENERALTYPES.yaml, so its"+
			" entity types are\nthemselves the resources (rule 6).")
	}
	fmt.Fprintf(w, "\n  %-12s %-30s %7s %7s  %s\n",
		"tag", "message", "var", "fields", "package segment")
	for _, r := range rs {
		fmt.Fprintf(w, "  %-12s %-30s %7d %7d  %s\n",
			r.Key.Name, r.Message, len(r.Variants), len(r.Fields), r.Segment)
	}
	for _, r := range rs {
		fmt.Fprintln(w)
		drawVariants(w, r)
	}
	fmt.Fprintf(w, "\nnext: erd %s/<segment>, e.g. erd %s/%s\n",
		s.namespace, s.namespace, rs[0].Segment)
}

// drawResource is level three: one resource in full.
func drawResource(w io.Writer, o *ontology.Ontology, r *model.Resource, n int) {
	fmt.Fprintf(w, "%s/%s\n\n", r.Namespace, r.Segment)
	fmt.Fprintf(w, "  general type   %s\n", r.Key)
	fmt.Fprintf(w, "  guid           %s\n", r.GUID)
	fmt.Fprintf(w, "  message        %s\n", r.Message)
	fmt.Fprintf(w, "  enum           %sType\n", r.Message)
	fmt.Fprintf(w, "  service        %s\n", r.ServiceName())
	fmt.Fprintf(w, "  package        %s\n", r.Package)
	fmt.Fprintf(w, "  resource name  %s\n", r.Pattern())
	fmt.Fprintf(w, "  kind           %s\n", kindName(r.Kind))
	if r.Description != "" {
		fmt.Fprintf(w, "\n  %s\n", wrap(r.Description, 72, "  "))
	}

	fmt.Fprintln(w, "\n  methods")
	for _, meth := range r.Methods() {
		fmt.Fprintf(w, "    %-6s %-48s %s\n", meth.Verb, meth.Path, meth.Name)
	}

	fmt.Fprintln(w)
	drawVariants(w, r)
	drawMixins(w, o, r)
	drawFields(w, r, n)
}

// drawVariants lists the canonical types, which rule 7 makes enum values
// rather than messages. The `implements` list is the whole reason a variant
// has more fields than it declares.
func drawVariants(w io.Writer, r *model.Resource) {
	if len(r.Variants) == 0 {
		fmt.Fprintf(w, "  %s — no canonical variants; the general type is the"+
			" resource on its own\n", r.Message)
		return
	}
	fmt.Fprintf(w, "  %sType — %d canonical %s\n",
		r.Message, len(r.Variants), plural(len(r.Variants), "variant"))
	for _, v := range r.Variants {
		fmt.Fprintf(w, "    %-34s %2d uses %2d opt_uses   implements %s\n",
			v.Key.Name, len(v.Uses), len(v.OptUses),
			strings.Join(short(v.Implements), " "))
	}
}

// drawMixins is the abstract functional groups the variants share. They are
// not resources and not enum values -- they are where the fields come from,
// and a reader looking for "why does this message have 450 fields" is looking
// for this table.
func drawMixins(w io.Writer, o *ontology.Ontology, r *model.Resource) {
	count := map[string]int{}
	for _, v := range r.Variants {
		for _, ref := range v.Implements {
			count[ref]++
		}
	}
	if len(count) == 0 {
		return
	}
	type row struct {
		ref  string
		n    int
		desc string
	}
	rows := make([]row, 0, len(count))
	for ref, n := range count {
		rows = append(rows, row{ref, n, describe(o, ref)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].ref < rows[j].ref
	})
	fmt.Fprintf(w, "\n  abstract functional groups the variants implement (%d)\n",
		len(rows))
	for _, x := range rows {
		fmt.Fprintf(w, "    %-28s %3d %-9s %s\n",
			x.ref, x.n, plural(x.n, "variant"), clip(x.desc, 46))
	}
}

// drawFields is the union rule 7 puts on the resource: every field any variant
// uses, all optional. The behaviour column is rule 8 -- where the field came
// from is what decides whether anything may write it.
func drawFields(w io.Writer, r *model.Resource, n int) {
	if n == 0 || len(r.Fields) == 0 {
		return
	}
	shown := r.Fields
	if n > 0 && len(shown) > n {
		shown = shown[:n]
	}
	writable := 0
	for _, f := range r.Fields {
		if f.Writable() {
			writable++
		}
	}
	fmt.Fprintf(w, "\n  %d standard fields — %d writable, %d read-only\n",
		len(r.Fields), writable, len(r.Fields)-writable)
	fmt.Fprintf(w, "    %-44s %-26s %-12s %s\n",
		"field", "type", "point type", "behaviour")
	for _, f := range shown {
		name := f.Name
		if f.Enumerated {
			name = "repeated " + name
		}
		fmt.Fprintf(w, "    %-44s %-26s %-12s %s\n",
			clip(name, 44), clip(f.ProtoType(), 26), f.Def.PointType,
			strings.Join(f.Behaviors(), "+"))
	}
	if len(shown) < len(r.Fields) {
		fmt.Fprintf(w, "    ... and %d more (-fields -1 for all)\n",
			len(r.Fields)-len(shown))
	}
}

func describe(o *ontology.Ontology, ref string) string {
	ns, name, found := strings.Cut(strings.TrimPrefix(ref, "/"), "/")
	if !found {
		ns, name = ontology.Global, ns
	}
	if e, ok := o.Types[ontology.TypeKey{Namespace: ns, Name: name}]; ok {
		return e.Description
	}
	return ""
}

// short drops the namespace from a reference in the namespace being printed,
// which is most of them, and keeps it where it is the point.
func short(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		_, name, found := strings.Cut(strings.TrimPrefix(ref, "/"), "/")
		if !found {
			name = ref
		}
		out = append(out, name)
	}
	return out
}

func kindName(k model.Kind) string {
	if k == model.Equipment {
		return "equipment (implements EQUIPMENT)"
	}
	return "not equipment (a system, zone or grouping)"
}

func variantCount(rs []*model.Resource) int {
	n := 0
	for _, r := range rs {
		n += len(r.Variants)
	}
	return n
}

func fieldCount(rs []*model.Resource) int {
	n := 0
	for _, r := range rs {
		n += len(r.Fields)
	}
	return n
}

func largest(rs []*model.Resource) string {
	best := rs[0]
	for _, r := range rs[1:] {
		if len(r.Fields) > len(best.Fields) {
			best = r
		}
	}
	return fmt.Sprintf("%s (%d fields)", best.Message, len(best.Fields))
}

// plural is the difference between "1 variants" and "1 variant". A generated
// line that reads as broken English reads as a generated line.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// wrap breaks a description onto lines, so that a 300-character ontology
// sentence does not become a 300-character terminal line.
func wrap(s string, width int, indent string) string {
	var b strings.Builder
	col := 0
	for i, word := range strings.Fields(s) {
		switch {
		case i == 0:
		case col+1+len(word) > width:
			b.WriteString("\n" + indent)
			col = 0
		default:
			b.WriteString(" ")
			col++
		}
		b.WriteString(word)
		col += len(word)
	}
	return b.String()
}
