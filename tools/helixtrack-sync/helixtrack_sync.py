#!/usr/bin/env python3
"""helixtrack_sync: registry <-> Helix-Track/Core ticket synchronisation (HRD-029).

Standard library only.  Talks to Core's unified `POST /do` endpoint using ONLY
actions that exist in Helix-Track/Core Application/internal/handlers
(authenticate, list/read/create/modify with object=project|ticket).

Mapping, conflict policy and the Core quirks this tool works around are
documented in docs/helixtrack-sync.md.  The tool never writes the registry
(docs/workable_items.db, docs/*.md); remote changes are only REPORTED.
Secrets are read from the environment (or a 0600 env file), never printed and
redacted from every recorded request/response.  Redirects are always refused,
and credentials are only ever sent over https or to a loopback host.
NOTE: `pull` (unless --no-probe) issues `modify` requests to read optimistic-lock versions; the
request carries the ticket's CURRENT field values and an impossible version so a conforming
server rejects it with 409 before writing anything (see CoreClient.probe_version).
"""
import argparse
import errno
import fcntl
import hashlib
import http.client
import ipaddress
import json
import os
import re
import sqlite3
import stat
import sys
import unicodedata
import urllib.error
import urllib.parse
import urllib.request

ENV_MAX_BYTES = 64 * 1024            # an env file is a handful of KEY=VALUE lines; anything bigger is refused unread
SCHEMA = "helix.helixtrack-sync.state.v1"
MARKER_PREFIX = "helix-reader-sync/v1"
PROBE_VERSION = 2147483647          # never a real version: yields 409 + current version, mutates nothing
PULL_ISSUES_PROBE_MODIFY = True     # documented behaviour, see module docstring
FIELDS = ("title", "description", "type", "status")
KEY_RE = re.compile(r"^\[(HRD-\d+)\]")
MARKER_RE = re.compile(r"^%s hrd=(HRD-\d+)[ \t]*$" % re.escape(MARKER_PREFIX), re.M)
REDACT_KEYS = {"jwt", "password", "token", "authorization", "secret", "password_hash"}

TYPE_MAP = {"Feature": "feature", "Task": "task", "Bug": "bug"}
STATUS_MAP = {"Queued": "open", "In progress": "in_progress", "Operator-blocked": "blocked",
              "Completed": "done", "Implemented": "done", "Fixed": "done"}
# Core -> registry. `done` is ambiguous (Completed|Implemented|Fixed) and review/testing/closed have no
# registry equivalent: those yield None so a human decides; the tool never guesses.
REVERSE_STATUS = {"open": "Queued", "in_progress": "In progress", "blocked": "Operator-blocked"}

# Outbox payload fields: pushed inside title/description/type/status, or deliberately NOT pushed because
# Core's ticket model has no field for them (see docs/helixtrack-sync.md) or they are server-controlled churn.
PUSHED_PAYLOAD_FIELDS = frozenset({"atm_id", "title", "type", "status", "description", "severity", "reopens_count",
                                   "affected_scope", "reproduction", "acceptance", "evidence_path"})
UNPUSHED_PAYLOAD_FIELDS = frozenset({"closure_criteria", "composes_with", "forensic_anchor", "assigned_to",
                                     "created_by", "created_at", "last_modified", "current_location"})


# ---------------------------------------------------------------- errors
MAX_RESPONSE_BYTES = 8 * 1024 * 1024
MSG_LIMIT = 300


def safe_text(value, limit=MSG_LIMIT):
    """Server-/tree-derived text made safe for a terminal, a log and a report: every character of Unicode
    category C* (controls incl. ESC, BEL, NUL, CR, C1 such as CSI 0x9b, format chars incl. bidi overrides and
    zero-width, surrogates, private use, unassigned) and the line/paragraph separators is escaped, then the text is
    capped.  Newlines are escaped too, so one message is always one line."""
    try:
        text = value if isinstance(value, str) else str(value)
    except Exception:
        text = "<unprintable %s>" % type(value).__name__
    out = []
    for ch in text:
        cat = unicodedata.category(ch)
        if cat[0] == "C" or cat in ("Zl", "Zp") or (ch != " " and cat == "Zs"):
            cp = ord(ch)
            out.append("\\x%02x" % cp if cp < 0x100 else "\\u%04x" % cp if cp < 0x10000 else "\\U%08x" % cp)
        else:
            out.append(ch)
    res = "".join(out)
    return res if len(res) <= limit else res[:limit] + "...[truncated %d]" % (len(res) - limit)


def emit_warning(text):
    print("warning: %s" % safe_text(text), file=sys.stderr)


class SyncError(Exception):
    def __str__(self):
        return safe_text(Exception.__str__(self))


class MappingError(SyncError):
    pass


class CoreError(SyncError):
    def __init__(self, message, status=None, code=None):
        super().__init__(message)
        self.status, self.code = status, code


class AuthError(CoreError):
    pass


class ForbiddenError(CoreError):
    pass


class NotFoundError(CoreError):
    pass


class ValidationError(CoreError):
    pass


class ConflictError(CoreError):
    pass


class VersionConflict(ConflictError):
    def __init__(self, message, current=None, status=409, code=3006):
        super().__init__(message, status, code)
        self.current = current


