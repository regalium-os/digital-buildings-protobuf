// Copyright 2026 RegaliumOS™.
// SPDX-License-Identifier: Apache-2.0

package catalog

// generalTypeNames is what the ontology's 1-4 character equipment tags are
// called in English. Rule 7 makes each of these a resource, and `FCU` is not a
// message name a consumer can read.
//
// Every name here comes from the general type's own description in the
// ontology, not from invention: AHU is "Tag for air-handling units", so it is
// AirHandlingUnit. A tag shared across namespaces (PMP, SENSOR, TK, VLV) maps
// once -- the packages differ, so the message names may agree.
//
// A canonical type whose prefix is not in this table is a build error, not a
// guess. See model.CheckCatalogue.
var generalTypeNames = map[string]string{
	// GLOBAL. Declared in entity_types/global.yaml rather than a
	// GENERALTYPES.yaml, and shared by several namespaces -- which is why
	// they resolve to one resource each rather than one per namespace.
	"PMP":            "Pump",
	"SENSOR":         "Sensor",
	"TK":             "Tank",
	"VLV":            "Valve",
	"USER_INTERFACE": "UserInterface",

	// ELECTRICAL
	"ATS":   "AutomaticTransferSwitch",
	"CB":    "CircuitBreaker",
	"BATT":  "Battery",
	"GEN":   "Generator",
	"PANEL": "ElectricalPanel",
	"TXMR":  "Transformer",
	"UPS":   "UninterruptiblePowerSupply",

	// HVAC
	"ADY":      "AirDryer",
	"AHU":      "AirHandlingUnit",
	"AION":     "AirIonizer",
	"ASHP":     "AirSourceHeatPump",
	"BLR":      "Boiler",
	"CDWS":     "CondensingWaterSystem",
	"CH":       "Chiller",
	"CO":       "ChangeoverUnit",
	"CHGS":     "GlycolSystem",
	"CHWRSR":   "ChilledWaterRiser",
	"CHWS":     "ChilledWaterSystem",
	"CMP":      "AirCompressor",
	"CRREF":    "ColdroomRefrigeration",
	"CT":       "CoolingTower",
	"CU":       "CompartmentUnit",
	"DC":       "DryCooler",
	"DFR":      "DuctFurnace",
	"DH":       "DuctHeater",
	"DHWT":     "DomesticHotWaterTank",
	"DMP":      "Damper",
	"DOAS":     "DedicatedOutdoorAirSystem",
	"DWST":     "DomesticWaterSystem",
	"FAN":      "Fan",
	"FARSR":    "FreshAirRiser",
	"FCU":      "FanCoilUnit",
	"FRZ":      "Freezer",
	"GTWS":     "GeothermalWaterSystem",
	"HOOD":     "Hood",
	"HWS":      "HeatingWaterSystem",
	"HWSRSR":   "HeatingWaterRiser",
	"HUM":      "Humidifier",
	"HX":       "HeatExchanger",
	"IGNORE":   "IgnoredDevice",
	"LANDLORD": "LandlordEquipment",
	"MAU":      "MakeUpAirUnit",
	"PCU":      "PollutionControlUnit",
	"RARSR":    "ReturnAirRiser",
	"RP":       "RadiantPanel",
	"RSR":      "DistributionRiser",
	"SDC":      "WindowShade",
	"TST":      "ThermalStorageTank",
	"UH":       "UnitHeater",
	"VAV":      "VariableAirVolumeUnit",
	"WEATHER":  "WeatherStation",
	"WSHP":     "WaterSourceHeatPump",
	"ZONE":     "Zone",

	// LIGHTING
	"ELT":  "EmergencyLuminaire",
	"LCM":  "LightingControlModule",
	"LGRP": "LuminaireGroup",
	"LKP":  "LightingKeypad",
	"LS":   "LightingSensor",
	"LT":   "Luminaire",
	"LTB":  "LightingBattery",
	"LTGW": "LightingGateway",

	// METERS
	"EM":  "ElectricalMeter",
	"FM":  "FlowMeter",
	"GM":  "GasMeter",
	"HM":  "HeatMeter",
	"MTR": "Meter",
	"WM":  "WaterMeter",

	// PHYSICAL_SECURITY
	"DOOR": "Door",

	// PLUMBING
	"RO":   "ReverseOsmosisUnit",
	"WSR":  "WaterSoftener",
	"WSTC": "WasteCompactor",

	// SAFETY
	"BGU":  "BreakGlassUnit",
	"COHS": "CarbonMonoxideHornStrobe",
	"EHT":  "HeatTracer",
	"FACP": "FireAlarmControlPanel",
	"FAS":  "FireAlarmSystem",
	"FD":   "FireDamper",
	"FDR":  "FireDoor",
	"FHP":  "FireHydrantPump",
	"FHS":  "FireHornStrobe",
	"FSS":  "FireSuppressionSystem",
	"HDS":  "HydrogenSensor",
	"LDS":  "LeakDetectionSystem",
	"PA":   "PublicAddressSystem",
	"RDT":  "RodentRepellentSystem",
	"SD":   "SmokeDetector",
	"SSS":  "SeismicDetector",
	"VBS":  "VibrationDetectionSystem",

	// TRANSPORT
	"ELV": "Elevator",

	// GATEWAYS and INFO_TECH. Both namespaces hold a single type, and both
	// names are abbreviations rather than English.
	"PASSTHROUGH": "PassthroughGateway",
	"PRNTR":       "Printer",
}

// GeneralTypeName returns the English message name for an equipment tag.
func GeneralTypeName(tag string) (string, bool) {
	n, ok := generalTypeNames[tag]
	return n, ok
}

// GeneralTypeTags is every tag the table names, for CheckCatalogue.
func GeneralTypeTags() []string {
	out := make([]string, 0, len(generalTypeNames))
	for t := range generalTypeNames {
		out = append(out, t)
	}
	return out
}
