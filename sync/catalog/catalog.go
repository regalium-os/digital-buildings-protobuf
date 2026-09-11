// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package catalog is where the ontology and AIP disagree, and where the
// ontology's own abbreviations are spelled out. Stage 5: pure tables, no
// logic beyond lookup.
//
// Rule 1: if a linter objects, fix the generator -- and for a name, fixing the
// generator means adding a row here. Every entry naming something the ontology
// no longer has is a build error (model.CheckCatalogue), not a rename that
// quietly stops applying.
package catalog

// FieldRenames maps an ontology field literal to the proto field name.
//
// This table is nearly empty, and that is a finding rather than an oversight.
// A sweep of all 1,565 literals reports zero AIP-140 preposition violations
// and zero whole-name reserved-word collisions, because Digital Buildings
// field names are already lower_snake_case built from a controlled subfield
// vocabulary. The four entries are AIP-142: a google.protobuf.Timestamp field
// must end in _time, and these are the only literals typed as one. See
// TimeFields, which is what makes them Timestamps in the first place.
var FieldRenames = map[string]string{
	// AIP-140 bans prepositions in field names, and `over` is one. The
	// ontology's `over` here is not a preposition at all -- it is half of
	// "overvoltage", an electrical term the subfield table happens to split.
	// Fusing the two words says the same thing and stops the linter reading
	// "over" as a relation between things.
	"source1_over_voltage_status":               "source1_overvoltage_status",
	"source1_phase1_phase2_over_voltage_status": "source1_phase1_phase2_overvoltage_status",
	"source1_phase2_phase3_over_voltage_status": "source1_phase2_phase3_overvoltage_status",
	"source1_phase3_phase1_over_voltage_status": "source1_phase3_phase1_overvoltage_status",
	"source2_over_voltage_status":               "source2_overvoltage_status",
	"source2_phase1_phase2_over_voltage_status": "source2_phase1_phase2_overvoltage_status",
	"source2_phase2_phase3_over_voltage_status": "source2_phase2_phase3_overvoltage_status",
	"source2_phase3_phase1_over_voltage_status": "source2_phase3_phase1_overvoltage_status",

	"next_event_start_timestamp":    "next_event_start_time",
	"next_event_end_timestamp":      "next_event_end_time",
	"ongoing_event_start_timestamp": "ongoing_event_start_time",
	"ongoing_event_end_timestamp":   "ongoing_event_end_time",
}

// TimeFields are literals typed as google.protobuf.Timestamp rather than as
// the raw epoch offset the ontology describes.
//
// The ontology defines its `timestamp` point type as "an instant in time,
// represented as a numeric offset from the epoch", which states neither the
// epoch nor the unit. These could have stayed int64 and tripped no rule --
// AIP-142 flags Timestamp fields not ending in _time, not fields ending in
// _timestamp. They are converted anyway, because an unqualified epoch offset
// is a puzzle for the consumer and the type solves it. The rename above is a
// consequence, not the motive.
var TimeFields = map[string]bool{
	"next_event_start_timestamp":    true,
	"next_event_end_timestamp":      true,
	"ongoing_event_start_timestamp": true,
	"ongoing_event_end_timestamp":   true,
}

// AcceptedTraps are literals that look like they trip an AIP rule but do not,
// recorded so the next reader does not "fix" them. Each was checked against
// the linter's own source, not against a recollection of the AIP.
//
// The value is why the rule does not apply.
var AcceptedTraps = map[string]string{
	// 164 literals end in _status. AIP-216's linter rule iterates over *enum*
	// names and flags those called Status or ending in Status. It does not
	// look at fields. The generated enum is named ...State to satisfy it; the
	// field keeps the ontology's name.
	"*_status": "AIP-216 core::0216::synonyms applies to enum names, not fields",

	// 216 literals contain `return`, `static` or `switch` as a subfield.
	// AIP-140's reserved-word rule matches the *whole* field name against its
	// list, so bypass_return_air_temperature_sensor is not a collision.
	"*_return_*": "AIP-140 core::0140::reserved-words matches whole field names",
}

// TypeRenames maps an entity type name to a message name, where the derived
// one would collide or read wrongly. Keyed by GUID, because the ontology's
// GUID is stable across upstream renames and a name-keyed entry is the one
// that stops applying silently (rule 2).
var TypeRenames = map[string]string{
	// PHYSICAL_SECURITY/REQUEST_TO_EXIT. AIP-136 bans prepositions in method
	// names and AIP-140 bans them in field names, so GetRequestToExit and
	// request_to_exit_id both trip. "Request to exit" is the industry term for
	// the sensor, and ExitRequestSensor is the same thing without the
	// preposition.
	"f7fb1157-ad14-4723-b060-3d4cfab881ba": "ExitRequestSensor",
}

// BooleanStates are the state sets modelled as a bool rather than an enum.
//
// Deliberately empty. 15 distinct two-state sets cover 565 fields, and a rule
// collapsing them would look like an obvious simplification: it is not one.
// OPEN/CLOSED has no agreed mapping to true/false, and PRESENT/ABSENT and
// NORMAL/REVERSED are worse -- a consumer that guesses wrong inverts a damper.
// A bool also has no third state, so an upstream bump adding UNKNOWN (which
// has already happened to ten fields) would be a wire break rather than an
// added enum value.
//
// The key is the sorted, comma-joined state set. Add an entry only with the
// reason recorded beside it.
var BooleanStates = map[string]string{}

// Rename returns the proto field name for an ontology literal.
func Rename(literal string) string {
	if to, ok := FieldRenames[literal]; ok {
		return to
	}
	return literal
}

// IsTime reports whether a literal is typed as a Timestamp.
func IsTime(literal string) bool { return TimeFields[literal] }
