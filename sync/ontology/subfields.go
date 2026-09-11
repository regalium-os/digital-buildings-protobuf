// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Subfield is one unit of meaning. Its category decides where it may sit in a
// field name, and the category order below *is* the field grammar.
type Subfield struct {
	Name        string
	Category    string
	Description string
}

// categoryOrder is the ontology's own construction format:
//
//	(<agg_desc>_)?(<agg>_)?(<descr>_)*(<component>_)?
//	(<meas_desc>_)?(<meas>_)?<pointtype>(_<num>)*
//
// Order is load-bearing, not documentation: field composition and the
// subfield-set annotation of rule 5 both read it. A category appearing
// upstream that is not listed here is an error, because the generator would
// have no idea where it belongs in a name.
var categoryOrder = []string{
	"aggregation_descriptor",
	"aggregation",
	"descriptor",
	"component",
	"measurement_descriptor",
	"measurement",
	"point_type",
}

// PointType is the category whose member terminates every field name and
// decides, by rule 8, whether the field is writable.
const PointType = "point_type"

// Measurement is the category that decides a field's dimensional units.
const Measurement = "measurement"

func parseSubfields(b []byte) (map[string]Subfield, []string, error) {
	var raw map[string]map[string]string
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, nil, err
	}
	known := map[string]bool{}
	for _, c := range categoryOrder {
		known[c] = true
	}
	out := map[string]Subfield{}
	var errs errorList
	for category, members := range raw {
		if !known[category] {
			errs.addf("unknown subfield category %q: the category order is the "+
				"field grammar, so a new one has to be placed in categoryOrder "+
				"before anything can use it", category)
			continue
		}
		for name, desc := range members {
			if prev, ok := out[name]; ok {
				errs.addf("subfield %q declared in both %s and %s; the ontology "+
					"requires subfields be unique within a namespace",
					name, prev.Category, category)
				continue
			}
			out[name] = Subfield{Name: name, Category: category, Description: desc}
		}
	}
	for _, c := range categoryOrder {
		if len(raw[c]) == 0 {
			errs.addf("subfield category %q is empty or missing", c)
		}
	}
	if err := errs.err("subfields.yaml"); err != nil {
		return nil, nil, err
	}
	return out, categoryOrder, nil
}

// CategoryRank gives a subfield's position in the field grammar, used to order
// the subfield set recorded in the rule 5 annotation.
func (o *Ontology) CategoryRank(subfield string) (int, error) {
	sf, ok := o.Subfields[subfield]
	if !ok {
		return 0, fmt.Errorf("unknown subfield %q", subfield)
	}
	for i, c := range o.Categories {
		if c == sf.Category {
			return i, nil
		}
	}
	return 0, fmt.Errorf("subfield %q has category %q, which is not ordered", subfield, sf.Category)
}
