import os,re,shutil,subprocess,sys,tempfile
SRC="/home/milosvasic/Projects/helix_reader/tools/helixtrack-sync"
M=[
("N1 lock file unlinked before release", "        if self.fd is not None:\n            os.close(self.fd)", "        if self.fd is not None:\n            os.unlink(self.path)\n            os.close(self.fd)"),
("N2 state remembered (desired) before readback verify", "    got = _verify(client, tid, desired)\n    _remember(state, key, tid, got.get(\"ticket_number\"), desired, version)\n    return tid", "    _remember(state, key, tid, None, desired, version)\n    got = _verify(client, tid, desired)\n    return tid"),
("N2b update: state remembered before verify", "    got = _verify(client, remote[\"id\"], desired)\n    _remember(state, key, remote[\"id\"], got.get(\"ticket_number\"), desired, version)", "    _remember(state, key, remote[\"id\"], None, desired, version)\n    got = _verify(client, remote[\"id\"], desired)"),
("N3 redirect refused only on GET", "        return None                      # every 3xx", "        if req.get_method() != 'GET': return urllib.request.HTTPRedirectHandler.redirect_request(self, req, fp, code, msg, headers, newurl)\n        return None                      # every 3xx"),
("N4 userinfo checked only on parsed netloc", 'any(c in base for c in "\\\\@#?")', '("@" in urllib.parse.urlparse(base).netloc or any(c in base for c in "\\\\#?"))'),
("N5 env mode checked after reading token", "    st = os.stat(path)\n    if not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) & 0o077:\n        raise SyncError(\"env file %s must be a regular file not accessible by group/others (chmod 600)\" % path)\n    with open(path, encoding=\"utf-8\") as fh:\n        for line in fh:\n            line = line.strip()\n            if line and not line.startswith(\"#\") and \"=\" in line:\n                k, v = line.split(\"=\", 1)\n                environ.setdefault(k.strip(), v.strip())\n",
 "    st = os.stat(path)\n    with open(path, encoding=\"utf-8\") as fh:\n        for line in fh:\n            line = line.strip()\n            if line and not line.startswith(\"#\") and \"=\" in line:\n                k, v = line.split(\"=\", 1)\n                environ.setdefault(k.strip(), v.strip())\n    if not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) & 0o077:\n        raise SyncError(\"env file %s must be a regular file not accessible by group/others (chmod 600)\" % path)\n"),
("N5b env file non-regular (fifo/dir) accepted", "not stat.S_ISREG(st.st_mode) or ", ""),
("N6 PROBE_VERSION constant is a real version", "PROBE_VERSION = 2147483647", "PROBE_VERSION = 2"),
("N7 push continues after first item error", "            report.fail(record_key(rec), exc)\n            if checkpoint and not dry_run:\n                checkpoint(state)\n            break", "            report.fail(record_key(rec), exc)\n            if checkpoint and not dry_run:\n                checkpoint(state)\n            continue"),
("N7b no checkpoint on abort", "            report.fail(record_key(rec), exc)\n            if checkpoint and not dry_run:\n                checkpoint(state)\n            break", "            report.fail(record_key(rec), exc)\n            break"),
("N8 push exit code uses has_drift", "return 2 if rep.has_conflicts else 0", "return 2 if rep.has_drift else 0"),
("N8b pull exit code uses has_conflicts", "return 2 if rep.has_drift else 0", "return 2 if rep.has_conflicts else 0"),
("N9 abort report written non-atomically", "            save_json_atomic(args.report, {\"mode\": args.command, \"aborted\": True,\n                                           \"error\": {\"type\": type(exc).__name__, \"message\": str(exc)}})", "            open(args.report,'w').write(json.dumps({\"mode\": args.command, \"aborted\": True,\n                                           \"error\": {\"type\": type(exc).__name__, \"message\": str(exc)}}))"),
("N10 history db mode rwc", "?mode=ro&immutable=1", "?mode=rwc&immutable=1"),
("N11 atomic save file mode 0644", "os.O_EXCL, 0o600)", "os.O_EXCL, 0o644)"),
("N12 lock file mode 0644", "os.O_RDWR | os.O_CREAT, 0o600)", "os.O_RDWR | os.O_CREAT, 0o644)"),
("N13 lock error exit code 1 not 3 path (LockError not SyncError)", "class LockError(SyncError):", "class LockError(Exception):"),
("N14 resume: duplicate extras ignored when ticket known", "            extras = [c for c in cands if c[\"id\"] != t[\"id\"]]", "            extras = []"),
("N15 HTTP 200 html swallowed as None body ok", "except ValueError:\n                return exc.code, None", "except ValueError:\n                return exc.code, {}"),
("N16 lock acquired after env/url validation", "        lock.acquire()\n        if args.env_file:\n            load_env_file(args.env_file, environ)\n        base, user, pw = environ.get(\"HT_BASE_URL\"), environ.get(\"HT_USER\"), environ.get(\"HT_PASSWORD\")\n        if not (base and user and pw):\n            raise SyncError(\"HT_BASE_URL, HT_USER and HT_PASSWORD must be set (environment or --env-file)\")\n        check_base_url(base)\n", "        if args.env_file:\n            load_env_file(args.env_file, environ)\n        base, user, pw = environ.get(\"HT_BASE_URL\"), environ.get(\"HT_USER\"), environ.get(\"HT_PASSWORD\")\n        if not (base and user and pw):\n            raise SyncError(\"HT_BASE_URL, HT_USER and HT_PASSWORD must be set (environment or --env-file)\")\n        check_base_url(base)\n        lock.acquire()\n"),
]
bad=0
for name,old,new in M:
    d=tempfile.mkdtemp(prefix="nm_")
    try:
        for f in ("helixtrack_sync.py","test_helixtrack_sync.py"): shutil.copy(os.path.join(SRC,f),d)
        p=os.path.join(d,"helixtrack_sync.py"); s=open(p).read()
        if old not in s: print("NOT APPLICABLE",name); bad+=1; continue
        open(p,"w").write(s.replace(old,new))
        try:
            o=subprocess.run([sys.executable,"-I","-m","unittest","discover","-s",d,"-p","test_*.py"],capture_output=True,text=True,timeout=120)
        except subprocess.TimeoutExpired:
            print("KILLED(hang)",name); continue
        t=o.stdout+o.stderr; f=re.findall(r"^(?:FAIL|ERROR): (\S+)",t,re.M)
        k=o.returncode!=0 and f
        if not k: bad+=1
        print("KILLED  " if k else "SURVIVED",name,"->",(f[0] if f else "suite passed"))
    finally: shutil.rmtree(d,True)
print("new_mutants=%d survivors_or_inapplicable=%d"%(len(M),bad))
