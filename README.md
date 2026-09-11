# digital-buildings-protobuf

Protobuf types for [Google's Digital Buildings ontology][db] and the building
configurations written against it.

The ontology ships as 2,637 entity types of hand-maintained YAML. This is that
ontology as a versioned, lint-clean gRPC surface: **1,165 `.proto` files across
113 packages**, generated from a pinned upstream revision and regenerated
rather than edited.

[db]: https://github.com/google/digitalbuildings

## What's in it

- **112 resources**, one per ontology *general type*, each with the six
  standard AIP methods. `FCU` becomes `FanCoilUnit` at
  `buildings/{building}/fanCoilUnits/{fan_coil_unit}`.
- **1,587 canonical types** as enum values on those resources, each carrying
  its exact field set in an annotation — so a validator can reject an
  `AHU_…` reporting a field its declared type does not have.
- **Zero lint findings** from both `buf lint` and Google's `api-linter`, with
  no config file and no suppressed rule.
- A generated `README.md` beside every package, from the same model as the
  schema.

## Using it

As a dependency, from the BSR:

```yaml
# buf.yaml
deps:
  - buf.build/regalium-os/digital-buildings
```

Locally — the ontology is a submodule, so clone recursively:

```sh
git clone --recurse-submodules https://github.com/oh-tarnished/digital-buildings-protobuf
just sync    # regenerate the schema from the pinned ontology
just docs    # regenerate the Markdown reference beside it
just ci      # everything CI checks
```

## Layout

```text
protobuf/digitalbuildings/<namespace>/<resource>/v1/<file>.proto
    package protobuf.digitalbuildings.<namespace>.<resource>.v1
```

`<namespace>` is one of the ontology's own thirteen plus `global`;
`<resource>` is the equipment class spelled out in English, not the 1–4
character tag — `hvac/fan_coil_unit`, not `hvac/fcu`. The tag is still on the
message, in `(annotations.entity_type).name`.

`protobuf/extensions/` is the other root: hand-written types for what the
ontology does not model, never touched by the generator.

## The pin

Generated from ontology revision [`4a794ac`][rev], dated 2026-09-09, recorded
in `sync/spec.yaml` and stamped into every generated banner — so "which
ontology is this from?" is answerable from any single `.proto` you hold.
Bumping it is a deliberate act; `docs/spec.md` has the procedure.

[rev]: https://github.com/google/digitalbuildings/tree/4a794acf6f01faa88a61f3740bf8d10dacee2483

## Reading further

| Where | What |
| --- | --- |
| [`CLAUDE.md`](CLAUDE.md) | the working rules, in full |
| [`docs/spec.md`](docs/spec.md) | the pinned revision, and how to move it |
| [`docs/generator.md`](docs/generator.md) | how `sync/` is laid out, stage by stage |
| [`docs/conventions.md`](docs/conventions.md) | naming, annotations, the AIP catalogue |
| [`docs/decisions.md`](docs/decisions.md) | what was considered and rejected |
| [`docs/references.md`](docs/references.md) | checked link index |

## Licence

Apache 2.0 — see [`LICENSE`](LICENSE).

Portions are derived from Google's Digital Buildings ontology, Copyright 2020
Google LLC, also under Apache 2.0; see [`NOTICE`](NOTICE) for attribution. This
project is not affiliated with, sponsored by, or endorsed by Google.
