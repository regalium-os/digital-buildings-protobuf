# References

A checked link index. `just links` verifies every URL here resolves, and
`.github/workflows/links.yml` runs it.

**Follow the link rather than working from memory.** AIP rule semantics have
been guessed wrong here more than once, and the two most consequential guesses
in this repository -- that AIP-216 flags `_status` *fields* and that AIP-140's
reserved-word rule matches *within* a field name -- were both wrong, in the
direction of inventing renames the linter never asked for. Rule 11 of
`docs/conventions.md` records what the source actually says.

## The source model

- [google/digitalbuildings](https://github.com/google/digitalbuildings) --
  the repository this schema is generated from
- [Ontology overview](https://github.com/google/digitalbuildings/blob/master/ontology/docs/overview.md)
- [Ontology concepts](https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology.md)
  -- subfields, the field grammar, equivalence, namespace elevation,
  enumeration, multistates, entity types. The authority for rules 5, 12 and 13.
- [Abstract model](https://github.com/google/digitalbuildings/blob/master/ontology/docs/model.md)
  -- general types, abstract functional groups, canonical types. The authority
  for rule 7.
- [HVAC model detail](https://github.com/google/digitalbuildings/blob/master/ontology/docs/model_hvac.md)
- [Ontology configuration](https://github.com/google/digitalbuildings/blob/master/ontology/docs/ontology_config.md)
  -- the YAML file formats `sync/load` and `sync/ontology` read
- [Building configuration](https://github.com/google/digitalbuildings/blob/master/ontology/docs/building_config.md)
  -- translations, links, connections, INITIALIZE/UPDATE. The authority for
  rules 4, 10 and 12.
- [Connections](https://github.com/google/digitalbuildings/blob/master/ontology/docs/connections.md)
- [YAML resources](https://github.com/google/digitalbuildings/tree/master/ontology/yaml/resources)
  -- the pinned input tree itself
- [FAQ](https://github.com/google/digitalbuildings/blob/master/ontology/docs/faq.md)

## The projects this is built on

- [protobuf-rfc](https://github.com/the-protobuf-project/protobuf-rfc) -- the
  numbering rules 1-15 keep
- [coversa-protobuf](https://github.com/oh-tarnished/coversa-protobuf) -- the
  same numbering applied to COVESA's Vehicle Signal Specification. Where a
  rule here differs, `CLAUDE.md` says so and why.

## AIP

The index: [aip.dev](https://aip.dev/). Rules that decide something in this
repository:

| AIP | What it decides here |
| --- | --- |
| [121](https://aip.dev/121) | resource-oriented design; why a general type is a resource |
| [122](https://aip.dev/122) | resource names; the `_name` suffix ban |
| [123](https://aip.dev/123) | resource types and name patterns (rule 11) |
| [126](https://aip.dev/126) | enums; the canonical-type enum of rule 7 |
| [127](https://aip.dev/127) | HTTP/JSON transcoding, `google.api.http` |
| [131](https://aip.dev/131) | `Get`; the buf-STANDARD conflict in rule 1 |
| [132](https://aip.dev/132) | `List` |
| [133](https://aip.dev/133) | `Create` |
| [134](https://aip.dev/134) | `Update`, `update_mask`, and rule 8's rejection |
| [135](https://aip.dev/135) | `Delete` |
| [140](https://aip.dev/140) | field names; prepositions, reserved words, abbreviations |
| [141](https://aip.dev/141) | quantities; no `uint32` (rule 3) |
| [142](https://aip.dev/142) | time and duration; the four `_timestamp` renames |
| [143](https://aip.dev/143) | standardised codes |
| [148](https://aip.dev/148) | standard fields; `uid` as the ontology GUID (rule 10) |
| [156](https://aip.dev/156) | singletons -- and why rule 7 has none |
| [164](https://aip.dev/164) | `Undelete` |
| [191](https://aip.dev/191) | file layout and the three `java_*` options |
| [203](https://aip.dev/203) | `google.api.field_behavior` (rule 14) |
| [215](https://aip.dev/215) | no cross-package references (rule 4) |
| [216](https://aip.dev/216) | states; `*State` not `*Status` on enums (rule 11) |

The linter's own rule pages are the authority on what is actually *checked* --
the AIP states intent, the rule page states behaviour, and rule 11 exists
because the two are not the same document:

- [linter.aip.dev](https://linter.aip.dev/) -- rule index
- [core::0140::reserved-words](https://linter.aip.dev/140/reserved-words) --
  whole-name match
- [the reserved word list itself](https://github.com/googleapis/api-linter/blob/main/rules/aip0140/reserved_words.go)
- [core::0142::time-field-names](https://linter.aip.dev/142/time-field-names)
- [core::0148::field-behavior](https://linter.aip.dev/148/field-behavior)
- [core::0216::synonyms](https://linter.aip.dev/216/synonyms) -- enum names only
- [googleapis/api-linter](https://github.com/googleapis/api-linter)

## Tooling

- [buf](https://buf.build/docs/) -- lint categories, `buf breaking`, managed
  mode (rule 3's `go_package` argument)
- [buf lint rules](https://buf.build/docs/lint/rules/) -- what `BASIC` selects
- [protovalidate](https://buf.build/docs/protovalidate/) -- the `buf.validate`
  constraints of rules 13 and 14
- [protovalidate standard rules](https://buf.build/docs/protovalidate/schemas/standard-rules/)
- [FlatBuffers](https://flatbuffers.dev/) -- the first schema target
- [Cap'n Proto](https://capnproto.org/language.html) -- the second, and the
  source of the keyword collision in the catalogue
- [just](https://just.systems/man/en/) -- the recipe runner

## Protobuf itself

- [Language guide (proto3)](https://protobuf.dev/programming-guides/proto3/)
- [Style guide](https://protobuf.dev/programming-guides/style/)
- [Field presence](https://protobuf.dev/programming-guides/field_presence/) --
  why every field on a resource is `optional` (rule 7)
- [Updating a message](https://protobuf.dev/programming-guides/proto3/#updating)
  -- what makes a moved field number the break rule 9 exists to prevent

## A note on `scripts/compile-schema.sh`

It is not redundant with `buf build`, and the reason belongs with the
references rather than in `CLAUDE.md`: `buf build` proves the *protobuf* is
valid, and says nothing about whether `flatc` and `capnp` accept what
`protoc-gen-buffers` derived from it. Those are different grammars with
different keyword sets and different ordinal models, and the catalogue's
Cap'n Proto entry is a case that passes `buf build` and fails `capnp`.

The VS Code
[Google API Linter extension](https://github.com/machanirobotics/google-api-linter-vscode)
(configured by `workspace.protobuf.yaml`) also disagrees with the CLI in
places -- it resolves imports differently and reports findings the CLI does
not. **The CLI is authoritative**; rule 1 is about what `just aip` reports.

One cause of that disagreement was **scope**, and it is fixed rather than
tolerated. `just aip` globs `find protobuf` and `buf.yaml` excludes `build`
and `modules`, but the extension defaults to the whole workspace folder -- so
it linted `modules/digitalbuildings/ibr/ibr_sdk/proto/ibr.proto`, the one
`.proto` in the pinned checkout. That file is Google's Internal Building
Representation, a spatial model this repository neither reads nor generates
from; it declares no package and no `java_*` options because it was never
written to AIP, and it produced 31 findings whose only available fix would be
editing another project's source inside a submodule. `workspace.protobuf.yaml`
now carries the same two excludes `buf.yaml` has.

That is scope, not suppression, and rule 1 stands: no rule is disabled, there
is still no `.api-linter.yaml`, and the count over what this repository does
own is unchanged at zero across 1,170 files. Rule 1's own remedy when two
tools disagree is to narrow a tool's scope rather than except a rule. A
finding the extension reports on a file under `protobuf/` is still a real
finding -- check it against `just aip` before believing either.
