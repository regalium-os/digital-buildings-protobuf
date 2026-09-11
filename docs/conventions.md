# Conventions

The second half of the working rules. `CLAUDE.md` holds rules 1-9 (structure,
layout, the type cut and the spec pin) and imports this file; these are the
ones about what goes *inside* a file. Same standing as the others -- none is
advisory.

## 10. Three identifiers, not one

The ontology and the building config each carry an identity, and neither is
ours. All three coexist:

- **`uid`** -- AIP-148, `OUTPUT_ONLY`, `(google.api.field_info).format =
  UUID4`. For an entity type this is **the ontology's own GUID**, not a second
  identifier invented beside it. The ontology assigns one to every entity type
  and it is stable across renames, which is exactly what AIP-148 asks a `uid`
  to be. Nothing else in this repository may mint one.
- **`code`** -- the building config's human-readable entity code,
  `FCU-123`. `OPTIONAL` and `IMMUTABLE`. It is locally unique rather than
  globally, is chosen at design time, and must survive a round trip unchanged
  or every re-import duplicates the entity.
- **`name`** -- the resource name, `IDENTIFIER`, assigned by the server.

The building config's *new* format keys entities by GUID and its *old* format
keys them by code. Both are representable because both fields exist; a
generator that collapsed them would make one of the two formats unreadable.

The generator holds numbers 1-15 for these and the connection list, so an
ontology field can never land on one. No literal collides today; the reserved
range is for the bump that introduces one.

## 11. AIP naming traps

Digital Buildings vocabulary is unusually well-behaved here, and the rules
below were **verified against the linter's own source**, not assumed. Three
things a reader coming from coversa-protobuf will expect to bite and which do
not:

- **AIP-140 prepositions.** None of the 1,565 field literals contain `of`,
  `in`, `since` or `including` -- the four that dominated coversa-protobuf's
  catalogue. Two others do bite, in nine literals total, and are below.
- **AIP-140 reserved words** matches the **whole field name** against
  [its list][rw], not each underscore-separated word. So
  `bypass_return_air_temperature_sensor` is fine and zero literals trip; the
  subfields `return`, `static` and `switch` are safe for the same reason.
- **AIP-216** flags **enum names** ending in `Status`, not fields. The 164
  `_status` literals keep their names.

  **The obvious conclusion -- name the enums `...State` -- is wrong, and cost
  394 findings.** [`core::0216::state-field-output-only`][sfo] flags every
  *field* whose enum type ends in `State` unless it is `OUTPUT_ONLY`, and says
  outright that the field name is ignored. Rule 8 makes `command` and
  `setpoint` fields writable and many carry multistates, so `State` would put
  api-linter in permanent conflict with the ontology. The generator uses
  **`...Value`**, which trips neither rule and is more accurate besides.

[sfo]: https://linter.aip.dev/216/state-field-output-only

[rw]: https://github.com/googleapis/api-linter/blob/main/rules/aip0140/reserved_words.go

What actually bites is three rules and thirteen fields. **AIP-142** requires a
`google.protobuf.Timestamp` field to end in `_time`. The ontology's
`timestamp` point type is defined as "an instant in time, represented as a
numeric offset from the epoch," and four literals use it:
`next_event_start_timestamp`, `next_event_end_timestamp`,
`ongoing_event_start_timestamp`, `ongoing_event_end_timestamp`. Either they
stay `int64` and keep the ontology's names, or they become `Timestamp` and are
renamed to `_time`. **They become `Timestamp` and are renamed**, because a
consumer holding an epoch offset with no stated epoch or unit has been handed
a puzzle. The four renames live in `TimeFields` in `sync/catalog`; the
ontology name stays in the comment.

**AIP-140 also bans prepositions**, and eight literals contain `over` --
`source1_over_voltage_status` and its phase variants. That `over` is not a
preposition at all: it is half of "overvoltage", which the subfield table
happens to split. Fusing the two words says the same thing.

**AIP-136 and AIP-140 both ban prepositions in what `REQUEST_TO_EXIT`
generates** -- `GetRequestToExit`, `request_to_exit_id` -- so it becomes
`ExitRequestSensor`, the same industry term without the preposition.

Two traps are applied **by rule rather than by list**, so a later ontology
release cannot introduce one unnoticed: the whole-name reserved-word check
above, and AIP-140's abbreviation table (`specification` -> `spec`, in seven
literals). Everything else is the catalogue at the foot of this file.

