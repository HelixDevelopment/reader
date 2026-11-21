// Package pse implements the OPDS-PSE (OPDS Page Streaming Extension)
// acquisition link: parsing it out of an OPDS feed and instantiating its
// URL template for a page. Nothing else.
//
// Provenance of every normative claim here, from texts retrieved
// 2026-11-21 (verbatim quotation, plus the list of things the extension does
// not say, in evidence/pse-spec/opds-pse-v1.2.md):
//
//   - https://github.com/opds-pse/opds-pse-spec returns HTTP 404; that
//     repository does not exist. The authoritative home of OPDS-PSE is
//     https://github.com/anansi-project/opds-pse, rendered at
//     https://anansi-project.github.io/docs/opds-pse/specs/v1.2, with the
//     original v1.0 at https://vaemendis.net/opds-pse/.
//   - Those texts define only: the namespace
//     "http://vaemendis.net/opds-pse/ns"; an acquisition <link> with
//     rel="http://vaemendis.net/opds-pse/stream"; type MUST be image/jpeg,
//     image/gif or image/png; pse:count MUST give the page count; href MUST
//     contain {pageNumber} and MAY contain {maxWidth}, which the client MUST
//     replace; pages are numbered 0..N-1; double pages are one page. v1.1
//     and v1.2 add pse:lastRead (numbering starts at 1) and pse:lastReadDate.
//   - Pages are fetched one per HTTP request. The retrieved specs define no
//     container format, no media type for one, no section or status framing,
//     and no Range conventions, so this package has none.
//
// What was deliberately not kept. An earlier revision of this package also
// carried a 4-byte-header / section-count / status-byte stream decoder and
// its own byte-range arithmetic, written against a design brief that
// described both as OPDS-PSE. Neither appears in v1.0, v1.1 or v1.2, and the
// page fetches the extension actually defines are served by the HTTP layer
// that already implements Range (server/internal/httpapi), including its
// mutation-verified 206/416 and If-Range behavior. Shipping a second,
// unwired implementation of a format no client emits would report a
// capability that does not exist, so both were removed rather than
// documented as provisional.
//
// All functions are pure: bytes and strings in, links and URLs out. No
// network, no HTTP framework, no database.
package pse
