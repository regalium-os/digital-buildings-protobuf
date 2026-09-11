// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package emit

import (
	"fmt"
	"strings"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/model"
	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// host is the API host every service declares (AIP-191).
const host = resourceType

// service writes the CRUD surface. Rule 7: every resource is a collection --
// there are no singletons here, because no statement in the ontology says a
// building holds exactly one of anything -- so every one gets the full set
// plus AIP-164 Undelete.
func (e *Emitter) service(r *model.Resource) *File {
	f := newFile(r.Dir, "service", r.Package, e.banner)
	f.importPath(importAnnotations)
	f.importPath(importClient)
	f.importPath(importEmpty)
	f.importPath(r.Dir + "/" + naming.Field(r.Message) + ".proto")
	f.importPath(r.Dir + "/messages.proto")

	plural := r.ServiceName()
	f.comment(0, fmt.Sprintf(
		"Manages %s.\n\n"+
			"AIP-131 names the service for the collection, not `%sService`: "+
			"that is buf STANDARD's convention and the two contradict, which is "+
			"why buf.yaml selects BASIC and lets api-linter own API design "+
			"(rule 1).", spaced(plural), r.Message))
	f.line(0, "service %s {", plural)
	f.line(1, "option (google.api.default_host) = %q;", host)
	f.line(0, "")

	// The names and routes are model's (sync/model/service.go); what is left
	// here is the proto text and the prose.
	for _, m := range r.Methods() {
		e.rpc(f, rpc{
			name: m.Name,
			in:   m.Name + "Request",
			out:  methodOutput(r, m),
			doc:  methodDoc(r, m, plural),
			http: httpOption(m),
			sig:  m.Sig,
		})
	}
	f.line(0, "}")
	return f
}

// methodOutput is the RPC's return type. Delete returns Empty (AIP-135), List
// its own response message, and everything else the resource.
func methodOutput(r *model.Resource, m model.Method) string {
	switch {
	case strings.HasPrefix(m.Name, "Delete"):
		return "google.protobuf.Empty"
	case strings.HasPrefix(m.Name, "List"):
		return m.Name + "Response"
	default:
		return r.Message
	}
}

func methodDoc(r *model.Resource, m model.Method, plural string) string {
	switch {
	case strings.HasPrefix(m.Name, "Get"):
		return fmt.Sprintf("Returns one %s.", spaced(r.Message))
	case strings.HasPrefix(m.Name, "List"):
		return fmt.Sprintf("Lists the %s in a building.", spaced(plural))
	case strings.HasPrefix(m.Name, "Create"):
		return fmt.Sprintf("Creates a %s.", spaced(r.Message))
	case strings.HasPrefix(m.Name, "Update"):
		return fmt.Sprintf("Updates a %s.\n\n"+
			"Rule 8: an update_mask naming an OUTPUT_ONLY field is rejected "+
			"rather than silently ignored. Most fields here are OUTPUT_ONLY -- "+
			"the equipment reports them and no API call sets them.", spaced(r.Message))
	case strings.HasPrefix(m.Name, "Delete"):
		return fmt.Sprintf("Deletes a %s.", spaced(r.Message))
	default:
		return fmt.Sprintf("Restores a deleted %s (AIP-164 "+
			"<https://aip.dev/164>).", spaced(r.Message))
	}
}

// httpOption renders one google.api.http block: the verb and its template,
// plus a body where the method has one.
func httpOption(m model.Method) string {
	s := m.Verb + ": " + fmt.Sprintf("%q", m.Path)
	if m.Body != "" {
		s += "\n      body: " + fmt.Sprintf("%q", m.Body)
	}
	return s
}

type rpc struct {
	name, in, out, doc, http, sig string
}

func (e *Emitter) rpc(f *File, r rpc) {
	f.comment(1, r.doc)
	f.line(1, "rpc %s(%s) returns (%s) {", r.name, r.in, r.out)
	f.line(2, "option (google.api.http) = {")
	f.line(3, "%s", r.http)
	f.line(2, "};")
	f.line(2, "option (google.api.method_signature) = %q;", r.sig)
	f.line(1, "}")
	f.line(0, "")
}
