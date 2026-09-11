// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

// The ontology's two range claims, and why they are not the same artefact.

// Bounds are the ontology's two different claims about range. A fixed bound is
// a constraint and becomes a protovalidate rule; a flexible bound is an
// expectation and becomes documentation. Conflating them would reject exactly
// the out-of-range telemetry an operator needs to see (rule 13).
type Bounds struct {
	FixedMin, FixedMax       *float64
	FlexibleMin, FlexibleMax *float64
}

// Dimensionless reports that a numeric field names no measurement subfield,
// and so is a plain integer rather than a quantity.
//
// The ontology states the rule as "a measurement subfield is required for any
// numeric point unless the point type is `count`", but its own subfield
// definitions are broader than that sentence: `counter` is "a special case of
// accumulator that assumes integer values and non-dimensional units", and
// `index` is "an integer which indicates the location of a particular data
// point in a list". Five literals land here -- the four `_counter` fields and
// `scene_index_command` -- and every one is an integer by the ontology's own
// definition, not a field missing its units. They are typed int64 rather than
// double, and `just survey` counts them so that a bump adding a genuinely
// unitless quantity is visible rather than silently integral.
func (f Field) Dimensionless() bool { return f.Kind == Numeric && f.Measurement == "" }

// HasFixed reports whether protovalidate should constrain this field.
func (b Bounds) HasFixed() bool { return b.FixedMin != nil || b.FixedMax != nil }

// Any reports whether there is a range worth documenting at all.
func (b Bounds) Any() bool {
	return b.HasFixed() || b.FlexibleMin != nil || b.FlexibleMax != nil
}
