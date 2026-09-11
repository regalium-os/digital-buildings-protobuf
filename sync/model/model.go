// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package model turns the parsed ontology into the resources the schema
// exposes. Stage 6: this is where rule 7's cut happens -- a general type
// becomes a resource, a canonical type becomes an enum value on it.
package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// Model is the whole schema, ready to emit.
type Model struct {
	Onto      *ontology.Ontology
	Resources []*Resource
	// Facets are the read-only ontology packages of rule 7: Subfield, Field,
	// State, Unit, ConnectionType, EntityType, Namespace.
	Facets []string
}

// Kind records whether a general type is equipment, for documentation and the
// survey. It does **not** decide the package: every resource is routed by the
// ontology's own namespace.
//
// An earlier revision of rule 6 put non-equipment general types under a
// separate system/ family, on the reasoning that the ontology models chilled
// and heating water systems as general types that are not EQUIPMENT. Running
// it disproved the discriminator. "Does not implement EQUIPMENT" selects
// CHWS, HWS, CDWS and GTWS correctly, but also LGRP (a luminaire group),
// WEATHER (a weather station) and FACILITIES/DOOR -- while *missing* CHGS and
// DWST, which are systems by name and description and do implement EQUIPMENT.
// The ontology is simply not consistent on this axis, so no derivable rule
// selects "system", and a hand-kept list would be a fourteenth table to go
// stale. The namespace is unambiguous and needs no judgement.
type Kind int

const (
	// Equipment is a general type that inherits EQUIPMENT.
	Equipment Kind = iota
	// NonEquipment is one that does not: a system, a riser, a zone grouping.
	NonEquipment
)

// Resource is one message with a google.api.resource annotation: a general
// type, carrying the union of every field any of its canonical variants uses.
type Resource struct {
	Key       ontology.TypeKey // the general type, e.g. HVAC/FCU
	Kind      Kind
	Message   string // FanCoilUnit
	Package   string // protobuf.digitalbuildings.hvac.fcu.v1
	Dir       string // protobuf/digitalbuildings/hvac/fcu/v1
	Namespace string // hvac
	Segment   string // fcu
	// Fields is the union across variants, sorted by literal.
	Fields []Field
	// Variants are the canonical types, which become enum values.
	Variants []Variant
	// Description is the general type's own ontology description.
	Description string
	GUID        string
}

// Field is one field on a resource, resolved.
type Field struct {
	// Literal is the ontology name, which stays in the comment and the
	// annotation even when Name differs.
	Literal string
	Name    string // the proto field name, after catalog.Rename
	Def     ontology.Field
	// Enumerated is rule 12: some entity type references this field with an
	// increment, so it is emitted repeated.
	Enumerated bool
	// Required is true when at least one variant `uses` rather than
	// `opt_uses` it. It does not make the proto field required -- every field
	// on a resource is optional (rule 7) -- but it is recorded.
	Required bool
	Number   int // assigned by the ordinal ledger, stage 8
}

// Variant is one canonical type: an enum value carrying its exact field sets.
type Variant struct {
	Key         ontology.TypeKey
	Value       string // FAN_COIL_UNIT_TYPE_FCU_DFSS_CSP_CHWDC
	GUID        string
	Description string
	Implements  []string // namespace-qualified, sorted
	Uses        []string // sorted literals
	OptUses     []string
}

// ReservedOrdinals is the top of the range rule 9 holds for the fields on
// every read path: name, uid, code, entity_type and the connection list.
const ReservedOrdinals = 15

// Facets are the ontology's own vocabulary, exposed read-only (rule 7).
var Facets = []string{
	"subfields", "fields", "states", "units",
	"connections", "entity_types", "namespaces",
}

// Build assembles the model. The order is the three passes of
// docs/generator.md: resolve inheritance, partition by general type, union the
// fields.
func Build(o *ontology.Ontology) (*Model, error) {
	m := &Model{Onto: o, Facets: Facets}
	groups, err := partition(o)
	if err != nil {
		return nil, err
	}
	// Collect rather than return on the first failure. A spec bump that adds
	// six general types should name all six, not make the reader rerun the
	// generator once per missing catalogue entry (rule 15).
	var failures []string
	for _, g := range groups {
		r, err := buildResource(o, g)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		m.Resources = append(m.Resources, r)
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return nil, fmt.Errorf("cannot build %d resources:\n  - %s",
			len(failures), strings.Join(failures, "\n  - "))
	}
	// Total order: the package path is unique, so comparing it leaves nothing
	// to map iteration.
	sort.Slice(m.Resources, func(i, j int) bool {
		return m.Resources[i].Package < m.Resources[j].Package
	})
	return m, CheckCatalogue(o)
}
