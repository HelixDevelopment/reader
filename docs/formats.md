# Format fulfillment matrix (Go core)

<!-- ATM-032. Governing rule: constitution §7.1 NO BLUFF. A cell says PASS
only for the stage named; detection is never rendering, parsing is never
conversion, storage of image bytes is never layout. Provenance cites tests
run in this tree (go1.27.3 toolchain per go.mod). Stages are independent:
PARSE PASS + RENDER NOT IMPLEMENTED is the normal state for text formats. -->

Stages, strictly independent:

| stage | meaning in this document |
|---|---|
| detect | filename/extension classification via `core/formats` registry. **Content/magic-byte sniffing is NOT IMPLEMENTED anywhere in the core**; an extension match never grants any other stage. |
| parse | bytes → HDM (`core/pkg/hdm`) through `core/pkg/parse/*`, with parseutil bounded-input guards, proven by tests |
| paginate | deterministic page packing (`core/pkg/sync` `Paginate`/`PageOf`) of a parsed HDM document, positions compatible with `sync.Position`/`Furthest` |
| render | put glyphs/pixels on screen. **NOT IMPLEMENTED for every format in the Go core.** The core produces HDM and page geometry only; the client renderer consumes them. `formats` registry keeps `Capabilities.Render=false`, guarded by `TestNoDescriptorClaimsUnimplementedRenderOrDRM` (core/formats/registry_test.go). |
| convert | HDM-pivot export (`core/pkg/convert`) to txt/xhtml/fb2/epub/cbz, proven by parse→export→re-parse chain tests |

## Primary formats