class ServerError(CoreError):
    pass


class TransportError(CoreError):
    pass


class ShapeError(CoreError):
    """The server answered, but not in the shape Core documents."""


class RedirectRefused(CoreError):
    pass


class ReadbackMismatch(SyncError):
    pass


class LockError(SyncError):
    pass


# ---------------------------------------------------------------- redaction / recording
def redact(obj):
    if isinstance(obj, dict):
        return {k: ("[REDACTED]" if k.lower() in REDACT_KEYS else redact(v)) for k, v in obj.items()}
    if isinstance(obj, list):
        return [redact(v) for v in obj]
    return obj


class Recorder:
    """Keeps (and optionally appends to a JSONL file) every request/response, secrets redacted."""

    def __init__(self, path=None):
        self.path, self.entries = path, []

    def add(self, method, path, headers, body, status, response):
        entry = {"seq": len(self.entries) + 1, "method": method, "path": path,
                 "request_headers": redact(headers or {}), "request": redact(body),
                 "http_status": status, "response": redact(response)}
        self.entries.append(entry)
        if self.path:
            fd = os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
            with os.fdopen(fd, "a", encoding="utf-8") as fh:
                fh.write(json.dumps(entry, ensure_ascii=False, sort_keys=True) + "\n")


# ---------------------------------------------------------------- url guard
def _is_loopback(host):
    host = host.lower()
    if host == "localhost":
        return True
    try:
        return ipaddress.ip_address(host).is_loopback
    except ValueError:
        return False


def check_base_url(base):
    """Credentials may only travel over https, or plain http to a loopback host.  Scheme is compared
    case-insensitively on the PARSED url (HTTP://... is plain http too)."""
    # Parser differential: urlparse() and the http client disagree about userinfo/backslashes
    # (http://evil@127.0.0.1, http://evil.com\\@127.0.0.1), so such a url could pass this guard while the
    # connection goes elsewhere.  Refuse them outright, together with fragments/queries and any
    # whitespace, control or non-ASCII character.  Nothing here echoes the (possibly secret) url.
    if not isinstance(base, str) or any(c in base for c in "\\@#?") or \
            any(ord(c) <= 0x20 or ord(c) >= 0x7f for c in base):
        raise SyncError("refusing to send credentials: base url is malformed or carries userinfo, backslash, "
                        "fragment, query, whitespace or non-ASCII characters")
    parsed = urllib.parse.urlparse(base or "")
    try:
        parsed.port
    except ValueError:
        raise SyncError("refusing to send credentials: base url has an invalid port")
    scheme, host = parsed.scheme.lower(), parsed.hostname or ""
    if scheme == "https" and host:
        return
    if scheme == "http" and host and _is_loopback(host):
        return
    raise SyncError("refusing to send credentials to %r: only https, or http to a loopback host, is allowed"
                    % (parsed.scheme + "://" + host if host else "<invalid url>"))


# ---------------------------------------------------------------- mapping
def _status_key(registry_status):
    return registry_status.split(" (")[0].strip()


def map_status(registry_status):
    try:
        return STATUS_MAP[_status_key(registry_status)]
    except KeyError:
        raise MappingError("no Core status mapping for registry status %r" % registry_status)


def map_type(registry_type):
    try:
        return TYPE_MAP[registry_type]
    except KeyError:
        raise MappingError("no Core ticket type mapping for registry type %r" % registry_type)


def record_key(record):
    return record["client_key"]


def build_fields(record, history=None):
    """Registry outbox record -> the four Core fields this tool owns."""
    p, key = record["payload"], record["client_key"]
    lines = [p.get("description", "").rstrip(), "",
             "Severity: %s" % p.get("severity", ""),
             "Registry status: %s" % p.get("status", ""),
             "Reopens: %s" % p.get("reopens_count", 0),
             "Affected scope: %s" % p.get("affected_scope", ""),
             "Reproduction: %s" % p.get("reproduction", ""),
             "Acceptance: %s" % p.get("acceptance", ""),
             "Evidence: %s" % p.get("evidence_path", "")]
    if history:
        lines += ["History:"] + ["- %s %s (%s): %s" % (h["on_date"], h["event_type"], h["actor"], h["reason"])
                                 for h in history]
    lines += ["", "---", "%s hrd=%s" % (MARKER_PREFIX, key)]
    return {"title": "[%s] %s" % (key, p["title"]), "description": "\n".join(lines),
            "type": map_type(p["type"]), "status": map_status(p["status"])}


def fields_hash(fields):
    canon = json.dumps({k: fields.get(k) for k in FIELDS}, sort_keys=True, ensure_ascii=False)
    return hashlib.sha256(canon.encode("utf-8")).hexdigest()


def remote_fields(ticket):
    return {k: ticket.get(k) for k in FIELDS}


