// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/describe"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// resourceType is the AIP-123 service name every resource type string uses.
const resourceType = "digitalbuildings.googleapis.com"

// message writes the resource message: name, uid, code, entity type, the
// connection list, then the field union.
func (e *Emitter) message(r *model.Resource) *File {
	f := newFile(r.Dir, naming.Field(r.Message), r.Package, e.banner)
	f.importPath(importFieldBehavior)
	f.importPath(importFieldInfo)
	f.importPath(importResource)
	f.importPath(importValidate)
	annotationImports(f)
	// Same proto package, but protobuf still resolves types per *file*: rule 2
	// puts one message per file, so the shared value types have to be imported
	// even though they are siblings.
	f.importPath(r.Dir + "/connection.proto")
	f.importPath(r.Dir + "/link.proto")
	f.importPath(r.Dir + "/field_translation.proto")
	if len(r.Variants) > 0 {
		f.importPath(r.Dir + "/" + naming.Field(r.Message+"Type") + ".proto")
	}

	desc := r.Description
	if desc == "" {
		desc = "A " + strings.ToLower(spaced(r.Message)) + "."
	}
	f.comment(0, fmt.Sprintf("%s\n\n"+
		"Ontology entity type `%s`, a general type carrying %d canonical "+
		"variant(s). Rule 7: the general type is the resource and each variant "+
		"is a value of %sType, so this message holds the union of every field "+
		"any variant uses -- all of them optional.\n\n"+
		"Reference: Digital Buildings abstract model. "+
		"https://github.com/google/digitalbuildings/blob/master/ontology/docs/model.md",
		desc, r.Key, len(r.Variants), r.Message))

	f.line(0, "message %s {", r.Message)
	f.line(1, "option (google.api.resource) = {")
	f.line(2, "type: \"%s/%s\"", resourceType, r.Message)
	f.line(2, "pattern: \"%s\"", r.Pattern())
	f.line(2, "singular: \"%s\"", r.Singular())
	f.line(2, "plural: \"%s\"", r.PluralName())
	f.line(1, "};")
	f.line(1, "option (protobuf.digitalbuildings.annotations.v1.entity_type) = {")
	f.line(2, "guid: \"%s\"", r.GUID)
	f.line(2, "namespace: \"%s\"", r.Key.Namespace)
	f.line(2, "general_type: \"%s\"", r.Key.Name)
	f.line(1, "};")
	f.line(0, "")

	e.standardFields(f, r)
	e.unionFields(f, r)

	if res := e.ledger.ReservedFor(r.Package + "." + r.Message); len(res) > 0 {
		f.line(0, "")
		f.comment(1, "Numbers whose field the ontology no longer declares. Never "+
			"reissued: a consumer may still hold a message that used one.")
		f.line(1, "reserved %s;", joinInts(res))
	}
	f.line(0, "}")
	return f
}

// unionFields writes the ontology fields, each with its number from the
// ledger rather than from its position (rule 9).
func (e *Emitter) unionFields(f *File, r *model.Resource) {
	msg := r.Package + "." + r.Message
	for i, fl := range r.Fields {
		if i > 0 {
			f.line(0, "")
		}
		f.comment(1, describe.Summary(e.onto, fl.Def, fl.Literal))
		n := e.ledger.Assign(msg, fl.Name)
		e.field(f, r, fl, n)
	}
}

// field writes one field: type, name, number, and its options.
func (e *Emitter) field(f *File, r *model.Resource, fl model.Field, n int) {
	typ := e.protoType(f, r, fl)
	label := "optional "
	if fl.Enumerated {
		// Rule 12: the ontology's `_1`, `_2` increment is instance data. The
		// base field is emitted once, repeated, and the index lives in the
		// translation rather than in the schema.
		label = "repeated "
	}
	opts := []string{"(google.api.field_behavior) = " + behavior(fl)}
	opts = append(opts, e.annotation(fl))
	if v := validate(fl); v != "" && !fl.Enumerated {
		opts = append(opts, v)
	}
	f.line(1, "%s%s %s = %d [", label, typ, fl.Name, n)
	for i, o := range opts {
		sep := ","
		if i == len(opts)-1 {
			sep = ""
		}
		f.line(2, "%s%s", o, sep)
	}
	f.line(1, "];")
}

// behavior implements rule 8: where a field comes from decides whether it can
// be written.
// behavior renders model.Field's rule 8 answer as proto options. Protobuf has
// no list syntax for a repeated option, so a second behavior is a second
// (google.api.field_behavior) line.
func behavior(fl model.Field) string {
	return strings.Join(fl.Behaviors(), ",\n    (google.api.field_behavior) = ")
}

func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(parts, ", ")
}

// spaced turns FanCoilUnit into "Fan Coil Unit", for a fallback description.
func spaced(s string) string {
	return strings.Join(strings.Split(naming.Field(s), "_"), " ")
}
