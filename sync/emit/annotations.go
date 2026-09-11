// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import "github.com/oh-tarnished/digital-buildings-protobuf/sync/model"

// annotationsPackage is the one custom vocabulary rule 5 permits.
const (
	annotationsPkg  = "protobuf.digitalbuildings.annotations.v1"
	annotationsDir  = "protobuf/digitalbuildings/annotations/v1"
	annotationsFile = annotationsDir + "/annotations.proto"
)

// Extension field numbers. The 1000-99999 range is for organisation-internal
// extensions; these are stable and must never be reused.
const (
	extField         = 50001
	extEntityType    = 50002
	extCanonicalType = 50003
)

// annotations writes the rule 5 vocabulary: three extensions, one package.
//
// The argument for it is the ontology's own: a field's identity is the *set of
// its subfields*, not its name -- "Applications should depend on the field
// set, not the string value." A proto field called zone_air_temperature_sensor
// has thrown that set away, and a consumer cannot test it for equivalence
// against air_zone_temperature_sensor without re-parsing the name against a
// subfield table it does not have.
func annotations(banner string) *File {
	f := newFile(annotationsDir, "annotations", annotationsPkg, banner)
	f.importPath(importDescriptor)

	f.comment(0, "Provenance the ontology carries and protobuf has nowhere to put.\n\n"+
		"This is the only custom vocabulary in the schema, and rule 5 of "+
		"CLAUDE.md is the argument for it. Provenance that a human reads still "+
		"goes in comments; what is here is what a *program* needs at runtime.")
	f.line(0, "extend google.protobuf.FieldOptions {")
	f.comment(1, "What this field is, in the ontology's own terms.")
	f.line(1, "FieldInfo field = %d;", extField)
	f.line(0, "}")
	f.line(0, "")

	f.line(0, "extend google.protobuf.MessageOptions {")
	f.comment(1, "The entity type this message was generated from.")
	f.line(1, "EntityTypeInfo entity_type = %d;", extEntityType)
	f.line(0, "}")
	f.line(0, "")

	f.line(0, "extend google.protobuf.EnumValueOptions {")
	f.comment(1, "The canonical type this enum value stands for, with the exact "+
		"field sets that make rule 7's flattening lossless.")
	f.line(1, "CanonicalTypeInfo canonical_type = %d;", extCanonicalType)
	f.line(0, "}")
	f.line(0, "")

	fieldInfoMessage(f)
	entityTypeMessage(f)
	canonicalTypeMessage(f)
	return f
}

func fieldInfoMessage(f *File) {
	f.comment(0, "A standard field's identity, as the ontology defines it.")
	f.line(0, "message FieldInfo {")
	f.comment(1, "The ontology literal, e.g. `zone_air_temperature_sensor`. This is "+
		"the name before any AIP rename, and is what a building config's "+
		"translation block keys on.")
	f.line(1, "string literal = 1;")
	f.line(0, "")
	f.comment(1, "The subfields composing the literal, in the ontology's own "+
		"construction order. This -- not the name -- is the field's identity: "+
		"two fields with the same set are equivalent however they are spelled.")
	f.line(1, "repeated string subfields = 2;")
	f.line(0, "")
	f.comment(1, "The terminating subfield: sensor, command, setpoint, alarm, ... "+
		"It decides whether the field is writable (rule 8).")
	f.line(1, "string point_type = 3;")
	f.line(0, "")
	f.comment(1, "The measurement subfield, empty for a multistate or a count. It "+
		"names the quantity kind and therefore the units.")
	f.line(1, "string measurement = 4;")
	f.line(0, "")
	f.comment(1, "The ontology's standard unit for that quantity. Values on the "+
		"wire are always in this unit -- it is pinned, not negotiated, because "+
		"two producers disagreeing about what a number means is the one failure "+
		"the ontology is careful to prevent.")
	f.line(1, "string standard_unit = 5;")
	f.line(0, "")
	f.comment(1, "Whether some entity type enumerates this field (`_1`, `_2`), "+
		"which is why it is repeated. See rule 12.")
	f.line(1, "bool enumerated = 6;")
	f.line(0, "")
	f.comment(1, "The ontology's expected range, where it declares one that is "+
		"not a hard constraint. A reading outside it is a fault to report, not "+
		"a message to reject, so it is documented here rather than enforced by "+
		"protovalidate (rule 13).")
	f.line(1, "optional double expected_min = 7;")
	f.line(0, "")
	f.comment(1, "Upper end of that expected range.")
	f.line(1, "optional double expected_max = 8;")
	f.line(0, "}")
	f.line(0, "")
}

func entityTypeMessage(f *File) {
	f.comment(0, "The general type a resource was generated from (rule 7).")
	f.line(0, "message EntityTypeInfo {")
	f.comment(1, "The ontology's own GUID for the entity type. Stable across "+
		"upstream renames, which is why the catalogue keys on it -- and why it "+
		"is also the resource's `uid`.")
	f.line(1, "string guid = 1;")
	f.line(0, "")
	f.comment(1, "The ontology namespace, e.g. `HVAC`.")
	f.line(1, "string namespace = 2;")
	f.line(0, "")
	f.comment(1, "The general type tag, e.g. `FCU`.")
	f.line(1, "string general_type = 3;")
	f.line(0, "}")
	f.line(0, "")
}

func canonicalTypeMessage(f *File) {
	f.comment(0, "A canonical type, carried on the enum value that stands for it.\n\n"+
		"This is what makes rule 7 lossless. Flattening 1,587 canonical types "+
		"into enum values on 112 resources gives up *compile-time* enforcement "+
		"of which fields a variant may set -- but not the information. A "+
		"validator reads these sets and rejects the same payloads a message per "+
		"canonical type would have refused to compile.")
	f.line(0, "message CanonicalTypeInfo {")
	f.comment(1, "The ontology's GUID for this canonical type.")
	f.line(1, "string guid = 1;")
	f.line(0, "")
	f.comment(1, "The namespace-qualified type, e.g. `HVAC/FCU_DFSS_CSP_CHWDC`.\n\n"+
		"Not `qualified_name`: AIP-122 bans the `_name` suffix on anything that "+
		"is not a resource name, and this is an ontology type path.")
	f.line(1, "string qualified_type = 2;")
	f.line(0, "")
	f.comment(1, "The abstract functional groups this type implements, "+
		"namespace-qualified.\n\n"+
		"The ontology calls this `implements`, which AIP-140 rejects as a "+
		"reserved word in a common language.")
	f.line(1, "repeated string implemented_types = 3;")
	f.line(0, "")
	f.comment(1, "Field literals the type requires. A conforming entity of this "+
		"type reports all of them.")
	f.line(1, "repeated string uses = 4;")
	f.line(0, "")
	f.comment(1, "Field literals the type may report but need not.")
	f.line(1, "repeated string opt_uses = 5;")
	f.line(0, "}")
}

// annotationImports adds the vocabulary import to a file that uses it.
func annotationImports(f *File) { f.importPath(annotationsFile) }

// unused keeps the model import referenced if the file is trimmed; it costs
// nothing and keeps the package boundary explicit.
var _ = model.Facets
