// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import (
	"fmt"
	"strings"
)

// Inheritance resolution. Split from entity_types.go, which declares the
// types; this file walks them.

// Resolve turns an `implements` reference into a key. The ontology's rules:
// a bare NAME is looked up in the referring namespace then falls back to
// global; /NAME is explicitly global; NS/NAME is explicit.
func (o *Ontology) Resolve(ref, from string) (TypeKey, bool) {
	if i := strings.Index(ref, "/"); i >= 0 {
		ns, name := ref[:i], ref[i+1:]
		if ns == "" {
			ns = Global
		}
		k := TypeKey{Namespace: ns, Name: name}
		_, ok := o.Types[k]
		return k, ok
	}
	if k := (TypeKey{Namespace: from, Name: ref}); o.has(k) {
		return k, true
	}
	k := TypeKey{Namespace: Global, Name: ref}
	return k, o.has(k)
}

func (o *Ontology) has(k TypeKey) bool {
	_, ok := o.Types[k]
	return ok
}

// ResolveFields returns a type's full field set, following implements. The
// ontology is a DAG with sharing -- an FCU variant reaches EQUIPMENT by
// several paths -- so a cycle is reported rather than followed.
func (o *Ontology) ResolveFields(k TypeKey) (map[string]bool, error) {
	return o.fields(k, map[TypeKey]bool{})
}

func (o *Ontology) fields(k TypeKey, stack map[TypeKey]bool) (map[string]bool, error) {
	if stack[k] {
		return nil, fmt.Errorf("inheritance cycle at %s", k)
	}
	e, ok := o.Types[k]
	if !ok {
		return nil, fmt.Errorf("unknown type %s", k)
	}
	stack[k] = true
	defer delete(stack, k)

	out := map[string]bool{}
	for _, f := range e.AllFields() {
		out[f] = true
	}
	for _, ref := range e.Implements {
		parent, ok := o.Resolve(ref, k.Namespace)
		if !ok {
			return nil, fmt.Errorf("%s implements %s, which is not defined", k, ref)
		}
		inherited, err := o.fields(parent, stack)
		if err != nil {
			return nil, err
		}
		for f := range inherited {
			out[f] = true
		}
	}
	return out, nil
}

// Enumerated reports every field a type references with an increment, which
// rule 12 turns into a repeated field.
func (o *Ontology) Enumerated(k TypeKey) map[string]bool {
	out := map[string]bool{}
	var walk func(TypeKey, map[TypeKey]bool)
	walk = func(k TypeKey, seen map[TypeKey]bool) {
		if seen[k] {
			return
		}
		seen[k] = true
		e, ok := o.Types[k]
		if !ok {
			return
		}
		for _, list := range [][]string{e.Uses, e.OptUses} {
			for _, f := range list {
				if base, enumerated := SplitEnumeration(f); enumerated {
					out[base] = true
				}
			}
		}
		for _, ref := range e.Implements {
			if parent, ok := o.Resolve(ref, k.Namespace); ok {
				walk(parent, seen)
			}
		}
	}
	walk(k, map[TypeKey]bool{})
	return out
}
