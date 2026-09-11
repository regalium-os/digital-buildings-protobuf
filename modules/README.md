# modules/

Every upstream this repository reads, and nothing it writes. One submodule
today: `digitalbuildings`, Google's ontology, pinned by `sync/spec.yaml`.

This file is a reader's map of what is actually in there — the seven facets,
the YAML shapes each one uses, and how a directory of hand-maintained YAML
becomes 112 resources. `docs/spec.md` is the pin's history and the procedure
for moving it; this is what the pin points at.

Nothing here is edited. A change to the ontology is made upstream and arrives
by moving the submodule; a change to how the ontology is *read* is made in
`sync/`. There is no third option, and a local edit under `modules/` is
reverted by the next `git submodule update` without saying so.

## Read it with the tool, not by grepping

```sh
just erd                             # every namespace, resources collapsed
just erd electrical                  # one namespace, general types and variants
just erd electrical/generator        # one resource, in full
just erd -view objects               # the primary objects and their relations
just erd -view mermaid hvac          # the same, as a Mermaid erDiagram
```

`sync/cmd/erd` reads this directory and never `protobuf/`. That is deliberate:
a view built by re-reading the emitted schema would agree with the schema by
construction, and so could not show that the schema says something the
ontology does not. Every count in this file was measured against this checkout rather than
carried over from prose, and most of them came out of `erd` itself.

`just survey` is the other half — the per-facet census, to be read after a
spec bump before anything is regenerated.

## The pin

```yaml
# sync/spec.yaml
ontology:
  repository: https://github.com/google/digitalbuildings
  commit: 4a794acf6f01faa88a61f3740bf8d10dacee2483
  date: 2026-09-09
  path: ontology/yaml/resources
```

`just spec` checks the pin against this checkout — commit, date, and a clean
tree — because a stale pin puts a revision into every generated banner that
the files were never generated from, and nothing else can see that is false.

## What is in the submodule

Only one path is an input. The rest is upstream's own tooling and prose, and
the generator does not read a byte of it.

| Path | Read by `sync` | What it is |
| --- | --- | --- |
| `ontology/yaml/resources/` | **yes, all of it** | the vocabulary: seven facets, 122 YAML files |
| `ontology/docs/` | no | upstream's prose, and the authority for everything below. `ontology.md` defines the components, `model.md` the general-type convention, `ontology_config.md` the YAML syntax and its validation rules, `connections.md` the direction of every relationship, `building_config.md` the instance format, `model_hvac.md` the HVAC conventions, `faq.md` the entity categories |
| `ontology/docs/learning/` | no | 16 PDF lesson decks covering the same ground as the Markdown |
| `ontology/rdf/` | no | a 2.2 MB RDF rendering **generated from** the YAML by upstream's own tool. It is an output, not a second source |
| `tools/`, `ibr/`, `styles/` | no | upstream's Python validator, the IBR format, lint config |

## The seven facets

`ontology/yaml/resources/` is seven facets in three shapes. The parser handles
each separately rather than generically, because the input is a
hand-maintained corpus under continuous change and a reader that quietly
accepts an unfamiliar shape produces a schema that is wrong in a way no linter
catches (rule 15).

| Facet | Path | Count | Shape |
| --- | --- | --- | --- |
| subfields | `subfields/subfields.yaml` | 391 in 7 categories | category → literal → definition |
| fields | `fields/telemetry_fields.yaml`, `fields/metadata_fields.yaml` | 1,565 (1,540 + 25) | a list under `literals:`, three value shapes |
| states | `states/states.yaml` | 67 | flat map, literal → definition |
| units | `units/units.yaml` | 69 kinds, 191 names | map of maps — **and five bare strings** |
| connections | `connections/connections.yaml` | 9 | map, name → `{description}` |
| entity types | `<NS>/entity_types/*.yaml`, `entity_types/*.yaml` | 2,637 | map, name → declaration |
| namespaces | the directory names themselves | 13, plus global | — |

### subfields — the field grammar

Seven categories, and **the order they appear in the file is the grammar**:

```text
aggregation_descriptor → aggregation → descriptor → component →
measurement_descriptor → measurement → point_type
```

| Category | Count | Examples |
| --- | --- | --- |
| `aggregation_descriptor` | 3 | `fivesecondrolling`, `tenminutefixed` |
| `aggregation` | 5 | `average`, `max`, `min`, `total` |
| `descriptor` | 217 | `discharge`, `zone`, `chilled`, `exhaust` |
| `component` | 41 | `fan`, `pump`, `coil`, `battery`, `damper` |
| `measurement_descriptor` | 42 | `run`, `linear`, `differential` |
| `measurement` | 69 | `temperature`, `pressure`, `flowrate` |
| `point_type` | 14 | `sensor`, `command`, `setpoint`, `alarm`, … |

