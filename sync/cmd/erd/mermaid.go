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

// The mermaid view: the same relationships as the objects view, as an
// erDiagram that pastes into Markdown.
//
// Which diagram depends on the scope, because there is no one useful drawing
// of 112 resources:
//
//	(no scope)    the meta-model -- how the ontology itself is shaped
//	a namespace   Building and that namespace's resources
//	a resource    one message, its enum, and the value objects beside it

func drawMermaid(w io.Writer, o *ontology.Ontology, m *model.Model,
	s scope, fields int) error {
	switch {
	case s.empty():
		metaModel(w, o, m)
	case s.segment == "":
		namespaceModel(w, m, s)
	default:
		for _, r := range inScope(m, s) {
			resourceModel(w, o, r, fields)
		}
	}
	return nil
}

// metaModel draws the ontology's own shape: the thing a reader needs before
// any individual resource means anything.
func metaModel(w io.Writer, o *ontology.Ontology, m *model.Model) {
	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "erDiagram")
	fmt.Fprintf(w, "    NAMESPACE ||--o{ GENERAL_TYPE : %q\n",
		fmt.Sprintf("declares (%d resources)", len(m.Resources)))
	fmt.Fprintf(w, "    GENERAL_TYPE ||--o{ CANONICAL_TYPE : %q\n",
		fmt.Sprintf("specialised by (%d)", variantCount(m.Resources)))
	fmt.Fprintln(w, `    CANONICAL_TYPE }o--o{ ABSTRACT_GROUP : "implements"`)
	fmt.Fprintln(w, `    GENERAL_TYPE }o--o{ ABSTRACT_GROUP : "implements"`)
	fmt.Fprintln(w, `    ABSTRACT_GROUP }o--o{ STANDARD_FIELD : "uses / opt_uses"`)
	fmt.Fprintln(w, `    GENERAL_TYPE ||--o{ STANDARD_FIELD : "union across variants"`)
	fmt.Fprintln(w, `    STANDARD_FIELD }o--|| POINT_TYPE : "terminates in"`)
	fmt.Fprintln(w, `    STANDARD_FIELD }o--o{ SUBFIELD : "composed of, ordered"`)
	fmt.Fprintln(w, `    STANDARD_FIELD }o--o| UNIT_KIND : "measurement subfield"`)
	fmt.Fprintln(w, `    STANDARD_FIELD }o--o{ STATE_VALUE : "multistate set"`)
	fmt.Fprintln(w, `    BUILDING ||--o{ GENERAL_TYPE : "parent of every resource name"`)
	fmt.Fprintln(w, `    GENERAL_TYPE ||--o{ CONNECTION : "has"`)
	fmt.Fprintln(w, `    CONNECTION }o--|| CONNECTION_TYPE : "typed by"`)
	fmt.Fprintln(w, `    CONNECTION }o--|| GENERAL_TYPE : "names the other end"`)
	fmt.Fprintln(w)
	block(w, "NAMESPACE", [][3]string{
		{"string", "name", "HVAC, ELECTRICAL, … plus GLOBAL"},
		{"int", "general_types", count(len(m.Resources))},
	})
	block(w, "GENERAL_TYPE", [][3]string{
		{"string", "tag", "FCU, ATS, TXMR — 1-4 characters"},
		{"string", "message", "FanCoilUnit — the resource"},
		{"uuid", "guid", "the ontology's own, and the resource uid"},
	})
	block(w, "CANONICAL_TYPE", [][3]string{
		{"string", "name", "FCU_DFSS_CSP_CHWDC — an enum value"},
		{"uuid", "guid", "rides on the enum value in an annotation"},
	})
	block(w, "ABSTRACT_GROUP", [][3]string{
		{"string", "name", count(abstractCount(o)) + " of them; never a message"},
	})
	block(w, "STANDARD_FIELD", [][3]string{
		{"string", "literal", count(len(o.Fields)) + " across two field files"},
		{"bool", "writable", "command and setpoint only (rule 8)"},
	})
	block(w, "SUBFIELD", [][3]string{
		{"string", "literal", count(len(o.Subfields)) + " in " +
			count(len(o.Categories)) + " ordered categories"},
	})
	block(w, "CONNECTION_TYPE", [][3]string{
		{"string", "name", strings.Join(connectionNames(o), ", ")},
	})
	fmt.Fprintln(w, "```")
}

