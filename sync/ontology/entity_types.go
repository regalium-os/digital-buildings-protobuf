// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"
	"sort"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/load"
	"gopkg.in/yaml.v3"
)

// generalTypesFile is the ontology's convention for where a namespace declares
// its general types. Rule 7 makes each of these a resource.
const generalTypesFile = "GENERALTYPES"

// globalFile is where the global namespace keeps its own general types. It is
// not called GENERALTYPES.yaml, and it mixes them with the abstract markers
// EQUIPMENT, NO_ANALYSIS, DEPRECATED, REMAP_REQUIRED and INCOMPLETE.
const globalFile = "global"

// EquipmentRoot is the abstract type every piece of equipment inherits.
const EquipmentRoot = "EQUIPMENT"

// isGeneralType recognises the ontology's two ways of declaring one.
//
// A namespace uses GENERALTYPES.yaml, by convention. The global namespace does
// not: global.yaml holds PMP, SENSOR, TK, VLV and USER_INTERFACE alongside
// EQUIPMENT itself and four markers that are not equipment at all. Taking the
// whole file would make EQUIPMENT a general type, and then every canonical
// type in the ontology would partition onto it.
//
// The discriminator is inheritance rather than position: a general type in
// global.yaml implements EQUIPMENT, and the markers do not.
func isGeneralType(file, namespace string, e Entity) bool {
	if file == generalTypesFile {
		return true
	}
	if namespace != Global || file != globalFile {
		return false
	}
	for _, ref := range e.Implements {
		if ref == EquipmentRoot || ref == "/"+EquipmentRoot {
			return true
		}
	}
	return false
}

// Entity is one entity type as declared, before inheritance is resolved.
type Entity struct {
	Key         TypeKey
	GUID        string
	Description string
	IsAbstract  bool
	IsCanonical bool
	Implements  []string // as written: NAME, NS/NAME, or /NAME
	Uses        []string // required fields
	OptUses     []string // optional fields
	// File is the ontology file that declared it, kept for diagnostics and to
	// recognise a GENERALTYPES member.
	File string
}

// AllFields is every field the type names directly, enumeration increments
// stripped. Inheritance is resolved in stage 6, not here.
func (e Entity) AllFields() []string {
	out := make([]string, 0, len(e.Uses)+len(e.OptUses))
	seen := map[string]bool{}
	for _, list := range [][]string{e.Uses, e.OptUses} {
		for _, f := range list {
			base, _ := SplitEnumeration(f)
			if !seen[base] {
				seen[base] = true
				out = append(out, base)
			}
		}
	}
	return out
}

func (o *Ontology) parseEntities(sources []load.Source) error {
	o.Types = map[TypeKey]Entity{}
	var errs errorList
	for _, src := range sources {
		var raw map[string]struct {
			GUID        string   `yaml:"guid"`
			Description string   `yaml:"description"`
			IsAbstract  bool     `yaml:"is_abstract"`
			IsCanonical bool     `yaml:"is_canonical"`
			Implements  []string `yaml:"implements"`
			Uses        []string `yaml:"uses"`
			OptUses     []string `yaml:"opt_uses"`
		}
		if err := yaml.Unmarshal(src.Data, &raw); err != nil {
			return fmt.Errorf("%s: %w", src.Path, err)
		}
		for name, v := range raw {
			key := TypeKey{Namespace: src.Namespace, Name: name}
			if prev, dup := o.Types[key]; dup {
				errs.addf("%s declared twice: %s and %s", key, prev.File, src.Path)
				continue
			}
			if v.GUID == "" {
				errs.addf("%s (%s) has no guid; catalogue entries are keyed by it",
					key, src.Path)
			} else if prevKey, dup := o.byGUID[v.GUID]; dup {
				errs.addf("guid %s is on both %s and %s", v.GUID, prevKey, key)
			} else {
				o.byGUID[v.GUID] = key
			}
			e := Entity{
				Key: key, GUID: v.GUID, Description: v.Description,
				IsAbstract: v.IsAbstract, IsCanonical: v.IsCanonical,
				Implements: v.Implements, Uses: v.Uses, OptUses: v.OptUses,
				File: src.Path,
			}
			o.Types[key] = e
			if isGeneralType(src.File, src.Namespace, e) {
				o.general[src.Namespace] = append(o.general[src.Namespace], key)
			}
		}
	}
	o.addSpaceTypes()
	for ns := range o.general {
		sort.Slice(o.general[ns], func(i, j int) bool { return o.general[ns][i].Name < o.general[ns][j].Name })
	}
	return errs.err("entity types")
}

// addSpaceTypes handles rule 6's third case: a namespace that declares no
// GENERALTYPES.yaml at all. FACILITIES, PHYSICAL_SECURITY, CARSON, INFO_TECH
// and UNTYPED are these, and there the entity types *are* the resources --
// BUILDING, FLOOR, ROOM, DOOR -- so the resource name takes the general-type
// segment of the package path.
//
// This is not cosmetic. PHYSICAL_SECURITY/DOOR_STD is canonical and implements
// FACILITIES/DOOR, so without this the ontology's own cross-namespace general
// type is invisible and two canonical types cannot be placed.
//
// A canonical or abstract type is excluded: the first is a variant of
// something else, and the second is a functional group, which rule 7 keeps out
// of the resource set deliberately.
func (o *Ontology) addSpaceTypes() {
	declares := map[string]bool{}
	for ns, keys := range o.general {
		if len(keys) > 0 {
			declares[ns] = true
		}
	}
	o.spaces = map[string]bool{}
	for key, e := range o.Types {
		if key.Namespace == Global || declares[key.Namespace] {
			continue
		}
		if e.IsCanonical || e.IsAbstract {
			continue
		}
		o.general[key.Namespace] = append(o.general[key.Namespace], key)
		o.spaces[key.Namespace] = true
	}
}

// IsSpaceNamespace reports whether a namespace declares no GENERALTYPES.yaml,
// and so its entity types are themselves the resources.
//
// These are resources whether or not anything specialises them, which the
// equipment namespaces are not: FACILITIES/BUILDING has no canonical variants
// and must still exist, because every resource name in the schema is patterned
// buildings/{building}/... and a parent nothing can create is not a parent.
func (o *Ontology) IsSpaceNamespace(ns string) bool { return o.spaces[ns] }