A field literal is those subfields concatenated in that order, and the
terminating `point_type` is what rule 8 reads to decide whether anything may
write the field:

```text
discharge_air_temperature_sensor
  discharge  descriptor
  air        descriptor
  temperature  measurement  → the quantity kind, and so the standard unit
  sensor     point_type     → OUTPUT_ONLY

exhaust_fan_run_command
  exhaust    descriptor
  fan        component
  run        measurement_descriptor
  command    point_type     → OPTIONAL, writable
```

Per-field limits: one each of aggregation descriptor, aggregation, measurement
descriptor, measurement and point type; up to ten descriptors and ten
components. Only the point type is required.

**`limit` and `max` are not synonyms, and the distinction is load-bearing.**
`model_hvac.md` reserves `max` and `min` for *aggregation* — the maximum among
several instances of a field — and uses the `limit` descriptor for a boundary
condition on one. So `high_limit_supply_air_temperature_setpoint` is a reset
band on a single setpoint, while `max_supply_air_temperature_setpoint` would
mean the largest setpoint across several devices. An operating limit is
therefore a **field** in its own right, and is unrelated to the `fixed_*` and
`flexible_*` value bounds below.

The ontology is explicit that **the subfield set is the field's identity, not
the name**: `zone_air_temperature_sensor` and `air_zone_temperature_sensor`
are the same field. A proto field name has thrown that set away, which is the
whole argument for the annotation vocabulary of rule 5.

### fields — three value shapes under one list

```yaml
literals:
- manufacturer_label                    # bare: 10 of them, a string
- air_pressure_sensor:                  # numeric: 953
    flexible_min: 500.0
    flexible_max: 200000.0
- air_pressure_status:                  # multistate: 602
  - ACTIVE
  - INACTIVE
```

Which of the three decides the proto type. A numeric range is always exactly
two entries, one min and one max, and each may independently be fixed or
flexible — which is why all four combinations appear.

The two bound kinds are **not the same claim**, and `ontology_config.md` is
explicit about why. "Fixed" means the value "should never be changed and
should always apply across all entities that have the field" — a constraint,
so it becomes a `buf.validate` rule. "Flexible" means the value "may be
adjusted through the range calculation pipeline, which periodically calculates
new ranges for fields by using the interquartile-range method on timeseries
data". A flexible bound is therefore a *fitted statistic that upstream
recalculates*, not a contract. Enforcing one would reject exactly the
out-of-range telemetry an operator needs to see, and would start failing
whenever upstream refits it — see rule 13.

19 of the numeric literals name no measurement subfield. They have no quantity
kind and so no unit, and are integers rather than quantities.

The split between the two files is rule 8's whole answer:
`metadata_fields.yaml` is configuration (`OPTIONAL` + `IMMUTABLE`), and in
`telemetry_fields.yaml` the point type decides — `command` and `setpoint` are
writable, the other twelve are `OUTPUT_ONLY`.

### states — and the YAML 1.1 trap

```yaml
ON: "Powered on."
OFF: "Powered off."
```

Unquoted. A **YAML 1.1** reader resolves those keys to the booleans `true` and
`false`, and the ontology's two commonest states — 111 fields' worth — arrive
named `True` and `False`, compiling into a perfectly valid and entirely wrong
enum. `gopkg.in/yaml.v3` follows the YAML 1.2 core schema and tags them
`!!str`; `yaml.v2` does not, and neither does a decode into `map[any]any`.
`sync/ontology` reads through `yaml.Node` and checks the tag so the reliance
is explicit, and `TestStatesAreNotBooleans` pins it.

602 multistate fields draw on only **40 distinct state sets**, so the schema
gets 40 shared enums rather than 602. Three sets cover 528 fields on their
own: `ACTIVE`/`INACTIVE`, `OPEN`/`CLOSED`, `ON`/`OFF`.

Which fields are multistate is also derivable, and upstream states it twice.
`ontology_config.md` says "any field with a status, alarm, mode, or command
point type"; `ontology.md` refines it — status and mode are *always*
multistates, and a command is one "if not given a measurement subfield". The
generator does not apply that rule: it reads the states the YAML actually
lists, which is more direct. The rule is still worth keeping as a cross-check,
because a field that should carry states and does not would pass silently.

### units — two value shapes under one key

```yaml
temperature:                 # a map: unit name → STANDARD, or a conversion
  kelvins: STANDARD
  degrees_celsius: {multiplier: 1, offset: 273.15}
diameter: distance           # a bare string naming another kind
```