// namespaceModel draws one namespace's resources hanging off Building, which
// is the shape of every resource name in the schema.
func namespaceModel(w io.Writer, m *model.Model, s scope) {
	rs := inScope(m, s)
	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "erDiagram")
	for _, r := range rs {
		fmt.Fprintf(w, "    BUILDING ||--o{ %s : %q\n",
			ident(r.Message), r.PluralName())
	}
	fmt.Fprintln(w)
	for _, r := range rs {
		fmt.Fprintf(w, "    %s ||--o{ CONNECTION : \"connections\"\n",
			ident(r.Message))
	}
	fmt.Fprintln(w)
	block(w, "BUILDING", [][3]string{
		{"string", "name", "buildings/{building} — the root resource"},
	})
	for _, r := range rs {
		block(w, ident(r.Message), [][3]string{
			{"string", "name", r.Pattern()},
			{ident(r.Message) + "Type", "entity_type",
				count(len(r.Variants)) + " canonical " +
					plural(len(r.Variants), "variant")},
			{"fields", "standard", count(len(r.Fields)) + " optional " +
				plural(len(r.Fields), "field")},
		})
	}
	block(w, "CONNECTION", [][3]string{
		{"ConnectionType", "type", "CONTAINS, FEEDS, CONTROLS, …"},
		{"string", "entity", "resource name of the other end, any namespace"},
	})
	fmt.Fprintln(w, "```")
}

// resourceModel draws one resource with the value objects beside it. The
// attributes are real proto fields, in ordinal order, so the drawing and the
// schema cannot disagree.
func resourceModel(w io.Writer, o *ontology.Ontology, r *model.Resource,
	nf int) {
	e := ident(r.Message)
	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "erDiagram")
	fmt.Fprintf(w, "    BUILDING ||--o{ %s : %q\n", e, r.PluralName())
	fmt.Fprintf(w, "    %s ||--o{ CONNECTION : \"connections\"\n", e)
	fmt.Fprintf(w, "    %s ||--o{ LINK : \"links\"\n", e)
	fmt.Fprintf(w, "    %s ||--o| TRANSLATION : \"translation\"\n", e)
	fmt.Fprintf(w, "    TRANSLATION ||--o{ FIELD_TRANSLATION : \"fields\"\n")
	fmt.Fprintf(w, "    FIELD_TRANSLATION ||--o| UNIT_MAPPING : \"units\"\n")
	fmt.Fprintf(w, "    FIELD_TRANSLATION ||--o| STATE_MAPPING : \"states\"\n")
	fmt.Fprintln(w)

	rows := [][3]string{
		{"string", "name", r.Pattern()},
		{"string", "uid", "the ontology GUID, format UUID4"},
		{"string", "code", "the building config entity code"},
		{ident(r.Message) + "Type", "entity_type",
			count(len(r.Variants)) + " canonical " +
				plural(len(r.Variants), "variant")},
	}
	shown := r.Fields
	if nf >= 0 && len(shown) > nf {
		shown = shown[:nf]
	}
	for _, f := range shown {
		typ := lastSegment(f.ProtoType())
		if f.Enumerated {
			// Rule 12: the ontology's numeric increment is one repeated
			// field, not one field per index.
			typ += "[]"
		}
		rows = append(rows, [3]string{typ, f.Name,
			f.Def.PointType + ", " + strings.Join(f.Behaviors(), "+")})
	}
	if len(shown) < len(r.Fields) {
		rows = append(rows, [3]string{"int", "_not_shown",
			fmt.Sprintf("%d further standard fields", len(r.Fields)-len(shown))})
	}
	block(w, e, rows)

	block(w, "CONNECTION", [][3]string{
		{"ConnectionType", "type", strings.Join(connectionNames(o), ", ")},
		{"string", "entity", "resource name, any namespace; reference type *"},
	})
	block(w, "LINK", [][3]string{
		{"string", "source", "resource name of the entity linked from"},
		{"map", "field_map", "that entity's fields onto this one's"},
	})
	block(w, "FIELD_TRANSLATION", [][3]string{
		{"string", "standard_field", "the ontology literal this point feeds"},
		{"string", "raw_field", "the device's own point name"},
		{"int32", "index", "rule 12: enumeration lives here, not in the schema"},
	})
	fmt.Fprintln(w, "```")
}

// block writes a mermaid attribute block. Mermaid wants a bare word for the
// type and the name, so anything else is sanitised rather than emitted and
// silently breaking the render.
func block(w io.Writer, entity string, rows [][3]string) {
	fmt.Fprintf(w, "    %s {\n", entity)
	for _, r := range rows {
		fmt.Fprintf(w, "        %s %s %q\n",
			identType(r[0]), ident(r[1]), strings.ReplaceAll(r[2], `"`, `'`))
	}
	fmt.Fprintln(w, "    }")
}

// identType is ident for the type column, where mermaid also accepts the []
// suffix that says the attribute is a list.
func identType(s string) string {
	if rest, ok := strings.CutSuffix(s, "[]"); ok {
		return ident(rest) + "[]"
	}
	return ident(s)
}

// ident reduces a string to what mermaid accepts unquoted: letters, digits,
// underscore and hyphen.
func ident(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9', c == '_', c == '-':
			b.WriteRune(c)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

// lastSegment turns google.protobuf.Timestamp into Timestamp. The full name is
// right in the schema and wrong in a diagram, where it is a column heading.
func lastSegment(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

func connectionNames(o *ontology.Ontology) []string {
	out := make([]string, 0, len(o.Connections))
	for k := range o.Connections {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func count(v int) string { return fmt.Sprintf("%d", v) }
