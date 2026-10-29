#!/usr/bin/env python3
"""SQLite-backed Helix workable-items registry and deterministic doc exporter.

The database is authoritative after initialization. Generated Markdown, HTML,
PDF, and HelixTrack outbox files are derived artifacts. No network synchronization
is attempted by this tool; an outbox is not evidence of a remote push.
"""
from __future__ import annotations

import argparse
import contextlib
import datetime as dt
import hashlib
import html
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import sys
import tempfile

SCHEMA_VERSION = "1"
ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DB = ROOT / "docs" / "workable_items.db"
DOCS = ROOT / "docs"
ACTIVE = {"Queued", "In progress", "Ready for testing", "In testing", "Reopened", "Operator-blocked"}
CLOSED = {
    "Bug": "Fixed (→ Fixed.md)",
    "Feature": "Implemented (→ Fixed.md)",
    "Task": "Completed (→ Fixed.md)",
}
TERMINAL = set(CLOSED.values()) | {"Obsolete (→ Fixed.md)"}
STATUSES = ACTIVE | TERMINAL
TYPES = {"Bug", "Feature", "Task"}
SEVERITIES = {"Critical", "High", "Medium", "Low"}
REOPEN_REASONS = {
    "test-failed", "manual-testing-detected", "captured-evidence-contradicts",
    "end-user-report", "cycle-re-discovered", "design-reconsidered",
}
ID_RE = re.compile(r"^ATM-([0-9]{3,})$")

SCHEMA = """
PRAGMA foreign_keys=ON;
CREATE TABLE IF NOT EXISTS items(
 atm_id TEXT PRIMARY KEY,
 type TEXT NOT NULL,
 status TEXT NOT NULL,
 severity TEXT NOT NULL,
 title TEXT NOT NULL,
 description TEXT NOT NULL,
 affected_scope TEXT NOT NULL DEFAULT '',
 reproduction TEXT NOT NULL DEFAULT '',
 acceptance TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 last_modified TEXT NOT NULL,
 current_location TEXT NOT NULL CHECK(current_location IN ('Issues','Fixed')),
 forensic_anchor TEXT NOT NULL DEFAULT '',
 closure_criteria TEXT NOT NULL DEFAULT '',
 composes_with TEXT NOT NULL DEFAULT '[]',
 evidence_path TEXT NOT NULL DEFAULT '',
 created_by TEXT NOT NULL DEFAULT '',
 assigned_to TEXT NOT NULL DEFAULT '',
 reopens_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS item_history(
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 atm_id TEXT NOT NULL REFERENCES items(atm_id) ON DELETE RESTRICT,
 event_type TEXT NOT NULL,
 actor TEXT NOT NULL CHECK(actor IN ('AI','User')),
 on_date TEXT NOT NULL,
 reason TEXT NOT NULL,
 evidence_path TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS obsolete_details(
 atm_id TEXT PRIMARY KEY REFERENCES items(atm_id) ON DELETE CASCADE,
 since TEXT NOT NULL,
 reason TEXT NOT NULL,
 superseding_item TEXT NOT NULL DEFAULT '',
 triple_check_evidence TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS operator_block_details(
 atm_id TEXT PRIMARY KEY REFERENCES items(atm_id) ON DELETE CASCADE,
 what TEXT NOT NULL,
 why_exhausted_alternatives TEXT NOT NULL,
 unblock_condition TEXT NOT NULL,
 who TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reopen_details(
 atm_id TEXT PRIMARY KEY REFERENCES items(atm_id) ON DELETE CASCADE,
 actor TEXT NOT NULL CHECK(actor IN ('AI','User')),
 on_date TEXT NOT NULL,
 reason TEXT NOT NULL,
 evidence_path TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS firebase_metadata(
 atm_id TEXT PRIMARY KEY REFERENCES items(atm_id) ON DELETE CASCADE,
 payload TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sync_outbox(
 atm_id TEXT NOT NULL REFERENCES items(atm_id) ON DELETE CASCADE,
 revision INTEGER NOT NULL,
 payload_hash TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('pending','pushed','failed')),
 remote_id TEXT,
 last_error TEXT,
 updated_at TEXT NOT NULL,
 PRIMARY KEY(atm_id,revision)
);
"""

ITEM_COLUMNS = [
    "atm_id", "type", "status", "severity", "title", "description",
    "affected_scope", "reproduction", "acceptance", "created_at",
    "last_modified", "current_location", "forensic_anchor", "closure_criteria",
    "composes_with", "evidence_path", "created_by", "assigned_to", "reopens_count",
]