Five keys are aliases rather than maps: **`diameter`, `length`, `level`,
`linearacceleration`, `massconcentration`**. A reader expecting a map gets a
string, which is a panic or a silently empty unit set depending on how it
decodes — both wrong answers rather than errors.

> `docs/conventions.md` rule 13 and `docs/generator.md` both list `impulse` as
> the fifth alias. It is not one: `impulse` is a real kind with
> `newton_seconds: STANDARD`, and it sits one line below `level: distance` in
> the file. The fifth alias is `massconcentration`. The parser derives the set
> rather than listing it, so the code is right and only the prose is wrong.

Every kind has exactly one `STANDARD`. The standard unit is pinned into the
field's annotation and restated in its comment; it is **not** put on the wire
beside the value, because two producers of the same field could then disagree
about what the number means.

### connections — the ERD's only true edges

Nine, global-only, and each is instance-level: a building config asserts them
between entities. An entity type *may* declare required connections — the
syntax is `<source type>: <connection type>` under a `connections:` key — but
upstream marks it "not yet implemented" and no type in this revision uses one.

`CONTAINS` · `CONTROLS` · `FEEDS` · `FULLY_AGGREGATES` · `HAS_PART` ·
`HAS_RANGE` · `MEASURES` · `MEASURES_TYPE` · `PARIALLY_AGGREGATES`

`PARIALLY_AGGREGATES` is an upstream typo for "partially". It is kept verbatim
because it is the wire value, and correcting it here would desynchronise this
schema from every config written against the ontology. Note that
`connections.md` writes the *corrected* spelling in its own example, so
upstream's prose and upstream's YAML disagree; the YAML is what validates.

#### The entity holding the connection is the target

This is the one thing about connections that is easy to get backwards, and
every definition is phrased to make it explicit: "**Source** provides some
media to **Target**". The `connections` block is declared on the **target**,
keyed by the **source**'s GUID, with a list of connection types as the value:

```yaml
# from connections.md: an AHU feeds air to a VAV.
# The block is on the VAV; the key is the AHU.
VAV-GUID:
  code: VAV 1-2
  connections:
    AHU-GUID:
    - FEEDS
```

All nine follow it: the room holds `CONTAINS` naming the building, the
luminaire holds `CONTROLS` naming the lighting control module, the pump holds
`HAS_PART` naming the chilled water system, the breaker meter holds
`FULLY_AGGREGATES` naming the panel meter.

So on a generated resource, **every entry in `connections` names the source**
— the entity acting *on* this one. The emitted `Connection.entity` is
currently documented as "the resource name of the entity at the other end",
which is direction-agnostic and leaves a consumer free to populate it either
way. Writing `ahu.connections = [{FEEDS, vav}]` reads naturally and is the
exact inverse of what the ontology means. Naming the field `source` would
close that, and it is a generator change (`sync/emit`), not an edit to the
1,165 emitted copies.

### entity types — four levels, and rule 7 cuts between two of them

This is the facet that decides the shape of the schema.

```text
namespace         ELECTRICAL              13 of them, plus global
  general type    ATS                     ── the resource: one message,
                                             one service, one package
    canonical     ATS_SRC1_SRC2           ── an enum value on that resource
    abstract      SS, IOBM, PWM, SRC1     ── a functional group: fields only,
                                             never a message
```

| Flag | Count | Upstream meaning | Becomes |
| --- | --- | --- | --- |
| `is_abstract: true` | 844 | "cannot be assigned directly to an entity" | nothing on its own; its fields flow into whatever implements it |
| `is_canonical: true` | 1,587 | "this is a preferred type in your model" | an enum value, carrying its exact field sets in an annotation |
| `allow_undefined_fields: true` | 23 | a **passthrough type** — see below | nothing; the flag is not read |
| declared in `GENERALTYPES.yaml` | 82 with variants | a convention, not a structure | a resource — the message |
| neither flag | 206 | an ordinary, assignable, non-preferred type | see **What the generator drops**, below |

**`is_abstract` is what prevents assignment to an entity, not `is_canonical`.**
That distinction decides how much the last row costs. `is_canonical` only
marks a type as curated — "the `is_canonical` flag lets the modeler
differentiate between 'official' curated types (canonical) and everything
else". So the ontology has **1,793 types a building config may legally name**,
and the schema's `entity_type` enums cover 1,587 of them.

#### Passthrough types

`allow_undefined_fields: true` is a first-class concept and not a comment. A
passthrough type "does not directly correspond to a logical entity in the
model. Instead, a passthrough entity provides translations that will be linked
to one or more other entities", and entities of the type may "define
translations for fields that are not listed as required or optional on this
type". It is mutually exclusive with `is_abstract`, and nothing may inherit
from it.

