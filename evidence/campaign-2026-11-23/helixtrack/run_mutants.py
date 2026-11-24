#!/usr/bin/env python3
"""Mutation check: apply one source mutation at a time to a COPY of helixtrack_sync.py and require the
unit suite to FAIL.  Prints one line per mutant plus the first FAIL/ERROR test names."""
import os, re, shutil, subprocess, sys, tempfile
HERE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(HERE, "..", "..", "..", "tools", "helixtrack-sync")
MUTANTS = [
 ("F1 url guard case-sensitive startswith (original bug)", "check_base_url",
  'scheme, host = parsed.scheme.lower(), parsed.hostname or ""\n    if scheme == "https" and host:\n        return\n    if scheme == "http" and host and _is_loopback(host):\n        return\n',
  'host = parsed.hostname or ""\n    if base.startswith("https://") or (base.startswith("http://") and _is_loopback(host)) or not base.startswith("http://"):\n        if base.startswith(("http","https")) or False:\n            return\n'),
 ("F1b url guard allows any http", "check_base_url", 'if scheme == "http" and host and _is_loopback(host):', 'if scheme == "http" and host:'),
 ("F1c transport constructor skips guard", "urllib_transport", "def urllib_transport(base_url, timeout=30):\n    check_base_url(base_url)\n", "def urllib_transport(base_url, timeout=30):\n"),
 ("F2a opener follows redirects", "build_opener", "handlers = [_NoRedirect()]", "handlers = []"),
 ("F2b client does not treat 3xx as error", "_raw", "if 300 <= status < 400:", "if False:"),
 ("F3 probe sends marker title", "probe_version", 'same = {k: cur[k] for k in ("title", "description", "status") if cur.get(k)}', 'same = {"title": "probe"}'),
 ("F3b accepted probe not detected", "probe_version", 'raise ServerError("version probe was accepted', 'return 0; raise ServerError("version probe was accepted'),
 ("F3c pull ignores --no-probe", "pull", "if probe else None", "if True else None"),
 ("F4a state ticket_id ignored when matching", "_resolve", 'known = (items.get(key) or {}).get("ticket_id")', "known = None"),
 ("F4b duplicate claimants ignored", "_resolve", "if extras:", "if False:"),
 ("F4c marker line ignored", "_claims", "keys.update(MARKER_RE.findall(ticket.get(\"description\") or \"\"))", "pass"),
 ("F5a remote-deleted silently re-created", "_resolve", 'problems[key] = ("remote-deleted"', 'continue; problems[key] = ("remote-deleted"'),
 ("F5b overwrite does not recreate", "_push_one", 'create_action, remote = "recreate", None', 'report.add(key, kind, **extra); return'),
 ("F5c remote-deleted not a conflict", "has_conflicts", '"duplicate", "remote-deleted")', '"duplicate")'),
 ("F6a no per-item checkpoint", "push", "None if dry_run else checkpoint, report)", "None, report)"),
 ("F6b create baseline recorded only after status modify", "_create", '    _remember(state, key, tid, None, dict(desired, status="open"), 1)\n', ""),
 ("F6c item errors propagate instead of partial report", "push", "except (CoreError, ReadbackMismatch) as exc:\n            report.fail(record_key(rec), exc)", "except ZeroDivisionError as exc:\n            report.fail(record_key(rec), exc)"),
 ("F6d lost-state interrupted create not reconciled", "_push_one", 'if base is None and rh == fields_hash(dict(desired, status="open")):', "if False:"),
 ("F6e state save not atomic", "save_json_atomic", "os.replace(tmp, path)", "open(path, 'w').write(open(tmp).read()); os.unlink(tmp)"),
 ("F7a dry-run checkpoints state", "multi", [("None if args.dry_run else checkpoint)", "checkpoint)"), ("None if dry_run else checkpoint, report)", "checkpoint, report)"), ("if checkpoint and not dry_run:", "if checkpoint:")], None),
 ("F7b dry-run final state write", "main", "            if not args.dry_run:\n                checkpoint(state)", "            checkpoint(state)"),
 ("F7c env-file mode check removed", "load_env_file", "stat.S_IMODE(st.st_mode) & 0o077", "False"),
 ("F7d pull drift exits 0", "main", "return 2 if rep.has_drift else 0", "return 0"),
 ("F7e push conflict exits 0", "main", "return 2 if rep.has_conflicts else 0", "return 0"),
 ("F7f aborted run exits 0", "main", "    if rep.aborted:\n        return 3\n", ""),
 ("F7g compare mismatch exits 0", "main", "return 0 if out[\"matching\"] == out[\"total\"] else 2", "return 0"),
 ("F8 unpushed field list drifts", "UNPUSHED", '"closure_criteria", "composes_with"', '"composes_with"'),
 ("R1 remote edits overwritten", "_push_one", "remote_edited = base is None or rh != base", "remote_edited = False"),
 ("R2 readback verification skipped", "_verify", "    if bad:\n        raise ReadbackMismatch", "    if False:\n        raise ReadbackMismatch"),
 ("R3 redaction disabled", "redact", 'k.lower() in REDACT_KEYS', "False"),
 ("R4 existing tickets ignored (duplicates)", "push", "tickets = [] if project_id is None else client.list_tickets(project_id)\n    matched, problems, report.orphans = _resolve(client, tickets, [record_key(r) for r in records], items)\n    for rec in records:\n        try:\n            _push_one(", "tickets = []\n    matched, problems, report.orphans = _resolve(client, tickets, [record_key(r) for r in records], items)\n    for rec in records:\n        try:\n            _push_one("),
 # ---- round 3 (hardening gaps)
 ("G1a HTTPException escapes _raw", "_raw", "urllib.error.URLError, http.client.HTTPException, ValueError)", "urllib.error.URLError, ValueError)"),
 ("G1b ValueError (HTML 200 body) escapes _raw", "_raw", "urllib.error.URLError, http.client.HTTPException, ValueError)", "urllib.error.URLError, http.client.HTTPException)"),
 ("G2a outbox structure not validated", "load_outbox", "if not isinstance(records, list) or not records:", "if False:"),
 ("G2b outbox record not-object accepted", "load_outbox", "        if not isinstance(r, dict):\n            raise", "        if False:\n            raise"),
 ("G2c outbox payload not required", "load_outbox", "        if not isinstance(payload, dict):\n            raise", "        if False:\n            raise"),
 ("G2d outbox duplicate keys accepted", "load_outbox", "        if key in seen:", "        if False:"),
 ("G2e state root type unchecked", "load_state", "    if not isinstance(doc, dict):\n        raise SyncError(\"state file", "    if False:\n        raise SyncError(\"state file"),
 ("G2f state items type unchecked", "load_state", "    if not isinstance(items, dict):", "    if False:"),
 ("G2g null/str state item accepted", "load_state", "        if not isinstance(item, dict):", "        if False:"),
 ("G2h state ticket_id unchecked", "load_state", 'if "ticket_id" in item and not', 'if False and not'),
 ("G2i corrupt json not wrapped", "_read_json_file", "except (OSError, ValueError, RecursionError) as exc:", "except (OSError, RecursionError) as exc:"),
 ("G2j history sqlite error not wrapped", "load_history", "except sqlite3.Error as exc:", "except ZeroDivisionError as exc:"),
 ("G2k state loaded after login", "main", "        state = load_state(args.state)\n        client = CoreClient(transport or urllib_transport(base), user, pw, Recorder(args.record))\n        client.login()\n",
  "        client = CoreClient(transport or urllib_transport(base), user, pw, Recorder(args.record))\n        client.login()\n        state = load_state(args.state)\n"),
 ("G3a flock not taken", "acquire", "fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)", "pass"),
 ("G3b lock blocks instead of failing", "acquire", "fcntl.LOCK_EX | fcntl.LOCK_NB)", "fcntl.LOCK_EX)"),
 ("G3c lock never released", "release", "        if self.fd is not None:\n            os.close(self.fd)", "        if False:\n            os.close(self.fd)"),
 ("G3d lock failure clobbers running report", "main", "if args.report and not isinstance(exc, LockError):", "if args.report:"),
 ("G3e lock taken after state load", "main", "        lock.acquire()\n        if args.env_file:", "        if args.env_file:"),
 ("G4a url userinfo allowed", "check_base_url", 'any(c in base for c in "\\\\@#?")', 'any(c in base for c in "\\\\#?")'),
 ("G4b url backslash allowed", "check_base_url", 'any(c in base for c in "\\\\@#?")', 'any(c in base for c in "@#?")'),
 ("G4c url fragment/query allowed", "check_base_url", 'any(c in base for c in "\\\\@#?")', 'any(c in base for c in "\\\\@")'),
 ("G4d url whitespace/non-ascii allowed", "check_base_url", "any(ord(c) <= 0x20 or ord(c) >= 0x7f for c in base)", "False"),
 ("G4e url bad port allowed", "check_base_url", "    try:\n        parsed.port\n    except ValueError:", "    try:\n        pass\n    except ValueError:"),
 ("G5a env mode ignores group bits", "load_env_file", "stat.S_IMODE(st.st_mode) & 0o077", "stat.S_IMODE(st.st_mode) & 0o007"),
 ("G5b env mode ignores other bits", "load_env_file", "stat.S_IMODE(st.st_mode) & 0o077", "stat.S_IMODE(st.st_mode) & 0o070"),
 ("G5c env mode ignores 0o001/0o004 only", "load_env_file", "stat.S_IMODE(st.st_mode) & 0o077", "stat.S_IMODE(st.st_mode) & 0o072"),
 ("G5d loopback by endswith('localhost')", "_is_loopback", 'if host == "localhost":', 'if host.endswith("localhost"):'),
 ("G5e history db opened read-write", "load_history", "?mode=ro&immutable=1", ""),
 ("G5f history db opened without immutable", "load_history", "?mode=ro&immutable=1", "?mode=ro"),
 ("G5g no read-confirm of absent known ticket", "_resolve", "t = client.read_ticket(known)          # confirm", "raise NotFoundError('x')  # confirm"),
 ("G5h title key not anchored", "multi", [('KEY_RE = re.compile(r"^\\[(HRD-\\d+)\\]")', 'KEY_RE = re.compile(r"\\[(HRD-\\d+)\\]")'), ('m = KEY_RE.match(', 'm = KEY_RE.search(')], None),
 ("G5i HTTP 200 with any errorCode accepted", "_do", "if status == 200 and code == -1:", "if status == 200:"),
 ("G5j unknown 4xx silently accepted", "_do", "        raise CoreError(msg, status, code)\n\n    def login", "        return {}\n\n    def login"),
 ("G5k re-login recursion unbounded", "_do", "return self._do(body, auth, _relogin=False)", "return self._do(body, auth, _relogin=True)"),
 ("G5l retry cap raised", "_apply_update", "if retries >= 1:", "if retries >= 50:"),
 ("G5m unpushable not a conflict for exit code", "has_conflicts", '("conflict", "unpushable", "duplicate"', '("conflict", "duplicate"'),
 ("G5n duplicate not a conflict for exit code", "has_conflicts", '"unpushable", "duplicate", "remote-deleted")', '"unpushable", "remote-deleted")'),
 ("G5o redact ignores lists", "redact", "    if isinstance(obj, list):\n        return [redact(v) for v in obj]\n", ""),
 ("G5p compare report not atomic", "main", "            if args.report:\n                save_json_atomic(args.report, out)\n            print(json.dumps({k: out[k]", "            if args.report:\n                open(args.report, 'w').write(json.dumps(out))\n            print(json.dumps({k: out[k]"),
 # ---- round 3b residual gaps (G6 series)
 ("G6a update baseline advanced before read-back verify", "_push_one", '    got = _verify(client, remote["id"], desired)\n    _remember(state, key, remote["id"], got.get("ticket_number"), desired, version)\n    extra =', '    _remember(state, key, remote["id"], None, desired, version)\n    got = _verify(client, remote["id"], desired)\n    extra ='),
 ("G6b create baseline advanced before read-back verify", "_create", '    got = _verify(client, tid, desired)\n    _remember(state, key, tid, got.get("ticket_number"), desired, version)', '    _remember(state, key, tid, None, desired, version)\n    got = _verify(client, tid, desired)'),
 ("G6c history db opened with create-capable mode", "load_history", "?mode=ro&immutable=1", "?immutable=1"),
 ("G6d env file regular-file check dropped", "load_env_file", "not stat.S_ISREG(st.st_mode) or ", ""),
 ("G6e env file opened blocking (FIFO hangs)", "load_env_file", "os.O_RDONLY | os.O_NONBLOCK", "os.O_RDONLY"),
 ("G6f env file open error not wrapped", "load_env_file", 'except OSError as exc:\n        raise SyncError("cannot open env file', 'except ZeroDivisionError as exc:\n        raise SyncError("cannot open env file'),
 ("G6g env file decode error not wrapped", "load_env_file", "except (OSError, ValueError) as exc:\n            raise SyncError(\"env file %s is unreadable", "except OSError as exc:\n            raise SyncError(\"env file %s is unreadable"),
 ("G6h env file mode check after environ is populated", "load_env_file", "        if not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) & 0o077:", "        if not stat.S_ISREG(st.st_mode):"),
 ("G6i atomic save files are group/other readable", "save_json_atomic", "os.O_EXCL, 0o600)", "os.O_EXCL, 0o666)"),
 ("G6j lock file group/other accessible", "acquire", "os.O_RDWR | os.O_CREAT, 0o600)", "os.O_RDWR | os.O_CREAT, 0o666)"),
 ("G6k abort report not atomic/0600", "multi", [('save_json_atomic(args.report, {"mode": args.command, "aborted": True,', 'open(args.report, "w").write(json.dumps({"mode": args.command, "aborted": True,'), ('"error": {"type": type(exc).__name__, "message": str(exc)}})\n        return 3', '"error": {"type": type(exc).__name__, "message": str(exc)}}))\n        return 3')], None),
 # ---- round 3c (env cap, fd leak, WAL-aware history)
 ("M13 env fd not closed on refusal path", "load_env_file", "        if fd is not None:\n            os.close(fd)", "        if False:\n            os.close(fd)"),
 ("H1 env size cap removed", "load_env_file", "if len(data) > ENV_MAX_BYTES:", "if False:"),
 ("H2 env file read unbounded", "load_env_file", "data = fh.read(ENV_MAX_BYTES + 1)", "data = fh.read()"),
 ("H3 env cap off by one (cap+0)", "load_env_file", "data = fh.read(ENV_MAX_BYTES + 1)", "data = fh.read(ENV_MAX_BYTES)"),
 ("H4 env cap constant inflated", "ENV_MAX_BYTES", "ENV_MAX_BYTES = 64 * 1024 ", "ENV_MAX_BYTES = 640 * 1024 "),
 ("H5 history always immutable (WAL ignored)", "load_history", "if wal_live and os.path.exists(ap + \"-shm\"):", "if False:"),
 ("H6 WAL-aware open without -shm check (creates files)", "load_history", "if wal_live and os.path.exists(ap + \"-shm\"):", "if wal_live:"),
 ("H7 stale-history warning dropped", "load_history", "if reason and warnings is not None:", "if False:"),
 ("H8 WAL-aware open failure not caught", "load_history", "            except sqlite3.Error as exc:\n                reason =", "            except ZeroDivisionError as exc:\n                reason ="),
 ("H9 warnings not written to report", "main", "    rep.warnings = warnings\n", ""),
 ("H10 warnings not printed to stderr", "main", 'print("warning: %s" % w, file=sys.stderr)', "pass"),
 ("H11 compare report drops warnings", "main", '"rows": rows,\n                   "warnings": warnings}', '"rows": rows}'),
 ("H12 WAL-aware open uses read-write mode", "load_history", 'return _query_history("file:%s?mode=ro" % q)', 'return _query_history("file:%s" % q)'),
]
def name_is_multi(x):
    return isinstance(x, list)


