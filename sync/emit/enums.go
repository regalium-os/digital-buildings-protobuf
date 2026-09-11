// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/ontology"
)

// stateSuffix names the multistate enums, and it is deliberately not "State".
//
// AIP-216 has two rules and they catch different things. core::0216::synonyms
// flags an enum called *Status, which is why the obvious name is out. But
// core::0216::state-field-output-only then flags every *field* whose enum type
// ends in "State" unless it is OUTPUT_ONLY -- and it says outright that the
// field name is ignored, so the trigger is purely the type name. A
// `..._command` field carrying OPEN/CLOSED is legitimately writable under rule
// 8, so naming its enum *State would put api-linter in direct conflict with
// the ontology across 394 fields.
//
// Neither suffix is right. These enums are domain multistates -- OPEN/CLOSED,
// ACTIVE/INACTIVE -- and not the resource lifecycle state AIP-216 is about, so
// "Value" says what they are and trips neither rule.
const stateSuffix = model.StateEnumSuffix

// capnpKeywords are the words Cap'n Proto reserves. It strips an enum's name
// prefix on the way out, so ...STATE_ON becomes `on`, which it then rejects.
// Inherited from coversa-protobuf; here it bites ON/OFF, the ontology's two
// commonest states, covering 111 fields.
var capnpKeywords = map[string]bool{
	"on": true, "off": true, "in": true, "of": true, "using": true,
	"import": true, "const": true, "enum": true, "struct": true,
	"union": true, "group": true, "interface": true, "annotation": true,
	"extends": true, "true": true, "false": true, "void": true,
	"text": true, "data": true, "list": true, "id": true,
}

// typeEnum writes the canonical-type enum: rule 7's other half.
func (e *Emitter) typeEnum(r *model.Resource) *File {
	name := r.Message + "Type"
	f := newFile(r.Dir, naming.Field(name), r.Package, e.banner)
	annotationImports(f)

	f.comment(0, fmt.Sprintf(
		"The canonical types of %s.\n\n"+
			"Rule 7: a message per canonical type would be 1,587 messages and "+
			"1,587 CRUD services, which is not a service surface. Each variant is "+
			"a value here instead, and its exact field sets ride on the value in "+
			"(annotations.canonical_type) -- so nothing the ontology states is "+
			"lost, only compile-time enforcement of it.\n\n"+
			"Names are the ontology's own, verbatim: `%s` *is* the identifier, "+
			"and expanding it would invent a second one.", r.Message, r.Variants[0].Key.Name))
	f.line(0, "enum %s {", name)
	f.comment(1, "Default. The entity's canonical type is unspecified.")
	f.line(1, "%s_UNSPECIFIED = 0;", naming.UpperSnake(name))

	for i, v := range r.Variants {
		f.line(0, "")
		desc := v.Description
		if desc == "" {
			desc = "Canonical type " + v.Key.Name + "."
		}
		f.comment(1, fmt.Sprintf("%s\n\nOntology type `%s`.", desc, v.Key))
		f.line(1, "%s = %d [", v.Value, i+1)
		f.line(2, "(protobuf.digitalbuildings.annotations.v1.canonical_type) = {")
		f.line(3, "guid: %q", v.GUID)
		f.line(3, "qualified_type: %q", v.Key.String())
		for _, im := range v.Implements {
			f.line(3, "implemented_types: %q", im)
		}
		for _, u := range v.Uses {
			f.line(3, "uses: %q", u)
		}
		for _, u := range v.OptUses {
			f.line(3, "opt_uses: %q", u)
		}
		f.line(2, "}")
		f.line(1, "];")
	}
	f.line(0, "}")
	return f
}

// stateEnums writes the multistate enums a package actually uses. Rule 4
// forbids a cross-package message reference and this repository declines the
// google.* exemption too, so a shared enum is copied into each package that
// needs it -- generated, never maintained.
func (e *Emitter) stateEnums(r *model.Resource) []*File {
	sets := map[string][]string{}
	for _, fl := range r.Fields {
		if fl.Def.Kind != ontology.Multistate {
			continue
		}
		sets[stateKey(fl.Def.States)] = fl.Def.States
	}
	keys := make([]string, 0, len(sets))
	for k := range sets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	files := make([]*File, 0, len(keys))
	for _, k := range keys {
		name := e.stateEnumName(sets[k])
		f := newFile(r.Dir, stateEnumFile(name), r.Package, e.banner)
		f.comment(0, fmt.Sprintf(
			"A multistate value used by %s.\n\n"+
				"One enum per distinct state set, not per field: across the ontology "+
				"602 multistate fields draw on only 40 sets, and minting an enum per "+
				"field would mean 305 different spellings of {ACTIVE, INACTIVE} that "+
				"no consumer could pass to a common function.\n\n"+
				"One set per file, not one file per package: rule 2 puts a single "+
				"message or a single enum in a file, and a package may use as many "+
				"as 19 of these.\n\n"+
				"Copied into this package rather than imported: AIP-215 forbids a "+
				"field referencing a type in another proto package (rule 4).", r.Message))
		e.stateEnum(f, name, sortedStates(sets[k]))
		files = append(files, f)
	}
	return files
}

func stateEnumFile(enum string) string { return model.StateEnumFile(enum) }

func (e *Emitter) stateEnum(f *File, name string, states []string) {
	f.comment(0, fmt.Sprintf("One of %s.", strings.Join(states, ", ")))
	f.line(0, "enum %s {", name)
	f.comment(1, "Default. The state is not reported.")
	f.line(1, "%s_UNSPECIFIED = 0;", naming.UpperSnake(name))
	for i, s := range states {
		f.line(0, "")
		desc := e.onto.States[s]
		if desc == "" {
			desc = "State " + s + "."
		}
		f.comment(1, desc)
		f.line(1, "%s = %d;", stateValue(name, s), i+1)
	}
	f.line(0, "}")
}

// stateValue builds the enum value, working around Cap'n Proto's keyword set.
func stateValue(enum, state string) string {
	v := naming.EnumValue(enum, state)
	if capnpKeywords[strings.ToLower(state)] {
		// Cap'n Proto strips the enum prefix, leaving a bare keyword. The
		// suffix is what keeps `ON` from colliding with capnp's `on`.
		v += "_VALUE"
	}
	return v
}