23 types carry it: the 22 `*_INITIAL` onboarding types and
`GATEWAYS/PASSTHROUGH`. **`sync` does not read the flag at all.** For the
`*_INITIAL` types that costs nothing today, since they are dropped anyway. For
`PASSTHROUGH` it does: it sits in `GATEWAYS/GENERALTYPES.yaml`, so it becomes
the resource `PassthroughGateway` — a closed message with zero standard fields
— when the one thing upstream says about it is that its field set is open.

A canonical type's general type is its name up to the first `_`, cross-checked
against the namespace's `GENERALTYPES.yaml`. That prefix is a *convention*,
not a rule the ontology enforces, and one type breaks it:
`ELECTRICAL/SWITCHBOARD` is canonical and implements `PANEL`. Walking
`implements` to a declared general type is the fallback, and it is not a guess
— it is the ontology stating the relationship outright, which is better
evidence than the name.

> **The resource set hangs on a file name upstream says it ignores.**
> `ontology_config.md`: "File names and subfolder hierarchy below the reserved
> folder names are ignored for the purposes of constructing the ontology. All
> files in all folders under a reserved folder will be read and consolidated
> into the model as if they had been defined in a single file." There is no
> `is_general_type` flag; `model.md` calls `GENERALTYPES.yaml` a convention and
> carries an open "TODO: structurally define how general types are identified".
>
> So `sync` reads a signal the ontology itself discards, and it is the signal
> that decides which 112 types become resources. Upstream renaming, splitting
> or merging that file changes the entire package layout without changing a
> single type — and because the file name is not part of the model, no upstream
> validation would flag it. There is no better signal available today; the
> exposure is worth knowing rather than fixing.

#### The file names inside `entity_types/` are a convention

| File | What it holds |
| --- | --- |
| `GENERALTYPES.yaml` | the namespace's general types — the resources |
| `ABSTRACT.yaml` | functional groups: `SS`, `IOBM`, `PWM`, `RMM` |
| `<TAG>.yaml` | the canonical variants of one general type: `ATS.yaml`, `FCU.yaml` |
| `INITIAL.yaml` | onboarding placeholders, `allow_undefined_fields: true`, explicitly "not to be permanently applied" |
| `ANALYSIS.yaml` (HVAC) | completeness tags: `CONTROL`, `OPERATIONAL` |
| `LOADTYPES.yaml` (METERS) | what a meter measures: `LOADTYPE_MAIN`, `LOADTYPE_PLUG` |
| `entity_types/global.yaml` | the global namespace — see below |
| `<Namespace>.yaml` | a namespace with no general types keeps everything in one file: `Facilities.yaml`, `Carson.yaml` |

Three declaration sites have to be recognised, and only the first is obvious:

1. **`GENERALTYPES.yaml`**, in the eight namespaces that have one: `ELECTRICAL`
   `GATEWAYS` `HVAC` `LIGHTING` `METERS` `PLUMBING` `SAFETY` `TRANSPORT`.
2. **`entity_types/global.yaml`**, which is *not* called `GENERALTYPES.yaml`
   and mixes `PMP`, `SENSOR`, `TK`, `VLV` and `USER_INTERFACE` in with
   `EQUIPMENT` itself and four markers that are not equipment at all
   (`NO_ANALYSIS`, `DEPRECATED`, `REMAP_REQUIRED`, `INCOMPLETE`). Taking the
   whole file would make `EQUIPMENT` a general type and every canonical type
   in the ontology would partition onto it. The discriminator there is
   inheritance, not position: a general type implements `EQUIPMENT` and a
   marker does not.
3. **A namespace with no `GENERALTYPES.yaml` at all** — `CARSON`,
   `FACILITIES`, `INFO_TECH`, `PHYSICAL_SECURITY`, `UNTYPED`. There the entity
   types *are* the resources: `BUILDING`, `FLOOR`, `ROOM`, `DOOR`. This is not
   cosmetic — `PHYSICAL_SECURITY/DOOR_STD` is canonical and implements
   `FACILITIES/DOOR`, so without it a cross-namespace general type is
   invisible and two canonical types cannot be placed.

#### `implements` is a DAG, and resolution is namespaced

```yaml
ATS_SRC1_SRC2:
  guid: "2b385a36-ee72-49ba-9d12-6731c9c502c1"
  description: "Standard automatic transfer switch."
  is_canonical: true
  implements: [ATS, SRC1, SRC2]     # bare: this namespace, then global
  uses:      [master_mode, control_mode, switch_position_mode]
  opt_uses:  [generator_status, fire_alarm]
```

- a **bare** name resolves in the referring namespace, then falls back to global
- a **`/`-prefixed** name is explicitly global: `/SS`, `/EQUIPMENT`
- **`NS/NAME`** is explicit: `CARSON/COORDINATE_BASIS`, `FACILITIES/DOOR`

