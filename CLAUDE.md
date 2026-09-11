# digital-buildings-protobuf — working conditions

Protobuf types for Google's Digital Buildings ontology and the building
configurations written against it, and the FlatBuffers and Cap'n Proto schemas
derived from them. Every rule below is a hard constraint, not a preference.
`.github/workflows/` enforces the ones that can be enforced mechanically, one
workflow per concern -- `lint`, `sync`, `schema`, `generate`, `breaking`,
`links` -- so a failure names what broke. The rest are on you.

The conventions are adapted from
[protobuf-rfc](https://github.com/the-protobuf-project/protobuf-rfc), whose
numbering this file keeps so the two can be read side by side, and follow
[coversa-protobuf](https://github.com/oh-tarnished/coversa-protobuf), which
applies the same numbering to COVESA's Vehicle Signal Specification. Where a
rule differs from either, it says so and why.

The generator is `sync/`, and it produces **1,165 `.proto` files across 113
packages** from the pinned ontology revision; `sync/cmd/docs` writes the
**113 `README.md` files** beside them. `just ci` is green: zero `buf lint`
findings and zero api-linter findings, with no config file and no suppressed
rule.

Several rules below carry a correction, because writing the generator
disproved a claim this file made before it existed. Those are marked where
they appear rather than quietly rewritten -- the wrong version is usually the
more tempting one.

## 1. No lint rule may be disabled

Not with an `except` list in `buf.yaml`, not with a `disabled_rules` entry in
an api-linter config, not with an inline `(-- api-linter: ... =disabled --)`
comment. There is no api-linter config file in this repo and there must not be
one. If a linter objects, the code is wrong -- and fixing it means fixing
**the generator**, never the emitted file.

When two linters genuinely contradict each other, do not except a rule --
narrow one tool's scope so they stop overlapping. That is why `buf.yaml` uses
`BASIC` rather than `STANDARD`: buf STANDARD wants `FanCoilUnitsService` and
`GetFanCoilUnitResponse` wrappers, AIP-131 wants `FanCoilUnits` returning
`FanCoilUnit`, and no code satisfies both. buf checks structure, api-linter
checks API design, and every rule in each selected category is enforced.

## 2. Nothing under `protobuf/digitalbuildings/` is written by hand

Both the `.proto` files and the `README.md` beside them are generated -- the
schema by `sync`, the reference by `docs`, from the same model rather than by
parsing the emitted protos back in.

The scope is the **generated root only**. `protobuf/extensions/` is
hand-written and `sync` never touches it; rule 16 is the boundary, and
`sync/emit/tree_test.go` asserts it rather than trusting it.

Every `.proto` is emitted by `sync` from the one specification pinned in
`sync/spec.yaml`: the YAML under `ontology/yaml/resources/` in Google's
digitalbuildings repository. Every file carries a `DO NOT EDIT` banner naming
that revision. Editing one is not a small shortcut, it is a change that the
next `just sync` silently reverts.

A fix belongs in the generator, and usually in a named table there:

| Change | Where |
| --- | --- |
| a field name a linter rejects | `FieldRenames` in `sync/catalog` |
| a field name a linter flags but that is right as it stands | `AcceptedTraps`, same file |
| a message name a linter rejects | `TypeRenames`, same file |
| what `AHU` is called in English | `generalTypeNames` in `sync/catalog` |
| a field better typed as a Timestamp or Duration | `TimeFields`, same file |
| a two-state multistate better modelled as a `bool` | `BooleanStates`, same file |
| an abbreviation the expander gets wrong | `abbreviations` in `sync/naming` |
| a plural English gets wrong | `irregularPlurals` in `sync/naming` |
| which general type an entity type belongs to | `generalTypes` in `sync/model` |
| a value type shared across packages | `sharedValueTypes` in `sync/model` |
| which ontology revision to read | `sync/spec.yaml`, then `just sync` |

Entity-type entries are keyed by **GUID**, not by name -- the ontology gives
every entity type one, it is stable across renames, and a name-keyed entry is
the one that stops applying silently. Field, subfield, state and unit entries
have no GUID upstream and are keyed by the literal. Either way, an entry
naming something the ontology no longer has is a build error: see
`model.CheckCatalogue`.

**One message or one enum per file.** `messages.proto` is the single
exception, because a request and its response are meaningless apart. This is
stricter than it first was: multistate enums used to share a `states.proto`
per package, and 16 of those ran past 200 lines while holding as many as 19
unrelated enums. Each now has its own file, named for the enum
(`closed_open_value.proto`), which is why the tree is 1,165 files rather than
830.

@docs/generator.md

## 3. AIP is the convention

api-linter must report zero violations across every file. In practice:

- `option java_package`, `java_outer_classname`, `java_multiple_files` on
  every file -- api-linter mandates all three (AIP-191)
- **no `go_package`**: the Go import path depends on where a consumer
  generates, so consumers set it through managed mode. The asymmetry with Java
  is not a preference -- api-linter requires one and not the other, and rule 1
  forbids excepting it
- a comment on every message, field, enum, enum value, service and RPC. The
  ontology documents every subfield, state, unit and entity type, and a field
  comment is composed from its subfields' own definitions; see
  `describe.Summary`
- `(google.api.field_behavior)` on every field
- `(google.api.resource)` on every resource, `(google.api.resource_reference)`
  on every field naming one
- `(google.api.http)` on every RPC, `(google.api.method_signature)` on the
  standard methods
- `int32`, never `uint32` (AIP-141). A `fixed_min: 0.0` is restored as a
  `buf.validate` range, not as an unsigned type
- messages before top-level enums; services before messages
- a `uid` carries `(google.api.field_info).format = UUID4` (AIP-148), and for
  an entity type that uid **is the ontology's own GUID**

Digital Buildings names cost far less here than VSS names did: they are
already `lower_snake_case`, and no literal is a whole-name reserved word. The
catalogue is 13 entries -- four `_timestamp` renames, eight `over_voltage`
fusions, one entity type -- against coversa-protobuf's 27. `docs/conventions.md`
rule 11 has them, and records three api-linter rules this repo first read
wrongly: two too broadly, and one (`core::0216::state-field-output-only`) not
broadly enough, which cost 394 findings.

## 4. No cross-package message references

AIP-215 forbids a field from referencing a message in another proto package
and exempts only `google.*`. A package must therefore contain everything it
references, which is what makes the namespace split in rule 6 possible rather
than merely tidy. Two consequences, and neither is a workaround:

- A **connection** is not a field. A floor does not hold its fan coil units;
  `Connection` carries a `ConnectionType` and the resource name of the other
  entity, with `(google.api.resource_reference).type = "*"` because the other
  end may be any resource in any namespace.
- A **link** is the same shape: it names the source entity by resource name
  and maps this entity's standard fields onto that entity's, which is what the
  building config's `links` block says.

`google.*` is the one exemption and this repository **does not take it** --
the single place it differs from protobuf-rfc's rule 4, inherited from
coversa-protobuf for the same reason: `protoc-gen-buffers` renders
`google.type.*` messages into neither target, silently. A value type used by
more than one package -- `Translation`, `FieldTranslation`, `UnitMapping`,
`StateMapping`, `Link`, `Connection`, `ValueRange` -- is **copied into each**,
generated rather than maintained. See `docs/decisions.md`.

## 5. This repository defines exactly one annotation vocabulary

It **consumes** several. The distinction matters and an earlier draft of this
rule collapsed it: "there is exactly one" was true of what we define and read
as a ban on importing anyone else's. We depend on googleapis and protovalidate
already, and `entity.v1` and `telemetry.v1` are the same kind of dependency --
vocabularies someone else maintains, versioned on the BSR, doing a job this
repository should not reimplement.

Extension field numbers are allocated across the-protobuf-project, and the
allocation is the reason nothing collides when several are applied to one file:

| Range | Vocabulary | Job |
| --- | --- | --- |
| 50001-50003 | `protobuf.digitalbuildings.annotations.v1` | ours, below |
| 50100-50102 | `telemetry.v1` | OpenTelemetry instrumentation |
| 51000-51002 | `entity.v1` (`store`) | table and column overrides |
| 52000-52003 | `cache.v1` | read-through cache decorators |
| 53000-53007 | `buffers.v1` | FlatBuffers and Cap'n Proto |

A fourth extension of ours takes **50004**, not the next round number. Check
this table before minting one; a collision is silent until two vocabularies
meet in a descriptor set.

protobuf-rfc bans custom annotations outright. This repository has one vocabulary,
`protobuf.digitalbuildings.annotations.v1`, and the argument is stronger here
than the unit argument was in coversa-protobuf. The ontology is explicit that
a field's identity is **the set of its subfields, not its name**:
"Applications should depend on the field set, not the string value." A proto
field called `zone_air_temperature_sensor` has thrown that set away -- a
consumer cannot test it for equivalence against `air_zone_temperature_sensor`
without re-parsing the name against a subfield table it does not have. So the
annotation carries what the name only renders. Three extensions:

- `(annotations.field)` on `FieldOptions` -- ordered subfields, point type,
  measurement, standard unit, flexible range, and whether the field was
  enumerated (rule 12 of `docs/conventions.md`)
- `(annotations.entity_type)` on `MessageOptions` -- GUID, namespace, general
  type
- `(annotations.canonical_type)` on `EnumValueOptions` -- GUID, the
  namespace-qualified name, and the exact `implements` / `uses` / `opt_uses`
  sets. This is what makes rule 7 lossless.

That is the whole licence: **provenance still goes in comments**, and no
second vocabulary may be added without an argument of the same kind.

## 6. Layout

```text
protobuf/digitalbuildings/<namespace>/<resource>/v1/<file>.proto
    package protobuf.digitalbuildings.<namespace>.<resource>.v1
```

`<namespace>` is one of the ontology's own thirteen, lowercased -- `carson`,
`electrical`, `facilities`, `gateways`, `hvac`, `info_tech`, `lighting`,
`meters`, `physical_security`, `plumbing`, `safety`, `transport`, `untyped` --
plus `global`, for the five general types the ontology declares in
`entity_types/global.yaml` and shares across namespaces (`PMP`, `SENSOR`,
`TK`, `VLV`, `USER_INTERFACE`). Those resolve to one resource each rather than
one per borrowing namespace, which is what the ontology means by declaring
them globally. The ontology allows exactly one level below global and forbids
hierarchical namespacing, so this segment is flat by construction and needs no
`domains` table of the kind coversa-protobuf has.

`<resource>` is the equipment class **spelled out in English**, snake_cased
from the resource's own message name: `hvac/fan_coil_unit`,
`electrical/automatic_transfer_switch`, `meters/electrical_meter`. It is
**not** the ontology's 1-4 character tag, and an earlier draft of this rule was
wrong to use one. `electrical/ats` is unreadable to anyone who does not
already know the ontology, and a directory name is the first thing a consumer
sees -- before any comment, annotation or `README`. Nothing is lost by
spelling it: the tag is still on the message, in
`(annotations.entity_type).name`, which is where a program looks for it.

The segment is derived from the message name rather than from the tag, so a
`TypeRenames` entry moves the directory with it -- `REQUEST_TO_EXIT` is
`ExitRequestSensor`, and its package is `physical_security/exit_request_sensor`.
That also makes the segment and the resource's own `.proto` file the same
string by construction, which is what makes the tree navigable by guessing.

**The segment stays flat, however long the name.** The longest is
`uninterruptible_power_supply` at 28 characters and the median is 15, which is
a directory name, not a problem. Splitting a long one into
`electrical/power_supply/uninterruptible` would need a head-and-modifier table
no rule derives -- one more hand-kept list to go stale, which is exactly what
this rule rejects a `domains` table for. A grouping the ontology does not
assert is a grouping this repository would have to defend.

The namespaces declaring no `GENERALTYPES.yaml` (`carson`, `facilities`,
`info_tech`, `physical_security`, `untyped`) name their types in English
already, so this is no change for them: `facilities/building`,
`facilities/floor`.

There is **no `system/` family**, and an earlier draft of this rule was wrong
to propose one. It said non-`EQUIPMENT` general types belong under `system/`,
on the reasoning that the ontology models chilled and heating water systems
that way. Running the generator disproved the discriminator: "does not
implement EQUIPMENT" selects `CHWS`, `HWS`, `CDWS` and `GTWS` correctly, but
also `LGRP` (a luminaire group), `WEATHER` and `FACILITIES/DOOR` -- while
*missing* `CHGS` and `DWST`, which are systems by name and description and do
implement `EQUIPMENT`. The ontology is not consistent on that axis, so no
derivable rule selects "system" and a hand-kept list would be one more table
to go stale. Systems live under their own namespace:
`hvac/chilled_water_system`.

Two packages sit outside the namespace grid: `digitalbuildings/annotations/v1`,
the vocabulary from rule 5, and `digitalbuildings/ontology/<facet>/v1`, the
nomenclature itself as read-only resources (`subfields`, `fields`, `states`,
`units`, `connections`, `entity_types`, `namespaces`). Both are **named**
rather than sitting at `digitalbuildings/v1`: a bare version directory beside
the namespace directories reads as a mistake, and
`protobuf.digitalbuildings.v1` would be a proper prefix of
`protobuf.digitalbuildings.hvac.fan_coil_unit.v1` -- a resolution hazard, not
a cosmetic one.

The buf module is rooted at the **repository root**, so `protobuf/` is itself
the first package segment and `PACKAGE_DIRECTORY_MATCH` holds with no
redundant directory inside it. `build/` is excluded; it holds exported copies
of googleapis and protovalidate for the schema step.

## 7. A general type is the resource; a canonical type is a value on it

The ontology has 2,637 entity types: 844 abstract functional groups and 1,587
canonical types. A message per canonical type would be 1,587 messages and
1,587 CRUD services, which is not a service surface. The cut is one level up.

**A general type is the resource.** `FCU` becomes `FanCoilUnit` at
`buildings/{building}/fanCoilUnits/{fan_coil_unit}`, carrying the union of
every field any FCU variant uses -- 301 of them, all `optional`. There are
**112 resources**: 82 general types carry canonical variants, and the rest are
declared general types with none yet, which are resources anyway because a
building config can name them and because `FACILITIES/BUILDING` is the parent
of every resource name in the schema. Each gets the standard methods in its
package's `service.proto`.

`FACILITIES/BUILDING` is the one **root** resource, named `buildings/{building}`.
Everything else hangs beneath it, so Building cannot: that would make its own
pattern `buildings/{building}/buildings/{building}`, and AIP-127 would then
have no pattern to match the HTTP template against.

**A canonical type is an enum value on that resource.** `FCU_DFSS_CSP_CHWDC`
becomes `FAN_COIL_UNIT_TYPE_FCU_DFSS_CSP_CHWDC`, keeping the ontology's name
verbatim because that name *is* the identifier. The variant's exact field set
rides on the enum value in `(annotations.canonical_type)`, so a validator can
still reject an `AHU_...` reporting a field its declared type does not have.
What is given up is *compile-time* enforcement of that, deliberately: see
`docs/decisions.md`, which also records why abstract groups are not messages.

**There are no singletons.** Coversa-protobuf splits its resources into
AIP-156 singletons and collections because a vehicle has exactly one cabin. No
such statement exists here -- a building may hold any number of anything,
including one -- so every resource is a collection and gets `Get`, `List`,
`Create`, `Update`, `Delete` and AIP-164 `Undelete`.

**The ontology packages are the exception, and read-only is their correct
method set.** `Subfield`, `Field`, `State`, `Unit`, `ConnectionType`,
`EntityType` and `Namespace` get `Get` and `List` only. This is not reduced
CRUD: the vocabulary is pinned by rule 9 and stamped into every banner, and a
vocabulary a client can extend at runtime is not pinned.

**The building configuration file is not a message.** Its `INITIALIZE` and
`UPDATE` modes are `Create`, `Update` and `Delete` on the resources above; its
`translation`, `links` and `connections` blocks are fields on them. See
`docs/decisions.md`.

## 8. Where a field comes from decides whether it can be written

The ontology keeps two field files, and the split is the answer.
`fields/metadata_fields.yaml` holds 25 literals, labels and design capacities:
configuration, so `OPTIONAL` and `IMMUTABLE` -- set at create, never reported.
`fields/telemetry_fields.yaml` holds 1,540, and there the **point type**
decides: `command` and `setpoint` are `OPTIONAL` and writable, and everything
else -- `sensor`, `alarm`, `status`, `mode`, `accumulator`, `counter`,
`count`, `timestamp`, `label`, `capacity`, `requirement`, `specification` --
is `OUTPUT_ONLY`, because the equipment reports it and no API call sets it.

`mode` is read-only despite reading like a control: the ontology defines it as
an observed "distinct mode of operation," and the thing that *sets* a mode is
a `command`. An `update_mask` naming an `OUTPUT_ONLY` field is rejected rather
than silently ignored.

## 9. The ontology revision is pinned and dated, and field numbers outlive it

`sync/spec.yaml` names the upstream commit this schema was generated from and
that commit's date, `YYYY-MM-DD`, the way MCP versions its specification. Both
are stamped into every generated banner, so "which ontology is this from?" is
answerable from any single `.proto` a consumer holds. The checkout is a
submodule under `modules/`, which holds every upstream this repository reads
and nothing it writes.

Bumping a pin is a deliberate act, not a side effect of pulling: move the
checkout, edit the pin, run `just sync`, and record what moved in
`docs/spec.md`. `just spec` checks the pin against its working tree -- commit,
date, and a clean tree -- because a stale pin puts a revision into every
banner that the files were never generated from, and nothing else can see that
is false.

**Field numbers come from `sync/ordinals.yaml`, never from position.** An
addition to coversa-protobuf's rule 9, and the reason is the shape of this
source: the ontology adds fields to existing types continuously, and they land
alphabetically in the middle of a `uses` list. Numbering by position would
renumber half of an 813-field `AirHandlingUnit` on a routine bump -- a wire
break that no linter, and no `buf breaking` run against a *regenerated* tree,
would catch. The ledger is append-only, committed, and `-merge` in
`.gitattributes` keeps git from resolving a conflict in it by guessing.
Numbers 1-15 are reserved for `name`, `uid`, `code`, `entity_type` and the
connection list, the fields on every read path.

@docs/spec.md

## 16. Two roots, and only one of them is generated

`protobuf/` holds two roots with different owners, different lifecycles and
the same standards.

| Root | Written by | Cleared by `sync` | Versioned with |
| --- | --- | --- | --- |
| `protobuf/digitalbuildings/` | `sync` and `docs` | yes, every run | the ontology pin |
| `protobuf/extensions/` | us | **never** | this repository |

The generated root is the ontology rendered as protobuf and nothing else. It
is wiped and rewritten on every `just sync`, so anything hand-written there is
deleted by the next person who regenerates -- silently, because a missing
package is an absence rather than a lint error. `sync/emit/tree.go` scopes the
clear to `digitalbuildings`, and `sync/emit/tree_test.go` fails if that scope
ever widens. That test is the rule; this paragraph is only its explanation.

The extensions root is for what Digital Buildings does not model: the
telemetry data plane, the asset register, protocol bindings, maintenance, and
the point lists for the namespaces the ontology leaves as stubs.

**Hand-write a fixed schema; generate an ontology-shaped one.** `PointSet` is
one message designed once, and a generator emitting fixed proto text would be
worse than the proto. Parking and security points are hundreds of entity types
from YAML, and there the generator earns everything it earns in the first root
-- ordinal stability, composed comments, lint compliance. The line is the
shape of the source, not the root.

**Reference across roots by resource name, never by import.** `Connection`
and `Link` already carry `(google.api.resource_reference).type = "*"`, so an
extension can name a `FanCoilUnit` without importing its package. This is not
a workaround for AIP-215: a type string is not a message reference, so no
cross-package dependency exists to forbid. An extension that `import`s a
generated package has made the ontology pin part of its own release cadence,
which is the coupling this split exists to prevent.

**Being hand-written buys no exemption.** `just aip` globs all of
`protobuf/`, so extensions are linted exactly as the generated root is.
Rule 1 applies in full: if api-linter objects to a hand-written file, the
file is wrong.

## Rules 10-15

Identifiers, the AIP naming traps and the rename catalogue, field enumeration,
bounds, units, states and the two annotation systems are in
`docs/conventions.md`, imported below and carrying the same weight; the split
is length.

@docs/conventions.md

## Where to look things up

`docs/references.md` is a checked link index: the AIPs and their linter rule
pages, buf and protovalidate docs, the Digital Buildings documentation this
schema models, and the two projects it is built on. **Follow the link rather
than working from memory** -- AIP rule semantics have been guessed wrong here
more than once, and `docs/conventions.md` rule 11 names two of them.

@docs/references.md

## Before you finish

Run what CI runs:

```sh
just ci
```

which runs, in order: the spec pin check, `buf format --diff --exit-code`,
`buf lint`, `buf build`, `api-linter`, a regenerate-and-diff,
the two schema targets, their compilers, the ordinal ledger and the `schema/`
diff. Each is also a workflow under `.github/workflows/` -- read the steps
there rather than trusting this list to stay current.

`scripts/compile-schema.sh` is not redundant with `buf build`; why, and the VS
Code linter extension's disagreement with the CLI, are in `docs/decisions.md`
and `docs/references.md`.
