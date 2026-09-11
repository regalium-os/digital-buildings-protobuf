// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// The objects view: what the primary objects are, how many of each the pinned
// revision has, and which relationships hold between them.
//
// The counts are computed rather than quoted. A number in prose goes stale at
// the next spec bump and nothing notices; a number printed from the model
// cannot.

func drawObjects(w io.Writer, o *ontology.Ontology, m *model.Model,
	s scope) error {
	rs := inScope(m, s)

	fmt.Fprintln(w, "primary objects")
	fmt.Fprintf(w, "\n  %-22s %7s  %s\n", "object", "count", "what one is")
	for _, row := range [][3]string{
		{"Building", "1", "the root resource; every name hangs beneath it"},
		{"Resource", n(len(rs)), "a general type: one message, one service"},
		{"CanonicalType", n(variantCount(rs)), "a variant: an enum value on its resource"},
		{"AbstractGroup", n(abstractCount(o)), "a functional group: fields, never a message"},
		{"StandardField", n(len(o.Fields)), "a field literal, from the two field files"},
		{"Subfield", n(len(o.Subfields)), "a word of the field grammar, in 7 categories"},
		{"StateValue", n(len(o.States)), "a multistate literal; sets collapse to enums"},
		{"UnitKind", n(len(o.Units)), "a quantity kind, with exactly one standard unit"},
		{"ConnectionType", n(len(o.Connections)), "how one entity relates to another"},
	} {
		fmt.Fprintf(w, "  %-22s %7s  %s\n", row[0], row[1], row[2])
	}

	fmt.Fprintln(w, "\n\nthe identity triple on every resource (rule 10)")
	fmt.Fprintln(w, `
  name     IDENTIFIER    the resource name, assigned by the server
  uid      OUTPUT_ONLY   the ontology's own GUID, format UUID4
  code     IMMUTABLE     the building config's entity code, e.g. FCU-123`)

	fmt.Fprintln(w, "\nrelationships")
	fmt.Fprintln(w, `
  Building        1 ──< Resource        by resource name, not by a field:
                                        buildings/{building}/fanCoilUnits/{…}
  Resource        N >──< Resource       via Connection, which carries a type and
                                        the other entity's resource name
  Resource        1 ──< Connection      repeated on the resource
  Resource        1 ──< Link            names a source entity and maps its
                                        standard fields onto this one's
  Resource        1 ──  CanonicalType   entity_type: which variant this is
  CanonicalType   N >──< AbstractGroup  implements: a DAG, resolved to a flat
                                        field set at stage 6
  Resource        1 ──< StandardField   the union across its variants, all
                                        optional
  StandardField   N >──  Subfield       ordered; the set is the field's identity
  StandardField   N >──  UnitKind       one measurement subfield, one standard
  StandardField   N >──< StateValue     multistates collapse to a shared enum

  No relationship above is a cross-package message reference. AIP-215 forbids
  one, so a Connection carries (google.api.resource_reference).type = "*" and
  names the other end by resource name (rule 4).`)

	fmt.Fprintf(w, "\n\nthe field grammar — %d subfields in %d ordered categories\n",
		len(o.Subfields), len(o.Categories))
	fmt.Fprintln(w, "\n  a literal is built in this order, and the order is the grammar:")
	fmt.Fprintf(w, "\n  %s\n", strings.Join(o.Categories, " → "))
	byCat := map[string]int{}
	for _, sf := range o.Subfields {
		byCat[sf.Category]++
	}
	fmt.Fprintln(w)
	for _, c := range o.Categories {
		fmt.Fprintf(w, "  %-24s %4d\n", c, byCat[c])
	}
	fmt.Fprintln(w, "\n  e.g. discharge_air_temperature_sensor")
	fmt.Fprintln(w, "       └ discharge (descriptor) air (descriptor)"+
		" temperature (measurement) sensor (point type)")

	fmt.Fprintf(w, "\n\nconnection types — %d, and the ERD's only true edges\n\n",
		len(o.Connections))
	names := make([]string, 0, len(o.Connections))
	for k := range o.Connections {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Fprintf(w, "  %-20s %s\n", k, clip(o.Connections[k], 54))
	}
	return nil
}

func abstractCount(o *ontology.Ontology) int {
	n := 0
	for _, e := range o.Types {
		if e.IsAbstract {
			n++
		}
	}
	return n
}

func n(v int) string { return fmt.Sprintf("%d", v) }
