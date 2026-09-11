// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package docs

import (
	"fmt"
	"path"
	"sort"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
)

// index renders protobuf/digitalbuildings/README.md: every resource in the
// schema, grouped by namespace.
//
// Its real job is the tag column. Rule 6 spells the directory in English so
// `electrical/automatic_transfer_switch` needs no glossary, but the ontology's
// own tag is what a building config and every upstream document use -- so the
// mapping has to be readable in one place, in both directions.
func (r *Renderer) index(m *model.Model, resources []*model.Resource) Page {
	var d doc
	d.line(r.bannerBlock())
	d.heading(1, "Digital Buildings, as protobuf")

	byNS := map[string][]*model.Resource{}
	for _, res := range resources {
		byNS[res.Namespace] = append(byNS[res.Namespace], res)
	}
	namespaces := make([]string, 0, len(byNS))
	for ns := range byNS {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	fields, variants := 0, 0
	for _, res := range resources {
		fields += len(res.Fields)
		variants += len(res.Variants)
	}
	d.line(fmt.Sprintf(
		"%s across %s, carrying %s and %s, generated from the pinned ontology "+
			"revision. Nothing here is written by hand (rule 2): the schema comes "+
			"from `sync`, this reference from `docs`, and both from the same model.",
		plural(len(resources), "resource", "resources"),
		plural(len(namespaces), "namespace", "namespaces"),
		plural(fields, "field", "fields"),
		plural(variants, "canonical type", "canonical types")))

	d.heading(2, "How to read this tree")
	d.line("A **general type** is the resource (rule 7). `FCU` is `FanCoilUnit` at " +
		"`buildings/{building}/fanCoilUnits/{fan_coil_unit}`, carrying the union of " +
		"every field any FCU variant uses. A **canonical type** such as " +
		"`FCU_DFSS_CSP_CHWDC` is not a message of its own -- it is a value on that " +
		"resource's enum, carrying its exact field set in an annotation.\n\n" +
		"Directories are the English name, not the ontology tag. The tag is in the " +
		"table below and on every message in `(annotations.entity_type)`.")

	for _, ns := range namespaces {
		list := byNS[ns]
		d.heading(2, fmt.Sprintf("%s (%s)", ns, plural(len(list), "resource", "resources")))
		rows := make([][]string, 0, len(list))
		for _, res := range list {
			link := path.Join(res.Segment, "v1")
			rows = append(rows, []string{
				fmt.Sprintf("[`%s`](%s/)", res.Message, link),
				code(res.Key.Name),
				itoa(len(res.Fields)),
				itoa(len(res.Variants)),
				sentence(res.Description),
			})
		}
		d.table([]string{"Resource", "Tag", "Fields", "Types", "What it is"}, rows)
	}

	d.heading(2, "Also here")
	d.table([]string{"Package", "What it is"}, [][]string{
		{code("annotations/v1"), "the one custom vocabulary (rule 5): the subfield set that *is* a field's identity, an entity type's GUID and namespace, and a canonical type's exact field sets"},
	})

	return Page{
		Path:    path.Join(indexDir, "README.md"),
		Content: d.String(),
	}
}

// indexDir is the root of the generated tree, which is also the root package
// directory PACKAGE_DIRECTORY_MATCH checks against.
const indexDir = "protobuf/digitalbuildings"

// sortStrings is sort.Strings, named here so fields.go does not have to import
// sort for one call.
func sortStrings(s []string) { sort.Strings(s) }
