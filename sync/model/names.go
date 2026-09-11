// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// What a resource is called, in the three forms AIP asks for: the singular and
// plural of google.api.resource, and the AIP-123 name pattern built from them.
//
// Split from resource.go, which builds the resource: these read the finished
// one. The path segment a resource lives at is not here -- it is set in
// buildResource, because the package path has to exist before anything is
// emitted into it.

// Singular is the resource's singular form for google.api.resource.
func (r *Resource) Singular() string { return naming.LowerCamel(r.Message) }

// PluralName is the collection identifier in the resource pattern.
func (r *Resource) PluralName() string { return naming.PluralCamel(r.Message) }

// IsRoot reports whether this resource is the top of the name hierarchy.
//
// FACILITIES/BUILDING is, and it is the only one. Every other resource is
// named buildings/{building}/..., so Building itself cannot be -- that would
// make its pattern buildings/{building}/buildings/{building}, and AIP-127
// then has no pattern to match the HTTP template against.
func (r *Resource) IsRoot() bool {
	return r.Key.Namespace == "FACILITIES" && r.Key.Name == "BUILDING"
}

// Pattern is the AIP-123 resource name pattern. Every resource is a
// collection (rule 7): there are no singletons here.
func (r *Resource) Pattern() string {
	if r.IsRoot() {
		return fmt.Sprintf("%s/{%s}", r.PluralName(), naming.Field(r.Message))
	}
	return fmt.Sprintf("buildings/{building}/%s/{%s}", r.PluralName(), naming.Field(r.Message))
}
