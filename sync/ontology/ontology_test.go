// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package ontology

import "testing"

// TestStatesAreNotBooleans pins the trap in states.yaml.
//
// The ontology writes `ON:` and `OFF:` unquoted. A YAML 1.1 reader resolves
// those to the booleans true and false, and the two commonest states in the
// ontology -- 111 fields' worth -- then arrive named "True" and "False" and
// compile into a perfectly valid, entirely wrong enum. gopkg.in/yaml.v3
// follows the YAML 1.2 core schema and tags them !!str, so it is safe today;
// yaml.v2 is not, and neither is a decode into map[any]any.
//
// This test is what makes that reliance explicit rather than incidental.
func TestStatesAreNotBooleans(t *testing.T) {
	src := []byte("ON: \"Powered on.\"\nOFF: \"Powered off.\"\nAUTO: \"Automatic.\"\n")
	states, err := parseStates(src)
	if err != nil {
		t.Fatalf("parseStates: %v", err)
	}
	for _, want := range []string{"ON", "OFF", "AUTO"} {
		if _, ok := states[want]; !ok {
			t.Errorf("state %q missing; got %v -- an unquoted ON/OFF was "+
				"resolved to a boolean", want, keys(states))
		}
	}
	for _, bad := range []string{"true", "false", "True", "False"} {
		if _, ok := states[bad]; ok {
			t.Errorf("state %q present: the YAML reader coerced a state name "+
				"to a boolean", bad)
		}
	}
}

// TestUnitAliases pins the other shape units.yaml mixes under one key: five
// measurements are a bare string naming another measurement rather than a map
// of units. A reader expecting a map gets a string, which is a panic or a
// silently empty unit set depending on how it decodes -- both wrong answers
// rather than errors.
func TestUnitAliases(t *testing.T) {
	src := []byte(`
distance:
  meters: STANDARD
  feet:
    multiplier: 0.3048
    offset: 0
diameter: distance
`)
	kinds, err := parseUnits(src)
	if err != nil {
		t.Fatalf("parseUnits: %v", err)
	}
	d, ok := kinds["diameter"]
	if !ok {
		t.Fatal("diameter missing")
	}
	if d.AliasOf != "distance" {
		t.Errorf("diameter.AliasOf = %q, want distance", d.AliasOf)
	}
	if d.Standard != "meters" {
		t.Errorf("diameter.Standard = %q, want meters -- the alias did not "+
			"resolve to its target's units", d.Standard)
	}
	if len(d.Units) != 2 {
		t.Errorf("diameter has %d units, want 2 from distance", len(d.Units))
	}
}

// TestUnitAliasChainIsAnError: the ontology has no alias chains, and one
// appearing upstream should be reported rather than followed.
func TestUnitAliasChainIsAnError(t *testing.T) {
	src := []byte("distance:\n  meters: STANDARD\ndiameter: distance\nwidth: diameter\n")
	if _, err := parseUnits(src); err == nil {
		t.Fatal("an alias chain was accepted; it should be reported")
	}
}

// TestSplitEnumeration covers rule 12's suffix handling, including the
// sub-grouping form the ontology allows.
func TestSplitEnumeration(t *testing.T) {
	for _, tc := range []struct {
		in         string
		base       string
		enumerated bool
	}{
		{"zone_air_temperature_sensor", "zone_air_temperature_sensor", false},
		{"zone_air_temperature_sensor_1", "zone_air_temperature_sensor", true},
		{"emergency_battery_status_383", "emergency_battery_status", true},
		{"supply_air_flowrate_sensor_2_1", "supply_air_flowrate_sensor", true},
		// Not an enumeration: the digit is part of the subfield.
		{"average_zone_air_co2_concentration_sensor", "average_zone_air_co2_concentration_sensor", false},
		{"input_phase1_current_sensor", "input_phase1_current_sensor", false},
	} {
		base, enum := SplitEnumeration(tc.in)
		if base != tc.base || enum != tc.enumerated {
			t.Errorf("SplitEnumeration(%q) = (%q, %v), want (%q, %v)",
				tc.in, base, enum, tc.base, tc.enumerated)
		}
	}
}

// TestSubfieldCategoryIsClosed: the category order is the field grammar, so a
// category appearing upstream that the generator does not order is an error
// rather than a silently mis-placed subfield.
func TestSubfieldCategoryIsClosed(t *testing.T) {
	src := []byte("point_type:\n  sensor: \"A sensor.\"\ninvented_category:\n  thing: \"A thing.\"\n")
	if _, _, err := parseSubfields(src); err == nil {
		t.Fatal("an unknown subfield category was accepted")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
