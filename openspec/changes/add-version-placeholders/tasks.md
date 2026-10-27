## 1. Spike: what the lint and the markup check do with the placeholder today

- [x] 1.1 Add conformance cases under `internal/dialect/testdata/conformance/` (`placeholder-known`: the placeholder in prose, in a fenced `bash` block, in an `https:` link destination and in the `description` front matter; expected output empty) and run `go test ./internal/dialect/...`; write the observed result in `design.md` "Context" (replace "Not yet run")
- [x] 1.2 In `internal/mdsafe/mdsafe_test.go`: a bundle-mode markup check of an authored page holding the same four forms reports nothing; if it reports, record the message in `design.md` and stop for the owner
- [ ] 1.3 `task check` green, then commit `test(lint): pin that the dialect admits the version placeholder form`

## 2. The name list and the lint rule

- [x] 2.1 New package `internal/placeholder`: the name table (name, kind, docs project), `Scan(line)` returning each `{{opm:` occurrence with its verdict (known; unknown name; unknown field; malformed), `Names()` in list order; table-driven tests in `placeholder_test.go` for every name, `opm-controler`, `cli.tag`, `{{opm:cli.version` unclosed, `{{ opm:cli.version}}` (not a placeholder, no match), two placeholders in a line
- [x] 2.2 `internal/dialect/page.go`: `record` calls `placeholders(l)` on every line (before the front-matter and fence returns), emitting the messages of the `dialect-lint` delta; tests in `dialect_test.go`
- [x] 2.3 Conformance cases: `placeholder-unknown-name` (message lists the known names), `placeholder-unknown-field`, `placeholder-unclosed`, `placeholder-in-fence-and-link` (accepted), `placeholder-shown-shortcode` (accepted); each with its expected `<file>:<line>` output
- [x] 2.4 `docs/contracts.md` C11: add the rule and the name list; sync `openspec/specs/dialect-lint` and the new `version-placeholders` spec (names, mapping) from this change
- [ ] 2.5 `task check` green, then commit `feat(lint): refuse unknown version placeholders`

## 3. Extra versions from a release manifest

- [x] 3.1 `schema/manifest.cue`: `units` and `#Unit`; `schema/config.cue`: `units` in `#Bundle` (design.md shows both); `internal/bundle/manifest.go`: `Units map[string]string` with `json:"units,omitempty"` after `Pins`; schema tests in `schema/load_test.go` (valid, `v0.1.1` refused, unit name `x` refused)
- [x] 3.2 `internal/config/config.go`: carry `Units` with the `file` default; test the default and the `..` refusal
- [x] 3.3 New `internal/build/units.go`: read `file` from the source tree, require a string matching `reSemVer`, set `s.m.Units`; call it from the assembly beside `pins`; tests in `units_test.go`: value recorded from a temp tree; missing file, missing key and `v0.1.1` each exit 2 naming file and key; no `units` declared leaves `manifest.json` bytes equal to the golden file of `build_test.go`
- [x] 3.4 `docs/contracts.md` C3 (`units`) and C6 (`units` entry); sync `bundle-format` and `opm-docs-cli` specs from this change
- [ ] 3.5 `task check` green, then commit `feat(bundle): record unit versions read from a release manifest`

## 4. The fill and the leftover check in pull

- [x] 4.1 `internal/placeholder`: `Fill(text, values)` replacing each well-formed placeholder, returning the names it could not supply; test that text outside placeholders is byte-identical
- [x] 4.2 New `internal/pull/placeholders.go`: `fillVersion(sv, set)` called at the end of `stageVersion` after `checkOverlap`; builds the value table (bundle values by docs project, unit values from `m.Units`, duplicate unit refused, `edge` refused); rewrites listed pages of each docs bundle with the file mode kept; `checkLeftover` over every staged tree, called in `Run` after `stageVersions`
- [x] 4.3 Tests in `internal/pull/placeholders_test.go` on the in-process registry: fill per site version (`v1.0` and `v1.1` get different `cli` values); a placeholder in prose, a fenced block and a link all filled, rest of the file unchanged; the extra version `opm-controller-module` recorded from a manifest and filled; no bundle for the name exits 2 naming version, page, placeholder; an `edge` value exits 2; two bundles recording one unit exits 2; a leftover in a tab bundle and in a section bundle exits 2 naming the page and leaves the previous trees and lock; `--frozen` fill equals the first pull; `--local` docs tree filled
- [x] 4.4 `docs/contracts.md` C7 and C16: the fill, the leftover check, the order in `Run`, the note that the unpacked tree differs from the layer; sync `bundle-pull` and `version-placeholders` specs
- [ ] 4.5 `task check` green, then commit `feat(pull): fill version placeholders per site version and refuse leftovers`

## 5. Refresh units on a module release (needs the owner's answer to the proposal question)

- [x] 5.1 `cmd/opm-docs/revise.go` and `internal/revise/revise.go`: flag `--refresh-units`, exclusive with `--fix` (exit 1); skip the fix checks; take the newest signed revision's patches; build with units read from the `main` checkout (a `--units-from` internal option of `build`); refuse when the units equal the newest revision's; tests in `revise_test.go` (module release builds revision 1 with the new unit; unchanged units exit 2; both flags exit 1; unsigned newest revision refused)
- [x] 5.2 `.github/workflows/publish.yml`: input `refresh-units` (boolean, default false); the revision step runs `revise --refresh-units` when it is true and `fix` is empty; the guard step fails when both or neither are given; `actionlint` green
- [x] 5.3 `docs/contracts.md` C5 and "Docs revisions": the refresh path; `docs/orchestration.md`: the adoption order (site bumps first, then cli, opm, opm-controller); sync `opm-docs-cli` and `publish-workflow` specs
- [x] 5.4 `task check` green, then commit `feat(workflow): refresh unit versions of a release in revision mode`