A type reaches `EQUIPMENT` by several paths, so resolution follows a DAG with
sharing and reports a cycle rather than following it. The resolved field set
is flat, and a resource carries the **union** across all its variants — every
field `optional`, because no variant has them all.

Every entity type carries a `guid`, and it is stable across upstream renames.
That is why catalogue entries for entity types are keyed by GUID and not by
name, and why a resource's `uid` **is** the ontology's GUID rather than a
second identifier invented beside it.

## The entity-relationship model

What the ontology itself is shaped like — `just erd -view mermaid`:

```mermaid
erDiagram
    NAMESPACE ||--o{ GENERAL_TYPE : "declares (112 resources)"
    GENERAL_TYPE ||--o{ CANONICAL_TYPE : "specialised by (1587)"
    CANONICAL_TYPE }o--o{ ABSTRACT_GROUP : "implements"
    GENERAL_TYPE }o--o{ ABSTRACT_GROUP : "implements"
    ABSTRACT_GROUP }o--o{ STANDARD_FIELD : "uses / opt_uses"
    GENERAL_TYPE ||--o{ STANDARD_FIELD : "union across variants"
    STANDARD_FIELD }o--|| POINT_TYPE : "terminates in"
    STANDARD_FIELD }o--o{ SUBFIELD : "composed of, ordered"
    STANDARD_FIELD }o--o| UNIT_KIND : "measurement subfield"
    STANDARD_FIELD }o--o{ STATE_VALUE : "multistate set"
    BUILDING ||--o{ GENERAL_TYPE : "parent of every resource name"
    GENERAL_TYPE ||--o{ CONNECTION : "has"
    CONNECTION }o--|| CONNECTION_TYPE : "typed by"
    CONNECTION }o--|| GENERAL_TYPE : "names the other end"

    NAMESPACE {
        string name "HVAC, ELECTRICAL, … plus GLOBAL"
        int general_types "112"
    }
    GENERAL_TYPE {
        string tag "FCU, ATS, TXMR — 1-4 characters"
        string message "FanCoilUnit — the resource"
        uuid guid "the ontology's own, and the resource uid"
    }
    CANONICAL_TYPE {
        string name "FCU_DFSS_CSP_CHWDC — an enum value"
        uuid guid "rides on the enum value in an annotation"
    }
    ABSTRACT_GROUP {
        string name "844 of them; never a message"
    }
    STANDARD_FIELD {
        string literal "1565 across two field files"
        bool writable "command and setpoint only (rule 8)"
    }
    SUBFIELD {
        string literal "391 in 7 ordered categories"
    }
    CONNECTION_TYPE {
        string name "CONTAINS, CONTROLS, FEEDS, FULLY_AGGREGATES, HAS_PART, HAS_RANGE, MEASURES, MEASURES_TYPE, PARIALLY_AGGREGATES"
    }
```

Two edges in that drawing are not fields, and cannot be. AIP-215 forbids a
field from referencing a message in another proto package, so a `Connection`
carries a type and the *resource name* of the other end, with
`(google.api.resource_reference).type = "*"`. A floor does not hold its fan
coil units.

## The docs describe more than the YAML defines

`ontology/docs/` is the authority on what things *mean*. It is not authority on
what *exists*: its examples cite several types and one connection that are not
in this revision's YAML at all.

| Cited in docs | In the YAML? | Where |
| --- | --- | --- |
| `ELECTRICAL/MSB` | no | `meter_systems.md`, twice |
| `LIGHTING/LIGHTING_FIXTURE` | no | `building_config.md` |
| `LIGHTING/SWITCH_GROUP` | no | `building_config.md` |
| `FACILITIES/ZONE` | no | `hvac_ahu.md` |
| `CONNECTS_TO` | no | `building_config.md`, four times |
| `PARTIALLY_AGGREGATES` | no — the YAML has `PARIALLY_AGGREGATES` | `connections.md` |

**The YAML is authoritative for what exists; the docs are authoritative for
what it means.** The generator reads only the YAML, which is right — but it
means a reader checking a doc example against the emitted schema will find
things missing that were never there.

One case is sharper than drift. `meter_systems.md` instructs the modeller to
"create a loadtype entity" of type `METERS/LOADTYPE_MAIN`, and all twelve
`LOADTYPE_*` types are `is_abstract: true` — which `ontology.md` defines as
"should only be used in inheritance and **not directly associated with any
entities**". Upstream's own documents disagree about whether those twelve can
be instantiated. The generator follows the flag and drops them, so a config
written to `meter_systems.md` names a type this schema cannot express.