| format | detect | parse | paginate | render | convert | provenance (exact test names) |
|---|---|---|---|---|---|---|
| EPUB | PASS (ext only) | PASS | PASS (reflowable text; fixed-layout NOT IMPLEMENTED) | **NOT IMPLEMENTED** | PASS → txt/xhtml/fb2/epub/cbz | detect: `TestMandatedThoriumAndLCPExtensions`, `FuzzDetect`. parse: `TestParseEPUB`, `TestParsePercentEncodedManifestHrefs`, `TestXHTMLRetainsListTableAndDivContent`, limits: `TestParseRejectsOversizedRequiredEntries`, `TestParseCorruptZip`, `TestParseMissingRootfile`, `FuzzParse`. spine page extraction: `TestSpineEnumeration`, `TestSpineDetectsRepeatsAndMissingManifestItems`, `TestSpineCompressedBombIsBounded`, `TestSpineRejectsExcessSpineItems`, `TestSpineFailsClosed`, `FuzzSpine` (core/pkg/parse/epub/spine.go). paginate: engine `TestPaginateExactPacking`/`TestPaginateSpansChapters` + parsed-doc proof `TestTxtPaginateExact` (chain tests paginate *parsed* docs; EPUB parsed docs exercise the same engine — synthetic-HDM engine tests + txt/mobi parsed-doc chains, not an EPUB-specific page-geometry test). convert: `TestExportEPUBStructure`, `TestExportEPUBRoundTripExact` (blocks/metadata/fingerprint byte-exact), `TestExportEPUBDeterministicBytes`, `TestExportEPUBWithImagesRoundTripsResources` (real stdlib-encoded PNG/JPEG), `TestEPUBToCBZRepack`; fail-closed: `TestExportEPUBFailClosed`. Note: synthetic-bytes fixtures exist (e.g. `cover.png` = `"\x89PNG-fake"` in `buildTestEPUB`); image *payload* claims are backed only by the real-encoded fixtures in core/pkg/convert/export_test.go. EPUB inline `<img>` positions are dropped on re-parse (resources survive) — see Limitations. |
| MOBI (old PalmDOC/PRC, uncompressed + PalmDOC-LZ77) | PASS (ext only) | PASS | PASS | **NOT IMPLEMENTED** | PASS → txt/xhtml/fb2/epub (text only; MOBI writing NOT IMPLEMENTED) | parse: `TestParseMOBINoCompression`, `TestParseMOBIPalmDOCCompression`, `TestPalmDOCTokenClasses`, paragraph-break fix `TestPalmDOCPagination` (</p>/<br> block breaks). limits: `TestBadRecordCount`, `TestTruncated`, `TestPalmDOCRejectsMalformedAndOverflow` (compressed-bomb: expansion > limit fails closed), `TestMOBIRejectsAggregateTextOverflow`, `FuzzParse`. paginate: `TestPalmDOCPagination` (compressed MOBI → exact pages + `sync.Furthest`/`PageOf` interop), `TestMobiToEPUBThenPaginate`. convert: `TestMobiToEPUBThenPaginate` (mobi→epub→parse, exact block texts), matrix row claim `TestDocumentedMatrix` {MOBI,EPUB,Direct}. **KF8/AZW3 parse: NOT IMPLEMENTED** (detected, stream caps only, server-side conversion per decision C-2; no AZW3 writer: `TestUnsupported`). |
| FB2 | PASS (ext only) | PASS incl. `<binary>` payloads + image refs | PASS | **NOT IMPLEMENTED** | PASS → epub/txt/xhtml (FB2 *export* is a documented text-only projection: `<binary>` images are dropped by `toFB2`; FB2-target-with-images NOT IMPLEMENTED) | parse: `TestParseFB2`, `TestFB2BinariesBecomeResources` (base64 binaries → HDM resources, inline/section `<image href="#id">` → BlockImage), `TestFB2DanglingImageHrefSkipped`, `TestFB2RejectsInvalidISBNMetadata`. limits: `TestFB2RejectsExcessiveNesting` (MaxXMLDepth), `TestFB2RejectsOversizedInput`, `TestFB2RejectsOversizedBinary` (base64 bomb > MaxImageBytes fails closed), `TestFB2BinariesFailClosed` (malformed base64, empty/dup ids), `TestFB2BinaryAggregateBoundIsArithmeticallySound`, `FuzzParse` + crash seed `testdata/fuzz/FuzzParse/4a3dc8cee8c1c385` (unclosed nested section; fixed in fb2.go + core/pkg/hdm walkers, regression: `TestFB2UnclosedNestedSectionsNoPanic`, `TestValidateRejectsNilChapter`). convert: `TestFB2BinariesToEPUBImages` (FB2 binary → EPUB image entry byte-identical), `TestTxtToEPUBFullChain` (txt→fb2→epub→txt exact). |
| TXT/Markdown-ish | PASS (ext only) | PASS | PASS | **NOT IMPLEMENTED** | PASS → fb2/xhtml/epub | parse: `TestChaptersAndHeadings`, `TestAllCapsHeading`, `TestNoHeadingsSingleChapter`, `TestEmpty`, limits: `TestTXTRejectsOversizedInput`, `TestLongUnwrappedLineWithinInputBudget`, `FuzzParse`. paginate: `TestTxtPaginateExact` (exact per-page block/char counts). convert: `TestTxtToEPUBFullChain` (exact final text after three hops), `TestTXT`, `TestTXTWrapDeterministic`, `TestFB2RoundTrip`, `TestXHTMLEscaping`. |
| CBZ | PASS (ext only) | PASS (container-level: page bytes + order; **image decoding/layout NOT IMPLEMENTED**) | PASS as fixed page enumeration (`cbz.Pages` = one physical page per image). Reflowable `Paginate` on CBZ docs is degenerate (zero-char blocks) — comic pagination is enumeration, not text packing. | **NOT IMPLEMENTED** | PASS → cbz (repack) ; export into EPUB spine only as image plates | enumeration: `TestPagesEnumeration`, `TestPagesIDsAgreeWithParse`, `TestPagesFailsClosed` (corrupt zip, no images, > MaxPages), `TestPagesDoesNotInflate` (corrupted central-directory CRC still enumerable → proves directory-only reads), `FuzzPages`; single-page extraction and the type a page stream may declare: `TestPagePayloadsFollowEnumeratedOrder`, `TestPageIndexOutOfRange`, `TestPageInflationIsBounded`, `TestStreamMediaTypeReadsOnlySignatures` (inflates signatures only), `TestStreamMediaTypeCases` (mixed types and unprovable payloads refuse), `TestSniffMediaType` (core/pkg/parse/cbz/pages.go). OPDS-PSE serving rules for these pages are in api.md, not a core claim. parse: `TestCBZPageOrder`, `TestCBZNoImages`, limits: `TestCBZRejectsOversizedImage` (per-entry + `MaxAggregateBytes`), `TestCBZRejectsExcessPages`, `FuzzParse`. convert: `TestEPUBToCBZRepack` (byte-exact payloads), `TestExportCBZFailClosed` (refuses non-PNG/JPEG, no silent packaging of unknown encodings). FB2/ZIP/CBT/CB7 registry rows stay detect/stream-only: RAR/7z decompression NOT IMPLEMENTED. |

## Every other registered format (PDF, AZW3/KF8, DOCX, DOC, ODT, RTF, HTML/HTMLZ, CBR/CBT/CB7/CBA, DJVU, CHM, LIT, SNB, LRF, PML, RB, WebPub, AudioBook, Divina, LPF, PNLD, DAISY, LCP-*, plain ZIP, audio)