def utcnow() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def connect(path: Path) -> sqlite3.Connection:
    path.parent.mkdir(parents=True, exist_ok=True)
    db = sqlite3.connect(path, timeout=30, isolation_level=None)
    db.row_factory = sqlite3.Row
    db.execute("PRAGMA foreign_keys=ON")
    db.execute("PRAGMA busy_timeout=30000")
    db.execute("PRAGMA journal_mode=WAL")
    db.execute("PRAGMA synchronous=FULL")
    db.executescript(SCHEMA)
    db.execute("INSERT OR IGNORE INTO meta(key,value) VALUES('schema_version',?)", (SCHEMA_VERSION,))
    return db


def row_dict(row: sqlite3.Row) -> dict:
    out = {key: row[key] for key in ITEM_COLUMNS}
    out["composes_with"] = json.loads(out["composes_with"])
    return out


def items(db: sqlite3.Connection, location: str | None = None) -> list[dict]:
    sql = "SELECT * FROM items"
    args: tuple = ()
    if location:
        sql += " WHERE current_location=?"
        args = (location,)
    sql += " ORDER BY CAST(substr(atm_id,5) AS INTEGER)"
    return [row_dict(row) for row in db.execute(sql, args)]


def canonical_item(item: dict) -> dict:
    return {key: item[key] for key in ITEM_COLUMNS}


