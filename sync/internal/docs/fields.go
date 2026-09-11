// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package docs

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// fields renders the field table: the part of the page a reader actually came
// for. One row per field, in the order the message declares them.
func (r *Renderer) fields(d *doc, res *model.Resource) {
	if len(res.Fields) == 0 {
		return
	}
	d.heading(2, "Fields")
	d.line("Every field is `optional` (rule 7): the resource carries the union of " +
		"every field any of its canonical types uses, and no single device reports " +
		"all of them.\n\n" +
		"**Writable** is rule 8. A `command` or a `setpoint` is `OPTIONAL` and can " +
		"be set; everything else is `OUTPUT_ONLY`, because the equipment reports it " +
		"and no API call sets it. An `update_mask` naming an `OUTPUT_ONLY` field is " +
		"rejected rather than silently ignored.")

	rows := make([][]string, 0, len(res.Fields))
	for _, f := range res.Fields {
		rows = append(rows, []string{
			code(f.Name),
			code(protoTypeCell(f)),
			writableCell(f),
			r.unitsCell(f),
			code(f.Literal),
			r.number(res, f),
		})
	}
	d.table([]string{"Field", "Type", "Writable", "Unit / states", "Ontology literal", "Field no."}, rows)
	r.states(d, res)
}

// protoTypeCell shows rule 12's repeated fields as repeated, because a
// consumer reading "double" and finding a list has been misled.
func protoTypeCell(f model.Field) string {
	t := f.ProtoType()
	if f.Enumerated {
		return "repeated " + t
	}
	return t
}

func writableCell(f model.Field) string {
	if f.Writable() {
		if f.Def.Origin == ontology.Metadata {
			// Configuration: settable once, at create.
			return "at create"
		}
		return "yes"
	}
	return "no"
}

// unitsCell says what the number means. For a quantity that is the ontology's
// standard unit -- pinned into the schema, never carried on the wire beside
// the value, because two producers could then disagree about what the number
// means. For a multistate it is the enum.
func (r *Renderer) unitsCell(f model.Field) string {
	switch f.Def.Kind {
	case ontology.Multistate:
		return code(model.StateEnumName(f.Def.States))
	case ontology.Numeric:
		if f.Def.Measurement == "" {
			return "count"
		}
		if k, ok := r.onto.Units[f.Def.Measurement]; ok && k.Standard != "" {
			return code(k.Standard)
		}
		return f.Def.Measurement
	default:
		return "--"
	}
}

// states lists the multistate enums this package emits and what each means, so
// the field table's enum names resolve without opening a second file.
func (r *Renderer) states(d *doc, res *model.Resource) {
	sets := map[string][]string{}
	for _, f := range res.Fields {
		if f.Def.Kind == ontology.Multistate {
			sets[model.StateEnumName(f.Def.States)] = model.SortedStates(f.Def.States)
		}
	}
	if len(sets) == 0 {
		return
	}
	d.heading(3, "Multistate enums")
	d.line(fmt.Sprintf(
		"%s, one per distinct state set rather than one per field. Across the "+
			"ontology 602 multistate fields draw on only 40 sets, so an enum per "+
			"field would mean 305 different spellings of {ACTIVE, INACTIVE} that no "+
			"consumer could pass to a common function (rule 13).\n\n"+
			"Each is copied into this package rather than imported: AIP-215 forbids "+
			"a field referencing a type in another proto package (rule 4).",
		plural(len(sets), "enum", "enums")))

	names := make([]string, 0, len(sets))
	for n := range sets {
		names = append(names, n)
	}
	sortStrings(names)

	rows := make([][]string, 0, len(names))
	for _, n := range names {
		vals := make([]string, 0, len(sets[n]))
		for _, s := range sets[n] {
			vals = append(vals, "`"+s+"`")
		}
		rows = append(rows, []string{
			code(model.StateEnumFile(n) + ".proto"),
			code(n),
			strings.Join(vals, ", "),
		})
	}
	d.table([]string{"File", "Enum", "States"}, rows)
}

// implementsCell lists the abstract functional groups a variant composes.
//
// This is the column that says what the variant *is*. A canonical type's own
// `uses` list is almost always empty -- it declares no fields directly and
// inherits every one of them through `implements` -- so a count of it would
// read 0 for nearly all 1,587 types and tell a reader nothing.
func implementsCell(v model.Variant) string {
	if len(v.Implements) == 0 {
		return "--"
	}
	names := make([]string, 0, len(v.Implements))
	for _, im := range v.Implements {
		if i := strings.LastIndex(im, "/"); i >= 0 {
			im = im[i+1:]
		}
		names = append(names, "`"+im+"`")
	}
	return strings.Join(names, ", ")
}

// variants lists the canonical types. Rule 7: a canonical type is a value on
// the resource's enum, not a message of its own -- 1,587 messages would not be
// a service surface.
func (r *Renderer) variants(d *doc, res *model.Resource) {
	if len(res.Variants) == 0 {
		return
	}
	d.heading(2, "Canonical types")
	d.line(fmt.Sprintf(
		"%s, each a value of `%sType`. The ontology's name is kept verbatim, "+
			"because that name *is* the identifier.\n\n"+
			"Each value carries its exact `implements` / `uses` / `opt_uses` sets in "+
			"`(annotations.canonical_type)`, so a validator can still reject a "+
			"variant reporting a field its declared type does not have. That is what "+
			"makes the cut in rule 7 lossless.",
		plural(len(res.Variants), "canonical type", "canonical types"), res.Message))

	rows := make([][]string, 0, len(res.Variants))
	for _, v := range res.Variants {
		rows = append(rows, []string{
			code(v.Value),
			code(v.Key.String()),
			implementsCell(v),
			sentence(v.Description),
		})
	}
	d.table([]string{"Enum value", "Ontology type", "Implements", "Description"}, rows)
}
