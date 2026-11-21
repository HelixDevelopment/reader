# OPDS-PSE specification — retrieval record

Retrieved 2026-11-21 by the coding agent working in this checkout. This file is
a record of *what the specification text says*, kept because `core/pkg/pse`
quotes it. It is not Helix's own normative claim.

## Sources and their status

| URL | Result |
|---|---|
| `https://github.com/opds-pse/opds-pse-spec` | **HTTP 404** — that repository does not exist. An earlier design brief in this project cited it. |
| `https://anansi-project.github.io/docs/opds-pse/specs/v1.2` | 200, text below (published March 05th 2023). |
| `https://vaemendis.net/opds-pse/` | v1.0 home, cited for version history. |
| `https://github.com/anansi-project/opds-pse` | authoritative repository for the above. |

## Verbatim normative text (v1.2)

> The OPDS-PSE extension simply consists in an additional type of Acquisition
> Link in an OPDS entry. This link is to be used by clients to request pages of
> a document, one page at a time.
>
> An OPDS-PSE feed MUST declare the OPDS-PSE namespace:
> `http://vaemendis.net/opds-pse/ns`

Example given by the specification (namespace declared on `<feed>`, prefix `pse`):

```xml
<link rel="http://vaemendis.net/opds-pse/stream"
      type="image/jpeg"
      href="/opds-comics/stream/1217?page={pageNumber}&width={maxWidth}"
      pse:count="35"
      pse:lastRead="10"
      pse:lastReadDate="2010-01-10T10:01:11Z"/>
```

> The Link **MUST** contain the following attributes:
>
> * `rel` **MUST** have the value `http://vaemendis.net/opds-pse/stream` that
>   indicates that this link is a OPDS-PSE page streaming link
> * `type` **MUST** indicate the type of image returned by the server (value
>   **MUST** be one of the following : `image/jpeg`, `image/gif` or `image/png`)
> * `count` **MUST** provide the number of pages of the document
> * `href` **MUST** provide the URL that the client has to use to get individual
>   pages of the document. This URL **MUST** contain the string `{pageNumber}`,
>   which the client **MUST** replace with the number of the desired page. This
>   URL **MAY** contain the string `{maxWidth}` that the client **MUST** replace
>   (if it exists) by the desired maximum width of the returned page.
>
> The Link **MAY** contain the following attributes:
>
> * `lastRead` **MUST** provide the last page read for this document. The
>   numbering starts at 1.
> * `lastReadDate` **MAY** provide the date of when the `lastRead` attribute was
>   last updated. Attribute **MUST** conform [Atom's Date
>   construct](https://www.rfc-editor.org/rfc/rfc4287#section-3.3).
>
> ### Notes
>
> * pages are numbered from `0` to `N-1` (`N` being the total number of pages).
> * double pages (one image containing two pages) **MUST** be treated as a
>   single page. If a split is to be done, it is the responsibility of the
>   client. The server **MUST NOT** split images.

## What the retrieved text does **not** say

Checked 2026-11-21 against v1.0, v1.1 and v1.2; none of these appear anywhere in
the extension, and Helix therefore does not implement or claim them:

- No container/framing format for a page response. A page is one image returned
  by one HTTP request; there is no header, no section count, no status byte.
- No media type for any such container.
- No `Range`/`206`/`416` convention for page streaming. Byte ranges belong to
  HTTP and to the ordinary file route (`server/internal/httpapi`), not here.
- No OPDS 2.0 / JSON representation. The link is defined only as an Acquisition
  Link inside an Atom OPDS entry, so Helix emits it only in the OPDS 1.2 feeds.
- No `{maxWidth}` server-side resizing requirement: the template is optional and
  the client substitutes a value; producing resized images is renderer work that
  is not implemented, so the Helix stream href carries only `{pageNumber}`.
- No `lastRead` write path. Reporting a reading position inside a feed link is
  defined by the extension; Helix does not currently write one, so it emits no
  `pse:lastRead`/`pse:lastReadDate` at all rather than emitting a placeholder.

`pse:lastReadDate` is constrained to Atom's Date construct (RFC 4287 §3.3, i.e.
RFC 3339 `date-time` or `date`), which `core/pkg/pse` validates on parse.
