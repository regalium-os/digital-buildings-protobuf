// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package describe composes every generated comment. Stage 7.
//
// Rule 3 requires a comment on every message, field, enum, enum value, service
// and RPC, and the ontology documents every subfield, state, unit and entity
// type -- so nothing here is invented. A field's summary is built from its
// own subfields' definitions, which is what makes 1,565 field comments
// possible without writing 1,565 sentences.
package describe

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// Wrap breaks a comment body at width, preserving paragraph breaks. Generated
// comments become the documentation in every target language, so they are
// wrapped rather than emitted as one long line.
func Wrap(s string, width int) []string {
	var out []string
	for i, para := range strings.Split(s, "\n\n") {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, wrapBlock(para, width)...)
	}
	return out
}

// wrapBlock wraps one paragraph. A block whose lines are indented is a list --
// the subfield breakdown of a field comment -- and collapsing it into flowing
// prose would run six definitions together into one unreadable sentence. Those
// keep one entry per line, with a hanging indent for continuations.
func wrapBlock(para string, width int) []string {
	lines := strings.Split(para, "\n")
	isList := false
	for _, l := range lines {
		if strings.HasPrefix(l, "  ") {
			isList = true
			break
		}
	}
	if !isList {
		return wrapParagraph(strings.Fields(para), width)
	}
	var out []string
	for _, l := range lines {
		if !strings.HasPrefix(l, "  ") {
			out = append(out, wrapParagraph(strings.Fields(l), width)...)
			continue
		}
		wrapped := wrapParagraph(strings.Fields(l), width-4)
		for i, w := range wrapped {
			if i == 0 {
				out = append(out, "  "+w)
			} else {
				out = append(out, "    "+w)
			}
		}
	}
	return out
}

func wrapParagraph(words []string, width int) []string {
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			out = append(out, line)
			line = w
			continue
		}
		line += " " + w
	}
	return append(out, line)
}

// Summary is a field's doc comment: the ontology's own definition of each
// subfield, in the order the field grammar puts them, then the facts a
// consumer needs at the point of use.
func Summary(o *ontology.Ontology, f ontology.Field, literal string) string {
	var b strings.Builder
	b.WriteString(sentence(f))

	if defs := subfieldDefs(o, f); defs != "" {
		b.WriteString("\n\n")
		b.WriteString(defs)
	}

	b.WriteString("\n\nOntology field: `")
	b.WriteString(literal)
	b.WriteString("`, point type `")
	b.WriteString(f.PointType)
	b.WriteString("`.")

	if f.Measurement != "" {
		if kind, ok := o.Units[f.Measurement]; ok && kind.Standard != "" {
			fmt.Fprintf(&b, " Measured as %s; values are in %s, the ontology's "+
				"standard unit for that quantity.", f.Measurement, kind.Standard)
		}
	}
	if f.Dimensionless() {
		b.WriteString(" Dimensionless: the ontology defines this point type as " +
			"an integer count rather than a measured quantity.")
	}
	if r := bounds(f); r != "" {
		b.WriteString("\n\n")
		b.WriteString(r)
	}
	b.WriteString("\n\nReference: Digital Buildings ontology. " +
		"https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology.md")
	return b.String()
}

// sentence renders the field name itself as prose, which is what the ontology
// intends: "Field names are intended to read naturally and be self-describing
// based on the composition of subfield definitions."
func sentence(f ontology.Field) string {
	words := strings.Join(f.Subfields[:len(f.Subfields)-1], " ")
	if words == "" {
		return "The " + f.PointType + "."
	}
	return "The " + words + " " + f.PointType + "."
}

// subfieldDefs lists each subfield's own definition, in grammar order. This is
// the whole reason a consumer can read a 450-field message: the name is a
// composition, and the composition is documented.
func subfieldDefs(o *ontology.Ontology, f ontology.Field) string {
	var lines []string
	seen := map[string]bool{}
	for _, s := range f.Subfields {
		if seen[s] {
			continue
		}
		seen[s] = true
		sf, ok := o.Subfields[s]
		if !ok || sf.Description == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s (%s): %s", s, sf.Category, sf.Description))
	}
	if len(lines) == 0 {
		return ""
	}
	return "Composed of:\n" + strings.Join(lines, "\n")
}

// bounds renders the ontology's two range claims, and says which is enforced.
// Rule 13: a fixed bound is a constraint, a flexible one is an expectation,
// and a reader has to be able to tell which they are looking at.
func bounds(f ontology.Field) string {
	var parts []string
	if f.Bounds.FixedMin != nil || f.Bounds.FixedMax != nil {
		parts = append(parts, "Valid range "+span(f.Bounds.FixedMin, f.Bounds.FixedMax)+
			", enforced by protovalidate.")
	}
	if f.Bounds.FlexibleMin != nil || f.Bounds.FlexibleMax != nil {
		parts = append(parts, "Expected range "+span(f.Bounds.FlexibleMin, f.Bounds.FlexibleMax)+
			". This is a data-quality expectation, not a constraint: a reading "+
			"outside it is a fault to report, not a message to reject.")
	}
	return strings.Join(parts, " ")
}

func span(min, max *float64) string {
	switch {
	case min != nil && max != nil:
		return fmt.Sprintf("%s to %s", num(*min), num(*max))
	case min != nil:
		return fmt.Sprintf("from %s", num(*min))
	case max != nil:
		return fmt.Sprintf("up to %s", num(*max))
	}
	return ""
}

func num(f float64) string { return strings.TrimSuffix(fmt.Sprintf("%g", f), ".0") }