# ---------------------------------------------------------------- client
class CoreClient:
    def __init__(self, transport, username, password, recorder=None):
        self._t, self._u, self._p, self._rec = transport, username, password, recorder
        self._jwt = None

    def _raw(self, body, auth=True):
        headers = {"Content-Type": "application/json"}
        if auth and self._jwt:
            body = dict(body, jwt=self._jwt)
            headers["Authorization"] = "Bearer " + self._jwt
        try:
            status, resp = self._t("POST", "/do", headers, body)
        except (OSError, urllib.error.URLError, http.client.HTTPException, ValueError, RecursionError,
                MemoryError) as exc:
            detail = (": " + safe_text(exc, 120)) if isinstance(exc, (ValueError, RecursionError)) else ""
            raise TransportError("transport failure: %s%s" % (type(exc).__name__, detail))
        if self._rec:
            self._rec.add("POST", "/do", headers, body, status, resp)
        if 300 <= status < 400:
            raise RedirectRefused("server answered HTTP %d (redirect); redirects are refused so credentials "
                                  "are never forwarded" % status, status)
        return status, resp

    def _do(self, body, auth=True, _relogin=True):
        status, resp = self._raw(body, auth)
        resp = resp if isinstance(resp, dict) else {}
        code, msg = resp.get("errorCode"), resp.get("errorMessage")
        if not isinstance(msg, str) or not msg:
            msg = "HTTP %s" % status
        if not isinstance(code, int) or isinstance(code, bool):
            code = None
        if status == 200 and code == -1:
            data = resp.get("data")
            if data is None:
                return {}
            if not isinstance(data, dict):
                raise ShapeError("response data is %s, expected an object" % type(data).__name__, status, code)
            return data
        if status == 401:
            if auth and _relogin and code in (1002, 1003):
                self.login()
                return self._do(body, auth, _relogin=False)
            raise AuthError(msg, status, code)
        if status == 403:
            raise ForbiddenError(msg, status, code)
        if status == 404:
            raise NotFoundError(msg, status, code)
        if status == 409:
            if code == 3006:
                loc = resp.get("errorMessageLocalised")
                m = re.search(r"Current version:\s*(\d+)", loc if isinstance(loc, str) else "")
                raise VersionConflict(msg, int(m.group(1)) if m else None)
            raise ConflictError(msg, status, code)
        if status == 400:
            raise ValidationError(msg, status, code)
        if status >= 500 or not resp:
            raise ServerError(msg, status, code)
        raise CoreError(msg, status, code)

    def login(self):
        data = self._do({"action": "authenticate", "data": {"username": self._u, "password": self._p}}, auth=False)
        if not isinstance(data.get("token"), str) or not data["token"]:
            raise AuthError("authenticate returned no token (is Core running with local auth?)")
        self._jwt = data["token"]

    @staticmethod
    def _obj(value, what):
        if not isinstance(value, dict):
            raise ShapeError("%s is %s, expected an object" % (what, type(value).__name__))
        return value

    @staticmethod
    def _id(value, what):
        if isinstance(value, bool) or not isinstance(value, (str, int)) or value == "":
            raise ShapeError("%s has a missing or invalid id" % what)
        return value

    @classmethod
    def _ticket(cls, value, what="ticket"):
        """Every server field the tool later uses is validated here: id (str|int), title/description/status/
        type (str or absent), version (int or absent)."""
        t = cls._obj(value, what)
        cls._id(t.get("id"), what)
        for f in ("title", "description", "status", "type", "modified", "created"):
            if t.get(f) is not None and not isinstance(t[f], (str, int)):
                raise ShapeError("%s field %s has an invalid type" % (what, f))
        for f in ("title", "description", "status", "type"):
            if t.get(f) is not None and not isinstance(t[f], str):
                raise ShapeError("%s field %s is not text" % (what, f))
        if t.get("version") is not None and (isinstance(t["version"], bool) or not isinstance(t["version"], int)):
            raise ShapeError("%s field version is not an integer" % what)
        return t

    @classmethod
    def _items(cls, data, what, item_check):
        items = data.get("items", [])
        if items is None:
            return []
        if not isinstance(items, list):
            raise ShapeError("%s items is %s, expected a list" % (what, type(items).__name__))
        return [item_check(i) for i in items]

    def ensure_project(self, key, name, create=True):
        def chk(p):
            self._obj(p, "project")
            if p.get("identifier") is not None and not isinstance(p["identifier"], str):
                raise ShapeError("project identifier is not text")
            return p
        for p in self._items(self._do({"action": "list", "object": "project", "data": {}}), "project list", chk):
            if p.get("identifier") == key:
                return self._id(p.get("id"), "project")
        if not create:
            return None
        data = self._do({"action": "create", "object": "project",
                         "data": {"key": key, "name": name, "description": "Helix Reader registry mirror"}})
        return self._id(self._obj(data.get("project"), "created project").get("id"), "created project")

    def list_tickets(self, project_id):
        data = self._do({"action": "list", "object": "ticket", "data": {"project_id": project_id}})
        return self._items(data, "ticket list", self._ticket)

    def read_ticket(self, ticket_id):
        return self._ticket(self._do({"action": "read", "object": "ticket", "data": {"id": ticket_id}}).get("ticket"))

    def create_ticket(self, project_id, fields):
        data = self._do({"action": "create", "object": "ticket", "data": {
            "project_id": project_id, "title": fields["title"], "description": fields["description"],
            "type": fields["type"]}})
        return self._id(self._obj(data.get("ticket"), "created ticket").get("id"), "created ticket")

    def modify_ticket(self, ticket_id, version, **fields):
        data = {"id": ticket_id, "version": version}
        data.update({k: v for k, v in fields.items() if k in ("title", "description", "status")})
        t = self._obj(self._do({"action": "modify", "object": "ticket", "data": data}).get("ticket"),
                      "modified ticket")
        v = t.get("version")
        if isinstance(v, bool) or not isinstance(v, int):
            raise ShapeError("modified ticket has no integer version")
        return v

    def probe_version(self, ticket_id):
        """Core's read/list omit `version`.  A modify carrying an impossible version is rejected with 409 BEFORE
        any write and reports the current version, so it is a side-effect-free version read on a conforming
        server.  The probe carries the ticket's CURRENT title/description/status, so even a server that ignores
        `version` and applies it changes no content (it is then reported as ServerError: locking not enforced)."""
        cur = self.read_ticket(ticket_id)
        same = {k: cur[k] for k in ("title", "description", "status") if cur.get(k)}
        try:
            self.modify_ticket(ticket_id, PROBE_VERSION, **same)
        except VersionConflict as exc:
            if exc.current is None:
                raise ServerError("version probe: conflict without current version")
            return exc.current
        raise ServerError("version probe was accepted: the server does not enforce optimistic locking "
                          "(a no-op write with identical content was applied)")


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None                      # every 3xx surfaces as an HTTPError; nothing is ever re-sent