def integrity(db: sqlite3.Connection) -> str:
    payload = {
        "items": [canonical_item(i) for i in items(db)],
        "history": [dict(r) for r in db.execute(
            "SELECT atm_id,event_type,actor,on_date,reason,evidence_path FROM item_history ORDER BY sequence")],
        "operator_blocks": [dict(r) for r in db.execute("SELECT * FROM operator_block_details ORDER BY atm_id")],
        "reopens": [dict(r) for r in db.execute("SELECT * FROM reopen_details ORDER BY atm_id")],
        "obsolete": [dict(r) for r in db.execute("SELECT * FROM obsolete_details ORDER BY atm_id")],
    }
    encoded = json.dumps(payload, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(encoded).hexdigest()


def checkpoint(db: sqlite3.Connection) -> None:
    db.execute("PRAGMA wal_checkpoint(TRUNCATE)")


def update_meta(db: sqlite3.Connection, direction: str) -> None:
    now = utcnow()
    db.execute("INSERT INTO meta(key,value) VALUES('last_sync_direction',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (direction,))
    db.execute("INSERT INTO meta(key,value) VALUES('last_sync_timestamp',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (now,))
    db.execute("INSERT INTO meta(key,value) VALUES('integrity_hash',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (integrity(db),))


def validate(db: sqlite3.Connection) -> list[str]:
    errors: list[str] = []
    seen = set()
    for item in items(db):
        atm = item["atm_id"]
        match = ID_RE.match(atm)
        if not match or int(match.group(1)) < 1 or atm in seen:
            errors.append(f"{atm}: invalid or duplicate ATM identifier")
        seen.add(atm)
        if item["type"] not in TYPES:
            errors.append(f"{atm}: invalid type")
        if item["status"] not in STATUSES:
            errors.append(f"{atm}: invalid status")
        if item["severity"] not in SEVERITIES:
            errors.append(f"{atm}: invalid severity")
        expected_location = "Issues" if item["status"] in ACTIVE else "Fixed"
        if item["current_location"] != expected_location:
            errors.append(f"{atm}: status/location mismatch")
        if item["status"] in TERMINAL and item["status"] != "Obsolete (→ Fixed.md)" and item["status"] != CLOSED.get(item["type"]):
            errors.append(f"{atm}: closure vocabulary does not match type")
        if len(item["description"].split()) < 6 or len(item["description"]) < 40:
            errors.append(f"{atm}: description is not comprehensive")
        for field in ("affected_scope", "reproduction", "acceptance"):
            if not item[field].strip():
                errors.append(f"{atm}: missing {field}")
        if item["status"] in TERMINAL and not item["evidence_path"].strip():
            errors.append(f"{atm}: terminal item lacks evidence")
        if item["status"] == "Reopened":
            row = db.execute("SELECT * FROM reopen_details WHERE atm_id=?", (atm,)).fetchone()
            if not row or not row["evidence_path"] or row["actor"] not in {"AI", "User"}:
                errors.append(f"{atm}: incomplete reopen attribution")
        if item["status"] == "Operator-blocked":
            row = db.execute("SELECT * FROM operator_block_details WHERE atm_id=?", (atm,)).fetchone()
            if not row or any(not row[k].strip() for k in ("what", "why_exhausted_alternatives", "unblock_condition", "who")):
                errors.append(f"{atm}: incomplete operator-block details")
            elif "Attempted:" not in row["why_exhausted_alternatives"]:
                errors.append(f"{atm}: missing self-resolution exhaustion audit")
        if item["reopens_count"] != db.execute("SELECT count(*) FROM item_history WHERE atm_id=? AND event_type='Reopened'", (atm,)).fetchone()[0]:
            errors.append(f"{atm}: reopen count does not match history")
    allocated = sorted(int(ID_RE.match(i["atm_id"]).group(1)) for i in items(db) if ID_RE.match(i["atm_id"]))
    if allocated and allocated != list(range(1, max(allocated) + 1)):
        errors.append("ATM allocation contains gaps")
    if db.execute("PRAGMA foreign_key_check").fetchall():
        errors.append("SQLite foreign key check failed")
    return errors


def md_escape(text: str) -> str:
    return text.replace("|", "\\|").replace("\n", " ")


def render_item(item: dict, db: sqlite3.Connection) -> str:
    lines = [
        f'## [{item["atm_id"]}] {item["title"]}',
        "",
        f'**Status:** {item["status"]}',
        f'**Type:** {item["type"]}',
        f'**Severity:** {item["severity"]}',
        f'**Reopens:** {item["reopens_count"]}',
    ]
    if item["status"] == "Reopened":
        detail = db.execute("SELECT * FROM reopen_details WHERE atm_id=?", (item["atm_id"],)).fetchone()
        lines.append(f'**Reopened-Details:** By: {detail["actor"]}; On: {detail["on_date"]}; Reason: {detail["reason"]}; Evidence: {detail["evidence_path"]}')
    if item["status"] == "Operator-blocked":
        detail = db.execute("SELECT * FROM operator_block_details WHERE atm_id=?", (item["atm_id"],)).fetchone()
        lines.append(f'**Operator-Block-Details:** WHAT: {detail["what"]} WHY: {detail["why_exhausted_alternatives"]} UNBLOCK CONDITION: {detail["unblock_condition"]} WHO: {detail["who"]}')
    lines += [
        "",
        f'**Description:** {item["description"]}',
        "",
        f'**Affected scope:** {item["affected_scope"]}',
        "",
        f'**Reproduction / current observation:** {item["reproduction"]}',
        "",
        f'**Acceptance:** {item["acceptance"]}',
        "",
        f'**Closure criteria:** {item["closure_criteria"]}',
        "",
        f'**Evidence:** {item["evidence_path"] or "Not yet available; no pass is claimed."}',
        "",
        f'**Forensic anchor:** {item["forensic_anchor"]}',
        "",
        f'**Composes with:** {", ".join(item["composes_with"]) or "None recorded"}',
        "",
        f'**Attribution:** created by {item["created_by"] or "UNKNOWN"}; assigned to {item["assigned_to"] or "UNKNOWN"}',
        "",
        f'**Created / modified:** {item["created_at"]} / {item["last_modified"]}',
        "",
    ]
    history = list(db.execute("SELECT * FROM item_history WHERE atm_id=? ORDER BY sequence", (item["atm_id"],)))
    if history:
        lines += ["**History:**", ""]
        for event in history:
            lines.append(f'- {event["on_date"]} — {event["event_type"]} by {event["actor"]}: {event["reason"]}; evidence: {event["evidence_path"] or "none recorded"}')
        lines.append("")
    return "\n".join(lines)


def summary_rows(rows: list[dict]) -> str:
    out = ["| ATM ID | Type | Severity | Status | Reopens | Description |", "|---|---|---|---|---:|---|"]
    out += [f'| {i["atm_id"]} | {i["type"]} | {i["severity"]} | {md_escape(i["status"])} | {i["reopens_count"]} | {md_escape(i["title"])} |' for i in rows]
    return "\n".join(out)


def markdown_documents(db: sqlite3.Connection) -> dict[str, str]:
    open_items, closed_items = items(db, "Issues"), items(db, "Fixed")
    counts = {status: sum(i["status"] == status for i in open_items) for status in sorted(ACTIVE)}
    rollup = ", ".join(f"{key}: {value}" for key, value in counts.items())
    documents = {
        "Issues.md": "# Helix Open Workable Items\n\nGenerated from `docs/workable_items.db`; do not edit directly.\n\n" + rollup + "\n\n" + "\n".join(render_item(i, db) for i in open_items),
        "Fixed.md": "# Helix Closed Workable Items\n\nGenerated from `docs/workable_items.db`; do not edit directly.\n\n" + "\n".join(render_item(i, db) for i in closed_items),
        "Issues_Summary.md": "# Helix Open Workable Items Summary\n\nGenerated from the authoritative SQLite registry.\n\n" + rollup + "\n\n" + summary_rows(open_items) + "\n",
        "Fixed_Summary.md": "# Helix Closed Workable Items Summary\n\nGenerated from the authoritative SQLite registry.\n\n" + summary_rows(closed_items) + "\n",
    }
    return documents


def html_documents(db: sqlite3.Connection) -> dict[str, str]:
    result = {}
    for name, markdown in markdown_documents(db).items():
        rows = items(db, "Issues" if name.startswith("Issues") else "Fixed")
        summary = name.endswith("Summary.md")
        title = html.escape(markdown.splitlines()[0].removeprefix("# "))
        if summary:
            body = ["<table><thead><tr><th>ATM ID</th><th>Type</th><th>Severity</th><th>Status</th><th>Reopens</th><th>Description</th></tr></thead><tbody>"]
            for i in rows:
                body.append("<tr>" + "".join(f"<td>{html.escape(str(v))}</td>" for v in (i["atm_id"], i["type"], i["severity"], i["status"], i["reopens_count"], i["title"])) + "</tr>")
            body.append("</tbody></table>")
        else:
            body = []
            for i in rows:
                body.append(f'<section id="{i["atm_id"]}"><h2>[{i["atm_id"]}] {html.escape(i["title"])}</h2>')
                body.append(f'<p><strong>Status:</strong> {html.escape(i["status"])}<br><strong>Type:</strong> {i["type"]}<br><strong>Severity:</strong> {i["severity"]}<br><strong>Reopens:</strong> {i["reopens_count"]}</p>')
                for label, key in (("Description", "description"), ("Affected scope", "affected_scope"), ("Reproduction / current observation", "reproduction"), ("Acceptance", "acceptance"), ("Closure criteria", "closure_criteria"), ("Evidence", "evidence_path")):
                    body.append(f'<h3>{label}</h3><p>{html.escape(i[key]) or "Not yet available; no pass is claimed."}</p>')
                body.append("</section>")
        result[name[:-3] + ".html"] = f'''<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>{title}</title><style>body{{font:15px/1.55 system-ui,sans-serif;max-width:1100px;margin:2rem auto;padding:0 1rem;color:#222}}table{{border-collapse:collapse;width:100%;font-size:13px}}th,td{{border:1px solid #bbb;padding:.45rem;text-align:left;vertical-align:top}}section{{border-top:2px solid #345;margin-top:2rem}}h1,h2{{color:#234}}code{{overflow-wrap:anywhere}}@media print{{body{{max-width:none;margin:0}}section{{break-inside:avoid}}}}</style></head><body><h1>{title}</h1><p>Generated from <code>docs/workable_items.db</code>; remote synchronization is not implied.</p>{''.join(body)}</body></html>'''
    return result


def pdf_documents(db: sqlite3.Connection, output_dir: Path) -> list[Path]:
    from reportlab.lib import colors
    from reportlab.lib.enums import TA_LEFT
    from reportlab.lib.pagesizes import A4, landscape
    from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
    from reportlab.lib.units import mm
    from reportlab.platypus import PageBreak, Paragraph, SimpleDocTemplate, Spacer, Table, TableStyle

    paths = []
    base = getSampleStyleSheet()
    body = ParagraphStyle("TrackerBody", parent=base["BodyText"], fontName="Helvetica", fontSize=9, leading=12, alignment=TA_LEFT, spaceAfter=5)
    small = ParagraphStyle("TrackerSmall", parent=body, fontSize=7, leading=9)
    heading = ParagraphStyle("TrackerHeading", parent=base["Heading2"], fontName="Helvetica-Bold", fontSize=13, leading=16, spaceBefore=8, spaceAfter=7, keepWithNext=True)
    title = ParagraphStyle("TrackerTitle", parent=base["Title"], fontName="Helvetica-Bold", fontSize=20, leading=24, spaceAfter=14)

    def footer(canvas, doc):
        canvas.saveState()
        canvas.setFont("Helvetica", 7)
        canvas.setFillColor(colors.HexColor("#555555"))
        canvas.drawString(18 * mm, 10 * mm, "Helix workable items — generated from SQLite; remote synchronization is not implied")
        canvas.drawRightString(doc.pagesize[0] - 18 * mm, 10 * mm, f"Page {doc.page}")
        canvas.restoreState()

    for stem, location in (("Issues", "Issues"), ("Fixed", "Fixed")):
        for summary in (False, True):
            name = stem + ("_Summary" if summary else "") + ".pdf"
            path = output_dir / name
            rows = items(db, location)
            page = landscape(A4) if summary else A4
            doc = SimpleDocTemplate(str(path), pagesize=page, rightMargin=16 * mm, leftMargin=16 * mm, topMargin=16 * mm, bottomMargin=18 * mm, title=f"Helix {stem} Workable Items", author="Helix Reader")
            story = [Paragraph(f"Helix {stem} Workable Items" + (" Summary" if summary else ""), title), Paragraph("Generated from docs/workable_items.db. This document does not assert remote HelixTrack synchronization.", body), Spacer(1, 4 * mm)]
            if summary:
                data = [[Paragraph(f"<b>{x}</b>", small) for x in ("ATM ID", "Type", "Severity", "Status", "Reopens", "Description")]]
                for i in rows:
                    data.append([Paragraph(html.escape(str(v)), small) for v in (i["atm_id"], i["type"], i["severity"], i["status"], i["reopens_count"], i["title"])])
                widths = [22 * mm, 20 * mm, 20 * mm, 42 * mm, 16 * mm, 140 * mm]
                table = Table(data, colWidths=widths, repeatRows=1)
                table.setStyle(TableStyle([("LINEABOVE", (0, 0), (-1, 0), 1, colors.HexColor("#334455")), ("LINEBELOW", (0, 0), (-1, 0), .6, colors.HexColor("#334455")), ("LINEBELOW", (0, -1), (-1, -1), 1, colors.HexColor("#334455")), ("VALIGN", (0, 0), (-1, -1), "TOP"), ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 4)]))
                story.append(table)
            else:
                for index, i in enumerate(rows):
                    if index:
                        story.append(PageBreak())
                    story.append(Paragraph(f'[{i["atm_id"]}] {html.escape(i["title"])}', heading))
                    story.append(Paragraph(f'<b>Status:</b> {html.escape(i["status"])} &nbsp; <b>Type:</b> {i["type"]} &nbsp; <b>Severity:</b> {i["severity"]} &nbsp; <b>Reopens:</b> {i["reopens_count"]}', body))
                    for label, key in (("Description", "description"), ("Affected scope", "affected_scope"), ("Reproduction / current observation", "reproduction"), ("Acceptance", "acceptance"), ("Closure criteria", "closure_criteria"), ("Evidence", "evidence_path"), ("Forensic anchor", "forensic_anchor")):
                        value = i[key] or "Not yet available; no pass is claimed."
                        story.append(Paragraph(f"<b>{label}:</b> {html.escape(value)}", body))
            doc.build(story, onFirstPage=footer, onLaterPages=footer)
            paths.append(path)
    return paths


def export(db: sqlite3.Connection, output_dir: Path = DOCS, include_pdf: bool = True) -> list[Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    written = []
    for documents in (markdown_documents(db), html_documents(db)):
        for name, content in documents.items():
            path = output_dir / name
            path.write_text(content, encoding="utf-8", newline="\n")
            written.append(path)
    if include_pdf:
        written.extend(pdf_documents(db, output_dir))
    update_meta(db, "db-to-docs")
    checkpoint(db)
    return written


def parse_markdown(path: Path, location: str) -> list[dict]:
    text = path.read_text(encoding="utf-8")
    sections = re.split(r"(?m)^## \[(ATM-[0-9]{3,})\] ", text)[1:]
    parsed = []
    for index in range(0, len(sections), 2):
        atm, section = sections[index], sections[index + 1]
        title, _, body = section.partition("\n")

        def field(label: str) -> str:
            match = re.search(rf"(?m)^\*\*{re.escape(label)}:\*\* (.*)$", body)
            return match.group(1).strip() if match else ""

        def multiline(label: str) -> str:
            match = re.search(rf"(?ms)^\*\*{re.escape(label)}:\*\* (.*?)(?:\n\n\*\*|\n\n- |\Z)", body)
            return match.group(1).strip() if match else ""

        created_modified = field("Created / modified")
        created, _, modified = created_modified.partition(" / ")
        status = field("Status")
        attribution = field("Attribution")
        attribution_match = re.match(r"created by (.*); assigned to (.*)$", attribution)
        created_by, assigned_to = attribution_match.groups() if attribution_match else ("", "")
        created_by = "" if created_by == "UNKNOWN" else created_by
        assigned_to = "" if assigned_to == "UNKNOWN" else assigned_to
        composes = field("Composes with")
        evidence = multiline("Evidence")
        if evidence == "Not yet available; no pass is claimed.":
            evidence = ""
        history = []
        for line in re.findall(r"(?m)^- (\d{4}-\d{2}-\d{2}) — (Opened|Updated|Reopened|Fixed|Implemented|Completed|Obsolete) by (AI|User): (.*); evidence: (.*)$", body):
            history.append({"on_date": line[0], "event_type": line[1], "actor": line[2], "reason": line[3], "evidence_path": "" if line[4] == "none recorded" else line[4]})
        parsed.append({
            "atm_id": atm, "type": field("Type"), "status": status,
            "severity": field("Severity"), "title": title.strip(),
            "description": multiline("Description"), "affected_scope": multiline("Affected scope"),
            "reproduction": multiline("Reproduction / current observation"),
            "acceptance": multiline("Acceptance"), "closure_criteria": multiline("Closure criteria"),
            "evidence_path": evidence, "forensic_anchor": multiline("Forensic anchor"),
            "created_at": created, "last_modified": modified or created,
            "current_location": location, "composes_with": json.dumps([] if composes == "None recorded" else [v.strip() for v in composes.split(",")]),
            "created_by": created_by, "assigned_to": assigned_to, "reopens_count": int(field("Reopens") or 0),
            "_history": history, "_reopened_details": field("Reopened-Details"), "_operator_block_details": field("Operator-Block-Details"),
        })
    return parsed


def import_markdown(db: sqlite3.Connection, docs_dir: Path) -> int:
    incoming = parse_markdown(docs_dir / "Issues.md", "Issues") + parse_markdown(docs_dir / "Fixed.md", "Fixed")
    if not incoming:
        raise ValueError("no workable-item sections found")
    db.execute("BEGIN IMMEDIATE")
    try:
        for item in incoming:
            columns = ITEM_COLUMNS
            db.execute(f"INSERT INTO items({','.join(columns)}) VALUES({','.join('?' for _ in columns)}) ON CONFLICT(atm_id) DO UPDATE SET " + ",".join(f"{c}=excluded.{c}" for c in columns if c != "atm_id"), [item[c] for c in columns])
            if not db.execute("SELECT 1 FROM item_history WHERE atm_id=? LIMIT 1", (item["atm_id"],)).fetchone():
                for event in item["_history"]:
                    db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (item["atm_id"], event["event_type"], event["actor"], event["on_date"], event["reason"], event["evidence_path"]))
            if item["_reopened_details"]:
                match = re.match(r"By: (AI|User); On: (\d{4}-\d{2}-\d{2}); Reason: (.*); Evidence: (.*)$", item["_reopened_details"])
                if not match:
                    raise ValueError(f'{item["atm_id"]}: malformed Reopened-Details')
                db.execute("INSERT INTO reopen_details(atm_id,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET actor=excluded.actor,on_date=excluded.on_date,reason=excluded.reason,evidence_path=excluded.evidence_path", (item["atm_id"], *match.groups()))
            if item["_operator_block_details"]:
                match = re.match(r"WHAT: (.*) WHY: (Attempted: .*) UNBLOCK CONDITION: (.*) WHO: (.*)$", item["_operator_block_details"])
                if not match:
                    raise ValueError(f'{item["atm_id"]}: malformed Operator-Block-Details')
                db.execute("INSERT INTO operator_block_details(atm_id,what,why_exhausted_alternatives,unblock_condition,who) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET what=excluded.what,why_exhausted_alternatives=excluded.why_exhausted_alternatives,unblock_condition=excluded.unblock_condition,who=excluded.who", (item["atm_id"], *match.groups()))
        update_meta(db, "md-to-db")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise
    return len(incoming)


def diff(db: sqlite3.Connection, docs_dir: Path = DOCS) -> list[str]:
    expected = markdown_documents(db)
    differences = []
    for name, content in expected.items():
        path = docs_dir / name
        if not path.exists() or path.read_bytes() != content.encode("utf-8"):
            differences.append(name)
    for name, content in html_documents(db).items():
        path = docs_dir / name
        if not path.exists() or path.read_bytes() != content.encode("utf-8"):
            differences.append(name)
    return differences


def helixtrack_outbox(db: sqlite3.Connection, path: Path) -> None:
    records = []
    for item in items(db):
        payload = canonical_item(item)
        records.append({
            "operation": "upsert", "client_key": item["atm_id"], "revision": item["reopens_count"] + 1,
            "payload": payload, "remote_status": "NOT RUN",
            "reason": "Exact HelixTrack API mapping and credentials have not been verified; this file is an offline outbox, not a remote push receipt.",
        })
    envelope = {"schema": "helix.workable-items.outbox.v1", "generated_at": utcnow(), "records": records}
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(envelope, indent=2, ensure_ascii=False, sort_keys=True) + "\n", encoding="utf-8")


def add_item(db: sqlite3.Connection, args: argparse.Namespace) -> str:
    db.execute("BEGIN IMMEDIATE")
    try:
        maximum = db.execute("SELECT COALESCE(MAX(CAST(substr(atm_id,5) AS INTEGER)),0) FROM items").fetchone()[0]
        atm = f"ATM-{maximum + 1:03d}"
        now = utcnow()
        item = {
            "atm_id": atm, "type": args.type, "status": "Queued", "severity": args.severity,
            "title": args.title, "description": args.description, "affected_scope": args.scope,
            "reproduction": args.reproduction, "acceptance": args.acceptance,
            "closure_criteria": args.acceptance, "forensic_anchor": args.anchor,
            "created_at": now, "last_modified": now, "current_location": "Issues",
            "composes_with": "[]", "evidence_path": "", "created_by": args.created_by,
            "assigned_to": args.assigned_to, "reopens_count": 0,
        }
        db.execute(f"INSERT INTO items({','.join(ITEM_COLUMNS)}) VALUES({','.join('?' for _ in ITEM_COLUMNS)})", [item[c] for c in ITEM_COLUMNS])
        db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (atm, "Opened", args.created_by if args.created_by in {"AI", "User"} else "User", now[:10], "new-work-item", ""))
        update_meta(db, "add")
        db.execute("COMMIT")
        return atm
    except Exception:
        db.execute("ROLLBACK")
        raise


def history(db: sqlite3.Connection, atm: str, event: str, actor: str, reason: str, evidence: str = "") -> None:
    db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (atm, event, actor, utcnow()[:10], reason, evidence))


