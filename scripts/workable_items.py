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
import posixpath
from pathlib import Path
import re
import shutil
import sqlite3
import stat
import sys
import tempfile
import unicodedata

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
# Legal non-closing status transitions (constitution 11.4.15 status machine). Closing,
# reopening and operator-blocking keep their own commands because each carries its own
# mandatory records (11.4.33 vocabulary, 11.4.55 reopen details, 11.4.21 block audit).
# Nothing here may target or leave a terminal status.
TRANSITIONS = {
    "Queued": {"In progress"},
    "Reopened": {"In progress"},
    "In progress": {"Queued"},
}
TRANSITION_EVENTS = {"In progress": "Started", "Queued": "Paused"}
# Evidence references on a status transition are resolved against this root, so a
# pointer to an artifact that does not exist is refused instead of recorded.
EVIDENCE_ROOT = ROOT
REOPEN_REASONS = {
    "test-failed", "manual-testing-detected", "captured-evidence-contradicts",
    "end-user-report", "cycle-re-discovered", "design-reconsidered",
}

# SINGLE SOURCE OF TRUTH for the ticket-id key (constitution 11.4.54 per-project key
# extension: the key is per-project DATA, never a literal baked into code). A different
# project changes this one value (or exports HELIX_WORKABLE_ITEMS_KEY). Everything below
# (ID_RE, minting, parsing, messages, exports) derives from it via set_project_key().
DEFAULT_PROJECT_KEY = "HRD"
KEY_RE = re.compile(r"^[A-Z]{2,8}$")
PROJECT_KEY = DEFAULT_PROJECT_KEY
ID_RE = re.compile(r"^$")  # replaced by set_project_key() immediately below


def set_project_key(key: str) -> None:
    """Declare the register's ticket-id key and rebuild every derived pattern."""
    global PROJECT_KEY, ID_RE
    if not KEY_RE.fullmatch(key or ""):
        raise ValueError(f"invalid project key {key!r}: expected 2-8 uppercase letters")
    PROJECT_KEY = key
    ID_RE = re.compile(rf"^{key}-([0-9]{{3,}})$")


def item_id(number: int) -> str:
    return f"{PROJECT_KEY}-{number:03d}"


try:
    set_project_key(os.environ.get("HELIX_WORKABLE_ITEMS_KEY", DEFAULT_PROJECT_KEY))
except ValueError as _key_error:
    print(f"error: HELIX_WORKABLE_ITEMS_KEY: {_key_error}", file=sys.stderr)
    raise SystemExit(2)

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


def busy_ms() -> int:
    """Lock-wait budget; HELIX_WORKABLE_ITEMS_BUSY_MS shortens it (tests, impatient operators)."""
    try:
        return max(0, int(os.environ.get("HELIX_WORKABLE_ITEMS_BUSY_MS", "30000")))
    except ValueError:
        return 30000


def connect(path: Path) -> sqlite3.Connection:
    path.parent.mkdir(parents=True, exist_ok=True)
    db = sqlite3.connect(path, timeout=busy_ms() / 1000, isolation_level=None)
    db.row_factory = sqlite3.Row
    db.execute("PRAGMA foreign_keys=ON")
    db.execute(f"PRAGMA busy_timeout={busy_ms()}")
    db.execute("PRAGMA journal_mode=WAL")
    db.execute("PRAGMA synchronous=FULL")
    db.executescript(SCHEMA)
    db.execute("INSERT OR IGNORE INTO meta(key,value) VALUES('schema_version',?)", (SCHEMA_VERSION,))
    return db


