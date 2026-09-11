# Decisions

Why the rules are what they are. Each of these gives something up; the entry
says what, so that a later reader can reopen the question with the cost in
front of them rather than rediscovering it.

## A general type is the resource, not a canonical type

The ontology has 1,587 canonical types. A message each would mean 1,587
messages and, if resource-hood followed, 1,587 CRUD services. That is not a
service surface -- it is a catalogue with an HTTP binding.

The cut is one level up, at the **general type**: 82 of them carry canonical
variants and 30 more are declared without any, giving 112 resources. Each
carries the union of
every field any of its variants uses. `FanCoilUnit` is 301 fields over 195
variants; `AirHandlingUnit` is 450 over 410.

**What this gives up: compile-time enforcement of a variant's field set.** A
`FanCoilUnit` declaring type `FCU_DFSS_CSP_CHWDC` can, as far as protobuf is
concerned, set any of the 301 fields, including ones that variant does not
have. With a message per canonical type, that would not compile.

**Why it is worth it, and why nothing is actually lost at runtime.** The exact
sets ride on the enum value in `(annotations.canonical_type)` --
`implements`, `uses` and `opt_uses`, verbatim. A validator reads the
annotation and rejects the same payloads the type system would have, and it is
a validator a consumer must run anyway: the ontology's own instance validator
exists because a building config can be wrong in ways no schema catches.

The deciding argument is that the alternative fails on its own terms. 410
message types for AHU variants would each still need every optional field its
abstract groups contribute, because `opt_uses` is genuinely optional -- so the
"exact" version is exact about which fields *may* appear and silent about
which *will*, which is the same guarantee, at 20x the file count.

## There is no `system/` package family

An earlier draft of rule 6 routed non-`EQUIPMENT` general types to a separate
`system/` family, reasoning that the ontology models chilled and heating water
systems as general types that are not equipment.

Running the generator disproved the discriminator outright. "Does not
implement `EQUIPMENT`" selects `CHWS`, `HWS`, `CDWS` and `GTWS` -- correct --
but also `LGRP` (a luminaire group), `WEATHER` (a weather station) and
`FACILITIES/DOOR`, none of which are systems. And it *misses* `CHGS` and
`DWST`, which are a glycol system and a domestic water system by their own
descriptions and do implement `EQUIPMENT`.

The ontology is not consistent on this axis, so no derivable rule selects
"system". The alternative was a hand-kept list -- a fourteenth table to go
stale on every bump, for a purely cosmetic grouping. Resources are routed by
the ontology's own namespace instead: `hvac/chilled_water_system`. **What this gives up:**
systems are not visually grouped. They were never a category the ontology
actually declares.

## Abstract functional groups are not messages

The third option was a message per abstract group (`SD`, `ZTC`, `DFSS`) with
canonical types composing them as submessages. It mirrors the ontology's own
composition most closely, and it was rejected on rule 4.

Abstract groups cross namespaces constantly -- `ZTC` is used by FCU, VAV, UH
and more. AIP-215 forbids a field referencing a message in another package, so
each group would be copied into every package using it, and a payload would
nest three or four levels deep to reach a temperature. The composition is
preserved instead where it costs nothing: in `(annotations.canonical_type)`.

## Shared value types are copied into each package, not imported

`Translation`, `FieldTranslation`, `UnitMapping`, `StateMapping`, `Link`,
`Connection` and `ValueRange` appear in many packages. Rule 4 forbids a
cross-package reference and exempts only `google.*` -- and this repository
declines even that exemption.

Inherited from coversa-protobuf, for a reason that is about tooling rather
than taste: `protoc-gen-buffers` renders `google.type.*` messages into neither
FlatBuffers nor Cap'n Proto, **silently**. A field simply vanished from the
emitted table with no diagnostic. A schema whose serialization targets quietly
disagree with it is worse than a schema with duplication in it.

**What this gives up: one definition.** Seven types are emitted into ~90
packages. It is tolerable only because they are *generated* -- `sharedValueTypes`
in `sync/model` is the single definition, and no human maintains the copies.

## The building configuration file is not a message

A `BuildingConfig` message with an `entities` map would be the direct
translation of the YAML, and it would be the wrong shape. The building config
is a **request format**, not a resource:

- Its `INITIALIZE` and `UPDATE` modes are `Create`, `Update` and `Delete` on
  the resources of rule 7.
- Its `translation`, `links` and `connections` blocks are fields *on* those
  resources.
- Its entity keys -- GUID in the new format, code in the old -- are `uid` and
  `code` (rule 10).

Modelling it as a message would give consumers two ways to say the same thing
and no way to say which is authoritative. A tool that wants to *ship* a
building config still can: it is the sequence of requests, and that sequence
is what an API can validate incrementally rather than all-or-nothing.

## An enumerated field is repeated, and the index is translation data

Rule 12 has the mechanism. The decision is that the ontology's `_1`, `_2`
suffix is **instance data that leaked into the vocabulary**, and the schema
should not follow it there.

The evidence is `emergency_battery_status_383`. No schema wants 383 fields for
one literal, and each new battery would otherwise be a schema change.

**What this gives up: a sparse index set survives only in the translation.** A
device reporting `_1` and `_7` and nothing between has that fact recorded in
`FieldTranslation.index`, not in the field's shape.

The ontology supports this directly: it warns that "the analysis applied will
be unable to apply meaningful distinctions to such fields" and tells modellers
to use distinct names when two points genuinely differ. An index that carried
meaning is an upstream modelling error, not information being discarded here.

## `flexible` bounds are documentation; only `fixed` bounds are constraints

599 field literals carry only `flexible_min`/`flexible_max`; 225 carry
`fixed_*` on both ends.