def _load_body(raw):
    if len(raw) > MAX_RESPONSE_BYTES:
        raise ValueError("response body exceeds %d bytes" % MAX_RESPONSE_BYTES)
    try:
        return json.loads(raw or b"null")
    except RecursionError:
        raise ValueError("response nesting too deep")


def urllib_transport(base_url, timeout=30):
    check_base_url(base_url)
    host = urllib.parse.urlparse(base_url).hostname or ""
    handlers = [_NoRedirect()]
    if _is_loopback(host):
        handlers.append(urllib.request.ProxyHandler({}))
    opener = urllib.request.build_opener(*handlers)

    def transport(method, path, headers, body):
        req = urllib.request.Request(base_url.rstrip("/") + path, json.dumps(body).encode("utf-8"),
                                     headers, method=method)
        try:
            with opener.open(req, timeout=timeout) as resp:
                return resp.status, _load_body(resp.read(MAX_RESPONSE_BYTES + 1))
        except urllib.error.HTTPError as exc:
            try:
                raw = exc.read(MAX_RESPONSE_BYTES + 1)
            finally:
                exc.close()
            try:
                return exc.code, _load_body(raw)
            except ValueError:
                return exc.code, None
    return transport


# ---------------------------------------------------------------- report
class Report:
    def __init__(self, mode, dry_run=False):
        self.mode, self.dry_run = mode, dry_run
        self.actions, self.orphans, self.proposals = [], [], []
        self.error, self.aborted = None, False
        self.warnings = []

    def add(self, key, action, **extra):
        self.actions.append(dict(key=key, action=action, **extra))

    def fail(self, key, exc):
        self.add(key, "error", error=type(exc).__name__, message=str(exc))
        self.error = {"key": key, "type": type(exc).__name__, "message": str(exc)}
        self.aborted = True

    @property
    def has_conflicts(self):
        return any(a["action"] in ("conflict", "unpushable", "duplicate", "remote-deleted", "unmapped") for a in self.actions)

    @property
    def has_drift(self):
        return any(a["action"] not in ("in-sync", "noop") for a in self.actions)

    def counts(self):
        out = {}
        for a in self.actions:
            out[a["action"]] = out.get(a["action"], 0) + 1
        return out

    def to_dict(self):
        return {"mode": self.mode, "dry_run": self.dry_run, "counts": self.counts(), "orphans": self.orphans,
                "has_conflicts": self.has_conflicts, "has_drift": self.has_drift, "aborted": self.aborted,
                "error": self.error, "proposals": self.proposals, "actions": self.actions,
                "warnings": self.warnings}


# ---------------------------------------------------------------- matching
def _claims(ticket):
    keys = set()
    m = KEY_RE.match(ticket.get("title") or "")
    if m:
        keys.add(m.group(1))
    keys.update(MARKER_RE.findall(ticket.get("description") or ""))
    return keys


def _resolve(client, tickets, keys, items):
    """Match registry keys to remote tickets: state ticket_id first, then marker line, then title prefix.
    Returns (matched, problems, orphans).  problems[key] = ("duplicate"|"remote-deleted", extra)."""
    by_id = {t["id"]: t for t in tickets}
    claims = {}
    for t in tickets:
        for k in _claims(t):
            claims.setdefault(k, []).append(t)
    matched, problems = {}, {}
    for key in keys:
        cands = claims.get(key, [])
        known = (items.get(key) or {}).get("ticket_id")
        if known:
            t = by_id.get(known)
            if t is None:
                try:
                    t = client.read_ticket(known)          # confirm: absent from list AND unreadable = deleted
                except NotFoundError:
                    t = None
            if t is None:
                problems[key] = ("remote-deleted", {"ticket_id": known,
                                                    "claimants": sorted(c["id"] for c in cands),
                                                    "reason": "ticket recorded in the sync state no longer exists remotely"})
                continue
            extras = [c for c in cands if c["id"] != t["id"]]
            if extras:
                problems[key] = ("duplicate", {"ticket_ids": [t["id"]] + sorted(c["id"] for c in extras),
                                               "reason": "other tickets claim this key"})
            else:
                matched[key] = t
        elif len(cands) == 1:
            matched[key] = cands[0]
        elif len(cands) > 1:
            problems[key] = ("duplicate", {"ticket_ids": sorted(c["id"] for c in cands),
                                           "reason": "several tickets claim this key"})
    orphans = sorted(k for k in claims if k not in set(keys))
    return matched, problems, orphans


