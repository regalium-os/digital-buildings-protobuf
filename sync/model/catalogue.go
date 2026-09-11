// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/catalog"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// CheckCatalogue fails when a catalogue entry names something the ontology no
// longer has. Rule 2: an entry that quietly stops applying is worse than one
// that breaks the build, because the schema silently changes shape instead.
func CheckCatalogue(o *ontology.Ontology) error {
	var errs []string
	for literal := range catalog.FieldRenames {
		if _, ok := o.Fields[literal]; !ok {
			errs = append(errs, fmt.Sprintf(
				"FieldRenames names %q, which is not a field literal", literal))
		}
	}
	for literal := range catalog.TimeFields {
		if _, ok := o.Fields[literal]; !ok {
			errs = append(errs, fmt.Sprintf(
				"TimeFields names %q, which is not a field literal", literal))
		}
	}
	for guid := range catalog.TypeRenames {
		if _, ok := o.TypeByGUID(guid); !ok {
			errs = append(errs, fmt.Sprintf(
				"TypeRenames names guid %s, which no entity type carries", guid))
		}
	}
	for guid := range generalTypes {
		if _, ok := o.TypeByGUID(guid); !ok {
			errs = append(errs, fmt.Sprintf(
				"generalTypes names guid %s, which no entity type carries", guid))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("stale catalogue entries:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