def connect_readonly(path: Path) -> sqlite3.Connection:
    """Connection for the read-only commands (validate, diff): no DDL, no WAL pragma, no
    inserts, so it works on a read-only database file and never changes it."""
    db = sqlite3.connect(f"file:{path}?mode=ro", uri=True, timeout=busy_ms() / 1000, isolation_level=None)
    db.row_factory = sqlite3.Row
    db.execute("PRAGMA foreign_keys=ON")
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
    sql += f" ORDER BY CAST(substr(atm_id,{len(PROJECT_KEY)+2}) AS INTEGER)"
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
    # firebase_metadata and sync_outbox are part of the registry's state, so they are
    # covered too. They are keyed in only when non-empty, which keeps the hash of every
    # registry that does not use them identical to the pre-existing value (no migration).
    extra = {
        "firebase": [dict(r) for r in db.execute("SELECT * FROM firebase_metadata ORDER BY atm_id")],
        "sync_outbox": [dict(r) for r in db.execute("SELECT * FROM sync_outbox ORDER BY atm_id,revision")],
    }
    payload.update({key: rows for key, rows in extra.items() if rows})
    # The rehash log and the per-item baseline are part of the audited state once present
    # (absent on a registry that never used them, which keeps its existing hash).
    for key in ("rehash_log", "item_hashes", "import_log"):
        row = db.execute("SELECT value FROM meta WHERE key=?", (key,)).fetchone()
        if row:
            payload[key] = row[0]
    encoded = json.dumps(payload, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(encoded).hexdigest()


def checkpoint(db: sqlite3.Connection) -> None:
    db.execute("PRAGMA wal_checkpoint(TRUNCATE)")


def item_snapshot_hashes(db: sqlite3.Connection) -> dict[str, str]:
    """Canonical per-item digest over the item row and every table row keyed by it, so a
    later `rehash` can name exactly which items changed since the last tool write."""
    out = {}
    for item in items(db):
        atm = item["atm_id"]
        parts = {"item": canonical_item(item)}
        for table, order in (("item_history", "sequence"), ("reopen_details", "atm_id"), ("operator_block_details", "atm_id"),
                             ("obsolete_details", "atm_id"), ("firebase_metadata", "atm_id"), ("sync_outbox", "revision")):
            parts[table] = [dict(r) for r in db.execute(f"SELECT * FROM {table} WHERE atm_id=? ORDER BY {order}", (atm,))]
        out[atm] = hashlib.sha256(json.dumps(parts, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    return out


def changed_items(db: sqlite3.Connection) -> list[str] | None:
    """Item ids whose content differs from the last tool-written baseline (None when no
    baseline has been recorded yet, e.g. a registry that predates item_hashes)."""
    row = db.execute("SELECT value FROM meta WHERE key='item_hashes'").fetchone()
    if row is None:
        return None
    try:
        known = json.loads(row[0])
    except ValueError:
        return None
    current = item_snapshot_hashes(db)
    return sorted(atm for atm in set(known) | set(current) if known.get(atm) != current.get(atm))


def update_meta(db: sqlite3.Connection, direction: str) -> None:
    now = utcnow()
    db.execute("INSERT INTO meta(key,value) VALUES('item_hashes',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (json.dumps(item_snapshot_hashes(db), sort_keys=True),))
    db.execute("INSERT INTO meta(key,value) VALUES('last_sync_direction',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (direction,))
    db.execute("INSERT INTO meta(key,value) VALUES('last_sync_timestamp',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (now,))
    db.execute("INSERT INTO meta(key,value) VALUES('integrity_hash',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (integrity(db),))


class RegistryTampered(ValueError):
    """The stored integrity hash does not match the registry content."""


def schema_problem(db: sqlite3.Connection) -> str | None:
    """Triggers and views can rewrite data after the fact without touching the hash."""
    objects = db.execute("SELECT type,name FROM sqlite_master WHERE type IN ('trigger','view') ORDER BY name").fetchall()
    if objects:
        return "unexpected schema objects: " + ", ".join(f"{r[0]} {r[1]}" for r in objects)
    return None


def hash_problem(db: sqlite3.Connection) -> str | None:
    """Every legitimate writer refreshes meta.integrity_hash inside its own transaction
    (update_meta), so a stored hash that differs from the recomputed one means the
    registry was edited behind the tool's back (raw SQL, byte flip, history edit)."""
    schema = schema_problem(db)
    if schema:
        return schema
    stored = db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()
    if stored is None:
        if db.execute("SELECT 1 FROM items LIMIT 1").fetchone():
            return "integrity hash missing: registry has items but meta.integrity_hash is absent"
        return None
    if stored[0] != integrity(db):
        return "integrity hash mismatch: registry content differs from the hash recorded by the last tool write"
    return None


def require_untampered(db: sqlite3.Connection) -> None:
    """Writers call this BEFORE writing: refreshing the hash over tampered content would
    launder the tampering into a valid-looking registry. `rehash` is the one deliberate,
    recorded way to accept an out-of-band edit."""
    problem = hash_problem(db)
    if problem:
        raise RegistryTampered(f"registry tampered; run validate ({problem}); after review use rehash --actor --reason")


TEXT_COLUMNS = ("title", "description", "affected_scope", "reproduction", "acceptance", "closure_criteria",
                "forensic_anchor", "evidence_path", "created_by", "assigned_to")


def validate(db: sqlite3.Connection, check_hash: bool = True) -> list[str]:
    """Validate inside ONE read transaction so the stored hash and the recomputed
    content come from the same snapshot even while another process writes."""
    own = not db.in_transaction
    if own:
        db.execute("BEGIN")
    try:
        return _validate(db, check_hash)
    finally:
        if own:
            db.execute("COMMIT")


def _validate(db: sqlite3.Connection, check_hash: bool) -> list[str]:
    errors: list[str] = []
    seen = set()
    problem = hash_problem(db) if check_hash else schema_problem(db)
    if problem:
        errors.append(problem)
    for row in db.execute("SELECT * FROM item_history"):
        for column in ("event_type", "actor", "on_date", "reason", "evidence_path"):
            if CONTROL_RE.search(row[column] or ""):
                errors.append(f'{row["atm_id"]}: history #{row["sequence"]} {column} contains control characters')
        if any(BOUNDARY_RE.search(row[column] or "") for column in ("reason", "evidence_path")):
            errors.append(f'{row["atm_id"]}: history #{row["sequence"]} carries the field boundary token')
    log = db.execute("SELECT value FROM meta WHERE key='rehash_log'").fetchone()
    if log:
        previous = ""
        for number, line in enumerate(log[0].split("\n"), 1):
            try:
                entry = json.loads(line)
            except ValueError:
                errors.append(f"rehash_log entry {number} is not valid JSON")
                break
            if entry.get("prev_entry") != previous:
                errors.append(f"rehash_log chain broken at entry {number}")
                break
            previous = hashlib.sha256(line.encode()).hexdigest()
    for item in items(db):
        atm = item["atm_id"]
        match = ID_RE.match(atm)
        if not match or int(match.group(1)) < 1 or atm in seen:
            errors.append(f"{atm}: invalid or duplicate {PROJECT_KEY} identifier")
        seen.add(atm)
        for column in TEXT_COLUMNS:
            if CONTROL_RE.search(item[column] or ""):
                errors.append(f"{atm}: {column} contains control characters")
            if len(item[column] or "") > MAX_FIELD_CHARS:
                errors.append(f"{atm}: {column} exceeds {MAX_FIELD_CHARS} characters")
        for column in ("created_by", "assigned_to"):
            if ASSIGN_RE.search(item[column] or ""):
                errors.append(f"{atm}: {column} carries the attribution boundary label")
        if not item["title"].strip():
            errors.append(f"{atm}: missing title")
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
        missing = sorted(set(range(1, max(allocated) + 1)) - set(allocated))
        errors.append(f"{PROJECT_KEY} allocation contains gaps: missing " + ", ".join(item_id(n) for n in missing[:20]))
    violations = db.execute("PRAGMA foreign_key_check").fetchall()
    if violations:
        affected = set()
        for table, rowid, parent, _fk in violations:
            row = db.execute(f"SELECT atm_id FROM {table} WHERE rowid=?", (rowid,)).fetchone()
            affected.add(f"{table}:{row[0] if row else rowid}")
        errors.append("SQLite foreign key check failed: " + ", ".join(sorted(affected)[:20]))
    for row in db.execute("SELECT * FROM operator_block_details"):
        for column in ("what", "why_exhausted_alternatives", "unblock_condition", "who"):
            if BLOCK_LABEL_RE.search(row[column] or ""):
                errors.append(f'{row["atm_id"]}: operator-block {column} carries a field boundary label')
    return errors


# Characters that could end a Markdown line or otherwise let a recorded value forge
# structure when the exports are re-imported: C0/C1 controls (tab excepted), DEL,
# the Unicode line/paragraph separators, and invisible/bidi-override characters
# (soft hyphen, zero-width, LRM/RLM, embeddings and overrides, word joiner, isolates, BOM)
# that can make a recorded line read differently from what it is.
CONTROL_RE = re.compile("[\x00-\x08\x0a-\x1f\x7f-\x9f\u00ad\u034f\u061c\u115f\u180e\u200b-\u200f\u2028-\u202e\u2060-\u2064\u2066-\u2069\u2800\u3164\ufe00-\ufe0f\u180b-\u180d\u180f\ufeff\ufff9-\ufffb\U000e0000-\U000e007f\U000e0100-\U000e01ef]")


# The history line is `... by ACTOR: REASON; evidence: EVIDENCE` and the reopen detail is
# `...; Reason: R; Evidence: E`, both split on this token when re-imported, so a value
# carrying it would shift the field boundary.
BOUNDARY_RE = re.compile(r";\s*evidence\s*:", re.IGNORECASE)
BOUNDARY_FIELDS = {"actor", "reason", "evidence"}
# Same idea for the attribution line (`created by X; assigned to Y`) and the operator-block
# line (`WHAT: .. WHY: Attempted: .. UNBLOCK CONDITION: .. WHO: ..`).
ASSIGN_RE = re.compile(r";\s*assigned\s+to\b", re.IGNORECASE)
BLOCK_LABEL_RE = re.compile(r"\s(?:WHY|UNBLOCK CONDITION|WHO):\s")
FIELD_TOKENS = {"actor": BOUNDARY_RE, "reason": BOUNDARY_RE, "evidence": BOUNDARY_RE,
                "created_by": ASSIGN_RE, "assigned_to": ASSIGN_RE,
                "what": BLOCK_LABEL_RE, "why": BLOCK_LABEL_RE, "unblock": BLOCK_LABEL_RE, "who": BLOCK_LABEL_RE}


MAX_FIELD_CHARS = 32768


def refuse_control_chars(**fields: str | None) -> None:
    """Refuse values that carry line breaks, other control characters, or (for the
    actor/reason/evidence fields) the history field-boundary token."""
    for name, value in fields.items():
        if value and len(value) > MAX_FIELD_CHARS:
            raise ValueError(f"{name} is too long ({len(value)} characters; limit {MAX_FIELD_CHARS})")
        if value and (CONTROL_RE.search(value) or any(unicodedata.category(ch) == "Cf" for ch in value)):
            raise ValueError(f"{name} must not contain line breaks or control characters")
        token = FIELD_TOKENS.get(name)
        if value and token and token.search(value):
            raise ValueError(f"{name} must not contain a field-boundary label ({token.pattern})")


def md_inline(text: str) -> str:
    """Neutralise Markdown image/link/raw-HTML openers in recorded text. Text without
    those sequences is returned unchanged, so existing exports stay byte-identical;
    `md_unescape` (applied by parse_markdown) reverses it for a lossless round trip."""
    return re.sub(r"<(?=[A-Za-z/!?])", r"\\<", text.replace("![", "!\\[").replace("](", "]\\("))


def md_unescape(text: str) -> str:
    return text.replace("\\<", "<").replace("]\\(", "](").replace("!\\[", "![")


def md_escape(text: str) -> str:
    # Backslash first, otherwise `a\|b` would turn into `a\\|b` and the pipe would escape.
    return md_inline(text.replace("\\", "\\\\")).replace("|", "\\|").replace("\n", " ")


def history_lines(db: sqlite3.Connection, atm_id: str, md: bool = False) -> list[str]:
    """Audit trail for one item, formatted identically wherever it is rendered.

    Markdown, HTML and PDF are all generated from this, so an event recorded in the
    registry cannot appear in one derived document and silently vanish from another.
    """
    # Defence in depth: values reach item_history only through writers that refuse
    # control characters, but a row planted by raw SQL must still not be able to start
    # a new line (and with it a forged event or item) in the exported documents.
    flat = (lambda value: md_inline(CONTROL_RE.sub(" ", value or ""))) if md else (lambda value: CONTROL_RE.sub(" ", value or ""))
    return [
        f'- {event["on_date"]} — {event["event_type"]} by {flat(event["actor"])}: {flat(event["reason"])}; evidence: {flat(event["evidence_path"]) or "none recorded"}'
        for event in db.execute("SELECT * FROM item_history WHERE atm_id=? ORDER BY sequence", (atm_id,))
    ]


def long_form_fields(item: dict, db: sqlite3.Connection) -> list[tuple[str, str]]:
    """Every field a derived long-form document must carry for one item.

    Markdown, HTML and PDF all iterate this list, so a value recorded in the
    registry — including lifecycle details and history cross-references — cannot
    reach one format and be dropped by another.
    """
    fields = [
        ("Status", item["status"]),
        ("Type", item["type"]),
        ("Severity", item["severity"]),
        ("Reopens", str(item["reopens_count"])),
    ]
    if item["status"] == "Reopened":
        detail = db.execute("SELECT * FROM reopen_details WHERE atm_id=?", (item["atm_id"],)).fetchone()
        fields.append(("Reopened-Details", f'By: {detail["actor"]}; On: {detail["on_date"]}; Reason: {detail["reason"]}; Evidence: {detail["evidence_path"]}'))
    if item["status"] == "Operator-blocked":
        detail = db.execute("SELECT * FROM operator_block_details WHERE atm_id=?", (item["atm_id"],)).fetchone()
        fields.append(("Operator-Block-Details", f'WHAT: {detail["what"]} WHY: {detail["why_exhausted_alternatives"]} UNBLOCK CONDITION: {detail["unblock_condition"]} WHO: {detail["who"]}'))
    fields += [
        ("Description", item["description"]),
        ("Affected scope", item["affected_scope"]),
        ("Reproduction / current observation", item["reproduction"]),
        ("Acceptance", item["acceptance"]),
        ("Closure criteria", item["closure_criteria"]),
        ("Evidence", item["evidence_path"] or "Not yet available; no pass is claimed."),
        ("Forensic anchor", item["forensic_anchor"]),
        ("Composes with", ", ".join(item["composes_with"]) or "None recorded"),
        ("Attribution", f'created by {item["created_by"] or "UNKNOWN"}; assigned to {item["assigned_to"] or "UNKNOWN"}'),
        ("Created / modified", f'{item["created_at"]} / {item["last_modified"]}'),
    ]
    return fields


def render_item(item: dict, db: sqlite3.Connection) -> str:
    # Same defence as history_lines: stored text can only reach a Markdown line through
    # here, so a value that somehow holds a line break cannot start a heading or event.
    flat = lambda value: md_inline(CONTROL_RE.sub(" ", value))
    lines = [f'## [{item["atm_id"]}] {flat(item["title"])}', ""]
    for label, value in long_form_fields(item, db):
        lines += [f"**{label}:** {flat(value)}", ""]
    events = history_lines(db, item["atm_id"], md=True)
    if events:
        lines += ["**History:**", ""] + events + [""]
    return "\n".join(lines)


def summary_rows(rows: list[dict]) -> str:
    out = ["| ID | Type | Severity | Status | Reopens | Description |", "|---|---|---|---|---:|---|"]
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
            body = ["<table><thead><tr><th>ID</th><th>Type</th><th>Severity</th><th>Status</th><th>Reopens</th><th>Description</th></tr></thead><tbody>"]
            for i in rows:
                body.append("<tr>" + "".join(f"<td>{html.escape(str(v))}</td>" for v in (i["atm_id"], i["type"], i["severity"], i["status"], i["reopens_count"], i["title"])) + "</tr>")
            body.append("</tbody></table>")
        else:
            body = []
            for i in rows:
                body.append(f'<section id="{i["atm_id"]}"><h2>[{i["atm_id"]}] {html.escape(i["title"])}</h2>')
                for label, value in long_form_fields(i, db):
                    body.append(f'<h3>{label}</h3><p>{html.escape(value) or "Not yet available; no pass is claimed."}</p>')
                events = history_lines(db, i["atm_id"])
                if events:
                    body.append("<h3>History</h3><ul>" + "".join(f"<li>{html.escape(event)}</li>" for event in events) + "</ul>")
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
                data = [[Paragraph(f"<b>{x}</b>", small) for x in ("ID", "Type", "Severity", "Status", "Reopens", "Description")]]
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
                    for label, value in long_form_fields(i, db):
                        story.append(Paragraph(f"<b>{label}:</b> {html.escape(value)}", body))
                    events = history_lines(db, i["atm_id"])
                    if events:
                        story.append(Paragraph("<b>History:</b>", body))
                        for event in events:
                            story.append(Paragraph(html.escape(event), small))
            doc.build(story, onFirstPage=footer, onLaterPages=footer)
            paths.append(path)
    return paths


def export(db: sqlite3.Connection, output_dir: Path = DOCS, include_pdf: bool = True) -> list[Path]:
    require_untampered(db)
    output_dir.mkdir(parents=True, exist_ok=True)
    written = []
    for documents in (markdown_documents(db), html_documents(db)):
        for name, content in documents.items():
            path = output_dir / name
            path.write_text(content, encoding="utf-8", newline="\n")
            written.append(path)
    if include_pdf:
        written.extend(pdf_documents(db, output_dir))
    db.execute("BEGIN IMMEDIATE")
    try:
        require_untampered(db)
        update_meta(db, "db-to-docs")
        db.execute("COMMIT")
    except Exception:
        db.execute("ROLLBACK")
        raise
    checkpoint(db)
    return written


HISTORY_LINE_RE = re.compile(r"(?m)^- (\d{4}-\d{2}-\d{2}) — (Opened|Updated|Started|Paused|Unblocked|Reopened|Fixed|Implemented|Completed|Obsolete) by (AI|User): (.*); evidence: (.*)$")
REOPEN_DETAIL_RE = re.compile(r"By: (AI|User); On: (\d{4}-\d{2}-\d{2}); Reason: (.*); Evidence: (.*)$")
BLOCK_DETAIL_RE = re.compile(r"WHAT: (.*) WHY: (Attempted: .*) UNBLOCK CONDITION: (.*) WHO: (.*)$")


def parse_markdown(path: Path, location: str) -> list[dict]:
    text = md_unescape(path.read_text(encoding="utf-8"))
    sections = re.split(rf"(?m)^## \[({PROJECT_KEY}-[0-9]{{3,}})\] ", text)[1:]
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
        history_section = body.partition("\n**History:**\n")[2]
        for candidate_line in re.findall(r"(?m)^- \d{4}-\d{2}-\d{2} — .*$", history_section):
            if not HISTORY_LINE_RE.match(candidate_line):
                raise ValueError(f"{atm}: unrecognised history line: {candidate_line[:100]!r}")
        for line in HISTORY_LINE_RE.findall(history_section):
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


def incoming_items(docs_dir: Path) -> list[dict]:
    incoming = parse_markdown(docs_dir / "Issues.md", "Issues") + parse_markdown(docs_dir / "Fixed.md", "Fixed")
    if not incoming:
        raise ValueError("no workable-item sections found")
    seen = set()
    for item in incoming:
        if item["atm_id"] in seen:
            raise ValueError(f'{item["atm_id"]}: duplicate section in the Markdown')
        seen.add(item["atm_id"])
        if item["_reopened_details"] and not REOPEN_DETAIL_RE.match(item["_reopened_details"]):
            raise ValueError(f'{item["atm_id"]}: malformed Reopened-Details')
        if item["_operator_block_details"] and not BLOCK_DETAIL_RE.match(item["_operator_block_details"]):
            raise ValueError(f'{item["atm_id"]}: malformed Operator-Block-Details')
    return incoming


def plan_import(db: sqlite3.Connection, docs_dir: Path) -> list[str]:
    """Row-level list of what importing would CHANGE on items that already exist
    (`ATM: column 'old' -> 'new'`), including the reopen and operator-block detail
    tables. New items are additions, not changes."""
    changes = []
    for item in incoming_items(docs_dir):
        atm = item["atm_id"]
        row = db.execute("SELECT * FROM items WHERE atm_id=?", (atm,)).fetchone()
        if not row:
            continue
        for column in ITEM_COLUMNS:
            if column != "atm_id" and row[column] != item[column]:
                changes.append(f'{atm}: {column} {str(row[column])[:60]!r} -> {str(item[column])[:60]!r}')
        for table, raw, pattern, columns in (
                ("reopen_details", item["_reopened_details"], REOPEN_DETAIL_RE, ("actor", "on_date", "reason", "evidence_path")),
                ("operator_block_details", item["_operator_block_details"], BLOCK_DETAIL_RE, ("what", "why_exhausted_alternatives", "unblock_condition", "who"))):
            if not raw:
                continue
            new = pattern.match(raw).groups()
            old = db.execute(f"SELECT {','.join(columns)} FROM {table} WHERE atm_id=?", (atm,)).fetchone()
            old = tuple(old) if old else None
            for column, was, now in zip(columns, old or (None,) * len(columns), new):
                if was != now:
                    changes.append(f'{atm}: {table}.{column} {str(was)[:60]!r} -> {str(now)[:60]!r}')
    return changes


def import_markdown(db: sqlite3.Connection, docs_dir: Path, allow_overwrite: bool = False,
                    actor: str | None = None, reason: str | None = None) -> int:
    """Rebuild registry rows from the generated Markdown (disaster recovery).

    The Markdown is untrusted input: the import transaction is validated before it
    commits (control characters, closure rules, ...), evidence on terminal items, reopen
    details and the history of new items must stay inside the repository, and existing
    items (and their reopen/operator-block details) are never silently rewritten:
    changing them needs `allow_overwrite` plus an actor and reason, which are recorded
    with the row-level changes in meta.import_log (covered by the integrity hash)."""
    incoming = incoming_items(docs_dir)
    refuse_control_chars(actor=actor, reason=reason)
    for item in incoming:
        if item["status"] in TERMINAL and item["evidence_path"].strip():
            repo_evidence(item["evidence_path"], must_exist=False)
        if item["_reopened_details"]:
            repo_evidence(REOPEN_DETAIL_RE.match(item["_reopened_details"]).group(4), must_exist=False)
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        changes = plan_import(db, docs_dir)
        if changes and not allow_overwrite:
            raise ValueError("import-md would change existing items (use --allow-overwrite --actor --reason after reviewing): " + "; ".join(changes[:10]))
        if changes:
            if actor not in ("AI", "User") or not (reason or "").strip():
                raise ValueError("import-md --allow-overwrite requires --actor (AI|User) and a non-empty --reason")
            entry = json.dumps({"on": utcnow(), "actor": actor, "reason": reason, "changes": changes}, sort_keys=True)
            log = db.execute("SELECT value FROM meta WHERE key='import_log'").fetchone()
            db.execute("INSERT INTO meta(key,value) VALUES('import_log',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", ((log[0] + "\n" if log else "") + entry,))
        for item in incoming:
            columns = ITEM_COLUMNS
            db.execute(f"INSERT INTO items({','.join(columns)}) VALUES({','.join('?' for _ in columns)}) ON CONFLICT(atm_id) DO UPDATE SET " + ",".join(f"{c}=excluded.{c}" for c in columns if c != "atm_id"), [item[c] for c in columns])
            if not db.execute("SELECT 1 FROM item_history WHERE atm_id=? LIMIT 1", (item["atm_id"],)).fetchone():
                for event in item["_history"]:
                    if event["evidence_path"]:
                        repo_evidence(event["evidence_path"], must_exist=False)
                    db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (item["atm_id"], event["event_type"], event["actor"], event["on_date"], event["reason"], event["evidence_path"]))
            if item["_reopened_details"]:
                match = REOPEN_DETAIL_RE.match(item["_reopened_details"])
                db.execute("INSERT INTO reopen_details(atm_id,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET actor=excluded.actor,on_date=excluded.on_date,reason=excluded.reason,evidence_path=excluded.evidence_path", (item["atm_id"], *match.groups()))
            if item["_operator_block_details"]:
                match = BLOCK_DETAIL_RE.match(item["_operator_block_details"])
                db.execute("INSERT INTO operator_block_details(atm_id,what,why_exhausted_alternatives,unblock_condition,who) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET what=excluded.what,why_exhausted_alternatives=excluded.why_exhausted_alternatives,unblock_condition=excluded.unblock_condition,who=excluded.who", (item["atm_id"], *match.groups()))
        update_meta(db, "md-to-db")
        commit_write(db, base)
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
    require_untampered(db)
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


def outbox_differences(db: sqlite3.Connection, path: Path) -> list[str]:
    """Compare a stored HelixTrack outbox with the one the database would produce now
    (ignoring generated_at). Returns [] when the file is consistent, else descriptive entries."""
    if not path.exists():
        return [f"{path.name}: missing"]
    with tempfile.TemporaryDirectory(prefix="helix-wi-outbox-") as tmp:
        fresh = Path(tmp) / "outbox.json"
        helixtrack_outbox(db, fresh)
        expected = json.loads(fresh.read_text(encoding="utf-8"))
    try:
        actual = json.loads(path.read_text(encoding="utf-8"))
    except ValueError:
        return [f"{path.name}: not valid JSON"]
    expected.pop("generated_at", None)
    if not isinstance(actual, dict):
        return [f"{path.name}: not an object"]
    actual.pop("generated_at", None)
    if actual == expected:
        return []
    have = {r.get("client_key"): r for r in actual.get("records", []) if isinstance(r, dict)}
    want = {r["client_key"]: r for r in expected["records"]}
    out = [f"{path.name}: {key} missing from outbox" for key in sorted(want.keys() - have.keys())]
    out += [f"{path.name}: {key} not in registry" for key in sorted(have.keys() - want.keys())]
    out += [f"{path.name}: {key} differs from registry" for key in sorted(want.keys() & have.keys()) if want[key] != have[key]]
    return out or [f"{path.name}: differs from registry"]


def write_baseline(db: sqlite3.Connection) -> set[str]:
    """First step inside every writer's transaction: refuse tampered content, then record
    the registry's pre-existing validation errors so only NEW ones can fail the commit."""
    require_untampered(db)
    return set(_validate(db, False))


def commit_write(db: sqlite3.Connection, baseline: set[str]) -> None:
    """Last step of every writer: the registry as it would be committed must not carry any
    validation error that was not already present; otherwise the caller rolls back."""
    introduced = [e for e in _validate(db, False) if e not in baseline]
    if introduced:
        raise ValueError("refused; the change would leave the registry invalid: " + "; ".join(introduced[:5]))
    db.execute("COMMIT")


def add_item(db: sqlite3.Connection, args: argparse.Namespace) -> str:
    refuse_control_chars(title=args.title, description=args.description, scope=args.scope,
                         reproduction=args.reproduction, acceptance=args.acceptance, anchor=args.anchor,
                         created_by=args.created_by, assigned_to=args.assigned_to)
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        maximum = db.execute("SELECT COALESCE(MAX(CAST(substr(atm_id,?) AS INTEGER)),0) FROM items", (len(PROJECT_KEY) + 2,)).fetchone()[0]
        atm = item_id(maximum + 1)
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
        commit_write(db, base)
        return atm
    except Exception:
        db.execute("ROLLBACK")
        raise


def history(db: sqlite3.Connection, atm: str, event: str, actor: str, reason: str, evidence: str = "") -> None:
    refuse_control_chars(actor=actor, reason=reason, evidence=evidence)
    db.execute("INSERT INTO item_history(atm_id,event_type,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?,?)", (atm, event, actor, utcnow()[:10], reason, evidence))


def require_item(db: sqlite3.Connection, atm: str) -> dict:
    row = db.execute("SELECT * FROM items WHERE atm_id=?", (atm,)).fetchone()
    if not row:
        raise ValueError(f"{atm}: item not found")
    return row_dict(row)


def close_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    refuse_control_chars(actor=args.actor, reason=args.reason, evidence=args.evidence)
    if not args.evidence.strip():
        raise ValueError("closure requires captured evidence")
    evidence = repo_evidence(args.evidence)
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        item = require_item(db, args.atm_id)
        if item["status"] in TERMINAL:
            raise ValueError(f'{args.atm_id}: already closed ({item["status"]}); reopen it first')
        expected = CLOSED[item["type"]]
        aliases = {"fixed": "Bug", "implemented": "Feature", "completed": "Task"}
        if args.status and aliases[args.status] != item["type"]:
            raise ValueError(f'{item["atm_id"]}: {item["type"]} requires {expected}')
        db.execute("UPDATE items SET status=?,current_location='Fixed',evidence_path=?,last_modified=? WHERE atm_id=?", (expected, evidence, utcnow(), args.atm_id))
        # A closed item is no longer blocked; the block itself stays on record in its history event.
        db.execute("DELETE FROM operator_block_details WHERE atm_id=?", (args.atm_id,))
        history(db, args.atm_id, expected.split(" ")[0], args.actor, args.reason, evidence)
        update_meta(db, "close")
        commit_write(db, base)
    except Exception:
        db.execute("ROLLBACK")
        raise


def reopen_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    refuse_control_chars(actor=args.actor, reason=args.reason, evidence=args.evidence)
    if not args.evidence.strip():
        raise ValueError("reopen requires positive evidence")
    evidence = repo_evidence(args.evidence)
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        item = require_item(db, args.atm_id)
        if item["status"] not in TERMINAL:
            raise ValueError(f'{args.atm_id}: only a closed item can be reopened (status is {item["status"]})')
        db.execute("UPDATE items SET status='Reopened',current_location='Issues',reopens_count=reopens_count+1,last_modified=? WHERE atm_id=?", (utcnow(), args.atm_id))
        db.execute("INSERT INTO reopen_details(atm_id,actor,on_date,reason,evidence_path) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET actor=excluded.actor,on_date=excluded.on_date,reason=excluded.reason,evidence_path=excluded.evidence_path", (args.atm_id, args.actor, utcnow()[:10], args.reason, evidence))
        history(db, args.atm_id, "Reopened", args.actor, args.reason, evidence)
        update_meta(db, "reopen")
        commit_write(db, base)
    except Exception:
        db.execute("ROLLBACK")
        raise


def block_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    refuse_control_chars(what=args.what, why=args.why, unblock=args.unblock, who=args.who)
    if not args.why.startswith("Attempted:"):
        raise ValueError("operator-block exhaustion audit must start with 'Attempted:'")
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        item = require_item(db, args.atm_id)
        if item["status"] in TERMINAL:
            raise ValueError(f'{args.atm_id}: a closed item cannot be operator-blocked; reopen it first')
        db.execute("UPDATE items SET status='Operator-blocked',current_location='Issues',last_modified=? WHERE atm_id=?", (utcnow(), args.atm_id))
        db.execute("INSERT INTO operator_block_details(atm_id,what,why_exhausted_alternatives,unblock_condition,who) VALUES(?,?,?,?,?) ON CONFLICT(atm_id) DO UPDATE SET what=excluded.what,why_exhausted_alternatives=excluded.why_exhausted_alternatives,unblock_condition=excluded.unblock_condition,who=excluded.who", (args.atm_id, args.what, args.why, args.unblock, args.who))
        history(db, args.atm_id, "Updated", "AI", "operator-blocked-after-exhaustion", "")
        update_meta(db, "operator-block")
        commit_write(db, base)
    except Exception:
        db.execute("ROLLBACK")
        raise


def normalise_evidence(part: str) -> str | None:
    """Return the normalised POSIX evidence reference, or None when it is not acceptable.

    Acceptable: a relative path that, with `..` and symlinks resolved, names a regular
    file or a directory strictly inside `evidence/` under EVIDENCE_ROOT. A directory is
    legitimate (an evidence bundle, e.g. a campaign folder). Refused: absolute paths,
    `.`/`..`, bare directories outside evidence/ such as `scripts` or `docs/..`, the
    evidence root itself, symlink escapes, and FIFOs/sockets/devices.
    """
    if not part or Path(part).is_absolute():
        return None
    clean = posixpath.normpath(part)
    if clean in (".", "..") or clean.startswith("../"):
        return None
    root = EVIDENCE_ROOT.resolve()
    allowed = root / "evidence"
    try:
        target = (root / clean).resolve()
        mode = target.stat().st_mode
    except (OSError, RuntimeError):
        return None
    if target == allowed or not target.is_relative_to(allowed):
        return None
    if not (stat.S_ISREG(mode) or stat.S_ISDIR(mode)):
        return None
    return clean


def repo_evidence(value: str, must_exist: bool = True) -> str:
    """Tier-2 evidence (closure, reopen, update): `;`-separated references that must be
    relative and, with `..` and symlinks resolved, stay inside the repository root;
    when `must_exist`, each must be an existing regular file or directory.
    Tier 1 (start/pause) is stricter: see normalise_evidence (inside evidence/ only).
    Returns the normalised `; `-joined references ('' for an empty value)."""
    parts = [part.strip() for part in (value or "").split(";") if part.strip()]
    root = EVIDENCE_ROOT.resolve()
    clean_parts = []
    for part in parts:
        clean = posixpath.normpath(part) if part and not Path(part).is_absolute() else None
        ok = clean is not None and clean not in (".", "..") and not clean.startswith("../")
        if ok:
            try:
                target = (root / clean).resolve()
                ok = target.is_relative_to(root) and target != root and ".git" not in target.relative_to(root).parts and not (must_exist and target.name == DEFAULT_DB.name)
                if ok and must_exist:
                    mode = target.stat().st_mode
                    ok = stat.S_ISREG(mode) or stat.S_ISDIR(mode)
            except (OSError, RuntimeError):
                ok = False
        if not ok:
            raise ValueError(f"evidence must be a relative path inside the repository{' that exists' if must_exist else ''}: {part!r}")
        clean_parts.append(clean)
    return "; ".join(clean_parts)


def transition_item(db: sqlite3.Connection, args: argparse.Namespace) -> tuple[str, str]:
    """Move an item between non-terminal statuses; returns (previous, new)."""
    target = args.target
    refuse_control_chars(actor=args.actor, reason=args.reason, evidence=args.evidence)
    if not (args.reason or "").strip():
        raise ValueError("status transition requires a reason")
    evidence = [part.strip() for part in (args.evidence or "").split(";") if part.strip()]
    if not evidence:
        raise ValueError("status transition requires an evidence/reference path")
    normalised = [normalise_evidence(part) for part in evidence]
    bad = [part for part, clean in zip(evidence, normalised) if clean is None]
    if bad:
        raise ValueError(f"evidence must be an existing file or directory under evidence/ (relative, no escapes): {', '.join(bad)}")
    evidence = normalised
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        item = require_item(db, args.atm_id)
        current = item["status"]
        if target not in TRANSITIONS.get(current, set()):
            raise ValueError(f'{args.atm_id}: illegal transition {current} -> {target}; '
                             "closing, reopening and operator-blocking use their own commands")
        db.execute("UPDATE items SET status=?,current_location='Issues',last_modified=? WHERE atm_id=?", (target, utcnow(), args.atm_id))
        history(db, args.atm_id, TRANSITION_EVENTS[target], args.actor, args.reason, "; ".join(evidence))
        update_meta(db, "start" if target == "In progress" else "pause")
        commit_write(db, base)
        return current, target
    except Exception:
        db.execute("ROLLBACK")
        raise


def unblock_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    """Operator-blocked -> Queued, recorded (actor, reason, evidence under evidence/).
    Without it the only way out of a block would be closing the item."""
    refuse_control_chars(actor=args.actor, reason=args.reason, evidence=args.evidence)
    if not (args.reason or "").strip():
        raise ValueError("unblock requires a reason")
    parts = [part.strip() for part in (args.evidence or "").split(";") if part.strip()]
    if not parts:
        raise ValueError("unblock requires an evidence/reference path")
    clean = [normalise_evidence(part) for part in parts]
    if any(c is None for c in clean):
        raise ValueError("evidence must be an existing file or directory under evidence/ (relative, no escapes)")
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        item = require_item(db, args.atm_id)
        if item["status"] != "Operator-blocked":
            raise ValueError(f'{args.atm_id}: unblock applies to Operator-blocked items (status is {item["status"]})')
        db.execute("UPDATE items SET status='Queued',current_location='Issues',last_modified=? WHERE atm_id=?", (utcnow(), args.atm_id))
        db.execute("DELETE FROM operator_block_details WHERE atm_id=?", (args.atm_id,))
        history(db, args.atm_id, "Unblocked", args.actor, args.reason, "; ".join(clean))
        update_meta(db, "unblock")
        commit_write(db, base)
    except Exception:
        db.execute("ROLLBACK")
        raise


def rehash(db: sqlite3.Connection, args: argparse.Namespace) -> dict:
    """Deliberately accept the current content as the new integrity baseline.

    This is the one sanctioned way past `registry tampered`, and it is NOT a security
    boundary: anyone who can write the database file can also run this command. It is
    an audit trail (the git history of the tracked database is the real control).
    Requirements: a reason; `--confirm` equal to the candidate hash printed by
    `--dry-run`; real change since the baseline; nothing else wrong with the registry.
    The log (meta.rehash_log, append-only, hash-chained, covered by the integrity hash)
    names the items that changed against the last tool-written baseline."""
    refuse_control_chars(actor=args.actor, reason=args.reason)
    if not (args.reason or "").strip():
        raise ValueError("rehash requires a reason")
    db.execute("BEGIN IMMEDIATE")
    try:
        schema = schema_problem(db)
        if schema:
            raise ValueError("rehash refused; " + schema)
        problems = validate(db, check_hash=False)
        if problems:
            raise ValueError("rehash refused; registry has other problems: " + "; ".join(problems[:5]))
        row = db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()
        previous = row[0] if row else ""
        candidate = integrity(db)
        if previous == candidate:
            raise ValueError("rehash refused; the stored hash already matches the content (nothing to accept)")
        changed = changed_items(db)
        if getattr(args, "dry_run", False):
            db.execute("ROLLBACK")
            return {"status": "DRY-RUN", "previous": previous, "candidate": candidate, "changed": changed}
        if not getattr(args, "confirm", None) or args.confirm != candidate:
            raise ValueError(f"rehash refused; --confirm must equal the candidate hash {candidate} (review it with --dry-run)")
        log = db.execute("SELECT value FROM meta WHERE key='rehash_log'").fetchone()
        last = log[0].split("\n")[-1] if log else None
        entry = json.dumps({"on": utcnow(), "actor": args.actor, "reason": args.reason, "previous": previous, "candidate": candidate,
                            "changed": changed, "prev_entry": hashlib.sha256(last.encode()).hexdigest() if last else ""}, sort_keys=True)
        db.execute("INSERT INTO meta(key,value) VALUES('rehash_log',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", ((log[0] + "\n" if log else "") + entry,))
        update_meta(db, "rehash")
        new = db.execute("SELECT value FROM meta WHERE key='integrity_hash'").fetchone()[0]
        db.execute("COMMIT")
        return {"status": "PASS", "previous": previous, "candidate": candidate, "new": new, "changed": changed}
    except Exception:
        if db.in_transaction:
            db.execute("ROLLBACK")
        raise


def update_item(db: sqlite3.Connection, args: argparse.Namespace) -> None:
    changes = {}
    for option, column in (("title", "title"), ("description", "description"), ("scope", "affected_scope"), ("reproduction", "reproduction"), ("acceptance", "acceptance"), ("severity", "severity")):
        value = getattr(args, option)
        if value is not None:
            changes[column] = value
            if column == "acceptance":
                changes["closure_criteria"] = value
    if not changes:
        raise ValueError("update requires at least one field")
    refuse_control_chars(actor=args.actor, reason=args.reason, evidence=args.evidence)
    refuse_control_chars(**{key: value for key, value in changes.items() if isinstance(value, str)})
    evidence = repo_evidence(args.evidence) if (args.evidence or "").strip() else ""
    changes["last_modified"] = utcnow()
    db.execute("BEGIN IMMEDIATE")
    try:
        base = write_baseline(db)
        require_item(db, args.atm_id)
        db.execute("UPDATE items SET " + ",".join(f"{key}=?" for key in changes) + " WHERE atm_id=?", [*changes.values(), args.atm_id])
        history(db, args.atm_id, "Updated", args.actor, args.reason, evidence)
        update_meta(db, "update")
        commit_write(db, base)
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
    import_parser.add_argument("--actor", choices=("AI", "User"), help="Required with --allow-overwrite.")
    import_parser.add_argument("--reason", help="Required with --allow-overwrite.")
    import_parser.add_argument("--allow-overwrite", action="store_true", help="Permit changing items that already exist (review the printed row-level changes first).")
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
    update_parser.add_argument("--evidence", default="", help="Artifact path recorded on the history event for this update")
    unblock_parser = sub.add_parser("unblock", help="Operator-blocked -> Queued (recorded)")
    unblock_parser.add_argument("atm_id")
    unblock_parser.add_argument("--actor", choices=("AI", "User"), required=True)
    unblock_parser.add_argument("--reason", required=True)
    unblock_parser.add_argument("--evidence", required=True, help="Existing file/directory under evidence/, ';'-separated")
    rehash_parser = sub.add_parser("rehash", help="Accept reviewed out-of-band edits as the new integrity baseline (recorded).")
    rehash_parser.add_argument("--actor", choices=("AI", "User"), required=True)
    rehash_parser.add_argument("--reason", required=True)
    rehash_parser.add_argument("--dry-run", action="store_true", help="Print the candidate hash and the items changed since the last tool write; write nothing.")
    rehash_parser.add_argument("--confirm", default=None, help="The candidate hash printed by --dry-run (required to write).")
    for name, helptext in (("start", "Queued/Reopened -> In progress"), ("pause", "In progress -> Queued")):
        transition_parser = sub.add_parser(name, help=helptext)
        transition_parser.add_argument("atm_id")
        transition_parser.add_argument("--actor", choices=("AI", "User"), required=True)
        transition_parser.add_argument("--reason", required=True)
        transition_parser.add_argument("--evidence", required=True, help="Existing file/directory under evidence/ (repo-relative), ';'-separated")
    args = parser.parse_args()
    db = None
    try:
        db = connect_readonly(args.db) if args.command in ("validate", "diff") else connect(args.db)
        if args.command == "validate":
            problems = validate(db)
            if problems:
                print("\n".join(problems), file=sys.stderr)
                return 1
            print(json.dumps({"status": "PASS", "items": len(items(db)), "integrity_hash": integrity(db)}, sort_keys=True))
        elif args.command == "diff":
            differences = diff(db)
            outbox = DOCS / "tracking" / "helixtrack-outbox.json"
            if outbox.exists():
                differences += outbox_differences(db, outbox)
            print(json.dumps({"status": "PASS" if not differences else "FAIL", "differences": differences}, sort_keys=True))
            return bool(differences)
        elif args.command == "export":
            paths = export(db, args.output, not args.no_pdf)
            print(json.dumps({"status": "PASS", "written": [str(p) for p in paths]}, sort_keys=True))
        elif args.command == "import-md":
            changes = plan_import(db, args.input) if args.allow_overwrite else []
            count = import_markdown(db, args.input, allow_overwrite=args.allow_overwrite, actor=args.actor, reason=args.reason)
            print(json.dumps({"status": "PASS", "imported": count, "overwritten": changes}, sort_keys=True))
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
        elif args.command in ("start", "pause"):
            args.target = "In progress" if args.command == "start" else "Queued"
            previous, new = transition_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "from": previous, "lifecycle": new}, sort_keys=True))
        elif args.command == "unblock":
            unblock_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": "Queued"}, sort_keys=True))
        elif args.command == "rehash":
            print(json.dumps(rehash(db, args), sort_keys=True))
        elif args.command == "update":
            update_item(db, args)
            print(json.dumps({"status": "PASS", "atm_id": args.atm_id, "lifecycle": "Updated"}, sort_keys=True))
        return 0
    except sqlite3.OperationalError as error:
        text = str(error).lower()
        if "locked" in text:
            print(f"error: {error}; another process holds the registry write lock, retry later", file=sys.stderr)
            return 2
        if "readonly" in text or "read-only" in text or "unable to open" in text:
            print(f"error: registry database cannot be opened for this command: {error}", file=sys.stderr)
            return 2
        raise
    except ValueError as error:
        print(f"error: {error}", file=sys.stderr)
        return 2
    finally:
        if db is not None:
            with contextlib.suppress(Exception):
                checkpoint(db)
            db.close()


if __name__ == "__main__":
    raise SystemExit(main())
