# Specification revision

Rule 9: the ontology revision is pinned, dated, and stamped into every
generated banner. This file is the pin's history and the procedure for moving
it.

## One source, not two

Coversa-protobuf pins two specifications because VSS and VDM are separate
projects on separate release cadences. Digital Buildings is one repository, so
there is one pin -- but it covers **seven facets** that move independently
inside it, and a bump that touches only one of them is the normal case:

| Facet | Path under `ontology/yaml/resources/` |
| --- | --- |
| subfields | `subfields/subfields.yaml` |
| fields | `fields/telemetry_fields.yaml`, `fields/metadata_fields.yaml` |
| states | `states/states.yaml` |
| units | `units/units.yaml` |
| connections | `connections/connections.yaml` |
| entity types | `<NAMESPACE>/entity_types/*.yaml`, `entity_types/*.yaml` |
| namespaces | the directory names themselves |

`just survey` reports counts per facet, which is how a bump's blast radius is
read before anything is regenerated.

The checkout is a submodule at `modules/digitalbuildings`. `modules/` holds
every upstream this repository reads and nothing it writes.

## Current

```yaml
# sync/spec.yaml
ontology:
  repository: https://github.com/google/digitalbuildings
  commit: 4a794acf6f01faa88a61f3740bf8d10dacee2483
  date: 2026-09-09
  path: ontology/yaml/resources
```

What that revision contains, as parsed:

| Facet | Count |
| --- | --- |
| namespaces | 13 |
| subfields | 391 in 7 categories |
| field literals | 1,565 (1,540 telemetry, 25 metadata) |
| states | 67 |
| unit families | 64 (plus 5 aliases), 191 unit names |
| connection types | 9 |
| entity types | 2,637 (844 abstract, 1,587 canonical) |
| general types carrying variants | 82, across 8 namespaces |

Derived, and what the schema is actually shaped by:

| Property | Value |
| --- | --- |
| resources | 112 (82 general types with variants, 30 without) |
| largest resource | `AirHandlingUnit`, 450 fields over 410 variants |
| median resource | 12 fields |
| emitted | 1,165 `.proto` files across 113 packages, plus 113 `README.md` |
| distinct multistate enums | 40, covering 602 multistate literals |
| numeric literals | 953 (931 telemetry, 22 metadata) |

## History

Nothing yet. The first entry goes here when the first bump lands, and each
entry records what moved per facet -- not just the commit range. A bump that
added 40 HVAC canonical types and a bump that renamed a subfield have very
different consequences and the log should say which happened.

## Bumping a pin

1. Move the submodule: `git -C modules/digitalbuildings fetch && git -C
   modules/digitalbuildings checkout <commit>`.
2. Edit `commit` and `date` in `sync/spec.yaml`. The date is the **commit's**
   date, `YYYY-MM-DD`, not today's.
3. `just spec` -- checks the pin against the working tree: commit, date, and
   that the tree is clean. A stale pin puts a revision into every banner that
   the files were never generated from, and nothing else can see that is
   false.
4. `just survey` -- read it before regenerating. This is where a removed field
   or a renamed entity type shows up as a number that moved.
5. `just sync`.
6. `git diff --stat protobuf/` -- and read it. See below for what to look for.
7. Record what moved in **History** above.
8. `just ci`.

## What a bump can break

**A catalogue entry that no longer applies** is a build error by design
(`model.CheckCatalogue`), so this one announces itself at step 5. Entity-type
entries are keyed by GUID and survive an upstream rename; field, state and
unit entries are keyed by the literal and do not.

**A field number that moved** is the failure this repository most needs to
catch, and `buf breaking` run against a regenerated tree cannot see it -- both
sides regenerate the same way. `sync/ordinals.yaml` is what catches it, and
`just verify-schema` fails if a slot moved. Rule 9 has the reasoning: the
ontology inserts new fields alphabetically into existing `uses` lists, so
roughly half of an 813-field message shifts position on a routine bump.

**A general type gaining its first canonical variant** adds a resource, a
package, a service and a directory. That is not a break, but it is a much
larger diff than the ontology commit suggests, and it is worth confirming the
general type is one this schema should expose rather than an artefact of an
upstream file being split.

**A general type losing its last canonical variant** removes a package. Do not
let `just sync` delete it silently: an empty package is still a published
import path. Deprecate rather than remove, and say so in History.

**A new state in an existing multistate set** changes a shared enum, which by
rule 13 is shared across every field using that set -- so a one-line ontology
change can touch hundreds of fields. Adding an enum value is wire-safe;
confirm the diff is only additions.

**A new subfield category or point type** is the one that needs thought rather
than procedure. The category order *is* the field grammar (`sync/ontology`)
and a new point type has to be classified writable or read-only under rule 8
before anything will build.