def require_item(db: sqlite3.Connection, atm: str) -> dict:
    row = db.execute("SELECT * FROM items WHERE atm_id=?", (atm,)).fetchone()
    if not row:
        raise ValueError(f"{atm}: item not found")
    return row_dict(row)


def close_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    item = require_item(db, args.atm_id)
    expected = CLOSED[item["type"]]
    aliases = {"fixed": "Bug", "implemented": "Feature", "completed": "Task"}
    if args.status and aliases[args.status] != item["type"]:
        raise ValueError(f'{item["atm_id"]}: {item["type"]} requires {expected}')
    if not args.evidence.strip():
        raise ValueError("closure requires captured evidence")
    db.execute("BEGIN IMMEDIATE")
    try:
        db.execute("UPDATE items SET status=?,current_location='Fixed',evidence_path=?,last_modified=? WHERE atm_id=?", (expected, args.evidence, utcnow(), args.atm_id))
        history(db, args.atm_id, expected.split(" ")[0], args.actor, args.reason, args.evidence)
        update_meta(db, "close")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise


def reopen_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    require_item(db, args.atm_id)
    if not args.evidence.strip():
        raise ValueError("reopen requires positive evidence")
    db.execute("BEGIN IMMEDIATE")
    try:
        db.execute("UPDATE items SET status='Reopened',current_location='Issues',reopens_count=reopens_count+1,last_modified=? WHERE atm_id=?", (utcnow(), args.atm_id))
        db.execute("INSERT INTO reopen_details(atm_id,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET actor=excluded.actor,on_date=excluded.on_date,reason=excluded.reason,evidence_path=excluded.evidence_path", (args.atm_id, args.actor, utcnow()[:10], args.reason, args.evidence))
        history(db, args.atm_id, "Reopened", args.actor, args.reason, args.evidence)
        update_meta(db, "reopen")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise


