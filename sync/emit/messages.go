// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// messages writes the request and response types.
//
// This is rule 2's single exception to one-message-per-file: a request and its
// response are meaningless apart, and splitting them would produce twelve
// files per package that nobody reads separately.
func (e *Emitter) messages(r *model.Resource) *File {
	f := newFile(r.Dir, "messages", r.Package, e.banner)
	f.importPath(importFieldBehavior)
	f.importPath(importResource)
	f.importPath(importFieldMask)
	f.importPath(importValidate)
	f.importPath(r.Dir + "/" + naming.Field(r.Message) + ".proto")

	plural := naming.PluralMessage(r.Message)
	msg, low := r.Message, naming.Field(r.Message)
	typ := resourceType + "/" + r.Message
	parent := resourceType + "/Building"

	e.getRequest(f, msg, low, typ)
	e.listRequest(f, msg, plural, low, parent, r.IsRoot())
	e.createRequest(f, msg, low, parent, r.IsRoot())
	e.updateRequest(f, msg, low)
	e.deleteRequest(f, msg, low, typ)
	e.undeleteRequest(f, msg, low, typ)
	return f
}

func (e *Emitter) getRequest(f *File, msg, low, typ string) {
	f.comment(0, fmt.Sprintf("Request for `Get%s`.", msg))
	f.line(0, "message Get%sRequest {", msg)
	f.comment(1, fmt.Sprintf("The name of the %s to return.", spaced(msg)))
	f.line(1, "string name = 1 [")
	f.line(2, "(google.api.field_behavior) = REQUIRED,")
	f.line(2, "(google.api.resource_reference).type = %q,", typ)
	f.line(2, "(buf.validate.field).string.min_len = 1")
	f.line(1, "];")
	f.line(0, "}")
	f.line(0, "")
	_ = low
}

func (e *Emitter) listRequest(f *File, msg, plural, low, parent string, root bool) {
	f.comment(0, fmt.Sprintf("Request for `List%s`.", plural))
	f.line(0, "message List%sRequest {", plural)
	if !root {
		f.comment(1, fmt.Sprintf("The building whose %s are listed.", spaced(plural)))
		f.line(1, "string parent = 1 [")
		f.line(2, "(google.api.field_behavior) = REQUIRED,")
		f.line(2, "(google.api.resource_reference).child_type = %q,", resourceType+"/"+msg)
		f.line(2, "(buf.validate.field).string.min_len = 1")
		f.line(1, "];")
		f.line(0, "")
	}
	f.comment(1, "Maximum number to return. The server may return fewer.")
	f.line(1, "int32 page_size = 2 [")
	f.line(2, "(google.api.field_behavior) = OPTIONAL,")
	f.line(2, "(buf.validate.field).int32.gte = 0")
	f.line(1, "];")
	f.line(0, "")
	f.comment(1, "A page token from a previous response.")
	f.line(1, "string page_token = 3 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "}")
	f.line(0, "")

	f.comment(0, fmt.Sprintf("Response for `List%s`.", plural))
	f.line(0, "message List%sResponse {", plural)
	f.comment(1, "The page of results.")
	f.line(1, "repeated %s %s = 1;", msg, naming.Plural(low)) // snake: a field, not a type
	f.line(0, "")
	f.comment(1, "Token for the next page, empty if there are no more.")
	f.line(1, "string next_page_token = 2;")
	f.line(0, "}")
	f.line(0, "")
}

func (e *Emitter) createRequest(f *File, msg, low, parent string, root bool) {
	f.comment(0, fmt.Sprintf("Request for `Create%s`.", msg))
	f.line(0, "message Create%sRequest {", msg)
	if !root {
		f.comment(1, "The building to create it in.")
		f.line(1, "string parent = 1 [")
		f.line(2, "(google.api.field_behavior) = REQUIRED,")
		f.line(2, "(google.api.resource_reference).child_type = %q,", resourceType+"/"+msg)
		f.line(2, "(buf.validate.field).string.min_len = 1")
		f.line(1, "];")
		f.line(0, "")
	}
	f.comment(1, "The id to use, which becomes the last segment of the name. "+
		"By convention this is the building config's entity code.")
	f.line(1, "string %s_id = 2 [(google.api.field_behavior) = OPTIONAL];", low)
	f.line(0, "")
	f.comment(1, "The resource to create.")
	f.line(1, "%s %s = 3 [(google.api.field_behavior) = REQUIRED];", msg, low)
	f.line(0, "}")
	f.line(0, "")
}

func (e *Emitter) updateRequest(f *File, msg, low string) {
	f.comment(0, fmt.Sprintf("Request for `Update%s`.", msg))
	f.line(0, "message Update%sRequest {", msg)
	f.comment(1, "The resource to update. Its `name` identifies it.")
	f.line(1, "%s %s = 1 [(google.api.field_behavior) = REQUIRED];", msg, low)
	f.line(0, "")
	f.comment(1, "The fields to update.\n\n"+
		"Rule 8: naming an OUTPUT_ONLY field here is rejected rather than "+
		"silently ignored. Only `command` and `setpoint` fields, and the "+
		"metadata fields at create time, can be written.")
	f.line(1, "google.protobuf.FieldMask update_mask = 2 [(google.api.field_behavior) = OPTIONAL];")
	f.line(0, "}")
	f.line(0, "")
}

func (e *Emitter) deleteRequest(f *File, msg, low, typ string) {
	f.comment(0, fmt.Sprintf("Request for `Delete%s`.", msg))
	f.line(0, "message Delete%sRequest {", msg)
	f.comment(1, fmt.Sprintf("The name of the %s to delete.", spaced(msg)))
	f.line(1, "string name = 1 [")
	f.line(2, "(google.api.field_behavior) = REQUIRED,")
	f.line(2, "(google.api.resource_reference).type = %q,", typ)
	f.line(2, "(buf.validate.field).string.min_len = 1")
	f.line(1, "];")
	f.line(0, "}")
	f.line(0, "")
	_ = low
}

func (e *Emitter) undeleteRequest(f *File, msg, low, typ string) {
	f.comment(0, fmt.Sprintf("Request for `Undelete%s`.", msg))
	f.line(0, "message Undelete%sRequest {", msg)
	f.comment(1, fmt.Sprintf("The name of the %s to restore.", spaced(msg)))
	f.line(1, "string name = 1 [")
	f.line(2, "(google.api.field_behavior) = REQUIRED,")
	f.line(2, "(google.api.resource_reference).type = %q,", typ)
	f.line(2, "(buf.validate.field).string.min_len = 1")
	f.line(1, "];")
	f.line(0, "}")
	_ = low
}
