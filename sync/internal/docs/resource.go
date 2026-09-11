// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package docs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// resource renders one package's README: what the resource is, where it came
// from in the ontology, what is in each file beside it, its fields, and its
// canonical types.
func (r *Renderer) resource(res *model.Resource) string {
	var d doc
	d.line(r.bannerBlock())
	d.heading(1, res.Message)
	if res.Description != "" {
		d.line(strings.Join(strings.Fields(res.Description), " "))
	}
	d.table(nil2(), summaryRows(res))

	d.heading(2, "Files")
	d.line("Rule 2: one message per file, plus its enums. `messages.proto` is the " +
		"single exception, because a request and its response are meaningless apart.")
	d.table([]string{"File", "Contains"}, fileRows(res))

	d.heading(2, "Service")
	d.line(fmt.Sprintf(
		"`%s` in `service.proto`. Every resource is a collection (rule 7): there "+
			"are no singletons here, so it gets the full standard method set plus "+
			"AIP-164 `Undelete`.\n\n"+
			"AIP-131 names the service for the collection rather than `%sService`; "+
			"rule 1 has why buf STANDARD is not selected.", res.ServiceName(), res.Message))
	d.table([]string{"Method", "HTTP"}, methodRows(res))

	r.fields(&d, res)
	r.variants(&d, res)

	d.heading(2, "Provenance")
	d.line(fmt.Sprintf(
		"Generated from `%s` in Google's Digital Buildings ontology.\n\n"+
			"- [Ontology concepts](https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology.md)\n"+
			"- [Abstract model](https://github.com/google/digitalbuildings/blob/master/ontology/docs/model.md)\n"+
			"- [Building configuration](https://github.com/google/digitalbuildings/blob/master/ontology/docs/building_config.md)",
		res.Key))
	return d.String()
}

// nil2 is a two-column header with no titles: the summary is a definition
// list, and GitHub has no Markdown for one.
func nil2() []string { return []string{" ", " "} }

func summaryRows(res *model.Resource) [][]string {
	writable := 0
	for _, f := range res.Fields {
		if f.Writable() {
			writable++
		}
	}
	kind := "equipment"
	if res.Kind == model.NonEquipment {
		kind = "not equipment (a system, a riser, a zone grouping)"
	}
	return [][]string{
		{"Ontology type", code(res.Key.String())},
		{"GUID", code(res.GUID)},
		{"Kind", kind},
		{"Proto package", code(res.Package)},
		{"Resource name", code(res.Pattern())},
		{"Fields", fmt.Sprintf("%s, %s writable", plural(len(res.Fields), "field", "fields"), itoa(writable))},
		{"Canonical types", itoa(len(res.Variants))},
	}
}

// fileRows lists the package's files in the order the emitter writes them,
// which is also the order a reader wants them: the message first.
func fileRows(res *model.Resource) [][]string {
	seg := res.Segment
	rows := [][]string{
		{code(seg + ".proto"), "the `" + res.Message + "` message"},
	}
	if len(res.Variants) > 0 {
		rows = append(rows, []string{
			code(seg + "_type.proto"),
			fmt.Sprintf("`%sType`, one value per canonical type (rule 7)", res.Message),
		})
	}
	rows = append(rows,
		[]string{code("service.proto"), "the `" + res.ServiceName() + "` service"},
		[]string{code("messages.proto"), "its request and response messages"},
		[]string{code("connection.proto"), "`Connection` -- how this entity relates to another (rule 4)"},
		[]string{code("link.proto"), "`Link` -- this entity's fields mapped onto another's"},
		[]string{code("field_translation.proto"), "`FieldTranslation`, `UnitMapping`, `StateMapping`, `ValueRange`"},
	)
	for _, name := range stateEnumFiles(res) {
		rows = append(rows, []string{code(name + ".proto"), "a multistate enum (rule 13)"})
	}
	return rows
}

// methodRows reads model.Methods -- the same list the service emitter renders,
// so the reference cannot document a route the service does not serve.
func methodRows(res *model.Resource) [][]string {
	ms := res.Methods()
	rows := make([][]string, 0, len(ms))
	for _, m := range ms {
		rows = append(rows, []string{
			code(m.Name),
			code(strings.ToUpper(m.Verb) + " " + m.Path),
		})
	}
	return rows
}

// stateEnumFiles is the set of multistate enum files in this package, sorted.
// It is derived the same way the emitter derives it -- from the distinct state
// sets across the resource's fields -- so the two cannot disagree.
func stateEnumFiles(res *model.Resource) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range res.Fields {
		if f.Def.Kind != ontology.Multistate {
			continue
		}
		name := model.StateEnumName(f.Def.States)
		file := model.StateEnumFile(name)
		if !seen[file] {
			seen[file] = true
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}
