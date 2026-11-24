"""Unit tests for helixtrack_sync (HRD-029).

The FakeCore below is a TEST DOUBLE whose behaviours were each observed on a
real self-hosted Helix-Track/Core instance (see docs/helixtrack-sync.md):
  * read/list never return `version`
  * modify ignores `type` and silently drops unknown status values
  * a stale `version` yields HTTP 409, errorCode 3006, current version in
    errorMessageLocalised
  * `modified` has one-second resolution
The unit tests prove the tool's logic; the LIVE run (evidence/campaign-2026-11-23/
helixtrack/) is the proof against the real server.
"""
import copy
import json
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import helixtrack_sync as hs  # noqa: E402

def slurp(path, mode="rb"):
    with open(path, mode) as fh:
        return fh.read()


PID = "proj-1"
SECRET_PW = "s3cret-password-value"
SECRET_JWT = "eyJhbGciOiJIUzI1NiJ9.SECRETPAYLOAD.SIG"


def rec(n, typ="Task", status="Queued", sev="High", title=None, reopens=0):
    key = "HRD-%03d" % n
    return {"client_key": key, "operation": "upsert", "revision": reopens + 1, "payload": {
        "atm_id": key, "title": title or ("Title %d" % n), "type": typ, "status": status,
        "severity": sev, "description": "Desc %d." % n, "affected_scope": "scope", "reproduction": "repro",
        "acceptance": "accept", "evidence_path": "evidence/x.log", "reopens_count": reopens,
        "current_location": "Issues"}}


class FakeCore:
    STATUSES = {"open", "in_progress", "blocked", "done", "review", "testing", "closed"}
    TYPES = {"bug", "epic", "feature", "story", "task"}

    def __init__(self):
        self.tickets = {}
        self.projects = {}
        self.n = 0
        self.calls = []
        self.clock = 1000
        self.fail_next = None          # (status, body) returned once
        self.before_modify = None      # callback(ticket) to simulate a concurrent writer
        self.ignore_status = False
        self.ignore_version = False    # server that does not enforce optimistic locking
        self.fail_on = None            # callable(action, obj, data) -> (status, body) | None

    def tick(self):
        self.clock += 1
        return self.clock

    def writes(self):
        return [c for c in self.calls if c[0] in ("create", "modify")]

    def transport(self, method, path, headers, body):
        if self.fail_next:
            out, self.fail_next = self.fail_next, None
            return out
        if path == "/health":
            return 200, {"status": "ok"}
        assert path == "/do", path
        action, obj, data = body.get("action"), body.get("object"), body.get("data") or {}
        probe = action == "modify" and data.get("version") == hs.PROBE_VERSION
        self.calls.append(("probe" if probe else action, obj))
        if action == "authenticate":
            if data.get("password") != SECRET_PW:
                return 401, {"errorCode": 1004, "errorMessage": "Invalid username or password"}
            return 200, {"errorCode": -1, "data": {"token": SECRET_JWT, "username": data["username"]}}
        if body.get("jwt") != SECRET_JWT:
            return 401, {"errorCode": 1003, "errorMessage": "Invalid or expired JWT token"}
        if self.fail_on and not probe:
            injected = self.fail_on(action, obj, data)
            if injected:
                return injected
        if (action, obj) == ("list", "project"):
            return 200, {"errorCode": -1, "data": {"items": list(self.projects.values()), "total": len(self.projects)}}
        if (action, obj) == ("create", "project"):
            pid = "proj-%d" % (len(self.projects) + 1)
            self.projects[pid] = {"id": pid, "identifier": data["key"], "title": data["name"]}
            return 200, {"errorCode": -1, "data": {"project": dict(self.projects[pid])}}
        if (action, obj) == ("list", "ticket"):
            items = [self.view(t) for t in self.tickets.values()
                     if not t["deleted"] and t["project_id"] == data.get("project_id")]
            return 200, {"errorCode": -1, "data": {"items": items, "total": len(items)}}
        if (action, obj) == ("read", "ticket"):
            t = self.tickets.get(data.get("id"))
            if not t or t["deleted"]:
                return 404, {"errorCode": 3000, "errorMessage": "Ticket not found"}
            return 200, {"errorCode": -1, "data": {"ticket": self.view(t)}}
        if (action, obj) == ("create", "ticket"):
            if not data.get("title"):
                return 400, {"errorCode": 1007, "errorMessage": "Missing title"}
            self.n += 1
            tid = "t-%d" % self.n
            now = self.tick()
            self.tickets[tid] = {"id": tid, "ticket_number": self.n, "title": data["title"],
                                 "description": data.get("description", ""), "type": data.get("type", "task"),
                                 "status": "open", "project_id": data["project_id"], "created": now,
                                 "modified": now, "version": 1, "deleted": False}
            return 200, {"errorCode": -1, "data": {"ticket": {"id": tid, "status": "open", "title": data["title"]}}}
        if (action, obj) == ("modify", "ticket"):
            t = self.tickets.get(data.get("id"))
            if not t or t["deleted"]:
                return 404, {"errorCode": 3000, "errorMessage": "Ticket not found"}
            if self.before_modify and data.get("version") != hs.PROBE_VERSION:
                cb, self.before_modify = self.before_modify, None
                cb(t)
            exp = data.get("version") or t["version"]
            if self.ignore_version:
                exp = t["version"]
            if exp != t["version"]:
                return 409, {"errorCode": 3006, "errorMessage": "Version conflict detected",
                             "errorMessageLocalised": "Current version: %d, Expected: %d" % (t["version"], exp)}
            upd = {}
            if data.get("title"):
                upd["title"] = data["title"]
            if "description" in data:
                upd["description"] = data["description"]
            if data.get("status") in self.STATUSES and not self.ignore_status:
                upd["status"] = data["status"]
            if not upd:
                return 400, {"errorCode": 1007, "errorMessage": "No fields to update"}
            t.update(upd)
            t["version"] += 1
            t["modified"] = self.tick()
            return 200, {"errorCode": -1, "data": {"ticket": {"id": t["id"], "updated": True, "version": t["version"]}}}
        if (action, obj) == ("remove", "ticket"):
            t = self.tickets.get(data.get("id"))
            if not t or t["deleted"]:
                return 404, {"errorCode": 3000, "errorMessage": "Ticket not found"}
            t["deleted"] = True
            return 200, {"errorCode": -1, "data": {"ticket": {"id": t["id"], "deleted": True}}}
        return 400, {"errorCode": 1005, "errorMessage": "unsupported"}

    @staticmethod
    def view(t):
        return {k: t[k] for k in ("id", "ticket_number", "title", "description", "type", "status",
                                  "project_id", "created", "modified")}

    def remote_edit(self, key, **fields):
        for t in self.tickets.values():
            if t["title"].startswith("[%s]" % key):
                t.update(fields)
                t["version"] += 1
                t["modified"] = self.tick()
                return t
        raise KeyError(key)


def client(fake, recorder=None):
    c = hs.CoreClient(fake.transport, "hrd_sync", SECRET_PW, recorder=recorder)
    c.login()
    return c


def actions(report):
    return {a["key"]: a["action"] for a in report.actions}


class MappingTests(unittest.TestCase):
    def test_type_and_title_key(self):
        f = hs.build_fields(rec(1, typ="Feature"))
        self.assertEqual(f["type"], "feature")
        self.assertTrue(f["title"].startswith("[HRD-001] "))
        self.assertEqual(hs.build_fields(rec(2, typ="Bug"))["type"], "bug")
        self.assertEqual(hs.build_fields(rec(3, typ="Task"))["type"], "task")

    def test_every_registry_status_maps_to_core_vocabulary(self):
        expect = {"Queued": "open", "In progress": "in_progress", "Operator-blocked": "blocked",
                  "Completed (→ Fixed.md)": "done", "Implemented (→ Fixed.md)": "done", "Fixed (→ Fixed.md)": "done"}
        for reg, core in expect.items():
            self.assertEqual(hs.map_status(reg), core, reg)
        with self.assertRaises(hs.MappingError):
            hs.map_status("Bogus")

    def test_description_carries_owned_fields_marker_history(self):
        hist = {"HRD-004": [{"on_date": "2026-10-28", "event_type": "Reopened", "actor": "AI", "reason": "why"}]}
        f = hs.build_fields(rec(4, sev="Critical", reopens=2), hist["HRD-004"])
        d = f["description"]
        for needle in ("Desc 4.", "Severity: Critical", "Reopens: 2", "Affected scope: scope",
                       "Acceptance: accept", "2026-10-28 Reopened (AI): why", "helix-reader-sync/v1 hrd=HRD-004"):
            self.assertIn(needle, d)

    def test_hash_deterministic_and_sensitive(self):
        a = hs.fields_hash(hs.build_fields(rec(1)))
        self.assertEqual(a, hs.fields_hash(hs.build_fields(rec(1))))
        self.assertNotEqual(a, hs.fields_hash(hs.build_fields(rec(1, sev="Medium"))))
        self.assertNotEqual(a, hs.fields_hash(hs.build_fields(rec(1, status="In progress"))))


