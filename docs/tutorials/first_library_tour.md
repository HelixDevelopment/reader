# Tutorial: a first tour of your library in the web and desktop clients

Goal: sign in, find a work with server-side search, read it, and change the text size and theme. About five minutes. Prerequisite: a running
server with a scanned library ([quick start](../quickstart.md)). The screenshots come from a generated 27-work fixture library.

## 1. Sign in

Open the client. Without credentials it tells you the server rejected the request; open **Settings** and enter the server address,
username and password.

![First start, before credentials](../../evidence/screenshots/web/web-error-401-before-sign-in.png)
![Settings form](../../evidence/screenshots/web/web-settings.png)

## 2. Find a work

Type in **Search titles**. The server does the matching; `Glass` leaves one work out of 27.

![Search result](../../evidence/screenshots/web/web-search-server-side.png)

Nothing matching? You get an explicit empty state rather than a blank grid.

![Empty state](../../evidence/screenshots/web/web-search-empty-state.png)

## 3. Open and read

Clear the search, open a plain-text work, press **Read TXT**.

![Work detail](../../evidence/screenshots/web/web-work-detail.png)
![Reading](../../evidence/screenshots/web/web-reader-txt.png)

## 4. Make it comfortable

**A+** four times makes the text 26 px; the **Theme** button in the header cycles system, light, dark.

![Larger text](../../evidence/screenshots/web/web-reader-large-text.png)
![Dark theme](../../evidence/screenshots/web/web-theme-toggle-2.png)

## 5. The same in the desktop client

The Linux desktop client shows the same screens in its own window:

![Desktop reader, dark](../../evidence/screenshots/desktop/desktop-reader-dark-theme.png)

## What this tutorial does not cover

EPUB reading (the web and desktop clients say they do not render EPUB and offer the original for download; use KOReader, see
[`koreader_opds_and_sync.md`](koreader_opds_and_sync.md)).
