# digital-buildings-protobuf dev tasks — run `just` to see recipes.
#
# Common flows:
#   just sync    # regenerate protobuf/ from the pinned ontology revision
#   just survey  # report what was parsed, before trusting a spec bump
#   just lint    # what CI checks: format, buf lint, build, api-linter, caps
#   just ci      # everything CI runs
#
# Requires: go, buf, api-linter.

# List recipes (default when you run bare `just`).
_default:
    @just --list

# Regenerate protobuf/ from the pinned ontology revision.
#
# buf format runs here rather than in the generator: reproducing buf's exact
# formatting in Go would mean reimplementing it, and the emitter would then
# drift from whatever buf does next. `just ci` checks the committed tree is
# already formatted, so the two cannot disagree silently.
[doc("Regenerate every .proto from the pinned ontology. Nothing under protobuf/ is hand-written.")]
sync:
    go run ./sync/cmd/sync
    buf format -w

# Regenerate the Markdown reference beside the schema.
#
# The second of the two targets built on model.Model. It never parses an
# emitted .proto back in -- a reference built by re-reading the output would
# agree with the output by construction, and so could not catch a generator
# that emitted the wrong thing.
[doc("Regenerate every README.md beside the protos, plus the index.")]
docs:
    go run ./sync/cmd/docs

# Report what the generator parsed: counts per facet, per namespace, per
# general type, and the state sets. Run this after a spec bump before
# anything else -- it is where a removed field shows up as a number that moved.
[doc("Survey the parsed ontology and the resource partition.")]
survey:
    @go run ./sync/cmd/sync -survey

# Check sync/spec.yaml against the checkout under modules/.
[doc("Verify the ontology pin matches the submodule.")]
spec:
    @go run ./sync/cmd/sync -survey >/dev/null

# Format Go sources in place.
fmt:
    gofmt -w sync

# Build, vet and test the generator.
[doc("Build and vet the generator, and run its tests.")]
test:
    go build ./sync/...
    go vet ./sync/...
    go test ./sync/...

# Everything the Lint job checks. Mutates nothing.
[doc("Format check, buf lint, buf build, api-linter, and the line caps.")]
lint: test aip
    @test -z "$(gofmt -l sync)" || { echo "unformatted Go (run: just fmt):"; gofmt -l sync; exit 1; }
    buf format --diff --exit-code
    buf lint
    buf build
    @just cap

# Run the Google API linter over every proto. No config file, no disabled rule.
#
# api-linter exits 0 even when it reports problems -- it is a reporter, and the
# CI action is what turns findings into a failure. A local recipe that printed
# a table of violations and still succeeded would be worse than no recipe, so
# the summary is captured and a non-empty table is made the failure it already
# is.
[doc("Run api-linter over protobuf/ (zero findings, no suppressions).")]
aip:
    #!/usr/bin/env sh
    set -eu
    d=$(mktemp -d); trap 'rm -rf "$d"' EXIT
    buf build -o "$d/desc.binpb" --as-file-descriptor-set
    api-linter --descriptor-set-in "$d/desc.binpb" \
        --output-format=summary $(find protobuf -name '*.proto') | tee "$d/out"
    if grep -q '| core::' "$d/out"; then
        echo "api-linter reported violations above. Rule 1: fix the generator," >&2
        echo "never the emitted file, and never except the rule." >&2
        exit 1
    fi