class PushTests(unittest.TestCase):
    def setUp(self):
        self.fake = FakeCore()
        self.fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        self.c = client(self.fake)
        self.recs = [rec(1), rec(2, typ="Feature", status="In progress"), rec(3, status="Fixed (→ Fixed.md)")]

    def test_first_push_creates_each_item_once_with_status(self):
        st = {}
        r = hs.push(self.c, PID, self.recs, st)
        self.assertEqual(set(actions(r).values()), {"create"})
        self.assertEqual(len(self.fake.tickets), 3)
        by = {t["title"][:9]: t for t in self.fake.tickets.values()}
        self.assertEqual(by["[HRD-001]"]["status"], "open")
        self.assertEqual(by["[HRD-002]"]["status"], "in_progress")
        self.assertEqual(by["[HRD-003]"]["status"], "done")
        self.assertEqual(by["[HRD-002]"]["type"], "feature")
        self.assertEqual(set(st["items"]), {"HRD-001", "HRD-002", "HRD-003"})

    def test_repush_is_idempotent_no_writes_no_duplicates(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        before = copy.deepcopy(self.fake.tickets)
        self.fake.calls.clear()
        r = hs.push(self.c, PID, self.recs, st)
        self.assertEqual(set(actions(r).values()), {"noop"})
        self.assertEqual(self.fake.writes(), [])
        self.assertEqual(self.fake.tickets, before)
        self.assertEqual(len(self.fake.tickets), 3)

    def test_lost_state_rematches_by_title_key_without_duplicates(self):
        hs.push(self.c, PID, self.recs, {})
        self.fake.calls.clear()
        st2 = {}
        r = hs.push(self.c, PID, self.recs, st2)
        self.assertEqual(len(self.fake.tickets), 3)
        self.assertEqual(self.fake.writes(), [])
        self.assertEqual(set(actions(r).values()), {"noop"})
        self.assertEqual(set(st2["items"]), {"HRD-001", "HRD-002", "HRD-003"})

    def test_registry_change_updates_with_version_from_probe(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        changed = [rec(1, status="In progress")] + self.recs[1:]
        r = hs.push(self.c, PID, changed, st)
        self.assertEqual(actions(r)["HRD-001"], "update")
        t = [t for t in self.fake.tickets.values() if t["title"].startswith("[HRD-001]")][0]
        self.assertEqual(t["status"], "in_progress")
        self.assertEqual(st["items"]["HRD-001"]["version"], t["version"])
        self.assertEqual(len(self.fake.tickets), 3)

    def test_remote_edit_is_never_silently_overwritten(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        self.fake.remote_edit("HRD-001", description="edited remotely")
        changed = [rec(1, status="In progress")] + self.recs[1:]
        self.fake.calls.clear()
        r = hs.push(self.c, PID, changed, st)
        self.assertEqual(actions(r)["HRD-001"], "conflict")
        self.assertTrue(r.has_conflicts)
        self.assertEqual(self.fake.writes(), [])
        t = [t for t in self.fake.tickets.values() if t["title"].startswith("[HRD-001]")][0]
        self.assertEqual(t["description"], "edited remotely")

    def test_overwrite_remote_flag_resolves_conflict_to_registry(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        self.fake.remote_edit("HRD-001", description="edited remotely")
        r = hs.push(self.c, PID, self.recs, st, overwrite_remote=True)
        self.assertEqual(actions(r)["HRD-001"], "overwrite")
        t = [t for t in self.fake.tickets.values() if t["title"].startswith("[HRD-001]")][0]
        self.assertEqual(t["description"], hs.build_fields(self.recs[0])["description"])

    def test_concurrent_writer_between_probe_and_modify_retries_once(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        changed = [rec(1, sev="Medium")] + self.recs[1:]
        # concurrent bump that does not alter content -> version moves, content equals base
        self.fake.before_modify = lambda t: t.__setitem__("version", t["version"] + 1)
        r = hs.push(self.c, PID, changed, st)
        self.assertEqual(actions(r)["HRD-001"], "update")
        self.assertEqual(r.actions[0].get("retries"), 1)

    def test_concurrent_content_edit_during_modify_becomes_conflict(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        changed = [rec(1, sev="Medium")] + self.recs[1:]

        def edit(t):
            t["description"] = "raced edit"
            t["version"] += 1
        self.fake.before_modify = edit
        r = hs.push(self.c, PID, changed, st)
        self.assertEqual(actions(r)["HRD-001"], "conflict")

    def test_type_change_is_reported_unpushable_not_silently_dropped(self):
        st = {}
        hs.push(self.c, PID, self.recs, st)
        changed = [rec(1, typ="Bug")] + self.recs[1:]
        r = hs.push(self.c, PID, changed, st)
        self.assertEqual(actions(r)["HRD-001"], "unpushable")

    def test_readback_mismatch_is_detected(self):
        st = {}
        hs.push(self.c, PID, [rec(2)], st)
        self.fake.ignore_status = True   # server returns 200 but drops the status change (Core does this)
        r = hs.push(self.c, PID, [rec(2, status="In progress")], st)
        self.assertTrue(r.aborted)
        self.assertEqual(r.error["type"], "ReadbackMismatch")

    def test_dry_run_performs_no_writes(self):
        r = hs.push(self.c, PID, self.recs, {}, dry_run=True)
        self.assertEqual(set(actions(r).values()), {"create"})
        self.assertTrue(r.dry_run)
        self.assertEqual(self.fake.writes(), [])
        self.assertEqual(self.fake.tickets, {})

    def test_duplicate_remote_keys_and_orphans_reported(self):
        hs.push(self.c, PID, self.recs, {})
        dup = copy.deepcopy(list(self.fake.tickets.values())[0])
        dup["id"] = "t-dup"
        self.fake.tickets["t-dup"] = dup
        self.fake.tickets["t-orph"] = dict(dup, id="t-orph", title="[HRD-099] ghost")
        r = hs.push(self.c, PID, self.recs, {})
        self.assertIn("duplicate", set(actions(r).values()))
        self.assertEqual(r.orphans, ["HRD-099"])


class PullTests(unittest.TestCase):
    def setUp(self):
        self.fake = FakeCore()
        self.fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        self.c = client(self.fake)
        self.recs = [rec(1), rec(2, typ="Feature"), rec(3)]
        self.st = {}
        hs.push(self.c, PID, self.recs, self.st)

    def test_clean_pull_is_all_in_sync(self):
        r = hs.pull(self.c, PID, self.recs, self.st)
        self.assertEqual(set(actions(r).values()), {"in-sync"})
        self.assertFalse(r.has_drift)

    def test_remote_edit_detected_with_fields_and_version(self):
        self.fake.remote_edit("HRD-002", status="review", title="[HRD-002] renamed")
        r = hs.pull(self.c, PID, self.recs, self.st)
        a = {x["key"]: x for x in r.actions}["HRD-002"]
        self.assertEqual(a["action"], "remote-edit")
        self.assertEqual(sorted(a["changed_fields"]), ["status", "title"])
        self.assertGreater(a["remote_version"], a["base_version"])
        self.assertTrue(r.has_drift)

    def test_pull_never_mutates_remote(self):
        self.fake.remote_edit("HRD-002", status="review")
        self.fake.calls.clear()
        hs.pull(self.c, PID, self.recs, self.st)
        self.assertEqual(self.fake.writes(), [])

    def test_pull_status_proposal_maps_to_registry_vocabulary(self):
        self.fake.remote_edit("HRD-001", status="in_progress")
        self.fake.remote_edit("HRD-002", status="review")
        r = hs.pull(self.c, PID, self.recs, self.st)
        props = {p["key"]: p for p in r.proposals}
        self.assertEqual(props["HRD-001"]["registry_status"], "In progress")
        self.assertIsNone(props["HRD-002"]["registry_status"])
        self.assertEqual(props["HRD-002"]["remote_status"], "review")

    def test_registry_ahead_and_both_changed_are_distinguished(self):
        self.fake.remote_edit("HRD-003", description="remote")
        newrecs = [rec(1, sev="Medium"), self.recs[1], rec(3, sev="Medium")]
        r = hs.pull(self.c, PID, newrecs, self.st)
        a = actions(r)
        self.assertEqual(a["HRD-001"], "registry-ahead")
        self.assertEqual(a["HRD-003"], "conflict")

    def test_missing_remote_and_deleted_ticket(self):
        r = hs.pull(self.c, PID, self.recs + [rec(9)], self.st)
        self.assertEqual(actions(r)["HRD-009"], "missing-remote")
        for t in self.fake.tickets.values():
            if t["title"].startswith("[HRD-001]"):
                t["deleted"] = True
        r = hs.pull(self.c, PID, self.recs, self.st)
        self.assertEqual(actions(r)["HRD-001"], "remote-deleted")


class FieldCompareTests(unittest.TestCase):
    def test_compare_registry_to_remote_field_by_field(self):
        fake = FakeCore()
        fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        c = client(fake)
        recs = [rec(1), rec(2, typ="Feature", status="In progress")]
        hs.push(c, PID, recs, {})
        rows = hs.compare(c, PID, recs)
        self.assertEqual(len(rows), 2)
        self.assertTrue(all(r["match"] for r in rows))
        self.assertEqual(set(rows[0]["fields"]), {"title", "description", "type", "status"})
        fake.remote_edit("HRD-001", status="done")
        rows = hs.compare(c, PID, recs)
        self.assertFalse(rows[0]["match"])
        self.assertFalse(rows[0]["fields"]["status"])


class ErrorMappingTests(unittest.TestCase):
    def mk(self, status, body):
        fake = FakeCore()
        fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        c = client(fake)
        fake.fail_next = (status, body)
        return c

    def test_http_errors_map_to_typed_exceptions(self):
        cases = [(401, {"errorCode": 1004, "errorMessage": "bad"}, hs.AuthError),
                 (403, {"errorCode": 2000, "errorMessage": "no"}, hs.ForbiddenError),
                 (404, {"errorCode": 3000, "errorMessage": "nf"}, hs.NotFoundError),
                 (400, {"errorCode": 1007, "errorMessage": "bad"}, hs.ValidationError),
                 (500, {"errorCode": 5000, "errorMessage": "boom"}, hs.ServerError),
                 (502, None, hs.ServerError)]
        for status, body, exc in cases:
            c = self.mk(status, body)
            with self.assertRaises(exc, msg=str(status)):
                c.list_tickets(PID)

    def test_version_conflict_carries_current_version(self):
        c = self.mk(409, {"errorCode": 3006, "errorMessage": "Version conflict detected",
                          "errorMessageLocalised": "Current version: 7, Expected: 3"})
        with self.assertRaises(hs.VersionConflict) as cm:
            c.modify_ticket("t-1", 3, title="x")
        self.assertEqual(cm.exception.current, 7)

    def test_transport_failure_maps(self):
        def boom(*a, **k):
            raise ConnectionRefusedError("refused")
        c = hs.CoreClient(boom, "u", "p")
        with self.assertRaises(hs.TransportError):
            c.login()

    def test_login_failure_is_auth_error(self):
        fake = FakeCore()
        c = hs.CoreClient(fake.transport, "u", "wrong")
        with self.assertRaises(hs.AuthError):
            c.login()

    def test_expired_jwt_triggers_single_relogin(self):
        fake = FakeCore()
        fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        c = client(fake)
        fake.fail_next = (401, {"errorCode": 1003, "errorMessage": "expired"})
        self.assertEqual(c.list_tickets(PID), [])
        self.assertEqual([x[0] for x in fake.calls].count("authenticate"), 2)

    def test_probe_version_does_not_mutate(self):
        fake = FakeCore()
        fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        c = client(fake)
        hs.push(c, PID, [rec(1)], {})
        tid = list(fake.tickets)[0]
        fake.remote_edit("HRD-001", description="z")
        snap = copy.deepcopy(fake.tickets)
        self.assertEqual(c.probe_version(tid), 2)
        self.assertEqual(fake.tickets, snap)


class RedactionTests(unittest.TestCase):
    def test_recorded_traffic_never_contains_secrets(self):
        fake = FakeCore()
        fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
        rc = hs.Recorder(None)
        c = client(fake, recorder=rc)
        hs.push(c, PID, [rec(1)], {})
        blob = json.dumps(rc.entries)
        self.assertGreater(len(rc.entries), 3)
        for secret in (SECRET_PW, SECRET_JWT, "SECRETPAYLOAD"):
            self.assertNotIn(secret, blob)
        self.assertIn("[REDACTED]", blob)

    def test_redact_handles_nested_and_headers(self):
        out = hs.redact({"jwt": "a", "data": {"password": "p", "token": "t", "x": 1},
                         "headers": {"Authorization": "Bearer abc"}})
        s = json.dumps(out)
        for bad in ('"a"', '"p"', '"t"', "abc"):
            self.assertNotIn(bad, s)
        self.assertEqual(out["data"]["x"], 1)


# ======================================================================= review round 2 (findings 1-8)
import contextlib  # noqa: E402
import http.server  # noqa: E402
import io  # noqa: E402
import tempfile  # noqa: E402
import threading  # noqa: E402


def fresh(n=3):
    fake = FakeCore()
    fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
    return fake, client(fake), [rec(i) for i in range(1, n + 1)]


def by_key(fake, key):
    return [t for t in fake.tickets.values() if t["title"].startswith("[%s]" % key)][0]


class UrlGuardTests(unittest.TestCase):                                   # finding 1
    def test_refuses_everything_but_https_or_loopback_http(self):
        for bad in ("HTTP://10.1.2.3:8080", "http://10.1.2.3", "Http://example.com", "ftp://127.0.0.1",
                    "file:///etc/passwd", "", "127.0.0.1:18417", "http://localhost.evil.com",
                    "http://127.0.0.1.evil.com", "http://127.0.0.1@evil.com/","https://"):
            with self.assertRaises(hs.SyncError, msg=bad):
                hs.check_base_url(bad)

    def test_allows_https_and_loopback_http_any_case(self):
        for ok in ("https://example.com", "HTTPS://Example.com:8443", "http://127.0.0.1:18417",
                   "HTTP://LOCALHOST:1", "http://[::1]:1", "http://127.0.0.2:5"):
            hs.check_base_url(ok)

    def test_transport_constructor_enforces_guard_too(self):
        with self.assertRaises(hs.SyncError):
            hs.urllib_transport("HTTP://10.9.9.9:1")


class _Sink(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        self.server.hits.append((self.path, self.headers.get("Authorization")))
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"{}")

    do_GET = do_POST

    def log_message(self, *a):
        pass


class _Redirector(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length") or 0))
        self.send_response(self.server.code)
        self.send_header("Location", self.server.target)
        self.end_headers()

    def log_message(self, *a):
        pass


def serve(handler, **attrs):
    srv = http.server.HTTPServer(("127.0.0.1", 0), handler)
    for k, v in attrs.items():
        setattr(srv, k, v)
    srv.hits = []
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    return srv


def stop(case, srv):
    case.addCleanup(srv.server_close)
    case.addCleanup(srv.shutdown)


class RedirectTests(unittest.TestCase):                                   # finding 2
    def test_3xx_from_any_transport_is_refused(self):
        for code in (301, 302, 303, 307, 308):
            c = hs.CoreClient(lambda *a, _c=code: (_c, None), "u", "p")
            with self.assertRaises(hs.RedirectRefused, msg=str(code)):
                c.login()

    def test_real_transport_never_follows_redirects_or_forwards_authorization(self):
        sink = serve(_Sink)
        stop(self, sink)
        for code in (301, 302, 303, 307, 308):
            red = serve(_Redirector, code=code, target="http://127.0.0.1:%d/do" % sink.server_address[1])
            stop(self, red)
            c = hs.CoreClient(hs.urllib_transport("http://127.0.0.1:%d" % red.server_address[1]), "u", "p")
            c._jwt = "tok"
            with self.assertRaises(hs.RedirectRefused, msg=str(code)):
                c.list_tickets("p")
        self.assertEqual(sink.hits, [])


class ProbeSafetyTests(unittest.TestCase):                                # finding 3
    def test_probe_sends_current_fields_and_never_a_marker_title(self):
        fake, c, recs = fresh(1)
        rc = hs.Recorder(None)
        c = client(fake, recorder=rc)
        hs.push(c, PID, recs, {})
        t = by_key(fake, "HRD-001")
        before = dict(t)
        c.probe_version(t["id"])
        probe_req = [e["request"] for e in rc.entries if e["request"].get("action") == "modify"][-1]
        self.assertEqual(probe_req["data"]["title"], before["title"])
        self.assertEqual(probe_req["data"]["description"], before["description"])
        self.assertNotEqual(probe_req["data"]["title"], "probe")

    def test_server_that_ignores_version_is_not_corrupted_and_is_detected(self):
        fake, c, recs = fresh(1)
        hs.push(c, PID, recs, {})
        t = by_key(fake, "HRD-001")
        title, desc, status = t["title"], t["description"], t["status"]
        fake.ignore_version = True
        with self.assertRaises(hs.ServerError):
            c.probe_version(t["id"])
        self.assertEqual((t["title"], t["description"], t["status"]), (title, desc, status))

    def test_pull_no_probe_issues_no_modify_requests(self):
        fake, c, recs = fresh(2)
        st = {}
        hs.push(c, PID, recs, st)
        fake.remote_edit("HRD-002", status="review")
        fake.calls.clear()
        r = hs.pull(c, PID, recs, st, probe=False)
        self.assertNotIn("probe", [x[0] for x in fake.calls])
        a = {x["key"]: x for x in r.actions}["HRD-002"]
        self.assertEqual(a["action"], "remote-edit")
        self.assertIsNone(a["remote_version"])

    def test_default_pull_documents_that_it_issues_probe_modifies(self):
        fake, c, recs = fresh(2)
        st = {}
        hs.push(c, PID, recs, st)
        fake.remote_edit("HRD-002", status="review")
        fake.calls.clear()
        hs.pull(c, PID, recs, st)
        self.assertEqual([x[0] for x in fake.calls].count("probe"), 1)   # only for drifted items
        self.assertEqual(hs.PULL_ISSUES_PROBE_MODIFY, True)


class MatchingTests(unittest.TestCase):                                   # finding 4
    def test_known_ticket_id_whose_title_lost_its_key_is_conflict_not_create(self):
        fake, c, recs = fresh(3)
        st = {}
        hs.push(c, PID, recs, st)
        fake.remote_edit("HRD-002", title="renamed without key")
        n = len(fake.tickets)
        fake.calls.clear()
        r = hs.push(c, PID, recs, st)
        self.assertEqual(actions(r)["HRD-002"], "conflict")
        self.assertIn("title", [a for a in r.actions if a["key"] == "HRD-002"][0]["changed_fields"])
        self.assertEqual(fake.writes(), [])
        self.assertEqual(len(fake.tickets), n)
        r = hs.pull(c, PID, recs, st)
        self.assertEqual(actions(r)["HRD-002"], "remote-edit")

    def test_marker_line_matches_when_state_lost_and_title_lost_key(self):
        fake, c, recs = fresh(2)
        hs.push(c, PID, recs, {})
        fake.remote_edit("HRD-001", title="no key here")
        n = len(fake.tickets)
        r = hs.push(c, PID, recs, {})
        self.assertEqual(actions(r)["HRD-001"], "conflict")
        self.assertEqual(len(fake.tickets), n)

    def test_forged_title_next_to_known_ticket_is_reported_duplicate(self):
        fake, c, recs = fresh(2)
        st = {}
        hs.push(c, PID, recs, st)
        real = by_key(fake, "HRD-001")
        fake.tickets["forged"] = dict(real, id="forged", description="attacker text", ticket_number=99)
        fake.calls.clear()
        r = hs.push(c, PID, recs, st)
        self.assertEqual(actions(r)["HRD-001"], "duplicate")
        self.assertTrue(r.has_conflicts)
        self.assertEqual(fake.writes(), [])
        self.assertEqual(hs.pull(c, PID, recs, st).actions[0]["action"], "duplicate")

    def test_orphans_include_marker_only_and_title_only_claims(self):
        fake, c, recs = fresh(1)
        hs.push(c, PID, recs, {})
        base = dict(by_key(fake, "HRD-001"))
        fake.tickets["o1"] = dict(base, id="o1", title="[HRD-099] ghost", description="x", ticket_number=7)
        fake.tickets["o2"] = dict(base, id="o2", title="retitled", ticket_number=8,
                                  description="body\n---\nhelix-reader-sync/v1 hrd=HRD-098")
        r = hs.push(c, PID, recs, {})
        self.assertEqual(r.orphans, ["HRD-098", "HRD-099"])


class RemoteDeletedTests(unittest.TestCase):                              # finding 5
    def setUp(self):
        self.fake, self.c, self.recs = fresh(3)
        self.st = {}
        hs.push(self.c, PID, self.recs, self.st)
        self.old_id = by_key(self.fake, "HRD-002")["id"]
        self.fake.tickets[self.old_id]["deleted"] = True

    def test_push_reports_remote_deleted_and_does_not_recreate(self):
        self.fake.calls.clear()
        r = hs.push(self.c, PID, self.recs, self.st)
        self.assertEqual(actions(r)["HRD-002"], "remote-deleted")
        self.assertTrue(r.has_conflicts)
        self.assertEqual(self.fake.writes(), [])
        self.assertEqual(len([t for t in self.fake.tickets.values() if not t["deleted"]]), 2)

    def test_pull_reports_remote_deleted(self):
        r = hs.pull(self.c, PID, self.recs, self.st)
        self.assertEqual(actions(r)["HRD-002"], "remote-deleted")
        self.assertTrue(r.has_drift)

    def test_overwrite_remote_recreates_explicitly(self):
        r = hs.push(self.c, PID, self.recs, self.st, overwrite_remote=True)
        self.assertEqual(actions(r)["HRD-002"], "recreate")
        live = [t for t in self.fake.tickets.values() if not t["deleted"]]
        self.assertEqual(len(live), 3)
        self.assertNotEqual(self.st["items"]["HRD-002"]["ticket_id"], self.old_id)
        self.assertEqual(set(actions(hs.push(self.c, PID, self.recs, self.st)).values()), {"noop"})

    def test_dry_run_reports_without_recreating(self):
        self.fake.calls.clear()
        r = hs.push(self.c, PID, self.recs, self.st, dry_run=True, overwrite_remote=True)
        self.assertEqual(actions(r)["HRD-002"], "recreate")
        self.assertEqual(self.fake.writes(), [])

    def test_never_synced_missing_item_is_still_a_plain_create(self):
        r = hs.push(self.c, PID, self.recs + [rec(9)], self.st)
        self.assertEqual(actions(r)["HRD-009"], "create")


class ResumeTests(unittest.TestCase):                                     # finding 6
    def test_mid_batch_503_is_reported_checkpointed_and_resumable(self):
        fake, c, recs = fresh(5)
        count = {"create": 0}

        def fail(action, obj, data):
            if (action, obj) == ("create", "ticket"):
                count["create"] += 1
                if count["create"] == 3:
                    return 503, {"errorCode": 5000, "errorMessage": "unavailable"}
        fake.fail_on = fail
        st, saves = {}, []
        r = hs.push(c, PID, recs, st, checkpoint=lambda s: saves.append(copy.deepcopy(s)))
        self.assertTrue(r.aborted)
        self.assertEqual(r.error["type"], "ServerError")
        self.assertEqual(actions(r)["HRD-003"], "error")
        self.assertEqual(set(st["items"]), {"HRD-001", "HRD-002"})
        self.assertGreaterEqual(len(saves), 2)
        self.assertIn("HRD-002", saves[-1]["items"])
        fake.fail_on = None
        r2 = hs.push(c, PID, recs, st)
        self.assertFalse(r2.aborted)
        self.assertEqual(len(fake.tickets), 5)
        self.assertEqual(sorted(t["title"][:9] for t in fake.tickets.values()),
                         ["[HRD-001]", "[HRD-002]", "[HRD-003]", "[HRD-004]", "[HRD-005]"])
        self.assertFalse(r2.has_conflicts)

    def test_failed_status_modify_after_create_does_not_become_false_conflict(self):
        fake, c, recs = fresh(1)
        recs = [rec(1, status="In progress")]

        def fail(action, obj, data):
            if action == "modify" and "status" in data:
                return 503, {"errorCode": 5000, "errorMessage": "unavailable"}
        fake.fail_on = fail
        st = {}
        r = hs.push(c, PID, recs, st)
        self.assertTrue(r.aborted)
        self.assertIn("HRD-001", st["items"])                 # baseline recorded BEFORE the follow-up modify
        fake.fail_on = None
        r2 = hs.push(c, PID, recs, st)
        self.assertEqual(actions(r2)["HRD-001"], "update")
        self.assertFalse(r2.has_conflicts)
        self.assertEqual(by_key(fake, "HRD-001")["status"], "in_progress")
        self.assertEqual(len(fake.tickets), 1)

    def test_interrupted_create_with_lost_state_reconciles(self):
        fake, c, recs = fresh(1)
        recs = [rec(1, status="In progress")]
        fake.fail_on = lambda a, o, d: (503, {"errorCode": 5000, "errorMessage": "x"}) \
            if a == "modify" and "status" in d else None
        hs.push(c, PID, recs, {})
        fake.fail_on = None
        r = hs.push(c, PID, recs, {})                         # state file lost as well
        self.assertEqual(actions(r)["HRD-001"], "update")
        self.assertEqual(len(fake.tickets), 1)

    def test_state_checkpointed_between_create_and_status_modify_even_on_hard_crash(self):
        fake, c, _ = fresh(1)
        recs = [rec(1, status="In progress")]

        def crash(action, obj, data):
            if action == "modify" and "status" in data:
                raise KeyboardInterrupt()
        fake.fail_on = crash
        saves = []
        with self.assertRaises(KeyboardInterrupt):
            hs.push(c, PID, recs, {}, checkpoint=lambda s: saves.append(copy.deepcopy(s)))
        self.assertTrue(saves and "HRD-001" in saves[-1]["items"])   # baseline persisted before the crash

    def test_checkpoint_runs_after_every_item_and_never_in_dry_run(self):
        fake, c, recs = fresh(4)
        saves = []
        hs.push(c, PID, recs, {}, checkpoint=lambda s: saves.append(1))
        self.assertGreaterEqual(len(saves), 4)
        saves.clear()
        hs.push(c, PID, [rec(7)], {}, dry_run=True, checkpoint=lambda s: saves.append(1))
        self.assertEqual(saves, [])

    def test_pull_error_is_reported_not_raised(self):
        fake, c, recs = fresh(2)
        st = {}
        hs.push(c, PID, recs, st)
        fake.remote_edit("HRD-002", status="review")
        fake.fail_on = lambda a, o, d: (503, {"errorCode": 5000, "errorMessage": "x"}) if a == "read" else None
        r = hs.pull(c, PID, recs, st)
        self.assertTrue(r.aborted)
        self.assertEqual(r.error["type"], "ServerError")
        self.assertEqual(actions(r)["HRD-002"], "error")


class MainTests(unittest.TestCase):                                       # finding 7
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.addCleanup(__import__("shutil").rmtree, self.tmp, True)
        self.fake, _, self.recs = fresh(3)
        self.outbox = self.p("outbox.json")
        with open(self.outbox, "w") as fh:
            json.dump({"records": self.recs}, fh)
        self.state, self.report = self.p("state.json"), self.p("report.json")
        self.env = self.p("ht.env")
        self.write_env(0o600)
        self.fake.calls.clear()

    def p(self, name):
        return os.path.join(self.tmp, name)

    def write_env(self, mode, base="http://127.0.0.1:18417"):
        with open(self.env, "w") as fh:
            fh.write("HT_BASE_URL=%s\nHT_USER=hrd_sync\nHT_PASSWORD=%s\n" % (base, SECRET_PW))
        os.chmod(self.env, mode)

    def run_main(self, cmd, *extra, environ=None):
        out, err = io.StringIO(), io.StringIO()
        argv = [cmd, "--outbox", self.outbox, "--state", self.state, "--report", self.report] + list(extra)
        if environ is None:
            argv += ["--env-file", self.env]
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = hs.main(argv, environ={} if environ is None else environ, transport=self.fake.transport)
        return code, out.getvalue(), err.getvalue()

    def test_clean_push_pull_compare_exit_zero(self):
        self.assertEqual(self.run_main("push")[0], 0)
        self.assertEqual(self.run_main("pull")[0], 0)
        self.assertEqual(self.run_main("compare")[0], 0)
        self.assertEqual(len(self.fake.tickets), 3)

    def test_push_conflict_exits_2(self):
        self.run_main("push")
        self.fake.remote_edit("HRD-001", description="remote")
        code, out, _ = self.run_main("push")
        self.assertEqual(code, 2)
        self.assertEqual(json.loads(out)["counts"]["conflict"], 1)

    def test_pull_drift_exits_2_and_compare_mismatch_exits_2(self):
        self.run_main("push")
        self.fake.remote_edit("HRD-001", status="review")
        self.assertEqual(self.run_main("pull")[0], 2)
        self.assertEqual(self.run_main("compare")[0], 2)

    def test_env_file_0644_refused_exit_3_and_0600_accepted(self):
        self.write_env(0o644)
        code, _, err = self.run_main("push")
        self.assertEqual(code, 3)
        self.assertIn("env file", err)
        self.assertNotIn(SECRET_PW, err)
        self.assertEqual(self.fake.calls, [])
        self.write_env(0o600)
        self.assertEqual(self.run_main("push")[0], 0)

    def test_plain_http_and_uppercase_http_refused_before_any_request(self):
        for base in ("HTTP://10.1.2.3:8080", "http://10.1.2.3:8080", "Http://example.org"):
            self.write_env(0o600, base)
            code, _, err = self.run_main("push")
            self.assertEqual(code, 3, base)
            self.assertIn("refusing", err)
            self.assertEqual(self.fake.calls, [], base)
        self.assertFalse(os.path.exists(self.state))

    def test_dry_run_never_writes_state_or_remote(self):
        code, out, _ = self.run_main("push", "--dry-run")
        self.assertEqual(code, 0)
        self.assertFalse(os.path.exists(self.state))
        self.assertEqual(self.fake.tickets, {})
        self.run_main("push")
        before = slurp(self.state)
        self.fake.remote_edit("HRD-001", description="r")
        self.run_main("push", "--dry-run", "--overwrite-remote")
        self.assertEqual(slurp(self.state), before)

    def test_state_is_written_atomically_with_no_temp_leftovers(self):
        self.run_main("push")
        self.assertEqual(sorted(os.listdir(self.tmp)), ["ht.env", "outbox.json", "report.json", "state.json", "state.json.lock"])
        self.assertEqual(oct(os.stat(self.state).st_mode & 0o777), "0o600")

    def test_mid_batch_503_exit_3_partial_state_and_report_then_resume(self):
        n = {"c": 0}

        def fail(action, obj, data):
            if (action, obj) == ("create", "ticket"):
                n["c"] += 1
                if n["c"] == 2:
                    return 503, {"errorCode": 5000, "errorMessage": "unavailable"}
        self.fake.fail_on = fail
        code, out, _ = self.run_main("push")
        self.assertEqual(code, 3)
        st = json.loads(slurp(self.state, "r"))
        self.assertEqual(set(st["items"]), {"HRD-001"})
        rep = json.loads(slurp(self.report, "r"))
        self.assertTrue(rep["aborted"])
        self.assertEqual(rep["error"]["type"], "ServerError")
        self.fake.fail_on = None
        self.assertEqual(self.run_main("push")[0], 0)
        self.assertEqual(len(self.fake.tickets), 3)

    def test_atomic_save_keeps_old_file_when_the_write_fails(self):
        target = self.p("atomic.json")
        hs.save_json_atomic(target, {"v": 1})
        real = json.dump

        def boom(*a, **k):
            raise RuntimeError("disk full")
        json.dump = boom
        try:
            with self.assertRaises(RuntimeError):
                hs.save_json_atomic(target, {"v": 2})
        finally:
            json.dump = real
        self.assertEqual(json.loads(slurp(target, "r")), {"v": 1})
        self.assertEqual([f for f in os.listdir(self.tmp) if "tmp" in f], [])

    def test_pull_no_probe_flag(self):
        self.run_main("push")
        self.fake.remote_edit("HRD-001", status="review")
        self.fake.calls.clear()
        self.assertEqual(self.run_main("pull", "--no-probe")[0], 2)
        self.assertNotIn("probe", [c[0] for c in self.fake.calls])

    def test_login_failure_exit_3_without_leaking_password(self):
        self.write_env(0o600)
        with open(self.env, "a") as fh:
            fh.write("")
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            code = hs.main(["push", "--outbox", self.outbox, "--state", self.state, "--env-file", self.env],
                           environ={"HT_PASSWORD": "wrong-password-xyz"}, transport=self.fake.transport)
        self.assertEqual(code, 3)
        self.assertNotIn("wrong-password-xyz", out.getvalue() + err.getvalue())


class FieldCoverageTests(unittest.TestCase):                              # finding 8
    def test_every_outbox_payload_key_is_either_pushed_or_explicitly_unpushed(self):
        self.assertEqual(hs.UNPUSHED_PAYLOAD_FIELDS, {
            "closure_criteria", "composes_with", "forensic_anchor", "assigned_to", "created_by", "created_at",
            "last_modified", "current_location"})
        self.assertEqual(hs.PUSHED_PAYLOAD_FIELDS & hs.UNPUSHED_PAYLOAD_FIELDS, set())
        root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
        path = os.environ.get("HRD_OUTBOX") or os.path.join(root, "docs", "tracking", "helixtrack-outbox.json")
        if not os.path.exists(path):
            self.skipTest("outbox not present")
        keys = set()
        for r in json.loads(slurp(path, "r"))["records"]:
            keys |= set(r["payload"])
        self.assertEqual(keys - hs.PUSHED_PAYLOAD_FIELDS - hs.UNPUSHED_PAYLOAD_FIELDS, set())

    def test_unpushed_fields_do_not_change_the_remote_hash(self):
        a = rec(1)
        b = copy.deepcopy(a)
        b["payload"].update(last_modified="2030-01-01", assigned_to="X", composes_with=["HRD-002"],
                            created_by="Y", forensic_anchor="z", closure_criteria="c", created_at="t",
                            current_location="Fixed")
        self.assertEqual(hs.fields_hash(hs.build_fields(a)), hs.fields_hash(hs.build_fields(b)))


if __name__ == "__main__":
    unittest.main()


# ======================================================================= review round 3 (hardening gaps 1-5)
import fcntl  # noqa: E402
import stat  # noqa: E402
import http.client  # noqa: E402
import sqlite3  # noqa: E402
import subprocess  # noqa: E402
import time  # noqa: E402


class _HtmlOk(http.server.BaseHTTPRequestHandler):
    """HTTP 200 whose body is HTML, not JSON (captive portal / proxy error page)."""
    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length") or 0))
        body = b"<html><body>maintenance</body></html>"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


class Gap1TransportErrorTests(unittest.TestCase):
    def test_http_client_and_value_errors_become_transport_error(self):
        for exc in (http.client.BadStatusLine("garbage"), http.client.IncompleteRead(b"ab", 10),
                    http.client.RemoteDisconnected("eof"), ValueError("Expecting value"),
                    UnicodeDecodeError("utf-8", b"\xff", 0, 1, "bad")):
            def boom(*a, _e=exc, **k):
                raise _e
            c = hs.CoreClient(boom, "u", "p")
            with self.assertRaises(hs.TransportError, msg=type(exc).__name__):
                c.login()

    def test_real_transport_html_200_body_is_transport_error_not_traceback(self):
        srv = serve(_HtmlOk)
        stop(self, srv)
        c = hs.CoreClient(hs.urllib_transport("http://127.0.0.1:%d" % srv.server_address[1]), "u", "p")
        with self.assertRaises(hs.TransportError):
            c.login()

    def test_main_exit_3_with_partial_report_when_transport_raises_midway(self):
        for exc in (http.client.BadStatusLine("x"), ValueError("html")):
            t = MainTests("test_clean_push_pull_compare_exit_zero")
            t.setUp()
            try:
                n = {"c": 0}
                real = t.fake.transport

                def flaky(method, path, headers, body, _real=real, _n=n, _e=exc):
                    _n["c"] += 1
                    if _n["c"] == 5:
                        raise _e
                    return _real(method, path, headers, body)
                t.fake.transport = flaky
                err = io.StringIO()
                argv = ["push", "--outbox", t.outbox, "--state", t.state, "--report", t.report, "--env-file", t.env]
                with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
                    code = hs.main(argv, environ={}, transport=flaky)
                self.assertEqual(code, 3, type(exc).__name__)
                rep = json.loads(slurp(t.report, "r"))
                self.assertTrue(rep["aborted"])
                self.assertEqual(rep["error"]["type"], "TransportError")
            finally:
                __import__("shutil").rmtree(t.tmp, True)


class _MainBase(unittest.TestCase):
    """MainTests' fixture without inheriting its tests."""
    setUp = MainTests.setUp
    p = MainTests.p
    write_env = MainTests.write_env
    run_main = MainTests.run_main


class Gap2CorruptInputTests(_MainBase):

    def put(self, path, data):
        with open(path, "wb") as fh:
            fh.write(data)

    def assert_refused_before_login(self, what):
        self.fake.calls.clear()
        code, _, err = self.run_main("push")
        self.assertEqual(code, 3, what)
        self.assertNotIn("Traceback", err, what)
        self.assertEqual(self.fake.calls, [], what + ": a request was made before validation")
        self.assertTrue(json.loads(slurp(self.report, "r"))["aborted"], what)

    def test_corrupt_state_files_exit_3_before_login(self):
        for what, data in (("empty", b""), ("truncated", b'{"items": {"HRD-001": {"tic'), ("nul", b"\x00\x00\x00"),
                           ("list", b"[]"), ("items-list", b'{"items": []}'), ("items-str", b'{"items": "x"}'),
                           ("item-str", b'{"items": {"HRD-001": "x"}}'), ("item-null", b'{"items": {"HRD-001": null}}'),
                           ("item-list", b'{"items": {"HRD-001": []}}'),
                           ("bad-ticket-id", b'{"items": {"HRD-001": {"ticket_id": 5}}}'),
                           ("bad-hash", b'{"items": {"HRD-001": {"ticket_id": "t", "base_hash": 7}}}'),
                           ("bad-schema", b'{"schema": "other", "items": {}}'), ("scalar", b"7"),
                           ("binary", b"\xff\xfe\x00")):
            self.put(self.state, data)
            self.assert_refused_before_login("state " + what)
        self.put(self.state, b'{"schema": "%s", "items": {}}' % hs.SCHEMA.encode())
        self.assertEqual(self.run_main("push")[0], 0)

    def test_corrupt_outbox_exit_3_before_login(self):
        good = self.recs[0]
        for what, doc in (("list", []), ("no-records", {}), ("empty-records", {"records": []}),
                          ("records-str", {"records": "x"}), ("record-str", {"records": ["HRD-001"]}),
                          ("record-null", {"records": [None]}),
                          ("no-payload", {"records": [{"client_key": "HRD-001"}]}),
                          ("payload-str", {"records": [{"client_key": "HRD-001", "payload": "x"}]}),
                          ("no-key", {"records": [{"payload": good["payload"]}]}),
                          ("bad-key", {"records": [dict(good, client_key="X-1")]}),
                          ("dup-key", {"records": [good, good]}),
                          ("no-title", {"records": [dict(good, payload={k: v for k, v in good["payload"].items()
                                                                        if k != "title"})]}),
                          ("desc-null", {"records": [dict(good, payload=dict(good["payload"], description=None))]})):
            self.put(self.outbox, json.dumps(doc).encode())
            self.assert_refused_before_login("outbox " + what)
        for what, data in (("empty", b""), ("truncated", b'{"records": [{'), ("nul", b"\x00")):
            self.put(self.outbox, data)
            self.assert_refused_before_login("outbox " + what)

    def test_unreadable_history_db_exit_3_before_login(self):
        db = self.p("hist.db")
        self.put(db, b"this is not a database" * 50)
        self.fake.calls.clear()
        code, _, err = self.run_main("push", "--db", db)
        self.assertEqual(code, 3)
        self.assertNotIn("Traceback", err)
        self.assertEqual(self.fake.calls, [])


class Gap3LockTests(_MainBase):
    def test_held_lock_refuses_second_instance_exit_3_without_requests_or_report_clobber(self):
        fd = os.open(self.state + ".lock", os.O_RDWR | os.O_CREAT, 0o600)
        self.addCleanup(os.close, fd)
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        import signal

        def hung(*a):
            raise AssertionError("main() blocked on the held lock instead of failing fast")
        old = signal.signal(signal.SIGALRM, hung)
        self.addCleanup(signal.signal, signal.SIGALRM, old)
        self.addCleanup(signal.alarm, 0)
        for cmd in ("push", "pull", "compare"):
            self.fake.calls.clear()
            signal.alarm(10)
            code, _, err = self.run_main(cmd)
            signal.alarm(0)
            self.assertEqual(code, 3, cmd)
            self.assertIn("lock", err)
            self.assertEqual(self.fake.calls, [])
            self.assertFalse(os.path.exists(self.report), "must not clobber the running instance's report")
            self.assertFalse(os.path.exists(self.state))
        fcntl.flock(fd, fcntl.LOCK_UN)
        self.assertEqual(self.run_main("push")[0], 0)

    def test_lock_is_released_after_every_exit_path(self):
        self.assertEqual(self.run_main("push")[0], 0)
        self.write_env(0o644)
        self.assertEqual(self.run_main("push")[0], 3)             # error path
        self.write_env(0o600)
        self.fake.remote_edit("HRD-001", description="x")
        self.assertEqual(self.run_main("push")[0], 2)             # conflict path
        self.assertEqual(self.run_main("compare")[0], 2)          # compare early-return path
        fd = os.open(self.state + ".lock", os.O_RDWR)
        self.addCleanup(os.close, fd)
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)            # raises if still held

    def test_two_concurrent_processes_second_is_refused(self):
        gate = threading.Event()

        class Slow(http.server.BaseHTTPRequestHandler):
            def do_POST(h):
                h.rfile.read(int(h.headers.get("Content-Length") or 0))
                h.server.hits.append(1)
                gate.wait(20)
                body = json.dumps({"errorCode": 1004, "errorMessage": "stop"}).encode()
                h.send_response(401)
                h.send_header("Content-Length", str(len(body)))
                h.end_headers()
                h.wfile.write(body)

            def log_message(h, *a):
                pass
        srv = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Slow)
        srv.daemon_threads = True
        srv.hits = []
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        self.addCleanup(srv.server_close)
        self.addCleanup(srv.shutdown)
        self.addCleanup(gate.set)
        self.write_env(0o600, "http://127.0.0.1:%d" % srv.server_address[1])
        script = os.path.join(os.path.dirname(os.path.abspath(__file__)), "helixtrack_sync.py")
        argv = [sys.executable, "-I", script, "push", "--outbox", self.outbox, "--state", self.state,
                "--report", self.report, "--env-file", self.env]
        a = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            t0 = time.time()
            while not srv.hits and time.time() - t0 < 15:
                time.sleep(0.05)
            self.assertEqual(len(srv.hits), 1, "first instance never reached the server")
            b = subprocess.run(argv, capture_output=True, text=True, timeout=8)
            self.assertEqual(b.returncode, 3, b.stderr)
            self.assertIn("lock", b.stderr)
            self.assertEqual(len(srv.hits), 1, "second instance must not have contacted the server")
            # the running first instance wrote its in-progress marker; the refused second one wrote nothing else
            self.assertEqual(json.loads(slurp(self.report, "r"))["error"]["type"], "InProgress")
        finally:
            gate.set()
            ao, ae = a.communicate(timeout=20)
        self.assertEqual(a.returncode, 3, ae)                        # first one ended on its own AuthError
        self.assertEqual(self.run_main("push")[0], 0)                # lock free again


class Gap4UrlDifferentialTests(unittest.TestCase):
    def test_userinfo_backslash_and_friends_refused(self):
        for bad in ("http://evil@127.0.0.1", "http://evil.com\\@127.0.0.1", "http://127.0.0.1\\@evil.com",
                    "http://user:pw@127.0.0.1:8080", "https://user@example.com", "http://127.0.0.1:80#@evil.com",
                    "http://127.0.0.1?@evil.com", "http://127.0.0.1\\evil.com", "http://127.0.0.1 ",
                    "http://127.0.0.1\t", "http://127.0.0.1\n", "http://127.0.0.1:bad", "http://127.0.0.1:99999",
                    "http://127.0.0.1\u2024evil", None, 5,
                    "http://127.0.0.1/api\\x", "http://127.0.0.1/#frag", "http://127.0.0.1/?q=1", "http://localhost/a?b"):
            with self.assertRaises(hs.SyncError, msg=repr(bad)):
                hs.check_base_url(bad)
            if isinstance(bad, str):
                with self.assertRaises(hs.SyncError, msg=repr(bad)):
                    hs.urllib_transport(bad)

    def test_refusal_message_never_echoes_userinfo(self):
        with self.assertRaises(hs.SyncError) as cm:
            hs.check_base_url("http://admin:hunter2@127.0.0.1")
        self.assertNotIn("hunter2", str(cm.exception))

    def test_plain_urls_still_accepted(self):
        for ok in ("http://127.0.0.1:18417", "http://localhost:8080/api", "https://core.example.com", "HTTP://[::1]:80"):
            hs.check_base_url(ok)


class Gap5GuardTests(_MainBase):
    def test_env_file_any_group_or_other_bit_refused(self):
        for mode in (0o640, 0o604, 0o660, 0o606, 0o644, 0o601, 0o610, 0o602, 0o620, 0o604, 0o664, 0o666, 0o755):
            self.write_env(mode)
            code, _, err = self.run_main("push")
            self.assertEqual(code, 3, oct(mode))
            self.assertIn("env file", err, oct(mode))
            self.assertEqual(self.fake.calls, [], oct(mode))
        for mode in (0o600, 0o400, 0o700):
            os.chmod(self.env, 0o600)
            self.write_env(0o600)
            os.chmod(self.env, mode)
            self.assertNotIn("env file", self.run_main("push")[2], oct(mode))

    def test_is_loopback_rejects_lookalike_hostnames(self):
        for host in ("evillocalhost", "localhost.evil.com", "notlocalhost", "xlocalhost", "127.0.0.1.evil.com", ""):
            self.assertFalse(hs._is_loopback(host), host)
            if host:
                with self.assertRaises(hs.SyncError, msg=host):
                    hs.check_base_url("http://%s:8080" % host)
        for host in ("localhost", "LOCALHOST", "127.0.0.1", "::1", "127.9.9.9"):
            self.assertTrue(hs._is_loopback(host), host)

    def test_history_db_opened_read_only_and_left_untouched(self):
        d = tempfile.mkdtemp()
        self.addCleanup(__import__("shutil").rmtree, d, True)
        db = os.path.join(d, "reg.db")
        con = sqlite3.connect(db)
        con.execute("PRAGMA journal_mode=WAL")
        con.execute("CREATE TABLE item_history(sequence INTEGER, atm_id TEXT, event_type TEXT, actor TEXT, "
                    "on_date TEXT, reason TEXT)")
        con.execute("INSERT INTO item_history VALUES(1,'HRD-001','Reopened','AI','2026-01-01','why')")
        con.commit()
        con.close()
        before = (slurp(db), os.stat(db).st_mtime_ns, sorted(os.listdir(d)))
        os.chmod(d, 0o555)                       # a plain read-write open of a WAL db needs to create -shm/-wal
        self.addCleanup(os.chmod, d, 0o755)
        got = hs.load_history(db)
        self.assertEqual(got["HRD-001"][0]["reason"], "why")
        self.assertEqual((slurp(db), os.stat(db).st_mtime_ns, sorted(os.listdir(d))), before)
        os.chmod(d, 0o755)
        hs.load_history(db)
        self.assertEqual((slurp(db), os.stat(db).st_mtime_ns, sorted(os.listdir(d))), before)

    def test_known_ticket_missing_from_list_but_readable_is_not_remote_deleted(self):
        fake, c, recs = fresh(2)
        st = {}
        hs.push(c, PID, recs, st)
        real = c.list_tickets
        c.list_tickets = lambda pid: [t for t in real(pid) if not t["title"].startswith("[HRD-001]")]
        for rep in (hs.push(c, PID, recs, st), hs.pull(c, PID, recs, st)):
            self.assertNotIn("remote-deleted", actions(rep).values())
            self.assertIn(actions(rep)["HRD-001"], ("noop", "in-sync"))
        self.assertEqual(len(fake.tickets), 2)                 # and nothing re-created

    def test_key_must_be_anchored_at_title_start(self):
        fake, c, recs = fresh(1)
        proj = fake.projects["proj-1"]["id"]
        fake.transport("POST", "/do", {}, {"action": "authenticate", "data": {"username": "u", "password": SECRET_PW}})
        for title in ("Notes on [HRD-001] migration", "see [HRD-001]", " [HRD-001] leading space", "x[HRD-001] y"):
            self.assertEqual(hs._claims({"title": title, "description": ""}), set(), title)
        self.assertEqual(hs._claims({"title": "[HRD-001] ok", "description": ""}), {"HRD-001"})
        # end-to-end: an unrelated ticket merely mentioning the key is neither claimed nor overwritten
        t = fake.transport("POST", "/do", {}, {"action": "create", "object": "ticket", "jwt": SECRET_JWT,
                                               "data": {"project_id": proj, "title": "Notes on [HRD-001] x",
                                                        "description": "mine"}})
        rep = hs.push(c, PID, recs, {})
        self.assertEqual(actions(rep)["HRD-001"], "create")
        self.assertEqual(len(fake.tickets), 2)

    def test_http_200_with_non_ok_error_code_and_unknown_4xx_fail(self):
        for status, body in ((200, {"errorCode": 1005, "errorMessage": "nope"}), (200, {"errorCode": 0}),
                             (200, {"data": {"items": []}}), (200, {}), (200, []), (200, None),
                             (418, {"errorCode": 1, "errorMessage": "teapot"}), (422, {"errorCode": 9}),
                             (429, {"errorCode": 9, "errorMessage": "slow"})):
            fake = FakeCore()
            fake.projects["proj-1"] = {"id": PID, "identifier": "HRD", "title": "x"}
            c = client(fake)
            fake.fail_next = (status, body)
            with self.assertRaises(hs.CoreError, msg=repr((status, body))):
                c.list_tickets(PID)

    def test_persistent_401_relogin_is_bounded(self):
        calls = {"auth": 0, "do": 0}

        def t(method, path, headers, body):
            if body["action"] == "authenticate":
                calls["auth"] += 1
                return 200, {"errorCode": -1, "data": {"token": "tok"}}
            calls["do"] += 1
            return 401, {"errorCode": 1003, "errorMessage": "expired"}
        c = hs.CoreClient(t, "u", "p")
        c.login()
        with self.assertRaises(hs.AuthError):
            c.list_tickets(PID)
        self.assertEqual(calls, {"auth": 2, "do": 2})

    def test_version_retry_is_capped_at_one(self):
        fake, c, recs = fresh(1)
        st = {}
        hs.push(c, PID, recs, st)
        fake.remote_edit("HRD-001", status="open")           # not content: keep hash == base
        real = fake.transport
        n = {"m": 0}

        def moving(method, path, headers, body):
            d = body.get("data") or {}
            if body.get("action") == "modify" and d.get("version") != hs.PROBE_VERSION:
                n["m"] += 1
                if n["m"] > 6:
                    raise AssertionError("retry loop is unbounded")
                by_key(fake, "HRD-001")["version"] += 1       # another writer wins every race, content unchanged
            return real(method, path, headers, body)
        c2 = hs.CoreClient(moving, "hrd_sync", SECRET_PW)
        c2.login()
        rec2 = [rec(1, sev="Low")]
        rep = hs.push(c2, PID, rec2, st)
        a = rep.actions[0]
        self.assertEqual(a["action"], "conflict")
        self.assertEqual(a["reason"], "version kept moving")
        self.assertEqual(a["retries"], 1)
        self.assertEqual(n["m"], 2)

    def test_every_conflict_kind_drives_exit_code_2(self):
        for kind in ("conflict", "unpushable", "duplicate", "remote-deleted"):
            r = hs.Report("push")
            r.add("HRD-001", kind)
            self.assertTrue(r.has_conflicts, kind)
        r = hs.Report("push")
        for kind in ("create", "update", "noop"):
            r.add("HRD-001", kind)
        self.assertFalse(r.has_conflicts)

    def test_unpushable_type_change_exits_2_via_main(self):
        self.assertEqual(self.run_main("push")[0], 0)
        by_key(self.fake, "HRD-001")["type"] = "bug"          # registry says Task: Core cannot modify type
        code, out, _ = self.run_main("push")
        self.assertEqual(json.loads(out)["counts"].get("unpushable"), 1)
        self.assertEqual(code, 2)

    def test_redact_handles_lists_at_any_depth(self):
        out = hs.redact({"a": [{"jwt": "J1"}, [{"password": "P1"}, {"x": [{"token": "T1"}]}]], "b": ["Authorization"]})
        s = json.dumps(out)
        for bad in ("J1", "P1", "T1"):
            self.assertNotIn(bad, s)
        self.assertEqual(out["b"], ["Authorization"])
        self.assertEqual(hs.redact([{"secret": "S"}, 3]), [{"secret": "[REDACTED]"}, 3])

    def test_compare_report_is_written_atomically(self):
        self.run_main("push")
        replaced = []
        real = os.replace
        os.replace = lambda a, b, _r=real: (replaced.append(os.path.abspath(b)), _r(a, b))[1]
        try:
            self.assertEqual(self.run_main("compare")[0], 0)
        finally:
            os.replace = real
        self.assertIn(os.path.abspath(self.report), replaced)
        self.assertEqual(json.loads(slurp(self.report, "r"))["mode"], "compare")
        self.assertEqual([f for f in os.listdir(self.tmp) if "tmp" in f], [])


# ======================================================================= round 3 residual gaps (G6 series)
class Gap6ResidualTests(_MainBase):
    # ---- (a) baseline is never advanced before read-back verification succeeds
    def test_update_with_dropped_write_leaves_baseline_so_next_run_is_update_not_conflict(self):
        fake, c, recs = fresh(1)
        st = {}
        hs.push(c, PID, recs, st)
        base0 = st["items"]["HRD-001"]["base_hash"]
        ver0 = st["items"]["HRD-001"]["version"]
        dropped = {"n": 0}

        def drop(action, obj, data):                       # 200 + new version, but nothing applied
            if (action, obj) == ("modify", "ticket"):
                dropped["n"] += 1
                return 200, {"errorCode": -1, "data": {"ticket": {"id": data["id"], "updated": True, "version": 99}}}
        fake.fail_on = drop
        new = [rec(1, sev="Low")]
        r = hs.push(c, PID, new, st)
        self.assertEqual(r.error["type"], "ReadbackMismatch")
        self.assertEqual(dropped["n"], 1)
        self.assertEqual(st["items"]["HRD-001"]["base_hash"], base0, "baseline advanced before read-back verify")
        self.assertEqual(st["items"]["HRD-001"]["version"], ver0)
        fake.fail_on = None
        r2 = hs.push(c, PID, new, st)
        self.assertEqual(actions(r2), {"HRD-001": "update"})
        self.assertFalse(r2.has_conflicts)

    def test_create_with_ignored_status_keeps_as_created_baseline_then_resumes_as_update(self):
        fake, c, recs = fresh(1)
        recs = [rec(1, status="Fixed (→ Fixed.md)")]
        fake.ignore_status = True
        st = {}
        r = hs.push(c, PID, recs, st)
        self.assertTrue(r.aborted)
        desired = hs.build_fields(recs[0])
        self.assertEqual(st["items"]["HRD-001"]["base_hash"], hs.fields_hash(dict(desired, status="open")))
        self.assertNotEqual(st["items"]["HRD-001"]["base_hash"], hs.fields_hash(desired))
        fake.ignore_status = False
        r2 = hs.push(c, PID, recs, st)
        self.assertEqual(actions(r2), {"HRD-001": "update"})
        self.assertEqual(len(fake.tickets), 1)

    def test_create_with_silently_dropped_status_write_keeps_as_created_baseline(self):
        fake, c, _ = fresh(1)
        recs = [rec(1, status="Fixed (→ Fixed.md)")]

        def drop(action, obj, data):                       # 200 OK but nothing applied
            if (action, obj) == ("modify", "ticket"):
                return 200, {"errorCode": -1, "data": {"ticket": {"id": data["id"], "updated": True, "version": 9}}}
        fake.fail_on = drop
        st = {}
        r = hs.push(c, PID, recs, st)
        self.assertEqual(r.error["type"], "ReadbackMismatch")
        desired = hs.build_fields(recs[0])
        self.assertEqual(st["items"]["HRD-001"]["base_hash"], hs.fields_hash(dict(desired, status="open")))
        fake.fail_on = None
        r2 = hs.push(c, PID, recs, st)
        self.assertEqual(actions(r2), {"HRD-001": "update"})
        self.assertFalse(r2.has_conflicts)

    # ---- (b) history DB: read-only, a missing DB is never created
    def test_missing_history_db_is_refused_and_not_created(self):
        missing = self.p("nope.db")
        with self.assertRaises(hs.SyncError):
            hs.load_history(missing)
        self.assertFalse(os.path.exists(missing))
        code, _, err = self.run_main("push", "--db", missing)
        self.assertEqual(code, 3)
        self.assertNotIn("Traceback", err)
        self.assertFalse(os.path.exists(missing))
        self.assertEqual(self.fake.calls, [])
        self.assertEqual([f for f in os.listdir(self.tmp) if f.endswith((".db", "-wal", "-shm", "-journal"))], [])

    # ---- (c) env file: checked before it is read, must be a regular file
    def test_refused_env_file_is_never_read_environment_untouched(self):
        for mode in (0o644, 0o640, 0o604):
            self.write_env(0o600)
            os.chmod(self.env, mode)
            environ = {}
            with self.assertRaises(hs.SyncError, msg=oct(mode)):
                hs.load_env_file(self.env, environ)
            self.assertEqual(environ, {}, oct(mode))
        os.chmod(self.env, 0o600)
        environ = {"HT_USER": "keep"}
        hs.load_env_file(self.env, environ)
        self.assertEqual(environ["HT_USER"], "keep")          # setdefault semantics preserved
        self.assertIn("HT_PASSWORD", environ)

    def test_env_file_must_be_regular_file_fifo_dir_missing_all_exit_3_without_hanging(self):
        import signal

        def hung(*a):
            raise AssertionError("hung on a non-regular env file")
        old = signal.signal(signal.SIGALRM, hung)
        self.addCleanup(signal.signal, signal.SIGALRM, old)
        self.addCleanup(signal.alarm, 0)
        fifo, dr, missing = self.p("fifo.env"), self.p("dir.env"), self.p("missing.env")
        os.mkfifo(fifo, 0o600)
        os.mkdir(dr, 0o700)
        for path in (fifo, dr, missing):
            environ = {}
            signal.alarm(8)
            with self.assertRaises(hs.SyncError, msg=path):
                hs.load_env_file(path, environ)
            signal.alarm(0)
            self.assertEqual(environ, {})
            self.env = path
            signal.alarm(8)
            code, _, err = self.run_main("push")
            signal.alarm(0)
            self.assertEqual(code, 3, path)
            self.assertNotIn("Traceback", err)
            self.assertEqual(self.fake.calls, [])

    def test_env_file_non_utf8_is_syncerror_and_environment_untouched(self):
        with open(self.env, "wb") as fh:
            fh.write(b"HT_USER=a\n\xff\xfe=\n")
        os.chmod(self.env, 0o600)
        environ = {}
        with self.assertRaises(hs.SyncError):
            hs.load_env_file(self.env, environ)
        self.assertEqual(environ, {})

    # ---- (d) 0600 + atomic for every file the tool creates
    def with_umask0(self):
        old = os.umask(0)
        self.addCleanup(os.umask, old)

    def mode(self, path):
        return stat.S_IMODE(os.stat(path).st_mode)

    def test_state_report_compare_report_and_lock_are_0600_even_under_umask_0(self):
        self.with_umask0()
        self.assertEqual(self.run_main("push")[0], 0)
        for f in (self.state, self.report, self.state + ".lock"):
            self.assertEqual(self.mode(f), 0o600, f)
        os.unlink(self.report)
        self.assertEqual(self.run_main("compare")[0], 0)
        self.assertEqual(self.mode(self.report), 0o600)

    def test_abort_report_is_0600_and_written_atomically(self):
        self.with_umask0()
        self.write_env(0o644)
        replaced, real = [], os.replace
        os.replace = lambda a, b, _r=real: (replaced.append(os.path.abspath(b)), _r(a, b))[1]
        try:
            self.assertEqual(self.run_main("push")[0], 3)
        finally:
            os.replace = real
        self.assertIn(os.path.abspath(self.report), replaced)
        self.assertEqual(self.mode(self.report), 0o600)
        self.assertTrue(json.loads(slurp(self.report, "r"))["aborted"])
        self.assertEqual([f for f in os.listdir(self.tmp) if "tmp" in f], [])


# ======================================================================= round 3c (env size cap, fd leak, WAL-aware history)
import shutil  # noqa: E402


def _make_wal_db(d):
    """item_history table checkpointed into the main file + one MORE row living only in the -wal."""
    db = os.path.join(d, "reg.db")
    w = sqlite3.connect(db)
    w.execute("PRAGMA journal_mode=WAL")
    w.execute("CREATE TABLE item_history(sequence INTEGER, atm_id TEXT, event_type TEXT, actor TEXT, "
              "on_date TEXT, reason TEXT)")
    w.execute("INSERT INTO item_history VALUES(1,'HRD-001','Reopened','AI','2026-01-01','why1')")
    w.commit()
    w.execute("PRAGMA wal_checkpoint(TRUNCATE)")
    w.execute("PRAGMA wal_autocheckpoint=0")
    w.execute("INSERT INTO item_history VALUES(2,'HRD-001','Closed','AI','2026-01-02','pending-in-wal')")
    w.commit()
    return w, db


class Gap7Tests(_MainBase):
    # ---- (1) env file size cap
    def test_env_file_size_is_capped(self):
        cap = hs.ENV_MAX_BYTES
        self.assertEqual(cap, 64 * 1024)
        body = b"HT_USER=u\n"
        with open(self.env, "wb") as fh:                       # exactly at the cap: accepted
            fh.write(body + b"#" * (cap - len(body)))
        os.chmod(self.env, 0o600)
        environ = {}
        hs.load_env_file(self.env, environ)
        self.assertEqual(environ, {"HT_USER": "u"})
        with open(self.env, "ab") as fh:                       # one byte over: refused
            fh.write(b"#")
        environ = {}
        with self.assertRaises(hs.SyncError):
            hs.load_env_file(self.env, environ)
        self.assertEqual(environ, {})

    def test_huge_sparse_env_file_refused_cleanly_exit_3(self):
        with open(self.env, "ab") as fh:                       # valid settings first, then 64 MiB of sparse padding
            fh.truncate(64 * 1024 * 1024)
        os.chmod(self.env, 0o600)
        code, _, err = self.run_main("push")
        self.assertEqual(code, 3)
        self.assertIn("env file", err)
        self.assertNotIn("Traceback", err)
        self.assertNotIn("MemoryError", err)
        self.assertEqual(self.fake.calls, [])

    def test_env_file_cap_reads_at_most_cap_plus_one_bytes(self):
        with open(self.env, "wb") as fh:
            fh.write(b"HT_USER=u\n")
        os.chmod(self.env, 0o600)
        sizes = []
        real_fdopen = os.fdopen

        def spy(fd, *a, **k):
            fh = real_fdopen(fd, *a, **k)
            real_read = fh.read
            fh.read = lambda n=-1: (sizes.append(n), real_read(n))[1]
            return fh
        os.fdopen = spy
        try:
            hs.load_env_file(self.env, {})
        finally:
            os.fdopen = real_fdopen
        self.assertEqual(sizes, [hs.ENV_MAX_BYTES + 1])

    # ---- (2) no descriptor leaks on the refusal routes
    def test_refused_env_files_do_not_leak_file_descriptors(self):
        bad_mode = self.p("mode.env")
        with open(bad_mode, "w") as fh:
            fh.write("HT_USER=u\n")
        os.chmod(bad_mode, 0o644)
        fifo, dr = self.p("f.env"), self.p("d.env")
        os.mkfifo(fifo, 0o600)
        os.mkdir(dr, 0o700)
        big = self.p("big.env")
        with open(big, "wb") as fh:
            fh.truncate(hs.ENV_MAX_BYTES + 5)
        os.chmod(big, 0o600)

        def nfd():
            return len(os.listdir("/proc/self/fd"))
        for path in (bad_mode, fifo, dr, big):                   # warm up
            with self.assertRaises(hs.SyncError):
                hs.load_env_file(path, {})
        before = nfd()
        for _ in range(25):
            for path in (bad_mode, fifo, dr, big):
                with self.assertRaises(hs.SyncError):
                    hs.load_env_file(path, {})
        self.assertEqual(nfd(), before, "file descriptors leaked on the refusal path")

    # ---- (3) WAL-aware history
    def test_uncheckpointed_wal_rows_are_visible_and_nothing_is_created_or_modified(self):
        d = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, d, True)
        w, db = _make_wal_db(d)
        self.addCleanup(w.close)
        files = sorted(os.listdir(d))
        self.assertIn("reg.db-wal", files)
        before = (slurp(db), slurp(db + "-wal"), files)
        warnings, uris, real = [], [], sqlite3.connect

        def spy(target, *a, **k):
            uris.append(target)
            return real(target, *a, **k)
        sqlite3.connect = spy
        try:
            hist = hs.load_history(db, warnings)
        finally:
            sqlite3.connect = real
        self.assertEqual([h["reason"] for h in hist["HRD-001"]], ["why1", "pending-in-wal"])
        self.assertEqual(warnings, [])
        self.assertTrue(uris and all("mode=ro" in u for u in uris), uris)      # never a read-write open
        self.assertEqual((slurp(db), slurp(db + "-wal"), sorted(os.listdir(d))), before)

    def test_wal_without_shm_falls_back_to_immutable_with_warning_and_creates_nothing(self):
        d, d2 = tempfile.mkdtemp(), tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, d, True)
        self.addCleanup(shutil.rmtree, d2, True)
        w, db = _make_wal_db(d)
        self.addCleanup(w.close)
        copy = os.path.join(d2, "c.db")
        shutil.copyfile(db, copy)
        shutil.copyfile(db + "-wal", copy + "-wal")              # -wal with content, no -shm
        warnings = []
        hist = hs.load_history(copy, warnings)
        self.assertEqual([h["reason"] for h in hist["HRD-001"]], ["why1"])      # stale view, but flagged
        self.assertEqual(len(warnings), 1)
        self.assertIn("-wal", warnings[0])
        self.assertIn("stale", warnings[0])
        self.assertEqual(sorted(os.listdir(d2)), ["c.db", "c.db-wal"])

    def test_wal_aware_open_failure_falls_back_with_warning(self):
        d = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, d, True)
        w, db = _make_wal_db(d)
        self.addCleanup(w.close)
        real = sqlite3.connect

        def picky(target, *a, **k):
            if "immutable=1" not in target:
                raise sqlite3.OperationalError("attempt to write a readonly database")
            return real(target, *a, **k)
        sqlite3.connect = picky
        try:
            warnings = []
            hist = hs.load_history(db, warnings)
        finally:
            sqlite3.connect = real
        self.assertEqual([h["reason"] for h in hist["HRD-001"]], ["why1"])
        self.assertEqual(len(warnings), 1)
        self.assertIn("stale", warnings[0])

    def test_missing_db_still_refused_and_nothing_created(self):
        missing = self.p("none.db")
        with self.assertRaises(hs.SyncError):
            hs.load_history(missing, [])
        self.assertEqual([f for f in os.listdir(self.tmp) if f.startswith("none.db")], [])

    def test_history_warning_reaches_the_report_and_stderr(self):
        d, d2 = tempfile.mkdtemp(), tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, d, True)
        self.addCleanup(shutil.rmtree, d2, True)
        w, db = _make_wal_db(d)
        self.addCleanup(w.close)
        copy = os.path.join(d2, "c.db")
        shutil.copyfile(db, copy)
        shutil.copyfile(db + "-wal", copy + "-wal")
        for cmd in ("push", "pull", "compare"):
            code, _, err = self.run_main(cmd, "--db", copy)
            self.assertIn("warning", err.lower(), cmd)
            rep = json.loads(slurp(self.report, "r"))
            self.assertEqual(len(rep["warnings"]), 1, cmd)
            self.assertIn("stale", rep["warnings"][0])


# ------------------------------------------------------------------ fix round 1 (review NO-GO of 662a1f2)
import unicodedata  # noqa: E402

HOSTILE = ("bad\x1b]0;PWNED\x07\x1b]52;c;AAAA\x07\x1b[2J\x1b[31mRED\x9b31m\x85‮⁦x\x00y\rz​"
           " \x7f")


def _no_controls(case, text, what=""):
    for ch in text:
        if ch in "\n":
            continue
        cat = unicodedata.category(ch)
        case.assertFalse(cat[0] == "C" or cat in ("Zl", "Zp"), "%s: unescaped U+%04X in %r" % (what, ord(ch), text))


class SafeTextTests(unittest.TestCase):
    def test_every_control_class_is_escaped(self):
        out = hs.safe_text(HOSTILE, 10000)
        _no_controls(self, out, "safe_text")
        self.assertIn("RED", out)
        self.assertNotIn("\x1b", out)

    def test_length_capped_and_non_str_accepted(self):
        self.assertLessEqual(len(hs.safe_text("a" * 100000)), 400)
        self.assertEqual(hs.safe_text(None), "None")


class B2TerminalInjectionTests(_MainBase):
    """Server text must never reach stderr/report unescaped (login, listing, compare, per-item, warnings)."""

    def hostile(self, action, obj, status):
        def inj(a, o, d):
            if (a, o) == (action, obj):
                return status, {"errorCode": 1004, "errorMessage": HOSTILE, "errorMessageLocalised": HOSTILE}
        self.fake.fail_on = inj

    def check(self, cmd, action, obj, status, what):
        self.hostile(action, obj, status)
        code, out, err = self.run_main(cmd)
        self.assertEqual(code, 3, what)
        for stream, text in (("stderr", err), ("stdout", out), ("report", slurp(self.report, "r"))):
            self.assertNotIn("\x1b", text, what + " " + stream)
            self.assertNotIn("\x9b", text, what + " " + stream)
            self.assertNotIn("\x07", text, what + " " + stream)
            self.assertNotIn("‮", text, what + " " + stream)
        _no_controls(self, err, what + " stderr")

    def test_login_401(self):
        orig = self.fake.transport

        def tr(m, p, h, b):
            if b.get("action") == "authenticate":
                return 401, {"errorCode": 1004, "errorMessage": HOSTILE}
            return orig(m, p, h, b)
        self.fake.transport = tr
        code, out, err = self.run_main("push")
        self.assertEqual(code, 3)
        _no_controls(self, err, "login stderr")
        for text in (err, out, slurp(self.report, "r")):
            self.assertNotIn("\x1b", text)
            self.assertNotIn("\x07", text)

    def test_listing_500_push_pull_compare(self):
        for cmd in ("push", "pull", "compare"):
            self.check(cmd, "list", "ticket", 500, "list " + cmd)

    def test_project_list_401_and_403(self):
        for st in (401, 403, 404, 400, 409):
            self.check("pull", "list", "project", st, "project %d" % st)

    def test_per_item_error_message_in_report_is_escaped(self):
        self.check("push", "create", "ticket", 400, "create")

    def test_warning_text_is_escaped(self):
        buf = io.StringIO()
        with contextlib.redirect_stderr(buf):
            hs.emit_warning("x\x1b[2Jy")
        self.assertNotIn("\x1b", buf.getvalue())

    def test_exception_str_is_sanitised_for_every_syncerror(self):
        for exc in (hs.CoreError(HOSTILE), hs.AuthError(HOSTILE), hs.SyncError(HOSTILE), hs.MappingError(HOSTILE),
                    hs.ReadbackMismatch(HOSTILE), hs.VersionConflict(HOSTILE)):
            _no_controls(self, str(exc), type(exc).__name__)


class B3MalformedResponseTests(_MainBase):
    def inject(self, match, status, body):
        def inj(a, o, d):
            if (a, o) == match:
                return status, body
        self.fake.fail_on = inj

    def assert_aborted(self, cmd, what):
        with open(self.report, "w") as fh:
            fh.write('{"stale": "previous run said synced"}')
        code, out, err = self.run_main(cmd)
        self.assertEqual(code, 3, what)
        self.assertNotIn("Traceback", err, what)
        rep = json.loads(slurp(self.report, "r"))
        self.assertTrue(rep.get("aborted"), what)
        self.assertNotIn("stale", rep, what)
        # must be the DESIGNED rejection (a CoreError subclass), not the last-resort safety net
        self.assertNotIn("Unexpected", rep["error"]["type"], what + ": " + json.dumps(rep["error"]))
        self.assertIn(rep["error"]["type"], ("ShapeError", "ServerError", "AuthError"), what)

    def test_malformed_shapes_abort_cleanly(self):
        cases = [
            ("ticket item without id", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": [{"title": "x"}]}}),
            ("items not list", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": {"a": 1}}}),
            ("items empty dict", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": {}}}),
            ("items of ints", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": [1, 2]}}),
            ("data is list", ("list", "ticket"), 200, {"errorCode": -1, "data": [1, 2]}),
            ("data is string", ("list", "ticket"), 200, {"errorCode": -1, "data": "x"}),
            ("title not str", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": [{"id": "a", "title": 5}]}}),
            ("id is dict", ("list", "ticket"), 200, {"errorCode": -1, "data": {"items": [{"id": {"a": 1}}]}}),
            ("project item no id", ("list", "project"), 200,
             {"errorCode": -1, "data": {"items": [{"identifier": "HRD"}]}}),
            ("project item not dict", ("list", "project"), 200, {"errorCode": -1, "data": {"items": [7]}}),
            ("token not str", ("authenticate", None), 200, {"errorCode": -1, "data": {"token": {"a": 1}}}),
            ("errorMessage not str", ("list", "ticket"), 500, {"errorCode": 1, "errorMessage": {"x": 1}}),
        ]
        for what, match, status, body in cases:
            if match[0] == "authenticate":
                orig = self.fake.transport

                def tr(m, p, h, b, _o=orig, _body=body):
                    if b.get("action") == "authenticate":
                        return 200, _body
                    return _o(m, p, h, b)
                self.fake.transport, saved = tr, orig
                try:
                    self.assert_aborted("push", what)
                finally:
                    self.fake.transport = saved
                continue
            self.inject(match, status, body)
            for cmd in ("push", "pull", "compare"):
                self.assert_aborted(cmd, "%s/%s" % (what, cmd))

    def test_create_and_modify_and_read_shapes(self):
        self.inject(("create", "ticket"), 200, {"errorCode": -1, "data": {"ticket": {}}})
        self.assert_aborted("push", "create no id")
        self.inject(("create", "ticket"), 200, {"errorCode": -1, "data": {"ticket": "x"}})
        self.assert_aborted("push", "create ticket str")
        self.fake.fail_on = None
        self.assertEqual(self.run_main("push")[0], 0)
        self.fake.remote_edit("HRD-001", description="remote edit forces a version probe (read)")
        self.inject(("read", "ticket"), 200, {"errorCode": -1, "data": {"ticket": [1]}})
        self.assert_aborted("pull", "read ticket list")
        self.fake.fail_on = None
        orig = self.fake.transport

        def tr(m, p, h, b):
            if b.get("action") == "modify":
                return 409, {"errorCode": 3006, "errorMessage": "c", "errorMessageLocalised": 7}
            return orig(m, p, h, b)
        self.fake.transport = tr
        self.assert_aborted("pull", "localised not str")

    def test_deep_nesting_and_huge_body_via_real_transport(self):
        class H(http.server.BaseHTTPRequestHandler):
            def log_message(self, *a):
                pass

            def do_POST(self):
                n = int(self.headers.get("Content-Length") or 0)
                self.rfile.read(n)
                body = self.server.body
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
        for what, body in (("nested", b"[" * 200000 + b"]" * 200000),
                           ("huge", b'{"errorCode":-1,"data":{"items":[],"pad":"' + b"a" * (hs.MAX_RESPONSE_BYTES + 10) + b'"}}')):
            srv = serve(H, body=body)
            stop(self, srv)
            base = "http://127.0.0.1:%d" % srv.server_address[1]
            self.write_env(0o600, base)
            with open(self.report, "w") as fh:
                fh.write('{"stale": 1}')
            out, err = io.StringIO(), io.StringIO()
            argv = ["push", "--outbox", self.outbox, "--state", self.state, "--report", self.report,
                    "--env-file", self.env]
            with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
                code = hs.main(argv, environ={})
            self.assertEqual(code, 3, what)
            self.assertNotIn("Traceback", err.getvalue(), what)
            rep = json.loads(slurp(self.report, "r"))
            self.assertTrue(rep.get("aborted"), what)
            self.assertEqual(rep["error"]["type"], "TransportError", what)
            self.assertIn("exceeds" if what == "huge" else "nesting", rep["error"]["message"], what)

    def test_client_level_shapes_and_raw_recursion(self):
        def t(resp):
            return lambda m, p, h, b: resp
        c = hs.CoreClient(t((200, {"errorCode": -1, "data": {"ticket": {"id": "t"}}})), "u", "p")
        with self.assertRaises(hs.ShapeError):
            c.modify_ticket("t", 1, title="x")
        c = hs.CoreClient(t((200, {"errorCode": -1, "data": {"ticket": {"id": "t", "version": "2"}}})), "u", "p")
        with self.assertRaises(hs.ShapeError):
            c.modify_ticket("t", 1, title="x")

        def deep(*a):
            raise RecursionError("x")
        with self.assertRaises(hs.TransportError):
            hs.CoreClient(deep, "u", "p").list_tickets("p")

    def test_unexpected_exception_still_aborts_report(self):
        orig = hs.push

        def boom(*a, **k):
            raise RuntimeError("x\x1b[2J")
        hs.push = boom
        self.addCleanup(setattr, hs, "push", orig)
        with open(self.report, "w") as fh:
            fh.write('{"stale": 1}')
        code, _, err = self.run_main("push")
        self.assertEqual(code, 3)
        self.assertNotIn("\x1b", err)
        self.assertTrue(json.loads(slurp(self.report, "r")).get("aborted"))

    def test_report_marked_aborted_at_start_even_if_killed_midway(self):
        with open(self.report, "w") as fh:
            fh.write('{"stale": 1}')
        seen = {}

        def spy(*a, **k):
            seen["r"] = json.loads(slurp(self.report, "r"))
            raise hs.TransportError("stop")
        orig = hs.push
        hs.push = spy
        self.addCleanup(setattr, hs, "push", orig)
        self.run_main("push")
        self.assertTrue(seen["r"].get("aborted"))
        self.assertNotIn("stale", seen["r"])


class NonBlockingFoldInTests(_MainBase):
    def test_unmapped_status_skips_that_record_not_the_batch(self):
        self.recs[0]["payload"]["status"] = "Reopened"
        with open(self.outbox, "w") as fh:
            json.dump({"records": self.recs}, fh)
        code, out, _ = self.run_main("push")
        counts = json.loads(out)["counts"]
        self.assertEqual(counts.get("create"), 2)
        self.assertEqual(counts.get("unmapped"), 1)
        self.assertEqual(code, 2)
        self.assertEqual(len(self.fake.tickets), 2)

    def test_record_file_is_0600(self):
        rec_path = self.p("traffic.jsonl")
        self.run_main("push", "--record", rec_path)
        self.assertEqual(stat.S_IMODE(os.stat(rec_path).st_mode), 0o600)

    def test_dry_run_performs_zero_writes_even_if_main_passed_a_checkpoint(self):
        self.run_main("push", "--dry-run")
        self.assertEqual(self.fake.writes(), [])
        c = client(self.fake)
        self.fake.calls.clear()
        rep = hs.push(c, PID, self.recs, {}, dry_run=True, checkpoint=lambda s: self.fail("checkpoint in dry-run"))
        self.assertEqual(self.fake.writes(), [])
        self.assertEqual(sorted(actions(rep).values()), ["create"] * 3)
