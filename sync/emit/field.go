// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// protoType decides a field's type, and registers whatever import it needs.
//
// Four shapes, and the ontology decides which:
//   - a timestamp point type becomes google.protobuf.Timestamp (rule 11)
//   - a multistate becomes the shared enum for its state set (rule 13)
//   - a numeric field with a measurement subfield is a double
//   - a numeric field without one is an integer, and a bare literal a string
func (e *Emitter) protoType(f *File, r *model.Resource, fl model.Field) string {
	t := fl.ProtoType()
	// The type itself is model.Field's decision (rules 11, 13 and 141); what
	// is left here is the import it costs, which only the emitter knows.
	switch {
	case t == "google.protobuf.Timestamp":
		f.importPath(importTimestamp)
	case fl.Def.Kind == ontology.Multistate:
		f.importPath(r.Dir + "/" + stateEnumFile(t) + ".proto")
	}
	return t
}

// annotation renders the rule 5 field annotation: what the name only implies.
func (e *Emitter) annotation(fl model.Field) string {
	var b strings.Builder
	b.WriteString("(protobuf.digitalbuildings.annotations.v1.field) = {\n")
	fmt.Fprintf(&b, "      literal: %q\n", fl.Literal)
	for _, s := range fl.Def.Subfields {
		fmt.Fprintf(&b, "      subfields: %q\n", s)
	}
	fmt.Fprintf(&b, "      point_type: %q\n", fl.Def.PointType)
	if fl.Def.Measurement != "" {
		fmt.Fprintf(&b, "      measurement: %q\n", fl.Def.Measurement)
		if k, ok := e.onto.Units[fl.Def.Measurement]; ok && k.Standard != "" {
			fmt.Fprintf(&b, "      standard_unit: %q\n", k.Standard)
		}
	}
	if fl.Enumerated {
		b.WriteString("      enumerated: true\n")
	}
	// Rule 13: a flexible bound is an expectation, so it is recorded here and
	// deliberately not turned into a protovalidate constraint.
	if fl.Def.Bounds.FlexibleMin != nil {
		fmt.Fprintf(&b, "      expected_min: %s\n", literalFloat(*fl.Def.Bounds.FlexibleMin))
	}
	if fl.Def.Bounds.FlexibleMax != nil {
		fmt.Fprintf(&b, "      expected_max: %s\n", literalFloat(*fl.Def.Bounds.FlexibleMax))
	}
	b.WriteString("    }")
	return b.String()
}

// validate renders the protovalidate constraint, and only for a fixed bound.
//
// Rule 13: air_pressure_sensor has a flexible range of 500-200000 Pa, and a
// reading outside it is precisely the fault an operator needs to see. Making
// that a constraint would reject the message carrying it.
func validate(fl model.Field) string {
	b := fl.Def.Bounds
	if !b.HasFixed() || fl.Def.Kind != ontology.Numeric {
		return ""
	}
	typ := "double"
	if fl.Def.Dimensionless() {
		typ = "int64"
	}
	var parts []string
	if b.FixedMin != nil {
		parts = append(parts, fmt.Sprintf("(buf.validate.field).%s.gte = %s",
			typ, boundLiteral(*b.FixedMin, typ)))
	}
	if b.FixedMax != nil {
		parts = append(parts, fmt.Sprintf("(buf.validate.field).%s.lte = %s",
			typ, boundLiteral(*b.FixedMax, typ)))
	}
	parts = append(parts, "(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
	return strings.Join(parts, ",\n    ")
}

func boundLiteral(v float64, typ string) string {
	if typ == "int64" {
		return fmt.Sprintf("%d", int64(v))
	}
	return literalFloat(v)
}

// literalFloat renders a float that protobuf's text format accepts. %g would
// emit 1e+06, which the parser rejects for a double option.
func literalFloat(v float64) string {
	s := fmt.Sprintf("%f", v)
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "0"
	}
	return s
}

// The state-set helpers are model's (rule 13). They are re-exported here as
// one-line wrappers so the emitter reads the way it always did, and so there
// is exactly one place that decides what a state set is called.

func (e *Emitter) stateEnumName(states []string) string {
	key := model.StateKey(states)
	if n, ok := e.stateNames[key]; ok {
		return n
	}
	name := model.StateEnumName(states)
	e.stateNames[key] = name
	return name
}

func stateKey(states []string) string { return model.StateKey(states) }

func sortedStates(states []string) []string { return model.SortedStates(states) }
