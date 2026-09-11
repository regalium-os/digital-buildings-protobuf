// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/catalog"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// rootPackage is the proto package prefix. Rule 6: the buf module is rooted at
// the repository root, so protobuf/ is itself the first package segment.
const rootPackage = "protobuf.digitalbuildings"

// rootDir mirrors rootPackage on disk, which is what PACKAGE_DIRECTORY_MATCH
// checks.
const rootDir = "protobuf/digitalbuildings"

// buildResource unions one general type's variants into a resource.
func buildResource(o *ontology.Ontology, g group) (*Resource, error) {
	e, ok := o.Types[g.key]
	if !ok {
		return nil, fmt.Errorf("general type %s is not defined", g.key)
	}
	// An equipment tag is 1-4 characters (AHU, FCU, TXMR) and has to be looked
	// up: no rule turns "AHU" into "AirHandlingUnit". A space namespace names
	// its types in English already -- BUILDING, COORDINATE_BASIS, fixed_camera
	// -- so deriving the message name is correct there, and demanding a
	// catalogue entry would be busywork that goes stale.
	message, ok := catalog.GeneralTypeName(g.key.Name)
	if !ok {
		if !o.IsSpaceNamespace(g.key.Namespace) {
			return nil, fmt.Errorf(
				"general type %s has no English name; add %q to generalTypeNames in "+
					"sync/catalog -- a message called %q is not one a consumer can read",
				g.key, g.key.Name, g.key.Name)
		}
		message = naming.Message(g.key.Name)
	}
	if renamed, ok := catalog.TypeRenames[e.GUID]; ok {
		message = renamed
	}
	// Rule 6: the path segment is the English name, not the ontology tag --
	// `electrical/ats` tells a reader nothing. Taking it from `message` rather
	// than from the tag also keeps it in step through a TypeRenames entry, so
	// REQUEST_TO_EXIT's directory says exit_request_sensor too.
	r := &Resource{
		Key: g.key, Kind: classify(o, g.key), Message: message,
		Description: e.Description, GUID: e.GUID,
		Namespace: naming.Package(g.key.Namespace),
		Segment:   naming.Package(message),
	}
	r.Package = strings.Join([]string{rootPackage, r.Namespace, r.Segment, "v1"}, ".")
	r.Dir = strings.Join([]string{rootDir, r.Namespace, r.Segment, "v1"}, "/")

	if err := r.collect(o, g); err != nil {
		return nil, err
	}
	return r, nil
}

// collect unions the variants' resolved field sets and records each variant's
// own sets on the enum value, which is what makes rule 7 lossless.
func (r *Resource) collect(o *ontology.Ontology, g group) error {
	union := map[string]bool{}    // literal -> present
	required := map[string]bool{} // literal -> some variant `uses` it
	enumerated := map[string]bool{}

	// The general type's own fields are on every variant.
	base, err := o.ResolveFields(g.key)
	if err != nil {
		return err
	}
	for f := range base {
		union[f] = true
	}
	for f := range o.Enumerated(g.key) {
		enumerated[f] = true
	}

	for _, vk := range g.variants {
		fields, err := o.ResolveFields(vk)
		if err != nil {
			return err
		}
		for f := range fields {
			union[f] = true
		}
		for f := range o.Enumerated(vk) {
			enumerated[f] = true
		}
		v, err := variantOf(o, vk, g.key, r.Message)
		if err != nil {
			return err
		}
		for _, f := range v.Uses {
			required[f] = true
		}
		r.Variants = append(r.Variants, v)
	}

	literals := make([]string, 0, len(union))
	for f := range union {
		literals = append(literals, f)
	}
	sort.Strings(literals)
	for _, lit := range literals {
		def, ok := o.Fields[lit]
		if !ok {
			return fmt.Errorf("%s: field %q is not a literal", r.Key, lit)
		}
		r.Fields = append(r.Fields, Field{
			Literal:    lit,
			Name:       naming.FieldLiteral(catalog.Rename(lit)),
			Def:        def,
			Enumerated: enumerated[lit],
			Required:   required[lit],
		})
	}
	return nil
}

// variantOf records a canonical type's exact sets. These ride on the enum
// value in (annotations.canonical_type), so a validator can reject a variant
// reporting a field its declared type does not have (rule 7).
//
// The enum value is qualified by namespace when the variant does not come from
// the general type's own. PMP, SENSOR, TK and VLV are declared in the global
// namespace and specialised from several others, so HVAC/VLV_PM and
// PLUMBING/VLV_PM are two different types landing on one resource -- and the
// ontology is right that they are different, so the schema must keep them
// apart rather than collide. Qualifying on the namespace *difference* rather
// than on collision keeps the value stable: a later namespace adding its own
// VLV_PM does not rename the two that already exist.
func variantOf(o *ontology.Ontology, k, general ontology.TypeKey, enum string) (Variant, error) {
	e, ok := o.Types[k]
	if !ok {
		return Variant{}, fmt.Errorf("canonical type %s is not defined", k)
	}
	name := k.Name
	if k.Namespace != general.Namespace {
		name = k.Namespace + "_" + name
	}
	v := Variant{
		Key:         k,
		Value:       naming.EnumValue(enum+"Type", name),
		GUID:        e.GUID,
		Description: e.Description,
	}
	for _, ref := range e.Implements {
		if pk, ok := o.Resolve(ref, k.Namespace); ok {
			v.Implements = append(v.Implements, pk.String())
		} else {
			v.Implements = append(v.Implements, ref)
		}
	}
	v.Uses = baseLiterals(e.Uses)
	v.OptUses = baseLiterals(e.OptUses)
	sort.Strings(v.Implements)
	return v, nil
}

// baseLiterals strips enumeration increments and de-duplicates, so that a
// variant using _1 and _2 records the literal once (rule 12).
func baseLiterals(refs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, ref := range refs {
		base, _ := ontology.SplitEnumeration(ref)
		if !seen[base] {
			seen[base] = true
			out = append(out, base)
		}
	}
	sort.Strings(out)
	return out
}
