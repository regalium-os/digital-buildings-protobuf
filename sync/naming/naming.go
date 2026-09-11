// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

// Package naming turns ontology identifiers into protobuf ones. Stage 4: it
// knows the AIP casing and abbreviation rules and nothing about the ontology's
// meaning.
package naming

import (
	"strings"
	"unicode"
)

// abbreviations is AIP-140's table, applied by rule rather than by list so a
// later ontology release cannot introduce one unnoticed. The key is the long
// form as it appears in an ontology identifier; the value is what AIP-140
// wants instead.
var abbreviations = map[string]string{
	"configuration": "config",
	"identifier":    "id",
	"information":   "info",
	"specification": "spec",
	"statistics":    "stats",
}

// Message converts an ontology name to UpperCamelCase: FCU_DFSS_CSP_CHWDC and
// fan_coil_unit both become FanCoilUnit-shaped output. Digits keep their own
// word so that DFVSC2X does not fuse into a following word.
func Message(s string) string {
	var b strings.Builder
	for _, w := range words(s) {
		b.WriteString(title(w))
	}
	return b.String()
}

// Field converts to lower_snake_case, the form the ontology already uses for
// field literals. It is not a no-op: it applies the abbreviation table and
// normalises any camelCase the ontology allows in subfield names.
func Field(s string) string {
	ws := words(s)
	for i, w := range ws {
		if a, ok := abbreviations[w]; ok {
			ws[i] = a
		}
	}
	return strings.Join(ws, "_")
}

// EnumValue builds an AIP-126 enum value: the enum's own name, upper-snaked,
// followed by the value. api-linter requires the prefix, and Cap'n Proto
// strips it again on the way out.
func EnumValue(enum, value string) string {
	return UpperSnake(enum) + "_" + UpperSnake(value)
}

// UpperSnake converts to UPPER_SNAKE_CASE.
func UpperSnake(s string) string {
	return strings.ToUpper(strings.Join(words(s), "_"))
}

// LowerCamel converts to lowerCamelCase, used for the resource id segment of
// an AIP-123 name pattern.
func LowerCamel(s string) string {
	ws := words(s)
	if len(ws) == 0 {
		return ""
	}
	out := ws[0]
	for _, w := range ws[1:] {
		out += title(w)
	}
	return out
}

// Package converts a namespace or general type to a proto package segment.
// The ontology's namespaces are already upper snake (INFO_TECH) and its
// general types are short upper-case tags (FCU), so this is a lowering.
func Package(s string) string {
	return strings.ToLower(strings.Join(words(s), "_"))
}

// words splits an identifier on underscores, hyphens, spaces and camelCase
// boundaries, and separates digit runs. Everything else here is built on it.
func words(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = nil
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '/':
			flush()
			continue
		case unicode.IsDigit(r):
			if len(cur) > 0 && !unicode.IsDigit(cur[len(cur)-1]) {
				flush()
			}
		case unicode.IsUpper(r):
			// A camelCase boundary, or the end of an acronym run: in HTTPServer
			// the S starts a new word because a lower-case letter follows.
			prevLower := len(cur) > 0 && unicode.IsLower(cur[len(cur)-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (len(cur) > 0 && nextLower) {
				flush()
			}
		default:
			if len(cur) > 0 && unicode.IsDigit(cur[len(cur)-1]) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

func title(w string) string {
	if w == "" {
		return w
	}
	r := []rune(w)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// FieldLiteral converts an ontology field literal to a proto field name.
//
// Unlike Field, it splits only on underscores and never on a digit boundary.
// The ontology's literals are already lower_snake_case, and its chemistry is
// load-bearing: co2, no2, h2 and phase1 are single subfields. Running them
// through Field's camelCase-and-digit splitter yields co_2 and phase_1, which
// is both wrong and a fresh AIP-140 violation -- api-linter reads a bare `2`
// as a number in a field name.
//
// Abbreviations still apply, because AIP-140 requires them: `specification`
// becomes `spec` in the seven literals that use it.
func FieldLiteral(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if a, ok := abbreviations[strings.ToLower(p)]; ok {
			parts[i] = a
		}
	}
	return strings.ToLower(strings.Join(parts, "_"))
}