# The line caps.
#
# Two caps, and they are enforced differently because the files are different
# kinds of thing.
#
# Hand-written Go and Markdown are capped by line count alone. Exemptions and
# their reasoning are in docs/generator.md: README.md and sample-reference.md
# would be made worse by splitting, and CLAUDE.md already splits into docs/ via
# @-imports that are concatenated back into one context anyway.
#
# Generated protos are capped at 200 too, against the audited list in
# line-cap-exemptions.txt. The check runs both ways -- an unlisted file over
# the cap fails, and a listed file back under it fails -- because a list that
# only ever grows is a list nobody rereads. See that file's header.
[doc("Check the line caps: 200 for Go and protos, 250 for Markdown.")]
cap:
    #!/usr/bin/env sh
    set -eu
    over=0
    for f in $(find . -path ./gen -prune -o -path ./build -prune -o -path ./schema -prune \
        -o -path ./modules -prune -o -path ./.git -prune -o -path ./protobuf -prune \
        -o -name 'README.md' -prune -o -name 'sample-reference.md' -prune \
        -o -name 'CLAUDE.md' -prune \
        -o \( -name '*.md' -o -name '*.go' \) -print); do
        n=$(wc -l <"$f")
        case "$f" in
            *.go) cap=200 ;;
            *)    cap=250 ;;
        esac
        [ "$n" -gt "$cap" ] && { echo "$f: $n lines, over the $cap-line cap"; over=1; }
    done
    [ "$over" -eq 0 ] || { echo "Split the files above; do not compress them."; exit 1; }
    echo "All hand-written files within their line cap."

    d=$(mktemp -d); trap 'rm -rf "$d"' EXIT
    grep -vE '^[[:space:]]*(#|$)' line-cap-exemptions.txt | sort >"$d/listed"
    find protobuf -name '*.proto' -exec wc -l {} + \
        | grep -v ' total$' | awk '$1>200 {print $2}' | sort >"$d/over"
    comm -13 "$d/listed" "$d/over" >"$d/unlisted"
    comm -23 "$d/listed" "$d/over" >"$d/stale"
    if [ -s "$d/unlisted" ]; then
        echo "Over the 200-line cap and not exempt:" >&2
        sed 's/^/  /' "$d/unlisted" >&2
        echo "Split them in the generator. A file that truly cannot be split --" >&2
        echo "one message or one enum -- goes in line-cap-exemptions.txt." >&2
        over=1
    fi
    if [ -s "$d/stale" ]; then
        echo "Exempt but now under the cap -- delete these lines from" >&2
        echo "line-cap-exemptions.txt:" >&2
        sed 's/^/  /' "$d/stale" >&2
        over=1
    fi
    [ "$over" -eq 0 ] || exit 1
    echo "All $(wc -l <"$d/over" | tr -d ' ') protos over 200 lines are accounted for."

# Verify the committed tree matches what the generator produces.
[doc("Fail if protobuf/ is stale relative to the pinned ontology.")]
verify-sync: sync
    @git diff --exit-code -- protobuf/digitalbuildings/ \
        || { echo "protobuf/digitalbuildings/ is stale — run 'just sync' and commit"; exit 1; }

# Verify the committed reference matches what the docs generator produces.
[doc("Fail if the README files are stale relative to the pinned ontology.")]
verify-docs: docs
    @git diff --exit-code -- 'protobuf/digitalbuildings/**/README.md' docs/sample-reference.md \
        || { echo "the Markdown reference is stale — run 'just docs' and commit"; exit 1; }

# Fail if regenerating would move a field's target slot.
#
# This is the check `buf breaking` cannot do: run against a regenerated tree,
# both sides of a breaking comparison are internally consistent, so a slot that
# moved is invisible. The ledger is what sees it.
[doc("Check the ordinal ledger is unchanged by a fresh run.")]
verify-ordinals: sync
    @git diff --exit-code -- sync/ordinals.yaml \
        || { echo "sync/ordinals.yaml changed: a field slot moved or a field was"; \
             echo "added. Adding is fine — commit it. A *moved* slot is a wire break."; exit 1; }

# Check the licence header on every source file.
#
# Two lines, in this order, at the very top of every .proto and every .go file:
#
#     // Copyright <year> RegaliumOS™.
#     // SPDX-License-Identifier: Apache-2.0
#
# The generated protos take theirs from sync/emit/file.go and so cannot drift.
# The hand-written ones can, and did: point_value.proto kept a stale holder
# name across a rename because nothing looked. This is the thing that looks.
[doc("Check the SPDX and copyright header on every .proto and .go file.")]
headers:
    #!/usr/bin/env sh
    set -eu
    bad=0
    n=0
    for f in $(find protobuf sync \( -name '*.proto' -o -name '*.go' \) -print | sort); do
        n=$((n+1))
        l1=$(sed -n 1p "$f")
        l2=$(sed -n 2p "$f")
        case "$l1" in
            "// Copyright "*" RegaliumOS™.") ;;
            *) echo "$f:1: bad copyright line: $l1" >&2; bad=1; continue ;;
        esac
        [ "$l2" = "// SPDX-License-Identifier: Apache-2.0" ] || {
            echo "$f:2: bad SPDX line: $l2" >&2; bad=1; }
    done
    [ "$bad" -eq 0 ] || {
        echo "Fix the headers above. For a generated proto the fix is in" >&2
        echo "sync/emit/file.go, never in the emitted file." >&2
        exit 1; }
    echo "All $n source files carry the licence header."

# Everything CI runs.
[doc("Everything CI checks.")]
ci: lint headers verify-sync verify-docs verify-ordinals
