// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"sort"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// The shared value types of rule 4. Each is copied into every package that
// needs it rather than imported, because AIP-215 forbids a cross-package
// message reference and this repository declines the google.* exemption --
// protoc-gen-buffers renders google.type.* into neither serialization target,
// silently. They are generated, so nobody maintains the copies.

// connection writes Connection and its ConnectionType enum.
func (e *Emitter) connection(r *model.Resource) *File {
	f := newFile(r.Dir, "connection", r.Package, e.banner)
	f.importPath(importFieldBehavior)
	f.importPath(importResource)

	f.comment(0, "A directed relationship to another entity.\n\n"+
		"Rule 4: a connection is not a field. A floor does not hold its fan "+
		"coil units -- AIP-215 forbids a field referencing a message in another "+
		"proto package, and the other end may be any resource in any namespace. "+
		"So the target is a resource name with a wildcard reference.")
	f.line(0, "message Connection {")
	f.comment(1, "The kind of relationship.")
	f.line(1, "ConnectionType type = 1 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "The resource name of the entity at the other end.\n\n"+
		"`type: \"*\"` because a connection may point at any resource in any "+
		"namespace: a floor CONTAINS equipment of every kind.")
	f.line(1, "string entity = 2 [")
	f.line(2, "(google.api.field_behavior) = REQUIRED,")
	f.line(2, "(google.api.resource_reference).type = \"*\"")
	f.line(1, "];")
	f.line(0, "}")
	f.line(0, "")

	f.comment(0, "The ontology's connection types.")
	f.line(0, "enum ConnectionType {")
	f.comment(1, "Default. The relationship is unspecified.")
	f.line(1, "CONNECTION_TYPE_UNSPECIFIED = 0;")
	names := make([]string, 0, len(e.onto.Connections))
	for n := range e.onto.Connections {
		names = append(names, n)
	}
	sort.Strings(names)
	for i, n := range names {
		f.line(0, "")
		desc := e.onto.Connections[n]
		if n == "PARIALLY_AGGREGATES" {
			desc += "\n\nThe ontology spells this without the second `t`. Kept " +
				"verbatim: it is the wire value, and correcting it here would " +
				"desynchronise this schema from every config written against the " +
				"ontology."
		}
		f.comment(1, desc)
		f.line(1, "%s = %d;", naming.EnumValue("ConnectionType", n), i+1)
	}
	f.line(0, "}")
	return f
}

// link writes Link, the building config's `links` block.
func (e *Emitter) link(r *model.Resource) *File {
	f := newFile(r.Dir, "link", r.Package, e.banner)
	f.importPath(importFieldBehavior)
	f.importPath(importResource)

	f.comment(0, "A standard field mapped from another entity's telemetry.\n\n"+
		"This is the building config's `links` block: a virtual entity does not "+
		"emit telemetry itself, but has a timeseries assembled from reporting "+
		"devices. Same shape as a connection, and for the same reason (rule 4).")
	f.line(0, "message Link {")
	f.comment(1, "The resource name of the entity supplying the data.")
	f.line(1, "string source = 1 [")
	f.line(2, "(google.api.field_behavior) = REQUIRED,")
	f.line(2, "(google.api.resource_reference).type = \"*\"")
	f.line(1, "];")
	f.line(0, "")
	f.comment(1, "This entity's field literal, e.g. `zone_air_temperature_sensor`.")
	f.line(1, "string target_field = 2 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "The source entity's field literal it is taken from.")
	f.line(1, "string source_field = 3 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "}")
	return f
}

// translation writes FieldTranslation and the value types it needs.
func (e *Emitter) translation(r *model.Resource) *File {
	f := newFile(r.Dir, "field_translation", r.Package, e.banner)
	f.importPath(importFieldBehavior)

	f.comment(0, "How one native payload point maps onto a standard field.\n\n"+
		"The building config's `translation` block. A real device's payload "+
		"differs from the modelled type in field names, values, units and "+
		"multistate definitions, and this is where that is reconciled.")
	f.line(0, "message FieldTranslation {")
	f.comment(1, "The standard field literal this maps onto.")
	f.line(1, "string field = 1 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "The enumeration index, where the entity has more than one "+
		"point of this field.\n\n"+
		"Rule 12: the ontology's `_1`, `_2` increment is instance data that "+
		"leaked into the vocabulary, so the schema emits the base field once as "+
		"repeated and the index lives here. A device reporting `_1` and `_7` "+
		"and nothing between records that fact in this field, and the schema "+
		"does not change when it grows a third point.")
	f.line(1, "int32 index = 2 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")
	f.comment(1, "Path to the value in the device's native payload, e.g. "+
		"`points.temp_1.present_value`.")
	f.line(1, "string present_value = 3 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")
	f.comment(1, "Whether the device lacks this field entirely. The building "+
		"config writes `MISSING`.")
	f.line(1, "bool missing = 4 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")
	f.comment(1, "The device's native unit token, mapped to the ontology's "+
		"standard unit.")
	f.line(1, "repeated UnitMapping units = 5 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")
	f.comment(1, "The device's native state values, mapped to standard states.")
	f.line(1, "repeated StateMapping states = 6 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")
	f.comment(1, "The expected range of the value, in the standard unit.")
	f.line(1, "optional ValueRange value_range = 7 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "}")
	f.line(0, "")

	f.comment(0, "A device's native unit token and what it means.")
	f.line(0, "message UnitMapping {")
	f.comment(1, "The ontology's unit name, e.g. `degrees_celsius`.")
	f.line(1, "string unit = 1 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "The token the device's payload uses, e.g. `degC`.")
	f.line(1, "string native_value = 2 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "}")
	f.line(0, "")

	f.comment(0, "A device's native state value and what it means.")
	f.line(0, "message StateMapping {")
	f.comment(1, "The ontology's state name, e.g. `OPEN`.")
	f.line(1, "string state = 1 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "The values the device reports for it. More than one native "+
		"value may map to the same standard state.")
	f.line(1, "repeated string native_values = 2 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "}")
	f.line(0, "")

	f.comment(0, "An expected minimum and maximum, in the standard unit.")
	f.line(0, "message ValueRange {")
	f.comment(1, "Lower bound.")
	f.line(1, "double min_value = 1 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "")
	f.comment(1, "Upper bound.")
	f.line(1, "double max_value = 2 [(google.api.field_behavior) = REQUIRED];")
	f.line(0, "}")
	return f
}

var _ = fmt.Sprintf
