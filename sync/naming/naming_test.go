// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package naming

import "testing"

func TestMessage(t *testing.T) {
	for in, want := range map[string]string{
		"FCU":              "Fcu",
		"fan_coil_unit":    "FanCoilUnit",
		"BUILDING":         "Building",
		"COORDINATE_BASIS": "CoordinateBasis",
		"fixed_camera":     "FixedCamera",
		"REQUEST_TO_EXIT":  "RequestToExit",
		// A digit run takes the letters that follow it as one token, so 2X is
		// "2x". Canonical type names only reach enum values, which go through
		// UpperSnake and keep the ontology spelling verbatim, so this shape
		// never reaches a consumer-visible identifier.
		"FCU_DFVSC2X_CHWZTC":  "FcuDfvsc2xChwztc",
		"multi_imager_camera": "MultiImagerCamera",
	} {
		if got := Message(in); got != want {
			t.Errorf("Message(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFieldLiteralKeepsDigits is the regression this cost a lint round to
// find. Field splits on digit boundaries so that DFVSC2X reads as three words
// in a message name -- but a field literal must not be split that way: the
// ontology's co2, no2, h2 and phase1 are single subfields, and turning them
// into co_2 and phase_1 is both wrong and a fresh AIP-140 violation, because
// api-linter reads a bare `2` as a number in a field name. 175 fields tripped.
func TestFieldLiteralKeepsDigits(t *testing.T) {
	for in, want := range map[string]string{
		"average_zone_air_co2_concentration_sensor": "average_zone_air_co2_concentration_sensor",
		"input_phase1_current_sensor":               "input_phase1_current_sensor",
		"high_zone_air_h2_concentration_alarm":      "high_zone_air_h2_concentration_alarm",
		"zone_air_no2_concentration_sensor":         "zone_air_no2_concentration_sensor",
		// The abbreviation table still applies: AIP-140 requires it.
		"differential_pressure_specification": "differential_pressure_spec",
	} {
		if got := FieldLiteral(in); got != want {
			t.Errorf("FieldLiteral(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	for in, want := range map[string]string{
		"FanCoilUnit":                "fan_coil_units",
		"AirHandlingUnit":            "air_handling_units",
		"UninterruptiblePowerSupply": "uninterruptible_power_supplies",
		"Boiler":                     "boilers",
		"HeatExchanger":              "heat_exchangers",
		"WindowShade":                "window_shades",
		"Analysis":                   "analyses",
		"Building":                   "buildings",
		// Only the last word is pluralised.
		"LuminaireGroup": "luminaire_groups",
	} {
		if got := Plural(in); got != want {
			t.Errorf("Plural(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestPluralMessage: a message or RPC name needs PascalCase, and using Plural
// directly yields ListautomaticTransferSwitchesRequest -- 252 buf lint
// findings' worth.
func TestPluralMessage(t *testing.T) {
	for in, want := range map[string]string{
		"AutomaticTransferSwitch": "AutomaticTransferSwitches",
		"FanCoilUnit":             "FanCoilUnits",
		"ElectricalPanel":         "ElectricalPanels",
	} {
		if got := PluralMessage(in); got != want {
			t.Errorf("PluralMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEnumValue(t *testing.T) {
	got := EnumValue("FanCoilUnitType", "FCU_DFSS_CSP_CHWDC")
	want := "FAN_COIL_UNIT_TYPE_FCU_DFSS_CSP_CHWDC"
	if got != want {
		t.Errorf("EnumValue = %q, want %q", got, want)
	}
}

func TestPluralCamel(t *testing.T) {
	if got := PluralCamel("FanCoilUnit"); got != "fanCoilUnits" {
		t.Errorf("PluralCamel = %q, want fanCoilUnits", got)
	}
}
