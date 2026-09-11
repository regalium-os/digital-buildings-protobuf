// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Kind is what a field literal carries, which decides its proto type.
type Kind int

const (
	// Numeric is a dimensional value with bounds. 953 literals.
	Numeric Kind = iota
	// Multistate is an enumerated value with a state list. 602 literals.
	Multistate
	// Bare is a literal with neither: a label, or a count. 10 literals.
	Bare
)

// Origin records which of the two field files a literal came from, which is
// the whole of rule 8's writability answer for metadata.
type Origin int

const (
	// Telemetry is fields/telemetry_fields.yaml: what equipment reports.
	Telemetry Origin = iota
	// Metadata is fields/metadata_fields.yaml: configuration.
	Metadata
)

// Field is one standard field literal.
type Field struct {
	Name      string
	Kind      Kind
	Origin    Origin
	Subfields []string // in the ontology's construction order
	PointType string   // the terminating subfield: sensor, command, ...
	// Measurement is the measurement subfield, empty for a multistate or a
	// count. It names the quantity kind, and so the units.
	Measurement string
	States      []string // sorted, for a Multistate
	Bounds      Bounds
}

// enumSuffix matches the ontology's enumeration increment: zone_air_
// temperature_sensor_1, and the sub-grouping form _2_1. Rule 12 emits the base
// field once as repeated, so the suffix is stripped here and remembered.
var enumSuffix = regexp.MustCompile(`(_\d+)+$`)

// SplitEnumeration returns a field reference's base literal and whether it
// carried an enumeration increment.
func SplitEnumeration(ref string) (base string, enumerated bool) {
	base = enumSuffix.ReplaceAllString(ref, "")
	return base, base != ref
}

func parseFields(telemetry, metadata []byte) (map[string]Field, error) {
	out := map[string]Field{}
	var errs errorList
	for _, src := range []struct {
		data   []byte
		origin Origin
		name   string
	}{
		{telemetry, Telemetry, "telemetry_fields.yaml"},
		{metadata, Metadata, "metadata_fields.yaml"},
	} {
		if err := parseFieldFile(src.data, src.origin, out, &errs); err != nil {
			return nil, fmt.Errorf("%s: %w", src.name, err)
		}
	}
	return out, errs.err("field literals")
}

// parseFieldFile reads one field file. Both use the same shape: a `literals`
// list whose members are either a bare string, or a single-key map whose value
// is a bounds map or a state list.
func parseFieldFile(b []byte, origin Origin, out map[string]Field, errs *errorList) error {
	var doc struct {
		Literals []yaml.Node `yaml:"literals"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return err
	}
	if len(doc.Literals) == 0 {
		return fmt.Errorf("no `literals` list")
	}
	for _, n := range doc.Literals {
		f, err := parseLiteral(n, origin)
		if err != nil {
			errs.addf("%v", err)
			continue
		}
		if prev, dup := out[f.Name]; dup && prev.Origin != f.Origin {
			errs.addf("field %s is declared in both field files", f.Name)
			continue
		}
		out[f.Name] = f
	}
	return nil
}

func parseLiteral(n yaml.Node, origin Origin) (Field, error) {
	switch n.Kind {
	case yaml.ScalarNode:
		return newField(n.Value, origin, Bare, nil, Bounds{})
	case yaml.MappingNode:
		if len(n.Content) != 2 {
			return Field{}, fmt.Errorf("literal entry has %d keys, expected exactly 1", len(n.Content)/2)
		}
		name, val := n.Content[0].Value, n.Content[1]
		switch val.Kind {
		case yaml.SequenceNode:
			states, err := stateList(name, val)
			if err != nil {
				return Field{}, err
			}
			return newField(name, origin, Multistate, states, Bounds{})
		case yaml.MappingNode:
			var raw struct {
				FixedMin    *float64 `yaml:"fixed_min"`
				FixedMax    *float64 `yaml:"fixed_max"`
				FlexibleMin *float64 `yaml:"flexible_min"`
				FlexibleMax *float64 `yaml:"flexible_max"`
			}
			if err := val.Decode(&raw); err != nil {
				return Field{}, fmt.Errorf("%s: %w", name, err)
			}
			b := Bounds{raw.FixedMin, raw.FixedMax, raw.FlexibleMin, raw.FlexibleMax}
			if !b.Any() {
				return Field{}, fmt.Errorf("%s: a mapping with no recognised bound keys", name)
			}
			return newField(name, origin, Numeric, nil, b)
		default:
			return Field{}, fmt.Errorf("%s: expected a state list or a bounds map, got %s",
				name, kindName(val.Kind))
		}
	default:
		return Field{}, fmt.Errorf("expected a literal or a single-key map, got %s", kindName(n.Kind))
	}
}

func stateList(field string, n *yaml.Node) ([]string, error) {
	var states []string
	for _, s := range n.Content {
		if s.Tag != "!!str" {
			return nil, fmt.Errorf("%s: state %q decoded as %s, not a string: an "+
				"unquoted ON or OFF has been resolved to a boolean", field, s.Value, s.Tag)
		}
		states = append(states, s.Value)
	}
	if len(states) < 2 {
		return nil, fmt.Errorf("%s: a multistate needs at least two states", field)
	}
	return states, nil
}

func newField(name string, origin Origin, kind Kind, states []string, b Bounds) (Field, error) {
	if base, enumerated := SplitEnumeration(name); enumerated {
		return Field{}, fmt.Errorf("field literal %s carries an enumeration "+
			"increment; the ontology defines %s and enumerates it at the point of "+
			"use, not here", name, base)
	}
	parts := strings.Split(name, "_")
	if len(parts) == 0 || name == "" {
		return Field{}, fmt.Errorf("empty field literal")
	}
	return Field{
		Name:      name,
		Kind:      kind,
		Origin:    origin,
		Subfields: parts,
		PointType: parts[len(parts)-1],
		States:    states,
		Bounds:    b,
	}, nil
}
