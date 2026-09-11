// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import "github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"

// equipmentRoot is the abstract type every piece of equipment inherits.
const equipmentRoot = ontology.EquipmentRoot

// classify decides which package family a general type belongs to (rule 6).
func classify(o *ontology.Ontology, key ontology.TypeKey) Kind {
	if isEquipment(o, key, map[ontology.TypeKey]bool{}) {
		return Equipment
	}
	return NonEquipment
}

func isEquipment(o *ontology.Ontology, k ontology.TypeKey, seen map[ontology.TypeKey]bool) bool {
	if seen[k] {
		return false
	}
	seen[k] = true
	if k.Name == equipmentRoot {
		return true
	}
	e, ok := o.Types[k]
	if !ok {
		return false
	}
	for _, ref := range e.Implements {
		if pk, ok := o.Resolve(ref, k.Namespace); ok && isEquipment(o, pk, seen) {
			return true
		}
	}
	return false
}
