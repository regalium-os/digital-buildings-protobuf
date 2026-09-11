// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package naming

import "strings"

// irregularPlurals is where English defeats the rules below. Keyed by the
// lower-case singular word, which is the last word of a resource name -- so
// one entry covers every resource ending in it.
var irregularPlurals = map[string]string{
	"analysis":  "analyses",
	"index":     "indices",
	"status":    "statuses",
	"apparatus": "apparatuses",
	"bus":       "buses",
	"gas":       "gases",
	"lens":      "lenses",
}

// consonantY is every letter after which a trailing y becomes ies.
const consonantY = "bcdfghjklmnpqrstvwxz"

// Plural is the collection name for a resource. AIP-122 wants the collection
// identifier to be the plural of the singular noun, and api-linter checks the
// `plural` field of google.api.resource against it.
//
// Only the last word is pluralised: an AirHandlingUnit collection is
// airHandlingUnits, not airsHandlingsUnits.
func Plural(singular string) string {
	ws := words(singular)
	if len(ws) == 0 {
		return ""
	}
	last := ws[len(ws)-1]
	ws[len(ws)-1] = pluralWord(last)
	return strings.Join(ws, "_")
}

func pluralWord(w string) string {
	if w == "" {
		return w
	}
	if p, ok := irregularPlurals[w]; ok {
		return p
	}
	switch {
	case strings.HasSuffix(w, "s"), strings.HasSuffix(w, "x"),
		strings.HasSuffix(w, "z"), strings.HasSuffix(w, "ch"),
		strings.HasSuffix(w, "sh"):
		return w + "es"
	case strings.HasSuffix(w, "y") && len(w) > 1 &&
		strings.ContainsRune(consonantY, rune(w[len(w)-2])):
		return w[:len(w)-1] + "ies"
	default:
		return w + "s"
	}
}

// PluralCamel is Plural in the lowerCamelCase a resource pattern uses:
// buildings/{building}/fanCoilUnits/{fan_coil_unit}.
func PluralCamel(singular string) string { return LowerCamel(Plural(singular)) }

// PluralMessage is Plural in UpperCamelCase, for a message or RPC name:
// ListAutomaticTransferSwitches. Plural itself returns snake_case, which is
// what a repeated *field* wants, and using it for a type name yields
// "ListautomaticTransferSwitchesRequest".
func PluralMessage(singular string) string { return Message(Plural(singular)) }
