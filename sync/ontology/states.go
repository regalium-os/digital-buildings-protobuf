// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseStates reads states.yaml, a flat map of state name to definition.
//
// The trap here is ON: and OFF:, which states.yaml writes unquoted. A YAML 1.1
// reader resolves those to the booleans true and false, and the ontology's two
// commonest states -- 111 fields' worth -- then arrive named "True" and
// "False" and compile perfectly into a nonsense enum. gopkg.in/yaml.v3 follows
// the YAML 1.2 core schema and tags them !!str, so it is safe; a decode into
// map[any]any, or a move to yaml.v2, would not be. Decoding through
// yaml.Node keeps the reliance on that explicit rather than incidental, and
// TestStatesAreNotBooleans pins it.
func parseStates(b []byte) (map[string]string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a mapping, got %s", kindName(root.Kind))
	}
	out := map[string]string{}
	var errs errorList
	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		if k.Tag != "!!str" {
			errs.addf("state key %q decoded as %s, not a string: an unquoted ON "+
				"or OFF has been resolved to a boolean", k.Value, k.Tag)
			continue
		}
		if _, dup := out[k.Value]; dup {
			errs.addf("state %s declared twice", k.Value)
			continue
		}
		out[k.Value] = v.Value
	}
	return out, errs.err("states.yaml")
}

// parseConnections reads connections.yaml: a name and a description block.
func parseConnections(b []byte) (map[string]string, error) {
	var raw map[string]struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := map[string]string{}
	var errs errorList
	for name, v := range raw {
		if v.Description == "" {
			errs.addf("connection %s has no description; rule 3 requires a comment "+
				"on every enum value", name)
		}
		out[name] = v.Description
	}
	if len(out) == 0 {
		errs.addf("no connection types found")
	}
	return out, errs.err("connections.yaml")
}