One trap is the generator's own. Applying the abbreviation table means
splitting a literal into words, and the splitter that serves message names
also splits digit runs so `DFVSC2X` reads correctly. Run over a *field*
literal it turns `co2` into `co_2` and `phase1` into `phase_1` -- wrong, and a
fresh AIP-140 violation, because api-linter reads a bare `2` as a number in a
field name. 175 fields tripped. `naming.FieldLiteral` splits only on
underscores; `TestFieldLiteralKeepsDigits` pins it.

Resource **patterns** trap differently. AIP-123 requires a name to alternate
collection and identifier. Every resource here is a collection (rule 7), so
the shape is uniform -- `buildings/{building}/fanCoilUnits/{fan_coil_unit}` --
and the singleton hazard coversa-protobuf works around does not arise. The
ontology facets are top-level: `subfields/{subfield}`.

## 12. Enumeration is a repeated field, and the index is translation data

The ontology permits a numeric increment when a device has two points of
identical meaning: `zone_air_temperature_sensor_1`. This is not rare -- 1,439
of the 2,830 field references entity types make are enumerated, and the
indices run to `emergency_battery_status_383`.

Materialising them as numbered proto fields is not an option: it would put 383
fields into one message for one literal, and each new index would be a schema
change. So:

**The base field is emitted once, `repeated`.** `zone_air_temperature_sensor_1`
and `_2` are two elements of `repeated double zone_air_temperature_sensor`,
and `(annotations.field).enumerated` is `true`.

**The index belongs to the translation, not the schema.** `FieldTranslation`
carries an `int32 index`, which is where a sparse or non-contiguous set is
recorded -- the building config's `translation` block is already the thing
that says which device point feeds which standard field, and the index is the
same kind of fact. The schema stays fixed while a device grows a third sensor.

This is sound rather than merely convenient, and the ontology says so: it
warns that "the analysis applied will be unable to apply meaningful
distinctions to such fields," and directs modellers to use *distinct names*
when two points genuinely differ. An index that carried meaning would be a
modelling error upstream, not information this schema is discarding.

A field with no index is still emitted `repeated` if any entity type
enumerates it, so that adding an enumeration upstream is not a type change on
the wire.

## 13. Bounds and states come from the ontology, and `fixed` is not `flexible`

**The two bound kinds are not the same claim, and must not become the same
rule.** Of 1,565 literals, 599 carry only `flexible_*` bounds, 225 carry
`fixed_*` on both ends, and 129 carry `fixed_min` with `flexible_max`.

- **`fixed_*` is a constraint.** It becomes a `buf.validate` numeric rule. A
  percentage really cannot exceed 100.
- **`flexible_*` is an expectation.** It becomes a line in the field comment
  and a `(annotations.field)` range, and **never** a validate rule. These are
  data-quality hints: `air_pressure_sensor` expects 500-200000 Pa, but a
  sensor reading outside that is a fault to report, not a message to reject.
  Turning one into a constraint would drop exactly the telemetry an operator
  most needs to see.

`fixed_min: 0.0` is the common case and is where rule 3's "no `uint32`" lands:
the bound that AIP-141 costs the type is restored here, not by widening to an
unsigned type.

**Multistates collapse hard.** 602 multistate fields draw on only **40
distinct state sets**, so the generator emits 40 enums, not 602 -- one per
distinct set, named for what it represents (`OpenClosedState`,
`ActiveInactiveState`), and shared by every field using it. The three commonest
cover 528 fields on their own: `ACTIVE`/`INACTIVE` (305), `OPEN`/`CLOSED`
(124), `ON`/`OFF` (99).

Two-state sets are **not** silently made `bool`. 15 distinct two-state sets
cover 565 fields, and `OPEN`/`CLOSED` is not `true`/`false` in any direction a
reader would agree on. A set becomes a `bool` only by an explicit entry in
`BooleanStates` in `sync/catalog`, with the reason recorded.

**Units are pinned, not carried.** Each measurement subfield maps to exactly
one quantity kind, and the ontology names one `STANDARD` unit per family --
64 families, 191 unit names, and every family has a standard. The schema pins
the standard unit into `(annotations.field).standard_unit` and restates it in
the comment. It does **not** put the unit on the wire beside the value: two
producers of the same field could then disagree about what the number means,
which is the one failure the ontology is careful to prevent. The building
config's `units` block is a *translation* concern and lives in `UnitMapping`,
mapping a device's native unit token to the standard one.