# ---------------------------------------------------------------- sync core
def _state_items(state):
    state.setdefault("schema", SCHEMA)
    return state.setdefault("items", {})


def _diff_fields(desired, remote):
    return [f for f in FIELDS if desired.get(f) != remote.get(f)]


def _remember(state, key, ticket_id, number, fields, version):
    _state_items(state)[key] = {"ticket_id": ticket_id, "ticket_number": number,
                                "base_hash": fields_hash(fields), "version": version}


def _verify(client, ticket_id, desired):
    got = client.read_ticket(ticket_id)
    bad = _diff_fields(desired, remote_fields(got))
    if bad:
        raise ReadbackMismatch("read-back differs from what was pushed in fields %s (server accepted the write "
                               "but did not apply it)" % ",".join(bad))
    return got


def _apply_update(client, ticket_id, desired, base_hash, overwrite):
    """Modify with optimistic locking.  Returns (outcome, version, retries, detail)."""
    retries = 0
    while True:
        version = client.probe_version(ticket_id)
        try:
            new_version = client.modify_ticket(ticket_id, version, title=desired["title"],
                                               description=desired["description"], status=desired["status"])
            return "ok", new_version, retries, None
        except VersionConflict as exc:
            current = client.read_ticket(ticket_id)
            rh = fields_hash(remote_fields(current))
            if rh == fields_hash(desired):
                return "ok", exc.current, retries, "already-applied"
            if not overwrite and base_hash is not None and rh != base_hash:
                return "conflict", exc.current, retries, "remote content changed during update"
            if retries >= 1:
                return "conflict", exc.current, retries, "version kept moving"
            retries += 1


def _create(client, project_id, desired, state, key, checkpoint):
    tid = client.create_ticket(project_id, desired)
    # baseline BEFORE the follow-up status modify: if that modify fails, the next run sees
    # remote == baseline (registry ahead) instead of a false conflict.
    _remember(state, key, tid, None, dict(desired, status="open"), 1)
    if checkpoint:
        checkpoint(state)
    version = 1
    if desired["status"] != "open":
        version = client.modify_ticket(tid, client.probe_version(tid), status=desired["status"])
    got = _verify(client, tid, desired)
    _remember(state, key, tid, got.get("ticket_number"), desired, version)
    return tid


def _push_one(client, project_id, rec, state, matched, problems, dry_run, overwrite_remote, history, checkpoint,
              report):
    items = _state_items(state)
    key = record_key(rec)
    desired = build_fields(rec, (history or {}).get(key))
    dh = fields_hash(desired)
    create_action, remote = "create", matched.get(key)
    if key in problems:
        kind, extra = problems[key]
        if kind == "duplicate" or not overwrite_remote:
            report.add(key, kind, **extra)
            return
        create_action, remote = "recreate", None          # operator explicitly accepted re-creation
    if remote is None:
        if dry_run:
            report.add(key, create_action)
            return
        tid = _create(client, project_id, desired, state, key, checkpoint)
        report.add(key, create_action, ticket_id=tid)
        return
    rf = remote_fields(remote)
    rh = fields_hash(rf)
    base = (items.get(key) or {}).get("base_hash")
    if base is None and rh == fields_hash(dict(desired, status="open")):
        base = rh                                          # interrupted create whose state was lost: reconcile
    if rh == dh:
        if not dry_run and (items.get(key) or {}).get("base_hash") != dh:
            _remember(state, key, remote["id"], remote.get("ticket_number"), desired,
                      client.probe_version(remote["id"]))
        report.add(key, "noop", ticket_id=remote["id"])
        return
    if rf["type"] != desired["type"]:
        report.add(key, "unpushable", ticket_id=remote["id"], reason="Core modify cannot change ticket type",
                   registry_type=desired["type"], remote_type=rf["type"])
        return
    remote_edited = base is None or rh != base
    if remote_edited and not overwrite_remote:
        report.add(key, "conflict", ticket_id=remote["id"], changed_fields=_diff_fields(desired, rf),
                   reason="remote differs from registry and from the last pushed state; "
                          "not overwritten (use --overwrite-remote after review)")
        return
    action = "overwrite" if remote_edited else "update"
    if dry_run:
        report.add(key, action, ticket_id=remote["id"], changed_fields=_diff_fields(desired, rf))
        return
    outcome, version, retries, detail = _apply_update(client, remote["id"], desired, base, overwrite_remote)
    if outcome == "conflict":
        report.add(key, "conflict", ticket_id=remote["id"], reason=detail, retries=retries)
        return
    got = _verify(client, remote["id"], desired)
    _remember(state, key, remote["id"], got.get("ticket_number"), desired, version)
    extra = {"retries": retries} if retries else {}
    report.add(key, action, ticket_id=remote["id"], changed_fields=_diff_fields(desired, rf), **extra)


