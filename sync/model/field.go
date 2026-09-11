// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"slices"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/catalog"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// Rules 8 and 13 in one place: whether anything may write a field, and what
// type it has.
//
// Both are decisions about the *ontology* rather than about protobuf text, so
// they belong at stage 6 with the rest of the model. They used to live in
// sync/emit, which was fine while the schema was the only target. It stopped
// being fine when sync/internal/docs arrived: two stage-9 targets each
// deciding a field's type would drift, and the drift would be a reference
// documenting a type the schema does not have. Neither target decides now --
// they both read the answer from here.

// Behaviors is the AIP-203 field_behavior set for this field.
//
// Rule 8: where a field comes from decides whether it can be written. The
// ontology keeps two field files and the split is the answer.
func (f Field) Behaviors() []string {
	if f.Def.Origin == ontology.Metadata {
		// fields/metadata_fields.yaml: 25 literals, labels and design
		// capacities. Configuration -- set at create, never reported.
		return []string{"OPTIONAL", "IMMUTABLE"}
	}
	switch f.Def.PointType {
	case "command", "setpoint":
		return []string{"OPTIONAL"}
	default:
		// sensor, alarm, status, mode, accumulator, counter, count,
		// timestamp, label, capacity, requirement, specification. The
		// equipment reports it and no API call sets it.
		//
		// `mode` is read-only despite reading like a control: the ontology
		// defines it as an observed "distinct mode of operation", and the
		// thing that sets a mode is a `command`.
		return []string{"OUTPUT_ONLY"}
	}
}

// Writable reports whether an update_mask may name this field. Rule 8: naming
// an OUTPUT_ONLY field is rejected rather than silently ignored.
func (f Field) Writable() bool {
	return !slices.Contains(f.Behaviors(), "OUTPUT_ONLY")
}

// ProtoType is the field's proto type. Four shapes, and the ontology decides
// which:
//   - a timestamp point type becomes google.protobuf.Timestamp (rule 11)
//   - a multistate becomes the shared enum for its state set (rule 13)
//   - a numeric field with a measurement subfield is a double
//   - a numeric field without one is an integer, and a bare literal a string
func (f Field) ProtoType() string {
	if catalog.IsTime(f.Literal) {
		return "google.protobuf.Timestamp"
	}
	switch f.Def.Kind {
	case ontology.Multistate:
		return StateEnumName(f.Def.States)
	case ontology.Numeric:
		if f.Def.Dimensionless() {
			// AIP-141: int32/int64, never unsigned. A fixed_min of 0 is
			// restored as a protovalidate range, not as an unsigned type.
			return "int64"
		}
		return "double"
	default:
		return "string"
	}
}

// StateEnumSuffix names the multistate enums, and is deliberately not "State".
//
// core::0216::state-field-output-only flags every field whose enum type ends
// in "State" unless the field is OUTPUT_ONLY, and says outright that the field
// name is ignored. Rule 8 makes command and setpoint fields writable and many
// carry multistates, so "State" would put api-linter in permanent conflict
// with the ontology -- 394 findings' worth. "Value" trips neither rule.
const StateEnumSuffix = "Value"

// StateEnumName is the shared enum for a set of states.
//
// Rule 13: 602 multistate fields draw on only 40 distinct sets, so the
// generator emits 40 enums rather than 602 -- and equality between two fields
// with the same states becomes expressible. The name is built from the states
// themselves, so the same set always yields the same enum.
func StateEnumName(states []string) string {
	return naming.Message(strings.Join(SortedStates(states), "_")) + StateEnumSuffix
}

// StateEnumFile is the file a multistate enum lives in, without the .proto
// extension. Rule 2 puts one enum in one file, so the emitter that writes it
// and the reference that lists it have to agree on the name.
func StateEnumFile(enum string) string { return naming.Package(enum) }

// StateKey identifies a state set, for grouping fields that share one.
func StateKey(states []string) string { return strings.Join(SortedStates(states), ",") }

// SortedStates is the canonical order of a state set: the enum's value order,
// and the order the name is built from.
func SortedStates(states []string) []string {
	out := append([]string(nil), states...)
	sort.Strings(out)
	return out
}