detect: PASS (extension only) — parse: **NOT IMPLEMENTED** — paginate: **NOT IMPLEMENTED** — render: **NOT IMPLEMENTED** — convert: **NOT IMPLEMENTED** (all Blocked; same-format repack Blocked too, since no parser+exporter pair exists — enforced by `TestSameFormatRepackIsHonest`, `TestMatrixNeverOutrunsTheCode`). PDF specifically: page rendering requires a MuPDF-class engine (plan P-2, server-side), nothing in core produces or rasterizes PDF pages. DRM/LCP (`Protected=true` rows): fulfillment NOT IMPLEMENTED; registry forbids `DRMFulfill`/`Convert` claims (`TestProtectedFormatsDoNotClaimConversion`, `TestNoDescriptorClaimsUnimplementedRenderOrDRM`). `ContentAI` was removed from parsed-format caps because no core AI engine exists (`TestNoDescriptorClaimsUnimplementedAI`) — parsing is not AI.

## Conversion matrix as implemented (`core/formats/formats.go`)

| from \ to | TXT | XHTML | FB2 | EPUB | CBZ | MOBI | PDF/AZW3/… |
|---|---|---|---|---|---|---|---|
| EPUB | Direct | Direct | Direct | Direct (repack) | Direct (image plates) | Blocked | Blocked |
| FB2 | Direct | Direct | Direct (text-only projection¹) | Direct | Blocked² | Blocked | Blocked |
| MOBI | Direct | Direct | Direct | Direct | Blocked² | **Blocked (no PalmDOC writer)** | Blocked |
| TXT | Direct | Direct | Direct | Direct | Blocked² | Blocked | Blocked |
| CBZ | Blocked | Blocked | Blocked | Blocked | Direct (repack) | Blocked | Blocked |

¹ `toFB2` drops non-paragraph/quote blocks by design (documented loss, not silent: image-bearing sources re-exported to FB2 lose `<binary>`). ² Engine-wise FB2/TXT→CBZ would work when payloads are PNG/JPEG, but the matrix stays Blocked until a test claims the row (§7.1: matrix lists tested guarantees only; `TestMatrixNeverOutrunsTheCode`).

Same-format Direct exists only for EPUB/FB2/TXT/CBZ (parser+exporter both implemented). PDF→PDF, XHTML→XHTML, MOBI→MOBI are deliberately Blocked: no parser/exporter pair, no byte-passthrough implemented in core.

## Bounded-input policy (all parse paths)

`core/pkg/parse/parseutil/limits.go`: MaxEntryBytes 16 MiB, MaxImageBytes 32 MiB, MaxTextBytes 64 MiB, MaxAggregateBytes 128 MiB, MaxPages 10000, MaxArchiveEntries 100000, MaxXMLDepth 256. `ReadAll` rejects (never truncates) oversized entries. Fail-closed proof per format: tests named in the limits columns above; `TestReadAllBounds` for the primitive itself. Fuzz targets (run with seeds via `-run`, live via `-fuzz -fuzztime 8s`, log `evidence/campaign-2026-11-20/formats/fuzz.log`): `FuzzDetect`, `FuzzParse` (txt/fb2/epub/mobi/cbz), `FuzzSpine`, `FuzzPages`, `FuzzExportEPUB`, `FuzzPaginate`. The fb2 live fuzz found one real pre-existing panic (unclosed nested section → nil chapter); the crashing input is kept as a regression seed.

## Explicit NOT IMPLEMENTED (what would be required)

- **Rendering, all formats**: requires the client layout engine (glyph shaping, image layout, fixed-layout viewport). Core delivers HDM + page geometry only.
- **PDF parse/render**: requires MuPDF-class engine integration behind the parser contract + rasterization tests; currently detect+stream only.
- **DRM/LCP fulfillment**: requires an LCP client (license validation, decryption, state); registry forbids the claim.
- **EPUB fixed-layout pagination** and media overlays: enumeration only.
- **MOBI/KF8/AZW3**: KF8 container parse, Huffman (HUFF/CDIC) compression, EXTH-image extraction, and any MOBI *writing* are unimplemented; P-2 keeps AZW3 conversion server-side (Calibre).
- **Font subsetting**: nowhere in core.
- **Content/magic-byte detection**: `formats.Detect` is filename-only; a real sniffer must read container signatures (mimetype entry, PalmDB, `%PDF-`) and be tested before any claim upgrades.
- **Image formats beyond PNG/JPEG on export**: CBZ parse stores bytes of 6 extensions; convert exports only sniffed PNG/JPEG and refuses the rest rather than mislabeling them. GIF/WebP/AVIF export requires encoder support (no new dependencies currently) or honest rejection, which is implemented.