results, bad = [], 0
env = dict(os.environ, HRD_OUTBOX=os.path.abspath(os.path.join(HERE, '..', '..', '..', 'docs', 'tracking', 'helixtrack-outbox.json')))
for name, _, old, new in MUTANTS:
    d = tempfile.mkdtemp(prefix="mut_")
    try:
        for f in ("helixtrack_sync.py", "test_helixtrack_sync.py"):
            shutil.copy(os.path.join(SRC, f), d)
        os.makedirs(os.path.join(d, "..", "docs", "tracking"), exist_ok=True) if False else None
        p = os.path.join(d, "helixtrack_sync.py")
        s = open(p).read()
        pairs = old if name_is_multi(old) else [(old, new)]
        if any(o not in s for o, _ in pairs):
            print("MUTANT NOT APPLICABLE (pattern missing): %s" % name); bad += 1; continue
        for o, n in pairs:
            s = s.replace(o, n)
        open(p, "w").write(s)
        try:
            out = subprocess.run([sys.executable, "-I", "-m", "unittest", "discover", "-s", d, "-p", "test_*.py"],
                                 capture_output=True, text=True, timeout=120, env=env)
        except subprocess.TimeoutExpired:
            print("KILLED   %s -> suite HUNG (timeout 120s; a hang is a detected behaviour change)" % name)
            continue
        text = out.stdout + out.stderr
        failing = re.findall(r"^(?:FAIL|ERROR): (\S+)", text, re.M)
        killed = out.returncode != 0 and failing
        if not killed: bad += 1
        print("%-8s %s -> %s" % ("KILLED" if killed else "SURVIVED", name,
                                 ("%d tests failed; first: %s" % (len(failing), failing[0])) if failing else "suite passed"))
    finally:
        shutil.rmtree(d, True)
print("mutants=%d survivors_or_inapplicable=%d" % (len(MUTANTS), bad))
sys.exit(1 if bad else 0)
