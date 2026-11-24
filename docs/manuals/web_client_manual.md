# Web client manual (`clients/web`)

Every image on this page is a real capture of the built web client (version 0.1.0) talking to a real `helix-server` over a real
Redis-backed catalog, taken inside the isolated rootless container runner (`qa/isolation/run_capture_web.sh`, `docs/test-isolation.md`). Each
image has a row in [`evidence/screenshots/web/capture_manifest.json`](../../evidence/screenshots/web/capture_manifest.json) (client, version,
scenario, resolution, sha256, producing command); the directory is sealed with `SHA256SUMS`. The library shown is a generated fixture
(27 works: one EPUB with real metadata, 26 plain-text works named after their files), not real books.

## Connect to a server

On first start the client has no credentials. It says so instead of showing an empty page:

![Error shown before credentials are entered](../../evidence/screenshots/web/web-error-401-before-sign-in.png)

Open **Settings**, fill in the server address (leave it empty for the same origin), the username and the password. The password is kept for
the browser session only; only the address and username are stored in `localStorage`.

![Connection settings](../../evidence/screenshots/web/web-settings.png)

If the server cannot be reached, the client shows the reason and a **Retry** button:

![Unreachable server with Retry](../../evidence/screenshots/web/web-error-unreachable-retry.png)

## Browse the library: responsive layouts and themes

The library grid adapts to phone, tablet and desktop widths and follows the operating-system light/dark preference.

| Phone, light | Phone, dark |
|---|---|
| ![](../../evidence/screenshots/web/web-library-phone-light.png) | ![](../../evidence/screenshots/web/web-library-phone-dark.png) |

| Tablet, light | Tablet, dark |
|---|---|
| ![](../../evidence/screenshots/web/web-library-tablet-light.png) | ![](../../evidence/screenshots/web/web-library-tablet-dark.png) |

![Desktop, light](../../evidence/screenshots/web/web-library-desktop-light.png)

![Desktop, dark](../../evidence/screenshots/web/web-library-desktop-dark.png)

The header button **Theme: system / light / dark** overrides the OS preference (system, then light, then dark, then system again):

![Theme: light](../../evidence/screenshots/web/web-theme-toggle-1.png)
![Theme: dark](../../evidence/screenshots/web/web-theme-toggle-2.png)

## Paging and search (power features)

The server returns 24 works per page; the pager shows the range and enables **Next** / **Previous**.

![Page 1 of the library: 1-24 of 27](../../evidence/screenshots/web/web-library-paging.png)
![Page 2 of the library](../../evidence/screenshots/web/web-library-page-2.png)

Search is executed by the server, not by filtering the page in the browser: typing `Glass` narrows 27 works to one, and a query with no
match shows an explicit empty state.

![Search for "Glass"](../../evidence/screenshots/web/web-search-server-side.png)
![Search with no match](../../evidence/screenshots/web/web-search-empty-state.png)

## Work details and the text reader

A work opens to its details and one **Read** button per available format.

![Work detail (plain text)](../../evidence/screenshots/web/web-work-detail.png)

The reader renders plain text from the live server. **A−** / **A+** change the size in steps of 2 px between 12 and 32 (default 18):

![Reader at default size](../../evidence/screenshots/web/web-reader-txt.png)
![Reader after four A+ clicks (26 px)](../../evidence/screenshots/web/web-reader-large-text.png)

### EPUB in the web client: honest limitation

Title and author of an EPUB come from the file's own metadata. The web client does **not** render EPUB; it says so and offers the original
file for download (use KOReader or another EPUB reader for the actual reading):

![EPUB metadata](../../evidence/screenshots/web/web-detail-epub-metadata.png)
![EPUB limitation notice and Download original](../../evidence/screenshots/web/web-epub-honest-limitation.png)

## Video

[`web-walkthrough-desktop.webm`](../../evidence/screenshots/web/web-walkthrough-desktop.webm) is a continuous recording of the whole
desktop-size session above (Playwright `recordVideo` of the headless browser inside the container; no host screen capture).

## UNCONFIRMED

- Browsers other than Chromium (Firefox, WebKit) were not captured.
- Only the plain-text reader is shown; there is no in-browser EPUB/PDF/CBZ renderer to capture.
- The fixture library has no cover art, so covers are the generated initial-letter tiles.
