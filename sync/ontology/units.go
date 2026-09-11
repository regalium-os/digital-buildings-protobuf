// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// UnitKind is the set of units for one measurement subfield -- one quantity
// kind, in the ontology's terms. Exactly one member is the standard.
type UnitKind struct {
	Measurement string
	Standard    string
	Units       []Unit
	// AliasOf names the measurement this one delegates to, empty unless the
	// ontology wrote `diameter: distance` instead of a map.
	AliasOf string
}

// Unit is one named unit and its conversion to the standard.
type Unit struct {
	Name       string
	Standard   bool
	Multiplier float64
	Offset     float64
}

// parseUnits handles the two shapes units.yaml mixes under one key:
//
//	temperature:            # a map of unit -> STANDARD | {multiplier, offset}
//	  kelvins: STANDARD
//	  degrees_celsius: {multiplier: 1, offset: 273.15}
//	diameter: distance      # a bare string naming another measurement
//
// A reader expecting a map gets a string for the five alias keys, which is a
// panic or a silently empty unit set depending on how it decodes. Both are
// wrong answers rather than errors, which is why this is handled by node kind.
func parseUnits(b []byte) (map[string]UnitKind, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a mapping at the top level, got %s", kindName(root.Kind))
	}
	out := map[string]UnitKind{}
	var errs errorList
	for i := 0; i+1 < len(root.Content); i += 2 {
		name, val := root.Content[i].Value, root.Content[i+1]
		switch val.Kind {
		case yaml.ScalarNode:
			out[name] = UnitKind{Measurement: name, AliasOf: val.Value}
		case yaml.MappingNode:
			k, err := unitMap(name, val)
			if err != nil {
				errs.addf("%s: %v", name, err)
				continue
			}
			out[name] = k
		default:
			errs.addf("%s: expected a unit map or an alias name, got %s", name, kindName(val.Kind))
		}
	}
	if err := resolveAliases(out, &errs); err != nil {
		return nil, err
	}
	return out, errs.err("units.yaml")
}

func unitMap(measurement string, n *yaml.Node) (UnitKind, error) {
	k := UnitKind{Measurement: measurement}
	for i := 0; i+1 < len(n.Content); i += 2 {
		name, val := n.Content[i].Value, n.Content[i+1]
		u := Unit{Name: name, Multiplier: 1}
		switch {
		case val.Kind == yaml.ScalarNode && val.Value == "STANDARD":
			if k.Standard != "" {
				return k, fmt.Errorf("both %s and %s are STANDARD", k.Standard, name)
			}
			u.Standard = true
			u.Offset, u.Multiplier = 0, 1
			k.Standard = name
		case val.Kind == yaml.MappingNode:
			var conv struct {
				Multiplier float64 `yaml:"multiplier"`
				Offset     float64 `yaml:"offset"`
			}
			if err := val.Decode(&conv); err != nil {
				return k, fmt.Errorf("%s: %w", name, err)
			}
			u.Multiplier, u.Offset = conv.Multiplier, conv.Offset
		default:
			return k, fmt.Errorf("%s: expected STANDARD or a conversion map", name)
		}
		k.Units = append(k.Units, u)
	}
	if k.Standard == "" {
		return k, fmt.Errorf("no unit marked STANDARD")
	}
	sort.Slice(k.Units, func(i, j int) bool { return k.Units[i].Name < k.Units[j].Name })
	return k, nil
}

// resolveAliases points each alias at its target's units. One pass suffices
// because the ontology has no alias chains; a chain is reported rather than
// followed, so that adding one upstream is noticed.
func resolveAliases(kinds map[string]UnitKind, errs *errorList) error {
	for name, k := range kinds {
		if k.AliasOf == "" {
			continue
		}
		target, ok := kinds[k.AliasOf]
		if !ok {
			errs.addf("%s is an alias of %q, which is not a measurement", name, k.AliasOf)
			continue
		}
		if target.AliasOf != "" {
			errs.addf("%s aliases %s, which is itself an alias; chains are not followed",
				name, k.AliasOf)
			continue
		}
		k.Standard, k.Units = target.Standard, target.Units
		kinds[name] = k
	}
	return nil
}

func kindName(k yaml.Kind) string {
	switch k {
	case yaml.ScalarNode:
		return "a scalar"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a sequence"
	default:
		return "an unexpected node"
	}
}