def push(client, project_id, records, state, dry_run=False, overwrite_remote=False, history=None,
         checkpoint=None):
    report = Report("push", dry_run)
    items = _state_items(state)
    tickets = [] if project_id is None else client.list_tickets(project_id)
    matched, problems, report.orphans = _resolve(client, tickets, [record_key(r) for r in records], items)
    for rec in records:
        try:
            _push_one(client, project_id, rec, state, matched, problems, dry_run, overwrite_remote, history,
                      None if dry_run else checkpoint, report)
        except MappingError as exc:                      # this record cannot be mapped: skip it, keep the batch
            report.add(record_key(rec), "unmapped", reason=str(exc))
            continue
        except (CoreError, ReadbackMismatch) as exc:
            report.fail(record_key(rec), exc)
            if checkpoint and not dry_run:
                checkpoint(state)
            break
        if checkpoint and not dry_run:
            checkpoint(state)
    return report


def pull(client, project_id, records, state, history=None, probe=True):
    report = Report("pull")
    items = _state_items(state)
    tickets = client.list_tickets(project_id)
    matched, problems, report.orphans = _resolve(client, tickets, [record_key(r) for r in records], items)
    for rec in records:
        key = record_key(rec)
        try:
            desired = build_fields(rec, (history or {}).get(key))
            dh = fields_hash(desired)
            if key in problems:
                kind, extra = problems[key]
                report.add(key, kind, **extra)
                continue
            remote = matched.get(key)
            if remote is None:
                report.add(key, "missing-remote")
                continue
            rf = remote_fields(remote)
            rh = fields_hash(rf)
            st = items.get(key) or {}
            base = st.get("base_hash")
            info = {"ticket_id": remote["id"], "remote_modified": remote.get("modified")}
            if rh == dh:
                report.add(key, "in-sync", **info)
                continue
            changed = _diff_fields(desired, rf)
            if base is None:
                action = "diverged-no-baseline"
            elif rh == base:
                action = "registry-ahead"
            elif dh == base:
                action = "remote-edit"
            else:
                action = "conflict"
            extra = {}
            if action in ("remote-edit", "conflict", "diverged-no-baseline"):
                extra = {"remote_version": client.probe_version(remote["id"]) if probe else None,
                         "base_version": st.get("version")}
                if "status" in changed:
                    report.proposals.append({"key": key, "remote_status": rf["status"],
                                             "registry_status": REVERSE_STATUS.get(rf["status"]),
                                             "note": "proposal only; the tool never edits the registry"})
            report.add(key, action, changed_fields=changed, **info, **extra)
        except MappingError as exc:
            report.add(key, "unmapped", reason=str(exc))
        except (CoreError, ReadbackMismatch) as exc:
            report.fail(key, exc)
            break
    return report


def compare(client, project_id, records, history=None):
    tickets = client.list_tickets(project_id)
    matched, _, _ = _resolve(client, tickets, [record_key(r) for r in records], {})
    rows = []
    for rec in records:
        key = record_key(rec)
        desired = build_fields(rec, (history or {}).get(key))
        remote = matched.get(key)
        if remote is None:
            rows.append({"key": key, "match": False, "missing": True, "fields": {f: False for f in FIELDS}})
            continue
        rf = remote_fields(remote)
        flds = {f: desired[f] == rf[f] for f in FIELDS}
        rows.append({"key": key, "match": all(flds.values()), "ticket_id": remote["id"], "fields": flds})
    return rows


# ---------------------------------------------------------------- CLI
def load_env_file(path, environ):
    """Open first (O_NONBLOCK, so a FIFO cannot hang), then fstat THAT descriptor: it must be a regular file with
    no group/other access.  Nothing is read from a refused file and `environ` is only touched after the whole
    file parsed."""
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK)
    except OSError as exc:
        raise SyncError("cannot open env file %s: %s" % (path, exc.strerror))
    try:
        st = os.fstat(fd)
        if not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) & 0o077:
            raise SyncError("env file %s must be a regular file not accessible by group/others (chmod 600)" % path)
        try:
            with os.fdopen(fd, "rb") as fh:
                fd = None
                data = fh.read(ENV_MAX_BYTES + 1)             # never reads more than cap+1 bytes
            if len(data) > ENV_MAX_BYTES:
                raise SyncError("env file %s is larger than %d bytes; refusing to read it" % (path, ENV_MAX_BYTES))
            text = data.decode("utf-8")
        except (OSError, ValueError) as exc:
            raise SyncError("env file %s is unreadable (%s)" % (path, type(exc).__name__))
    finally:
        if fd is not None:
            os.close(fd)
    found = {}
    for line in text.splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            found.setdefault(k.strip(), v.strip())
    for k, v in found.items():
        environ.setdefault(k, v)


def _query_history(uri):
    con = sqlite3.connect(uri, uri=True)
    try:
        out = {}
        for atm, ev, actor, on, reason in con.execute(
                "SELECT atm_id,event_type,actor,on_date,reason FROM item_history ORDER BY sequence"):
            out.setdefault(atm, []).append({"event_type": ev, "actor": actor, "on_date": on, "reason": reason})
        return out
    finally:
        con.close()