## The building config is the instance format, and the schema is missing four of its fields

`building_config.md` is the format a real deployment writes. Most of it maps
onto the schema cleanly and confirms decisions already made — entity
`operation` values of `ADD`, `DELETE`, `UPDATE` and `EXPORT` are Create,
Delete, Update and Get; the per-entity `update_mask` is AIP-134's; the new
format keys entities by GUID and carries `code` as a field, which is why rule
10 needs all three identifiers.

Four things it defines have no representation in the emitted protos:

| Config field | What it does | Status |
| --- | --- | --- |
| `etag` | "required for all entities under an `UPDATE` configuration", compared against the backend datastore so an update only applies if the config is in sync | **absent**; this is AIP-154 optimistic concurrency, and the ontology's own update model depends on it |
| `cloud_device_id` | the registry id of the reporting device; "mandatory when a translation exists" | **absent** |
| `translate_like` | reuse another entity's translation wholesale | **absent** |
| `units.key` | the payload path where the device reports its unit, e.g. `pointset.points.temp_1.units` | **absent** — `UnitMapping` carries the unit and the native token but not where to read it from |

`UnitMapping` is also emitted `repeated`, while upstream states "only one unit
is allowed per field". The schema is laxer than the source, which is the
direction that lets an invalid config through.

What *is* modelled is modelled well: `FieldTranslation.missing` is upstream's
`MISSING` sentinel for a field the device lacks, `StateMapping.native_values`
is `repeated` because a standard state may map to several native values
(`CLOSED: ["2", "3"]`), and `ValueRange` matches `value_range`.

### Link got the direction right; Connection did not

Both describe the same shape — another entity acting on this one — and
`building_config.md` annotates the connections block in its own example with
"**Listed entities are sources on connections**". `Link` says so:

```proto
message Link {
  string source = 1 [(google.api.resource_reference).type = "*"];
  string target_field = 2;   // a field on this entity
  string source_field = 3;   // a field on the source entity
}
```

`Connection` does not — it calls the same thing `entity`, "the resource name
of the entity at the other end". Two messages, one direction, two names. The
fix is to make `Connection` read like `Link`.

## Three entity categories the schema flattens

`faq.md` and `building_config.md` divide every entity three ways, and the
division decides which blocks an entity carries:

| Category | What it is | Carries |
| --- | --- | --- |
| **logical** | the thing you actually want to model and analyse — a VAV, an AHU | a canonical type |
| **reporting** | the thing that sends telemetry — often a controller or gateway | a `translation`, and `cloud_device_id` |
| **virtual** | a logical device with no telemetry of its own; its data is assembled from others | `links` |

A reporting entity may also be logical, and virtual entities usually are. The
passthrough types above are the fourth corner: a gateway "receiving a throwaway
type purely for translation support" so that several virtual entities can link
through it.

The generated resources carry `translations`, `links` and `connections` all
three, on every resource, all optional — which is permissive enough to express
any of the categories and says nothing about which one an instance is. That is
a defensible flattening rather than a gap, but it is a flattening, and a
validator built on this schema cannot reject a virtual entity that also
declares a translation.

## Worked example: ELECTRICAL, end to end

`just erd electrical`:

| Tag | Message | Variants | Fields | Package segment |
| --- | --- | --- | --- | --- |
| `ATS` | `AutomaticTransferSwitch` | 2 | 28 | `automatic_transfer_switch` |
| `BATT` | `Battery` | 0 | 2 | `battery` |
| `CB` | `CircuitBreaker` | 1 | 7 | `circuit_breaker` |
| `PANEL` | `ElectricalPanel` | 1 | 4 | `electrical_panel` |
| `GEN` | `Generator` | 2 | 32 | `generator` |
| `TXMR` | `Transformer` | 1 | 7 | `transformer` |
| `UPS` | `UninterruptiblePowerSupply` | 7 | 61 | `uninterruptible_power_supply` |

Following `ATS` all the way down:

```text
ELECTRICAL/                                   the namespace
  GENERALTYPES.yaml                             declares ATS, is_abstract
    ATS  "Tag for automatic transfer switch (ATS) units."
  ATS.yaml                                      declares its canonical types
    ATS_SRC1_SRC2  implements [ATS, SRC1, SRC2]
    ATS_SPM        implements [ATS, SPM]
  ABSTRACT.yaml                                 declares the groups they mix in
    SRC1  "Source 1 parameters typically for ATS"
    SRC2  "Source 2 parameters typically for ATS"
    SPM   "ATS switch position monitoring."
```

becomes

