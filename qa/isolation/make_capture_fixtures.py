#!/usr/bin/env python3
"""Deterministic fixtures for the HRD-076 documentation capture runs. usage: make_capture_fixtures.py <outdir>
A larger library than the test fixtures (27 works) so search, paging and the empty state can be shown for real:
glass-meridian.epub (title/author from real EPUB metadata), 26 plain-text works named after their file."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))
from make_gui_fixtures import epub, LOREM

NAMES = ["orbital-gardens", "field-notes", "harbour-survey-log", "tidal-ledger-notes", "lantern-keepers-diary", "salt-road-journal",
         "midnight-cartography", "clockmakers-apprentice", "northern-signal-fires", "the-quiet-archive", "river-atlas-fragments",
         "winter-orchard-letters", "paper-moon-lectures", "copper-sky-chronicle", "driftwood-grammar", "amber-station-reports",
         "bell-tower-sketches", "meadow-engine-manual", "glacier-poems-collected", "kite-maker-handbook", "foxglove-field-guide",
         "lighthouse-weather-diary", "orchard-of-numbers", "slow-train-stories", "stone-bridge-sermons", "violet-hour-essays"]

def main(out):
    out = Path(out); out.mkdir(parents=True, exist_ok=True)
    epub(out / "glass-meridian.epub", "The Glass Meridian", "Elena Voss", "urn:uuid:11111111-1111-4111-8111-111111111111",
         [("Chapter One", LOREM), ("Chapter Two", LOREM)])
    for i, n in enumerate(NAMES):
        pretty = n.replace("-", " ")
        (out / f"{n}.txt").write_text(f"{pretty.capitalize()}, entry {i + 1}.\n\nA second paragraph proves paragraph splitting in the reader.\n\n"
                                      f"A third paragraph closes the page; every work in this fixture set is generated, not copyrighted material.\n\n"
                                      + "\n\n".join(f"Paragraph {k}: the survey continued along the shoreline, noting tides, birds and lamps." for k in range(4, 12)) + "\n")

if __name__ == "__main__":
    main(sys.argv[1])