def load_history(db_path, warnings=None):
    """Registry history, read-only, never creating or modifying any file (a missing DB is a SyncError).

    The registry DB is WAL-mode, and `immutable=1` ignores the -wal, so a DB whose writer has not
    checkpointed yet would give a stale history.  Therefore: when a non-empty `<db>-wal` AND its `-shm`
    exist, open `mode=ro` (honours the WAL; creates nothing because both files are already there).
    Otherwise, or when that open fails, fall back to `mode=ro&immutable=1`; if a non-empty -wal exists the
    fallback may be stale and a WARNING is appended to `warnings` (and shown in the report)."""
    ap = os.path.abspath(db_path)
    q = urllib.parse.quote(ap)
    try:
        wal_live = os.path.getsize(ap + "-wal") > 0
    except OSError:
        wal_live = False
    try:
        if wal_live and os.path.exists(ap + "-shm"):
            try:
                return _query_history("file:%s?mode=ro" % q)
            except sqlite3.Error as exc:
                reason = "the WAL-aware read-only open failed (%s)" % type(exc).__name__
        elif wal_live:
            reason = "%s-shm is absent so the WAL cannot be read without creating files" % os.path.basename(ap)
        else:
            reason = None
        out = _query_history("file:%s?mode=ro&immutable=1" % q)
        if reason and warnings is not None:
            warnings.append("registry history may be stale: %s-wal holds data but %s; history was read from the "
                            "main DB file only (immutable)" % (os.path.basename(ap), reason))
        return out
    except sqlite3.Error as exc:
        raise SyncError("registry history DB %s is unreadable (%s)" % (db_path, type(exc).__name__))


def _read_json_file(path, what):
    try:
        with open(path, encoding="utf-8") as fh:
            return json.load(fh)
    except (OSError, ValueError, RecursionError) as exc:      # JSONDecodeError/UnicodeDecodeError are ValueErrors
        raise SyncError("%s %s is missing, unreadable or not valid JSON (%s)" % (what, path, type(exc).__name__))


def load_outbox(path):
    """Validate the registry outbox BEFORE any request is made.  Returns the records list."""
    doc = _read_json_file(path, "outbox")
    records = doc.get("records") if isinstance(doc, dict) else None
    if not isinstance(records, list) or not records:
        raise SyncError("no records in %s (expected an object with a non-empty `records` list)" % path)
    seen = set()
    for i, r in enumerate(records):
        if not isinstance(r, dict):
            raise SyncError("outbox record #%d is not an object" % i)
        key = r.get("client_key")
        if not isinstance(key, str) or not re.fullmatch(r"HRD-\d+", key):
            raise SyncError("outbox record #%d has no valid client_key" % i)
        if key in seen:
            raise SyncError("outbox lists %s twice" % key)
        seen.add(key)
        payload = r.get("payload")
        if not isinstance(payload, dict):
            raise SyncError("outbox record %s has no payload object" % key)
        for f in ("title", "type", "status"):
            if not isinstance(payload.get(f), str) or not payload[f]:
                raise SyncError("outbox record %s payload.%s must be a non-empty string" % (key, f))
        if not isinstance(payload.get("description", ""), str):
            raise SyncError("outbox record %s payload.description must be a string" % key)
    return records


def load_state(path):
    """Validate the sync state BEFORE any request is made.  A missing file is a fresh state; anything
    else must be an object whose `items` is an object of objects (null/str/list entries are corrupt)."""
    if not path or not os.path.exists(path):
        return {}
    doc = _read_json_file(path, "state file")
    if not isinstance(doc, dict):
        raise SyncError("state file %s must be a JSON object" % path)
    if "schema" in doc and doc["schema"] != SCHEMA:
        raise SyncError("state file %s has unknown schema %r" % (path, doc["schema"]))
    items = doc.get("items", {})
    if not isinstance(items, dict):
        raise SyncError("state file %s: `items` must be an object" % path)
    for key, item in items.items():
        if not isinstance(item, dict):
            raise SyncError("state file %s: item %s must be an object" % (path, key))
        if "ticket_id" in item and not (isinstance(item["ticket_id"], str) and item["ticket_id"]):
            raise SyncError("state file %s: item %s has an invalid ticket_id" % (path, key))
        if item.get("base_hash") is not None and not isinstance(item["base_hash"], str):
            raise SyncError("state file %s: item %s has an invalid base_hash" % (path, key))
    return doc


class StateLock:
    """Exclusive, non-blocking flock on `<state>.lock` (single-instance guard).  The kernel drops the lock
    when the process dies, so a crash never leaves a stale lock; the file itself is left in place."""

    def __init__(self, path):
        self.path, self.fd = path, None

    def acquire(self):
        try:
            fd = os.open(self.path, os.O_RDWR | os.O_CREAT, 0o600)
        except OSError as exc:
            raise SyncError("cannot open lock file %s: %s" % (self.path, exc.strerror))
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except OSError as exc:
            os.close(fd)
            if exc.errno in (errno.EWOULDBLOCK, errno.EAGAIN):
                raise LockError("another helixtrack-sync instance holds the lock %s; refusing to run "
                                "concurrently (wait for it to finish)" % self.path)
            raise SyncError("cannot lock %s: %s" % (self.path, exc.strerror))
        self.fd = fd

    def release(self):
        if self.fd is not None:
            os.close(self.fd)
            self.fd = None