```text
protobuf/digitalbuildings/electrical/automatic_transfer_switch/v1/
  automatic_transfer_switch.proto        message AutomaticTransferSwitch
  automatic_transfer_switch_type.proto   enum    AutomaticTransferSwitchType
                                                   …_ATS_SRC1_SRC2
                                                   …_ATS_SPM
  service.proto                          service AutomaticTransferSwitches
  messages.proto                         the six requests and responses
  connection.proto, link.proto, …        the value types, copied per package
  <state set>_value.proto                one file per shared multistate enum

  package protobuf.digitalbuildings.electrical.automatic_transfer_switch.v1
  name    buildings/{building}/automaticTransferSwitches/{automatic_transfer_switch}
```

The directory segment is the English name, not the ontology tag.
`electrical/ats` is unreadable to anyone who does not already know the
ontology, and a directory name is the first thing a consumer sees — before any
comment, annotation or README. The tag is not lost: it is on the message, in
`(annotations.entity_type).name`, which is where a program looks for it.

The 28 fields on the message are the union of what both variants reach through
`implements` — `SRC1` and `SRC2` contribute the `source1_*` and `source2_*`
families, `EQUIPMENT` contributes `manufacturer_label` and `model_label`. Two
of the 28 are writable; the other 26 terminate in a read-only point type.

## What the ontology carries per namespace

| Namespace | Resources | Variants | Fields | Largest resource |
| --- | --- | --- | --- | --- |
| `hvac` | 46 | 1,392 | 4,020 | `AirHandlingUnit` (450) |
| `safety` | 17 | 30 | 77 | `LeakDetectionSystem` (16) |
| `lighting` | 9 | 45 | 76 | `LuminaireGroup` (29) |
| `electrical` | 7 | 14 | 141 | `UninterruptiblePowerSupply` (61) |
| `facilities` | 7 | 2 | 2 | `Door` (2) |
| `physical_security` | 7 | 0 | 0 | `BadgeReader` (0) |
| `meters` | 6 | 27 | 100 | `ElectricalMeter` (43) |
| `global` | 5 | 72 | 286 | `Sensor` (113) |
| `plumbing` | 3 | 3 | 18 | `WasteCompactor` (7) |
| `transport` | 1 | 2 | 13 | `Elevator` (13) |
| `carson`, `gateways`, `info_tech`, `untyped` | 1 each | 0 | 0 | — |
| | **112** | **1,587** | **4,733** | |

The distribution is the thing to take away: HVAC is 88% of the canonical types
and 85% of the fields. A change to `HVAC/AHU.yaml` is a large diff; a change
to `CARSON/Carson.yaml` is not.

## What the generator drops

206 entity types carry neither `is_abstract` nor `is_canonical`. 17 are in the
five namespaces with no `GENERALTYPES.yaml`, where they become resources. The
other **189 are in namespaces that do declare general types, and the partition
has no slot for them**. 12 are still reached — something `implements` them, so
their fields flow into whatever does. The remaining 177 are unreachable, and
124 of those declare fields of their own.

| Group | All 189 | Unreachable | …declaring fields |
| --- | --- | --- | --- |
| `*NONCANONICAL*`, `*NON_CANONICAL*` | 112 | 111 | 108 |
| unflagged, otherwise ordinary | 51 | 40 | 16 |
| `*_INITIAL` | 22 | 22 | 0 |
| `*_UNDEFINED` | 4 | 4 | 0 |

Upstream's own semantics settle two of the four rows. `model.md` says
"sometimes devices are so bespoke as to not be worth defining as `canonical`",
which is exactly what the first row is, so dropping it is right. The
`*_INITIAL` types are passthrough types — open field sets by design, declaring
no fields of their own — and a closed message is the wrong shape for one, so
dropping them is right too, as is `*_UNDEFINED`.

What it does *not* settle is assignability. `is_abstract` is the flag that
stops a type being attached to an entity, and none of these 177 carry it, so a
building config may legally name any of them. Dropping a type therefore means
the schema cannot express a config the ontology accepts — for the bespoke
first row that is a deliberate trade, and for the second row it is not.

**The second group is the one that matters**, and `ELECTRICAL/BATT_STD` is the
clearest case:

```yaml
BATT_STD:
  guid: "d5b5721e-1194-4b1b-8e0e-ed24bc1fab70"
  description: "Standard battery."
  implements: [BATT]
  uses:     [voltage_sensor, current_sensor]
  opt_uses: [remaining_charge_time_sensor, battery_charge_status, …]
```

No `is_canonical: true`, so `Battery` is emitted with **no type enum at all
and two fields** — `manufacturer_label` and `model_label`, inherited from
`EQUIPMENT` — while the ontology describes a standard battery with eleven.
`ELECTRICAL/UPS_UPSTR` is the same shape and says `is_canonical: false`
outright; `METERS/EM_ION`, `HVAC/ZONE_HVAC`, `HVAC/HUM_RHHC`,
`LIGHTING/LTGW_BS` and the eight `HVAC/DWST_*` types are others. The largest
by declared fields is `HVAC/FCU_RHC_DFVSC_RTC` at 14.

