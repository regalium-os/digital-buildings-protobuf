// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ordinals

import (
	"path/filepath"
	"testing"
)

const msg = "protobuf.digitalbuildings.hvac.fcu.v1.FanCoilUnit"

// TestSlotsSurviveInsertion is rule 9's whole point.
//
// The ontology adds fields to existing types alphabetically, so a field can
// appear in the middle of a `uses` list on a routine bump. A generator
// numbering by position would renumber everything after it -- a wire break
// that no linter sees, and that `buf breaking` cannot see either, because run
// against a regenerated tree both sides are internally consistent.
func TestSlotsSurviveInsertion(t *testing.T) {
	l := &Ledger{Messages: map[string]*Message{}}
	first := l.Assign(msg, "zone_air_temperature_sensor")
	second := l.Assign(msg, "zone_air_temperature_setpoint")

	// A later bump inserts a field that sorts before both.
	inserted := l.Assign(msg, "aisle_brightness_percentage_command")

	if got := l.Assign(msg, "zone_air_temperature_sensor"); got != first {
		t.Errorf("existing slot moved: %d -> %d", first, got)
	}
	if got := l.Assign(msg, "zone_air_temperature_setpoint"); got != second {
		t.Errorf("existing slot moved: %d -> %d", second, got)
	}
	if inserted <= second {
		t.Errorf("inserted field got %d, which is not a fresh slot above %d",
			inserted, second)
	}
}

// TestReservedRangeIsHeld: numbers 1-15 belong to name, uid, code,
// entity_type and the connection list.
func TestReservedRangeIsHeld(t *testing.T) {
	l := &Ledger{Messages: map[string]*Message{}}
	if n := l.Assign(msg, "first_field"); n <= Reserved {
		t.Errorf("first ontology field got %d, inside the reserved range", n)
	}
}

// TestRetiredNumbersAreNotReissued: a consumer may still hold a message that
// used a removed field's number, so it goes to `reserved` rather than back
// into circulation.
func TestRetiredNumbersAreNotReissued(t *testing.T) {
	l := &Ledger{Messages: map[string]*Message{}}
	doomed := l.Assign(msg, "field_upstream_will_remove")
	l.Assign(msg, "field_that_stays")

	l.Retire(msg, map[string]bool{"field_that_stays": true})

	res := l.ReservedFor(msg)
	if len(res) != 1 || res[0] != doomed {
		t.Fatalf("ReservedFor = %v, want [%d]", res, doomed)
	}
	if n := l.Assign(msg, "a_brand_new_field"); n == doomed {
		t.Errorf("retired number %d was reissued", n)
	}
}

// TestRoundTrip: a reload assigns nothing new, which is what makes
// `just verify-schema` able to fail on a moved slot.
func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ordinals.yaml")
	l := &Ledger{Messages: map[string]*Message{}, path: path}
	want := map[string]int{}
	for _, f := range []string{"alpha_sensor", "beta_command", "gamma_setpoint"} {
		want[f] = l.Assign(msg, f)
	}
	if err := l.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for f, n := range want {
		if got := reloaded.Assign(msg, f); got != n {
			t.Errorf("after reload %s = %d, want %d", f, got, n)
		}
	}
	if reloaded.Dirty() {
		t.Error("reloading and reassigning the same fields marked the ledger dirty; " +
			"verify-schema would report a spurious change")
	}
}