def save_json_atomic(path, obj):
    """temp file in the same directory (0600, O_EXCL) -> fsync -> rename over the target."""
    directory = os.path.dirname(os.path.abspath(path))
    tmp = os.path.join(directory, ".%s.tmp.%d" % (os.path.basename(path), os.getpid()))
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            json.dump(obj, fh, indent=2, sort_keys=True)
            fh.write("\n")
            fh.flush()
            os.fsync(fh.fileno())
        os.replace(tmp, path)
    except BaseException:
        if os.path.exists(tmp):
            os.unlink(tmp)
        raise


def main(argv=None, environ=None, transport=None):
    environ = os.environ if environ is None else environ
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("command", choices=["push", "pull", "compare"])
    ap.add_argument("--outbox", default="docs/tracking/helixtrack-outbox.json")
    ap.add_argument("--db", help="registry DB, opened read-only, only for item history")
    ap.add_argument("--state", required=True, help="sync state JSON (created/updated by push)")
    ap.add_argument("--project-key", default="HRD")
    ap.add_argument("--project-name", default="Helix Reader")
    ap.add_argument("--env-file", help="KEY=VALUE file (mode 0600) providing HT_BASE_URL/HT_USER/HT_PASSWORD")
    ap.add_argument("--dry-run", action="store_true", help="push: plan only, no writes (state file untouched)")
    ap.add_argument("--overwrite-remote", action="store_true")
    ap.add_argument("--no-probe", action="store_true", help="pull: do not issue version-probe modify requests")
    ap.add_argument("--record", help="append redacted request/response JSONL here")
    ap.add_argument("--report", help="write the JSON report here")
    args = ap.parse_args(argv)
    rep, warnings = None, []
    lock = StateLock(args.state + ".lock")
    try:
        lock.acquire()
        if args.report:      # a previous run's report must never survive as if it described this run
            save_json_atomic(args.report, {"mode": args.command, "aborted": True, "warnings": [],
                                           "error": {"type": "InProgress",
                                                     "message": "run started and not finished"}})
        if args.env_file:
            load_env_file(args.env_file, environ)
        base, user, pw = environ.get("HT_BASE_URL"), environ.get("HT_USER"), environ.get("HT_PASSWORD")
        if not (base and user and pw):
            raise SyncError("HT_BASE_URL, HT_USER and HT_PASSWORD must be set (environment or --env-file)")
        check_base_url(base)
        records = load_outbox(args.outbox)               # all inputs validated BEFORE login
        history = load_history(args.db, warnings) if args.db else None
        for w in warnings:
            emit_warning(w)
        state = load_state(args.state)
        client = CoreClient(transport or urllib_transport(base), user, pw, Recorder(args.record))
        client.login()
        pid = client.ensure_project(args.project_key, args.project_name,
                                    create=(args.command == "push" and not args.dry_run))
        if args.command == "push":
            def checkpoint(st):
                st.update(project_id=pid, project_key=args.project_key)
                save_json_atomic(args.state, st)
            rep = push(client, pid, records, state, args.dry_run, args.overwrite_remote, history,
                       None if args.dry_run else checkpoint)
            if not args.dry_run:
                checkpoint(state)
        elif pid is None:
            raise SyncError("project %s not found on the instance" % args.project_key)
        elif args.command == "pull":
            rep = pull(client, pid, records, state, history, probe=not args.no_probe)
        else:
            rows = compare(client, pid, records, history)
            out = {"mode": "compare", "total": len(rows), "matching": sum(r["match"] for r in rows), "rows": rows,
                   "warnings": warnings}
            if args.report:
                save_json_atomic(args.report, out)
            print(json.dumps({k: out[k] for k in ("mode", "total", "matching")}))
            return 0 if out["matching"] == out["total"] else 2
    except SyncError as exc:
        print("error: %s: %s" % (type(exc).__name__, safe_text(exc)), file=sys.stderr)
        if args.report and not isinstance(exc, LockError):     # never clobber the running instance's report
            save_json_atomic(args.report, {"mode": args.command, "aborted": True, "warnings": warnings,
                                           "error": {"type": type(exc).__name__, "message": str(exc)}})
        return 3
    except Exception as exc:                                  # last resort: honest abort, never a traceback
        print("error: unexpected %s: %s" % (type(exc).__name__, safe_text(exc)), file=sys.stderr)
        if args.report:
            save_json_atomic(args.report, {"mode": args.command, "aborted": True, "warnings": warnings,
                                           "error": {"type": "Unexpected" + type(exc).__name__,
                                                     "message": safe_text(exc)}})
        return 3
    finally:
        lock.release()
    rep.warnings = warnings
    out = rep.to_dict()
    if args.report:
        save_json_atomic(args.report, out)
    print(json.dumps({"mode": out["mode"], "dry_run": out["dry_run"], "counts": out["counts"],
                      "orphans": out["orphans"], "has_conflicts": out["has_conflicts"],
                      "aborted": out["aborted"], "error": out["error"]}))
    if rep.aborted:
        return 3
    if args.command == "push":
        return 2 if rep.has_conflicts else 0
    return 2 if rep.has_drift else 0


if __name__ == "__main__":
    sys.exit(main())
