// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
)

// standardFields are the AIP fields on every resource, in the 1-15 range rule
// 9 reserves for them.
func (e *Emitter) standardFields(f *File, r *model.Resource) {
	f.comment(1, fmt.Sprintf("Resource name, %q. Assigned by the server.", r.Pattern()))
	f.line(1, "string name = 1 [")
	f.line(2, "(google.api.field_behavior) = IDENTIFIER,")
	f.line(2, "(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
	f.line(1, "];")
	f.line(0, "")

	f.comment(1, "The ontology's GUID for this entity.\n\n"+
		"Rule 10: this is the ontology's own identifier, not a second one "+
		"minted beside it. Digital Buildings assigns a stable UUID to every "+
		"entity and it survives renames, which is exactly what AIP-148 asks a "+
		"`uid` to be.")
	f.line(1, "string uid = 2 [")
	f.line(2, "(google.api.field_behavior) = OUTPUT_ONLY,")
	f.line(2, "(google.api.field_info).format = UUID4,")
	f.line(2, "(buf.validate.field).string.uuid = true,")
	f.line(2, "(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
	f.line(1, "];")
	f.line(0, "")

	f.comment(1, "The building config's human-readable entity code, e.g. "+
		"`FCU-123`.\n\n"+
		"Locally unique rather than globally, chosen at design time, and it "+
		"must survive a round trip unchanged or every re-import duplicates the "+
		"entity. The old building-config format keys entities by this; the new "+
		"one keys them by `uid`. Both are representable because both exist.")
	f.line(1, "string code = 3 [")
	f.line(2, "(google.api.field_behavior) = OPTIONAL,")
	f.line(2, "(google.api.field_behavior) = IMMUTABLE,")
	f.line(2, "(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE")
	f.line(1, "];")
	f.line(0, "")

	if len(r.Variants) > 0 {
		f.comment(1, "The canonical type this entity conforms to.\n\n"+
			"The variant's exact field sets ride on the enum value in "+
			"(annotations.canonical_type), so a validator can reject an entity "+
			"reporting a field its declared type does not have.")
		f.line(1, "%sType entity_type = 4 [(google.api.field_behavior) = OPTIONAL];", r.Message)
		f.line(0, "")
	}

	f.comment(1, "Directed relationships to other entities.\n\n"+
		"Rule 4: a connection is not a field. AIP-215 forbids referencing a "+
		"message in another proto package, and the other end of a connection "+
		"may be any resource in any namespace -- so it is carried as a resource "+
		"name with a wildcard reference.")
	f.line(1, "repeated Connection connections = 5 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")

	f.comment(1, "Standard fields mapped from another entity's telemetry.\n\n"+
		"The building config's `links` block: a virtual entity reports data "+
		"linked from reporting devices rather than emitting it itself.")
	f.line(1, "repeated Link links = 6 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")

	f.comment(1, "How this entity's native payload maps onto the standard "+
		"fields, when it reports telemetry directly.")
	f.line(1, "repeated FieldTranslation translations = 7 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "")

	f.comment(1, "Held for the fields on every read path (rule 9).")
	f.line(1, "reserved 8 to %d;", model.ReservedOrdinals)
	f.line(0, "")
}
