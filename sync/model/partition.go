// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// generalTypes overrides which general type a canonical type belongs to, for
// the case where the name prefix is not the answer. Keyed by the canonical
// type's GUID, per rule 2.
//
// Empty: all 1,587 canonical types partition cleanly on their prefix, which is
// the ontology's own naming convention ("each child type begins its name with
// <general type>_").
var generalTypes = map[string]ontology.TypeKey{}

// group is one general type and the canonical types that belong to it.
type group struct {
	key      ontology.TypeKey
	variants []ontology.TypeKey
}

// partition assigns every canonical type to a general type. A canonical type
// whose prefix names nothing in GENERALTYPES.yaml is an error rather than a
// guess (rule 15): silently inventing a resource would publish a package.
func partition(o *ontology.Ontology) ([]group, error) {
	declared := map[ontology.TypeKey]bool{}
	for _, ns := range append([]string{ontology.Global}, o.Namespaces...) {
		for _, k := range o.GeneralTypes(ns) {
			declared[k] = true
		}
	}
	byGeneral := map[ontology.TypeKey][]ontology.TypeKey{}
	// Every declared general type is a resource, with or without variants.
	// Rule 7 says "a general type is the resource" and puts no variant-count
	// caveat on it, and the types this reaches are the ones that need it most:
	// FACILITIES/BUILDING is the parent of every resource name in the schema,
	// and GATEWAYS/PASSTHROUGH is what a building config names for a device
	// that only carries translations. Neither has a canonical variant, and a
	// parent nothing can create is not a parent.
	for gk := range declared {
		byGeneral[gk] = nil
	}
	var errs []string
	for key, e := range o.Types {
		if !e.IsCanonical {
			continue
		}
		gk, err := generalTypeOf(o, key, e, declared)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		byGeneral[gk] = append(byGeneral[gk], key)
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return nil, fmt.Errorf("cannot place %d canonical types:\n  - %s",
			len(errs), strings.Join(errs, "\n  - "))
	}
	out := make([]group, 0, len(byGeneral))
	for gk, vs := range byGeneral {
		// Sort on the whole key, not just the name. HVAC/VLV_PM and
		// PLUMBING/VLV_PM are different types with identical names landing on
		// one globally-declared resource, so a name-only comparison leaves
		// their order to map iteration -- and enum numbers come from position,
		// so the two would swap wire values between runs.
		sort.Slice(vs, func(i, j int) bool {
			if vs[i].Name != vs[j].Name {
				return vs[i].Name < vs[j].Name
			}
			return vs[i].Namespace < vs[j].Namespace
		})
		out = append(out, group{key: gk, variants: vs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key.String() < out[j].key.String() })
	return out, nil
}

func generalTypeOf(o *ontology.Ontology, key ontology.TypeKey, e ontology.Entity,
	declared map[ontology.TypeKey]bool) (ontology.TypeKey, error) {

	if gk, ok := generalTypes[e.GUID]; ok {
		if _, exists := o.Types[gk]; !exists {
			return gk, fmt.Errorf("%s: generalTypes names %s, which the ontology "+
				"does not have", key, gk)
		}
		return gk, nil
	}
	tag := key.Name
	if i := strings.Index(tag, "_"); i >= 0 {
		tag = tag[:i]
	}
	// The ontology's own convention: "each child type begins its name with
	// <general type>_". 1,586 of 1,587 canonical types partition on this.
	// The general type is normally declared in the canonical type's own
	// namespace; a few namespaces inherit theirs from global.
	for _, ns := range []string{key.Namespace, ontology.Global} {
		gk := ontology.TypeKey{Namespace: ns, Name: tag}
		if declared[gk] {
			return gk, nil
		}
	}
	// The convention is a convention, not a rule the ontology enforces:
	// ELECTRICAL/SWITCHBOARD is canonical, implements PANEL, and is named
	// nothing like it. Walking `implements` is not a guess -- it is the
	// ontology stating the relationship outright, and it is better evidence
	// than the name. Ambiguity is still an error.
	found := inheritedGeneralTypes(o, key, declared)
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return ontology.TypeKey{}, fmt.Errorf(
			"%s has prefix %q, which names no general type in %s or GLOBAL, and it "+
				"implements none either; add an entry to generalTypes in sync/model "+
				"keyed by guid %s", key, tag, key.Namespace, e.GUID)
	default:
		names := make([]string, len(found))
		for i, k := range found {
			names[i] = k.String()
		}
		sort.Strings(names)
		return ontology.TypeKey{}, fmt.Errorf(
			"%s implements %d general types (%s), so which resource it belongs to "+
				"is ambiguous; add an entry to generalTypes in sync/model keyed by "+
				"guid %s", key, len(found), strings.Join(names, ", "), e.GUID)
	}
}

// inheritedGeneralTypes finds the declared general types a canonical type
// reaches through `implements`, stopping at the first one on each branch so
// that a general type inheriting another does not report both.
func inheritedGeneralTypes(o *ontology.Ontology, key ontology.TypeKey,
	declared map[ontology.TypeKey]bool) []ontology.TypeKey {

	var out []ontology.TypeKey
	seen := map[ontology.TypeKey]bool{}
	found := map[ontology.TypeKey]bool{}
	var walk func(ontology.TypeKey)
	walk = func(k ontology.TypeKey) {
		if seen[k] {
			return
		}
		seen[k] = true
		e, ok := o.Types[k]
		if !ok {
			return
		}
		for _, ref := range e.Implements {
			pk, ok := o.Resolve(ref, k.Namespace)
			if !ok {
				continue
			}
			if declared[pk] {
				if !found[pk] {
					found[pk] = true
					out = append(out, pk)
				}
				continue
			}
			walk(pk)
		}
	}
	walk(key)
	return out
}
