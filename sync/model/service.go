// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"

	"github.com/oh-tarnished/digital-buildings-protobuf/sync/naming"
)

// The service surface: what the collection is called and where its methods are
// routed.
//
// This is here rather than in sync/emit for the same reason field types are
// (sync/model/field.go): two stage-9 targets that each derived the routes would
// drift, and the drift would be a reference documenting a route the service
// does not serve. It did, briefly -- the first Markdown reference rebuilt the
// templates by hand and produced `GET /v1/buildings/*/fanCoilUnits/*` for a
// method actually routed at `GET /v1/{name=buildings/*/fanCoilUnits/*}`. The
// path template's binding is not decoration; it is what tells the transcoder
// which segment is the resource name.

// ServiceName is the AIP-131 service name: the collection, not
// <Message>Service. That is buf STANDARD's convention and the two contradict,
// which is why buf.yaml selects BASIC and lets api-linter own API design
// (rule 1).
func (r *Resource) ServiceName() string { return naming.PluralMessage(r.Message) }

// Method is one standard method, named and routed.
type Method struct {
	Name string // GetFanCoilUnit
	Verb string // GET
	Path string // /v1/{name=buildings/*/fanCoilUnits/*}
	Body string // the google.api.http body, empty where there is none
	Sig  string // the google.api.method_signature
}

// Methods is the six every resource gets. Rule 7: every resource here is a
// collection -- a building may hold any number of anything, including one --
// so there are no AIP-156 singletons and no reduced method sets.
func (r *Resource) Methods() []Method {
	plural := r.ServiceName()
	field := naming.Field(r.Message)
	base, item, patch := r.httpBase(), r.httpItem(), r.httpPatch()

	listSig, createSig := "parent", "parent,"+field+","+field+"_id"
	if r.IsRoot() {
		listSig, createSig = "", field+","+field+"_id"
	}
	return []Method{
		{"Get" + r.Message, "get", item, "", "name"},
		{"List" + plural, "get", base, "", listSig},
		{"Create" + r.Message, "post", base, field, createSig},
		{"Update" + r.Message, "patch", patch, field, field + ",update_mask"},
		{"Delete" + r.Message, "delete", item, "", "name"},
		{"Undelete" + r.Message, "post", item + ":undelete", "*", "name"},
	}
}

// httpBase is the collection route, httpItem one resource, and httpPatch the
// same as httpItem but binding the body's own name field, which is what AIP-134
// asks for.
func (r *Resource) httpBase() string {
	if r.IsRoot() {
		// The root collection has no parent segment to bind.
		return "/v1/" + r.PluralName()
	}
	return fmt.Sprintf("/v1/{parent=buildings/*}/%s", r.PluralName())
}

func (r *Resource) httpItem() string {
	if r.IsRoot() {
		return fmt.Sprintf("/v1/{name=%s/*}", r.PluralName())
	}
	return fmt.Sprintf("/v1/{name=buildings/*/%s/*}", r.PluralName())
}

func (r *Resource) httpPatch() string {
	field := naming.Field(r.Message)
	if r.IsRoot() {
		return fmt.Sprintf("/v1/{%s.name=%s/*}", field, r.PluralName())
	}
	return fmt.Sprintf("/v1/{%s.name=buildings/*/%s/*}", field, r.PluralName())
}
