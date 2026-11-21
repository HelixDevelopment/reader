# ATM-032 — Per-format fulfillment matrix, Go core (campaign 2026-11-20)

Deliverable doc: `docs/formats.md` (matrix + provenance + NOT IMPLEMENTED list).
This file records what was verified and how. Raw logs in this directory:
`verify.log` (vet/build/test/race/coverage), `fuzz-seeds.log` (seed corpus via
`-run ^Fuzz`), `fuzz.log` (live `-fuzz` 8s per target), `coverage.out`.

Toolchain: go.mod pins `go 1.27.3` (auto-downloaded toolchain; host `go`
binary is 1.26.8 and defers per GOTOOLCHAIN=auto).

## Tree state

- `go vet ./...` — clean. `go build ./...` — clean.
- `go test ./... -count=1` — 22/22 packages ok.
- `go test -race ./... -count=1` — 22/22 packages ok.
- Coverage (statement, whole core module): **87.29% before → 87.32% after**
  (1037/1188 → 1377/1577; 389 new statements, 340 of them covered = 87.4%
  coverage on new code; total stays ≥ the 87.3% baseline claim).

## Stages (kept strictly independent, constitution §7.1)

detect | parse | paginate | render | convert. **render = NOT IMPLEMENTED for
every format in the Go core** (HDM + page geometry only; registry guard
`TestNoDescriptorClaimsUnimplementedRenderOrDRM`). detect = extension/filename
only; magic-byte sniffing explicitly NOT IMPLEMENTED.

## Cells moved to PASS in this campaign (with proving tests)

| stage/format | new capability | proving tests |
|---|---|---|
| paginate (all HDM text formats) | `core/pkg/sync/paginate.go`: deterministic greedy page packing over HDM in `sync.Position` coordinates; `PageOf` lookup; `Furthest`-compatible | `TestPaginateExactPacking`, `TestPaginateSpansChapters`, `TestPaginateOversizedBlockGetsOwnPage`, `TestPaginateImageBlocksAnchorWithoutChars`, `TestPaginateUsesFlatTOCOrderIncludingChildren`, `TestPaginateFailClosed` (5 sub-cases), `TestPageOfBoundariesAndOrdering`, `FuzzPaginate` |
| convert TXT/FB2/MOBI→EPUB | real EPUB 3 container exporter (stored mimetype, container.xml, OPF, NCX, per-chapter XHTML, sniffed PNG/JPEG resources), byte-deterministic, fails closed instead of dropping content | `TestExportEPUBStructure`, `TestExportEPUBRoundTripExact` (blocks+meta+fingerprint exact), `TestExportEPUBDeterministicBytes`, `TestExportEPUBWithImagesRoundTripsResources` (stdlib-encoded real PNG/JPEG), `TestExportEPUBFailClosed` (12 sub-cases), `TestExportEPUBTitleInjection`, `FuzzExportEPUB` |
| convert CBZ (page packaging) | HDM→CBZ exporter, refuses unknown image encodings | `TestEPUBToCBZRepack`, `TestExportCBZFailClosed` |
| convert chains | txt→fb2→epub→txt and mobi→epub full chains with exact expected text | `TestTxtToEPUBFullChain`, `TestMobiToEPUBThenPaginate`, `TestFB2BinariesToEPUBImages`, `TestTxtPaginateExact` |
| parse CBZ page enumeration | `cbz.Pages`: directory-only page list (no inflation) | `TestPagesEnumeration`, `TestPagesIDsAgreeWithParse`, `TestPagesFailsClosed`, `TestPagesDoesNotInflate` (corrupted central-dir CRC still enumerable → proof of no inflate), `FuzzPages` |
| parse EPUB spine extraction | `epub.Spine`: ordered spine pages, sizes, dup/ghost detection, MaxPages cap | `TestSpineEnumeration`, `TestSpineDetectsRepeatsAndMissingManifestItems`, `TestSpineCompressedBombIsBounded`, `TestSpineRejectsExcessSpineItems`, `TestSpineFailsClosed` (9 sub-cases), `FuzzSpine` |
| parse FB2 images | `<binary>` base64 → HDM resources with per-entry/aggregate limits; `<image href="#id">` → BlockImage; malformed base64 and dup ids rejected | `TestFB2BinariesBecomeResources`, `TestFB2DanglingImageHrefSkipped`, `TestFB2BinariesFailClosed`, `TestFB2RejectsOversizedBinary`, `TestFB2BinaryAggregateBoundIsArithmeticallySound` |
| parse MOBI pagination | PalmDOC text paginated with sync positions; `</p>`/`<br>` now become paragraph breaks (comment promised it, code didn't do it — real bug fixed) | `TestPalmDOCPagination`, `TestMobiToEPUBThenPaginate` |
| parse EPUB separators | `<hr>` → `BlockSeparator` (round-trip completeness with exporter) | asserted inside `TestExportEPUBRoundTripExact` |

## Fuzz (10 targets; seeds in `go test` always, live 8s each)

All PASS (fuzz.log). One **real pre-existing crash found and fixed**:
`<FictionBook><body><section><section></FictionBook>` (unclosed nested
section) → nil chapter → panic in `hdm.Validate` walker. Fixed in
`fb2.sectionToChapter` (skip empty children), hardened `core/pkg/hdm`
(`Validate` explicit error on nil chapter; walkers nil-safe). Regression
guards: corpus seed `core/pkg/parse/fb2/testdata/fuzz/FuzzParse/4a3dc8cee8c1c385`,
`TestFB2UnclosedNestedSectionsNoPanic`, `TestValidateRejectsNilChapter`.

## Bluffs removed (honesty corrections, not capability losses)

1. `formats.Convertible(f, f)` returned Direct for **all** formats incl.
   PDF/AZW3/DOCX ("re-pack" without parser or writer). Now Direct only for
   EPUB/FB2/TXT/CBZ (parser+exporter pair exists); MOBI→MOBI and
   XHTML→XHTML explicitly Blocked. Guard: `TestSameFormatRepackIsHonest`.
2. Matrix rows were claims without engines for any EPUB *target*; the
   exporter now exists and every Direct row is executable in a test.
   Structural guard against future drift: `TestMatrixNeverOutrunsTheCode`.
3. `parsedCaps.ContentAI = true` (any parsed book "supports AI") removed;
   no AI engine exists in core. Guard: `TestNoDescriptorClaimsUnimplementedAI`.
4. CBZ gained a truthful `Convert` capability (exporter + enumeration exist).

## Explicit NOT IMPLEMENTED (unchanged; requirements stated in docs/formats.md)

PDF page rendering & parsing; DRM/LCP fulfillment; font subsetting; image
decoding/layout/rendering (all formats); KF8/AZW3 parsing, Huffman
compression, MOBI writing; EPUB fixed-layout pagination; magic-byte content
detection; GIF/WebP/AVIF conversion targets (exporter refuses them rather
than mislabeling); RAR/7z comic containers.

## Synthetic-fixture caveat (per §7.1)

Old parse tests use synthetic image bytes (`"\x89PNG-fake"`); they prove
container plumbing only. All *payload-sensitive* claims (EPUB/CBZ export,
sniffing, repack) are backed by stdlib-encoded real PNG/JPEG fixtures
(`fixturePNG`/`fixtureJPEG` in `core/pkg/convert/export_test.go`, decoded
and verified in-test). No cell in `docs/formats.md` claims rendering for
anything.
