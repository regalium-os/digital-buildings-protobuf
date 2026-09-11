// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package ontology parses the Digital Buildings resource tree into typed
// values. Stage 3: it knows the ontology's shapes and nothing about protobuf.
//
// Every shape here is handled explicitly rather than generically. The input is
// a hand-maintained corpus under continuous change, and a parser that quietly
// accepts an unfamiliar shape produces a schema that is wrong in a way no
// linter catches (rule 15).
package ontology

import (
	"fmt"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/load"
)

// Ontology is the whole pinned vocabulary, parsed.
type Ontology struct {
	Subfields   map[string]Subfield  // literal -> definition
	Categories  []string             // subfield categories, in field-grammar order
	Fields      map[string]Field     // literal -> definition
	States      map[string]string    // literal -> definition
	Units       map[string]UnitKind  // measurement subfield -> its units
	Connections map[string]string    // CONTAINS, FEEDS, ... -> definition
	Types       map[TypeKey]Entity   // (namespace, name) -> entity type
	Namespaces  []string             // declared namespaces, excluding GLOBAL
	byGUID      map[string]TypeKey   // GUID -> the type that declared it
	general     map[string][]TypeKey // namespace -> its general types
	spaces      map[string]bool      // namespaces whose entity types are themselves resources
}

// TypeKey identifies an entity type. The ontology allows one level of
// namespace below global and forbids hierarchical namespacing, so two
// segments is the whole address space.
type TypeKey struct {
	Namespace string // HVAC, LIGHTING, ... or GLOBAL
	Name      string // FCU, FCU_DFSS_CSP_CHWDC, EQUIPMENT, ...
}

func (k TypeKey) String() string {
	if k.Namespace == Global {
		return "/" + k.Name
	}
	return k.Namespace + "/" + k.Name
}

// Global is the namespace every other namespace may reference directly.
const Global = "GLOBAL"

// Parse reads a loaded tree. The order matters only for error quality: a
// broken subfield table makes every field diagnostic meaningless, so it is
// reported first.
func Parse(t *load.Tree) (*Ontology, error) {
	o := &Ontology{
		Namespaces: t.Namespaces,
		byGUID:     map[string]TypeKey{},
		general:    map[string][]TypeKey{},
	}
	var err error
	if o.Subfields, o.Categories, err = parseSubfields(t.Subfields); err != nil {
		return nil, fmt.Errorf("subfields: %w", err)
	}
	if o.States, err = parseStates(t.States); err != nil {
		return nil, fmt.Errorf("states: %w", err)
	}
	if o.Units, err = parseUnits(t.Units); err != nil {
		return nil, fmt.Errorf("units: %w", err)
	}
	if o.Connections, err = parseConnections(t.Connections); err != nil {
		return nil, fmt.Errorf("connections: %w", err)
	}
	if o.Fields, err = parseFields(t.Telemetry, t.Metadata); err != nil {
		return nil, fmt.Errorf("fields: %w", err)
	}
	if err := o.parseEntities(t.Entities); err != nil {
		return nil, err
	}
	if err := o.classifyFields(); err != nil {
		return nil, err
	}
	return o, o.validate()
}

// classifyFields assigns each field its measurement subfield, which needs the
// subfield table and so cannot happen while the field file is being read. A
// numeric field without one has no quantity kind and therefore no unit, which
// the ontology forbids except for the `count` point type.
func (o *Ontology) classifyFields() error {
	var errs errorList
	for name, f := range o.Fields {
		for _, part := range f.Subfields {
			sf, ok := o.Subfields[part]
			if !ok {
				continue // reported by validate, with the whole field for context
			}
			if sf.Category == Measurement {
				if f.Measurement != "" {
					errs.addf("field %s has two measurement subfields, %s and %s",
						name, f.Measurement, part)
					continue
				}
				f.Measurement = part
			}
		}
		o.Fields[name] = f
	}
	return errs.err("field classification")
}

// TypeByGUID resolves the ontology's own stable identifier. Catalogue entries
// for entity types are keyed by GUID precisely because it survives a rename
// (rule 2).
func (o *Ontology) TypeByGUID(guid string) (TypeKey, bool) {
	k, ok := o.byGUID[guid]
	return k, ok
}

// GeneralTypes returns the types a namespace declared in GENERALTYPES.yaml.
func (o *Ontology) GeneralTypes(namespace string) []TypeKey {
	return o.general[namespace]
}

// validate checks the cross-facet references the ontology's own validator
// would. Every one of these is a silent wrong answer if left unchecked: a
// field naming a state that does not exist yields an enum with a missing
// value, not a parse error.
func (o *Ontology) validate() error {
	var errs errorList
	for name, f := range o.Fields {
		for _, s := range f.States {
			if _, ok := o.States[s]; !ok {
				errs.addf("field %s: state %s is not defined in states.yaml", name, s)
			}
		}
		if f.Measurement != "" {
			if _, ok := o.Units[f.Measurement]; !ok {
				errs.addf("field %s: measurement subfield %s has no units", name, f.Measurement)
			}
		}
		for _, s := range f.Subfields {
			if _, ok := o.Subfields[s]; !ok {
				errs.addf("field %s: subfield %s is not defined", name, s)
			}
		}
	}
	for key, e := range o.Types {
		for _, ref := range e.Implements {
			if _, ok := o.Resolve(ref, key.Namespace); !ok {
				errs.addf("%s: implements %s, which is not defined", key, ref)
			}
		}
		for _, f := range e.AllFields() {
			if _, ok := o.Fields[f]; !ok {
				errs.addf("%s: uses %s, which is not a field literal", key, f)
			}
		}
	}
	return errs.err("ontology is inconsistent")
}