def block_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    require_item(db, args.atm_id)
    if not args.why.startswith("Attempted:"):
        raise ValueError("operator-block exhaustion audit must start with 'Attempted:'")
    db.execute("BEGIN IMMEDIATE")
    try:
        db.execute("UPDATE items SET status='Operator-blocked',current_location='Issues',last_modified=? WHERE atm_id=?", (utcnow(), args.atm_id))
        db.execute("INSERT INTO operator_block_details(atm_id,what,why_exhausted_alternatives,unblock_condition,who) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET what=excluded.what,why_exhausted_alternatives=excluded.why_exhausted_alternatives,unblock_condition=excluded.unblock_condition,who=excluded.who", (args.atm_id, args.what, args.why, args.unblock, args.who))
        history(db, args.atm_id, "Updated", "AI", "operator-blocked-after-exhaustion", "")
        update_meta(db, "operator-block")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise


def update_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    require_item(db, args.atm_id)
    changes = {}
    for option, column in (("title", "title"), ("description", "description"), ("scope", "affected_scope"), ("reproduction", "reproduction"), ("acceptance", "acceptance"), ("severity", "severity")):
        value = getattr(args, option)
        if value is not None:
            changes[column] = value
            if column == "acceptance":
                changes["closure_criteria"] = value
    if not changes:
        raise ValueError("update requires at least one field")
    changes["last_modified"] = utcnow()
    db.execute("BEGIN IMMEDIATE")
    try:
        db.execute("UPDATE items SET " + ",".join(f"{key}=?" for key in changes) + " WHERE atm_id=?", [*changes.values(), args.atm_id])
        history(db, args.atm_id, "Updated", args.actor, args.reason, "")
        update_meta(db, "update")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", type=Path, default=DEFAULT_DB)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("validate")
    sub.add_parser("diff")
    export_parser = sub.add_parser("export")
    export_parser.add_argument("--output", type=Path, default=DOCS)
    export_parser.add_argument("--no-pdf", action="store_true")
    import_parser = sub.add_parser("import-md")
    import_parser.add_argument("--input", type=Path, default=DOCS)
    outbox_parser = sub.add_parser("helixtrack-outbox")
    outbox_parser.add_argument("--output", type=Path, default=DOCS / "tracking" / "helixtrack-outbox.json")
    add_parser = sub.add_parser("add")
    add_parser.add_argument("type", choices=sorted(TYPES))
    add_parser.add_argument("severity", choices=sorted(SEVERITIES))
    for name in ("title", "description", "scope", "reproduction", "acceptance"):
        add_parser.add_argument("--" + name, required=True)
    add_parser.add_argument("--anchor", default="User work-item request")
    add_parser.add_argument("--created-by", default="User")
    add_parser.add_argument("--assigned-to", default="Operator")
    close_parser = sub.add_parser("close")
    close_parser.add_argument("atm_id")
    close_parser.add_argument("--status", choices=("fixed", "implemented", "completed"), help="Optional type-check alias; canonical status is derived from item type.")
    close_parser.add_argument("--evidence", required=True)
    close_parser.add_argument("--actor", choices=("AI", "User"), default="AI")
    close_parser.add_argument("--reason", default="acceptance-evidence-captured")
    reopen_parser = sub.add_parser("reopen")
    reopen_parser.add_argument("atm_id")
    reopen_parser.add_argument("--actor", choices=("AI", "User"), required=True)
    reopen_parser.add_argument("--reason", required=True)
    reopen_parser.add_argument("--evidence", required=True)
    block_parser = sub.add_parser("operator-block")
    block_parser.add_argument("atm_id")
    block_parser.add_argument("--what", required=True)
    block_parser.add_argument("--why", required=True, help="Attempted: enumerate every exhausted self-resolution path.")
    block_parser.add_argument("--unblock", required=True)
    block_parser.add_argument("--who", required=True)
    update_parser = sub.add_parser("update")
    update_parser.add_argument("atm_id")
    update_parser.add_argument("--title")
    update_parser.add_argument("--description")
    update_parser.add_argument("--scope")
    update_parser.add_argument("--reproduction")
    update_parser.add_argument("--acceptance")
    update_parser.add_argument("--severity", choices=sorted(SEVERITIES))
    update_parser.add_argument("--actor", choices=("AI", "User"), default="AI")
    update_parser.add_argument("--reason", default="registry-detail-update")
    args = parser.parse_args()
    db = connect(args.db)
    try:
        if args.command == "validate":
            problems = validate(db)
            if problems:
                print("\n".join(problems), file=sys.stderr)
                return 1
            print(json.dumps({"status": "PASS", "items": len(items(db)), "integrity_hash": integrity(db)}, sort_keys=True))
        elif args.command == "diff":
            differences = diff(db)
            print(json.dumps({"status": "PASS" if not differences else "FAIL", "differences": differences}, sort_keys=True))
            return bool(differences)
        elif args.command == "export":
            paths = export(db, args.output, not args.no_pdf)
            print(json.dumps({"status": "PASS", "written": [str(p) for p in paths]}, sort_keys=True))
        elif args.command == "import-md":
            count = import_markdown(db, args.input)
            print(json.dumps({"status": "PASS", "imported": count}, sort_keys=True))
        elif args.command == "helixtrack-outbox":
            helixtrack_outbox(db, args.output)
            print(json.dumps({"status": "PASS", "remote_sync": "NOT RUN", "output": str(args.output)}, sort_keys=True))
        elif args.command == "add":
            print(add_item(db, args))
        elif args.command == "close":
            close_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": require_item(db, args.atm_id)["status"]}, sort_keys=True))
        elif args.command == "reopen":
            reopen_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": "Reopened"}, sort_keys=True))
        elif args.command == "operator-block":
            block_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": "Operator-blocked"}, sort_keys=True))
        elif args.command == "update":
            update_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": "Updated"}, sort_keys=True))
        return 0
    finally:
        with contextlib.suppress(Exception):
            checkpoint(db)
        db.close()


if __name__ == "__main__":
    raise SystemExit(main())