Emitting both as `buf.validate` rules would have been the obvious reading and
is wrong. `air_pressure_sensor` has a flexible range of 500-200000 Pa. A
sensor reading outside it is precisely the fault an operator needs to see, and
a validate rule would reject the message carrying it. The ontology uses
flexible bounds for data-quality measurement, not admission control.

So `fixed_*` becomes a constraint, `flexible_*` becomes a comment line and an
annotation range. **What this gives up:** nothing enforces a flexible range,
which is correct, but it does mean two similar-looking YAML keys produce very
different artefacts. `just survey` reports the split so it stays visible.

## Multistate enums are shared by state set, not minted per field

602 multistate fields draw on 40 distinct state sets. One enum per field would
be 602 enums, most of them `{ACTIVE, INACTIVE}` under 305 different names, and
a consumer could not pass one field's value to a function expecting another's.

One enum per distinct set gives 40, and equality between two fields with the
same states is expressible. **What this gives up:** if the ontology later adds
a state to one field's set and not another's, a shared enum splits in two, and
that is a rename for every field that moves to the new one. The alternative
front-loads that cost onto all 602. `docs/spec.md` lists it as a thing to look
for on a bump.

## A two-state set does not become a `bool`

15 distinct two-state sets cover 565 fields, so a rule making them `bool`
would touch a third of the schema and read as an obvious simplification.

It is not one. `OPEN`/`CLOSED` has no agreed mapping to `true`/`false`, and
`PRESENT`/`ABSENT`, `NORMAL`/`REVERSED` and `AUTO`/`MANUAL` are worse. A
consumer that guesses wrong inverts a damper. Worse, `bool` has no third
state, so a later ontology bump adding `UNKNOWN` -- which has already happened
to at least ten fields -- becomes a wire break rather than an added enum value.

A set becomes `bool` only by an explicit entry in `BooleanStates` with the
reason recorded. **What this gives up:** ergonomics, deliberately.

## `uid` is the ontology's GUID, and there is a third identifier

Coversa-protobuf keeps `uid` (ours, server-minted) strictly apart from the
source model's own id (`vdm_uid`), because VDM ids are whatever the
originating system chose.

Digital Buildings is different: it assigns a stable UUID to every entity type
and to every entity in a building config, and it is stable *across renames* --
which is exactly what AIP-148 asks a `uid` to be. Minting a second one beside
it would create two answers to "which entity is this."

So `uid` *is* the GUID, and the human-readable entity code becomes a separate
`code` field. Three identifiers, none redundant (rule 10). **What this gives
up:** the server cannot mint a `uid` for an entity that has no GUID yet. That
is the correct constraint -- the ontology's own tooling has a GUID generator,
and an entity without one is not yet onboarded.

## The four `_timestamp` fields become `Timestamp` and are renamed

The ontology defines the `timestamp` point type as "an instant in time,
represented as a numeric offset from the epoch" -- which states neither the
epoch nor the unit. Four literals use it.

They could have stayed `int64` with their ontology names, tripping no rule
(AIP-142 flags `Timestamp` fields *not* ending in `_time`, not fields ending
in `_timestamp`). They become `google.protobuf.Timestamp` and get `_time`
names anyway, because an unqualified epoch offset is a puzzle for the
consumer and the type solves it.

**What this gives up:** four field names that no longer match the ontology
literal. The literal is in the comment and in `(annotations.field)`, and this
is the only place in the schema where a field name diverges from its source.

## Multistate enums are suffixed `Value`, not `State`

AIP-216 flags an enum named `*Status`, so the obvious name for an
`OPEN`/`CLOSED` enum is `OpenClosedState`. That is wrong, and it cost 394
api-linter findings to discover.

`core::0216::state-field-output-only` flags every *field* whose enum type name
ends in `State` unless the field is `OUTPUT_ONLY`, and the rule documentation
says explicitly that the field name is ignored -- the trigger is purely the
type name. Rule 8 makes `command` and `setpoint` fields writable, and plenty of
them carry multistates, so `State` would put api-linter in permanent conflict
with the ontology.

`Value` trips neither rule and is more accurate besides: these are domain
multistates, not the resource lifecycle state AIP-216 concerns itself with.
**What this gives up:** nothing, other than the more familiar suffix.

## Field numbers come from a ledger, not from position

The ontology inserts new fields **alphabetically into existing `uses` lists**.
A generator numbering by position would therefore renumber roughly half of an
813-field message on a routine bump.

This is the failure mode this repository is most exposed to and the one
tooling is least able to see: `buf breaking` compares two trees, and if both
are regenerated from their own pins, both are internally consistent. The break
is real and invisible.

`sync/ordinals.yaml` is append-only, committed, and marked `-merge` in
`.gitattributes` so git cannot resolve a conflict in it by guessing -- a
conflict there means two branches each assigned a slot, which is a
compatibility decision a merge algorithm must not make.

## Both target compilers run, and `buf build` does not replace them

`just schema` emits `.fbs` and `.capnp` and then runs `flatc` and `capnp` on
them. `buf build` proves the protobuf is valid and says nothing about whether
the derived schemas compile: different grammars, different keyword sets,
different ordinal models. The Cap'n Proto keyword collision in the
`docs/conventions.md` catalogue is a case that passes `buf build` and fails
`capnp`, which is why the check is not redundant.

## One submodule

`modules/` holds `digitalbuildings` and nothing else. The `tools/`, `ibr/` and
`styles/` trees in that repository are not read: `ibr/` carries its own
`ibr.proto` for spatial data, which is a different model with a different
purpose, and vendoring it would put a schema this repository does not own
into a module it publishes. `buf.yaml` excludes `modules/` for exactly that
reason.