Five keys in `units.yaml` are aliases naming another family rather than maps
of their own (`diameter`, `length`, `level`, `linearacceleration`,
`impulse`). They resolve to their target; see `sync/ontology`.

## 14. Two annotations, two jobs -- and every citation carries its link

**`google.api.field_behavior`** declares intent, and api-linter requires it on
every field. `IDENTIFIER` on a resource `name`; `OUTPUT_ONLY` for anything the
server sets and for every field rule 8 makes read-only; `IMMUTABLE` on `code`
and on the metadata fields; `OPTIONAL` for a `command` or `setpoint`.

**`buf.validate`** (protovalidate) enforces, and only what rule 13 calls a
constraint. Where two rules would apply to one field they are **intersected**
into one, because protobuf accepts a single rule per field and emitting two is
a compile error.

One trap: a constraint on a possibly-unset field must carry
`(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE`, or an empty string fails
`string.uuid` and every create request is rejected. Every field on these
resources is `optional` (rule 7), so this applies nearly everywhere.

The custom vocabulary of rule 5 is a third thing and does neither job: it
carries provenance a consumer needs at runtime -- the subfield set that *is*
the field's identity. It never restates what `field_behavior` already says.

**Every citation carries its link.** These comments become the generated
documentation in every target language, so a citation without a URL is a
lookup the reader does by hand. AIP mentions get `AIP-131
<https://aip.dev/131>`; ontology references name the namespace-qualified type
or the field literal and link the Digital Buildings documentation.

## 15. The generator fails loudly or not at all

`sync` parses only the shapes the ontology actually uses and errors on
anything else. That is deliberate: the input is a hand-maintained YAML corpus
under continuous change, and a parser that quietly accepts an unfamiliar shape
produces a schema that is wrong in a way no linter catches.

The two shapes that already prove the point are in `sync/ontology` and both
are silent failures, not crashes:

- `states.yaml` has unquoted `ON:` and `OFF:` keys, which YAML 1.1 coerces to
  booleans. Load with string keys or the ontology's two commonest states
  arrive as `True` and `False` -- and 111 fields get an enum with two
  nonsense values that compiles perfectly.
- `units.yaml` mixes maps and alias strings under one key (rule 13).

The same applies downstream. An unknown point type is an error, not a skipped
field. A canonical type whose prefix names no general type is an error, not a
guess. A catalogue entry naming something the ontology no longer has is an
error, not a rename that quietly stops applying.

Run `just survey` after a spec bump before anything else. It reports what was
parsed -- counts per facet, per namespace, per general type -- and every field
name that trips a known AIP rule. It is how the catalogue below was built and
how it should be rechecked.

## The catalogue

Digital Buildings vocabulary and AIP rules disagree in the places below, and
that is the whole list. Applied by rule, not listed: AIP-140's whole-name
reserved-word check and its abbreviation table.

| Source | Proto | Why |
| --- | --- | --- |
| `next_event_start_timestamp` | `next_event_start_time` | AIP-142: a `Timestamp` field must end `_time` (rule 11) |
| `next_event_end_timestamp` | `next_event_end_time` | same |
| `ongoing_event_start_timestamp` | `ongoing_event_start_time` | same |
| `ongoing_event_end_timestamp` | `ongoing_event_end_time` | same |
| `source1_over_voltage_status` (+7 phase variants) | `source1_overvoltage_status` | AIP-140 bans prepositions; `over` here is half of "overvoltage" |
| `PHYSICAL_SECURITY/REQUEST_TO_EXIT` | `ExitRequestSensor` | AIP-136 and AIP-140 both ban the preposition |
| `*_specification` | `*_spec` | AIP-140's abbreviation table, applied by rule |
| a multistate's state set | `...Value` | **not** `...State`: `core::0216::state-field-output-only` would then demand every such field be `OUTPUT_ONLY` |
| `implements` (annotation field) | `implemented_types` | AIP-140 reserved word |
| `qualified_name` (annotation field) | `qualified_type` | AIP-122 bans the `_name` suffix |
| `ON` / `OFF` states | `..._ON_VALUE` / `..._OFF_VALUE` | Cap'n Proto strips the enum prefix and `on` is one of its keywords; inherited from coversa-protobuf |
| `PARIALLY_AGGREGATES` | kept verbatim | an upstream typo for "partially"; it is the wire value, and correcting it here would desynchronise this schema from every config written against the ontology |