Read against upstream, `is_canonical: false` on `BATT_STD` is a statement that
it is not a *curated* type — not that it is unusable. An entity can be a
`BATT_STD` today, and this schema has no way to say so.

A third slice of that group is 11 types sitting in `HVAC/ABSTRACT.yaml`
without `is_abstract: true` — `BSWTC`, `SSWTC`, `SRWISOVM` and so on. Those
are reached, because other types implement them, so nothing is lost today; the
flag is missing rather than the type.

This is a live decision, not a defect to patch quietly. Rule 15 says the
generator fails loudly or not at all, and a type matching no branch of the
partition is currently ignored rather than reported. Settling it means
choosing, per group, between *drop and say so*, *treat as canonical*, and
*error until a catalogue entry says which* — and then having
`sync/model/partition.go` enforce the choice rather than fall through.

## Bumping the pin

The procedure is `docs/spec.md`. In short: move the submodule, edit
`sync/spec.yaml`, `just spec`, **`just survey` and `just erd` before
regenerating**, then `just sync` and read `git diff --stat protobuf/`.

What a bump can break, in the order it will bite you:

- **a catalogue entry that no longer applies** — a build error by design
  (`model.CheckCatalogue`). Entity-type entries are keyed by GUID and survive
  an upstream rename; field, state and unit entries are keyed by the literal
  and do not.
- **a field number that moved** — the failure this repository most needs to
  catch, and the one `buf breaking` cannot see, because both sides of a
  comparison between regenerated trees are internally consistent.
  `sync/ordinals.yaml` is what sees it. The ontology inserts new fields
  *alphabetically* into existing `uses` lists, so numbering by position would
  renumber half of a 450-field message on a routine bump.
- **a general type gaining its first canonical variant** — a new resource,
  package, service and directory. Not a break, but a much larger diff than the
  commit range suggests.
- **a general type losing its last variant** — removes a package. An empty
  package is still a published import path: deprecate, do not delete.
- **a new state in an existing multistate set** — changes a shared enum, and
  by rule 13 that enum is shared across every field using the set. A one-line
  ontology change can touch hundreds of fields. Additions are wire-safe;
  confirm the diff is only additions.
- **a new subfield category or point type** — the one that needs thought
  rather than procedure. The category order *is* the field grammar, and a new
  point type has to be classified writable or read-only under rule 8 before
  anything will build.

## Upstream, for reading

Everything in `ontology/docs/`, in the order it pays to read it. All were read
for this file; the citations above are theirs, not this repository's.

| Document | What it settles |
| --- | --- |
| [`ontology.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology.md) | the components. Subfield categories and the field grammar, equivalence by subfield set, enumeration, namespace elevation, and the flag definitions — `is_abstract` blocks assignment, `is_canonical` marks curation, `allow_undefined_fields` makes a passthrough |
| [`ontology_config.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology_config.md) | the YAML syntax and every validation rule. Fixed vs flexible bounds and the IQR pipeline; unit aliases; that file names are ignored when constructing the ontology |
| [`model.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/model.md) | the general-type convention, `GENERALTYPES.yaml` and `ABSTRACT.yaml`, and why bespoke devices are left non-canonical |
| [`connections.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/connections.md) | all nine relationships, each with a worked example — and that the block always sits on the target |
| [`building_config.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/building_config.md) | the instance format: translations, links, `etag`, `cloud_device_id`, INITIALIZE/UPDATE and the per-entity operations |
| [`model_hvac.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/model_hvac.md) | HVAC modelling doctrine, the general-type "smell tests", the three system types, and `limit` vs `max` |
| [`faq.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/faq.md) | logical, reporting and virtual entities; what to model and what not to |
| [`hvac_ahu.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/hvac_ahu.md), [`hvac_fcu.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/hvac_fcu.md), [`hvac_chws.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/hvac_chws.md), [`hvac_hws.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/hvac_hws.md) | worked building configs per equipment class, with the connection patterns each requires |
| [`meter_systems.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/meter_systems.md) | load types and meter hierarchies, and the `MEASURES` / `MEASURES_TYPE` / `*_AGGREGATES` pattern |
| [`overview.md`](https://github.com/google/digitalbuildings/blob/master/ontology/docs/overview.md) | the extension and validation workflow: propose YAML, pass the type validator, then validate instances |
| `docs/learning/` | 16 PDF decks, two modules. Not read for this file — the Markdown above covers the same ground |
