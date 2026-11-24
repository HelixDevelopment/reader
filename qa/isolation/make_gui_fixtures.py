#!/usr/bin/env python3
"""Deterministic fixtures for isolated GUI runs. usage: make_gui_fixtures.py <web|koreader> <outdir>
web:      glass-meridian.epub, field-notes.txt, orbital-gardens.txt   (what clients/web/e2e/live.mjs expects)
koreader: 3 real EPUBs (valid OCF/OPF/XHTML with nav) + comic.cbz with 3 real decodable PNG pages."""
import sys, zipfile, zlib, struct
from pathlib import Path
STAMP = (2020, 1, 1, 0, 0, 0)

def zinfo(n, stored=False):
    i = zipfile.ZipInfo(n, STAMP); i.external_attr = 0o644 << 16
    i.compress_type = zipfile.ZIP_STORED if stored else zipfile.ZIP_DEFLATED
    return i

def epub(path, title, author, uid, chapters):
    with zipfile.ZipFile(path, "w") as z:
        z.writestr(zinfo("mimetype", True), "application/epub+zip")
        z.writestr(zinfo("META-INF/container.xml"), '<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>')
        items = "".join(f'<item id="c{i}" href="c{i}.xhtml" media-type="application/xhtml+xml"/>' for i in range(len(chapters)))
        spine = "".join(f'<itemref idref="c{i}"/>' for i in range(len(chapters)))
        z.writestr(zinfo("OEBPS/content.opf"), f'<?xml version="1.0" encoding="UTF-8"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:identifier id="id">{uid}</dc:identifier><dc:title>{title}</dc:title><dc:creator>{author}</dc:creator><dc:language>en</dc:language><meta property="dcterms:modified">2020-01-01T00:00:00Z</meta></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>{items}</manifest><spine>{spine}</spine></package>')
        lis = "".join(f'<li><a href="c{i}.xhtml">{t}</a></li>' for i, (t, _) in enumerate(chapters))
        z.writestr(zinfo("OEBPS/nav.xhtml"), f'<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"><head><title>Contents</title></head><body><nav epub:type="toc"><ol>{lis}</ol></nav></body></html>')
        for i, (t, body) in enumerate(chapters):
            z.writestr(zinfo(f"OEBPS/c{i}.xhtml"), f'<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>{t}</title></head><body><h1>{t}</h1>{"".join(f"<p>{p}</p>" for p in body)}</body></html>')

def png(w, h, rgb):
    raw = b"".join(b"\x00" + bytes(rgb) * w for _ in range(h))
    def ck(t, d): c = struct.pack(">I", len(d)) + t + d; return c + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
    return b"\x89PNG\r\n\x1a\n" + ck(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) + ck(b"IDAT", zlib.compress(raw)) + ck(b"IEND", b"")

LOREM = ["The quick survey of the harbour began at dawn, and the instruments were laid out in careful rows." * 3] * 8

def main(kind, out):
    out = Path(out); out.mkdir(parents=True, exist_ok=True)
    if kind == "web":
        epub(out / "glass-meridian.epub", "The Glass Meridian", "Elena Voss", "urn:uuid:11111111-1111-4111-8111-111111111111", [("Chapter One", LOREM), ("Chapter Two", LOREM)])
        (out / "field-notes.txt").write_text("Field notes, day one.\n\nA second paragraph proves paragraph splitting.\n\nA third paragraph closes the page.\n")
        (out / "orbital-gardens.txt").write_text("Orbital gardens grow in rings.\n\nEach ring turns once a day.\n\nThe tomatoes do not mind.\n")
    else:
        for n, (t, a) in enumerate([("The Glass Meridian", "Elena Voss"), ("Tidal Ledger", "Marek Sol"), ("Ninety Lanterns", "Ines Calder")]):
            epub(out / f"book{n+1}.epub", t, a, f"urn:uuid:2222222{n}-2222-4222-8222-22222222222{n}", [("Opening", LOREM), ("Middle", LOREM), ("Ending", LOREM)])
        with zipfile.ZipFile(out / "comic.cbz", "w") as z:
            for i, c in enumerate([(200, 30, 30), (30, 200, 30), (30, 30, 200)], 1):
                z.writestr(zinfo(f"p{i:02d}.png", True), png(240, 360, c))

if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
