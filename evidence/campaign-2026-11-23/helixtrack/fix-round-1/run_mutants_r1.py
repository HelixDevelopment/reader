#!/usr/bin/env python3
"""Fix-round-1 paired mutants: each mutation is applied to a scratch COPY of tools/helixtrack-sync and the unit
suite must FAIL.  A control (unmutated copy must pass) runs first.  Usage: python3 run_mutants_r1.py"""
import os, shutil, subprocess, sys, tempfile
HERE = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(HERE, "..", "..", "..", "..", "tools", "helixtrack-sync")
M = [
 ("B2 safe_text escapes nothing", 'cat = unicodedata.category(ch)\n        if cat[0] == "C"', 'cat = unicodedata.category(ch)\n        if False'),
 ("B2 safe_text keeps C1/format (no Cf)", 'if cat[0] == "C" or cat in ("Zl", "Zp")', 'if cat in ("Cc",)'),
 ("B2 safe_text no length cap", "return res if len(res) <= limit else", "return res if True else"),
 ("B2 SyncError.__str__ raw", "return safe_text(Exception.__str__(self))", "return Exception.__str__(self)"),
 ("B2 main error print raw", 'print("error: %s: %s" % (type(exc).__name__, safe_text(exc)), file=sys.stderr)', 'print("error: %s: %s" % (type(exc).__name__, exc.args[0] if exc.args else ""), file=sys.stderr)'),
 ("B2 warning raw", 'print("warning: %s" % safe_text(text), file=sys.stderr)', 'print("warning: %s" % text, file=sys.stderr)'),
 ("B2 last-resort print raw", 'print("error: unexpected %s: %s" % (type(exc).__name__, safe_text(exc)), file=sys.stderr)', 'print("error: unexpected %s: %s" % (type(exc).__name__, exc), file=sys.stderr)'),
 ("B3 ticket id not validated", 'cls._id(t.get("id"), what)', 'pass'),
 ("B3 items type not validated", 'if not isinstance(items, list):', 'if False:'),
 ("B3 item shape not validated", 'return [item_check(i) for i in items]', 'return items'),
 ("B3 data type not validated", 'if not isinstance(data, dict):\n                raise ShapeError("response data', 'if False:\n                raise ShapeError("response data'),
 ("B3 localised type not checked", 'loc if isinstance(loc, str) else ""', 'loc'),
 ("B3 token type not checked", 'if not isinstance(data.get("token"), str) or not data["token"]:', 'if not data.get("token"):'),
 ("B3 RecursionError escapes transport body", "    except RecursionError:\n        raise ValueError", "    except ZeroDivisionError:\n        raise ValueError"),
 ("B3 RecursionError escapes _raw", "ValueError, RecursionError,\n                MemoryError)", "ValueError)"),
 ("B3 no body size cap", 'if len(raw) > MAX_RESPONSE_BYTES:', 'if False:'),
 ("B3 stale report not replaced at start", 'if args.report:      # a previous run', 'if False:      # a previous run'),
 ("B3 last-resort except removed", "    except Exception as exc:                                  # last resort", "    except ZeroDivisionError as exc:                                  # last resort"),
 ("B3 created ticket id unchecked", 'return self._id(self._obj(data.get("ticket"), "created ticket").get("id"), "created ticket")', 'return data["ticket"]["id"]'),
 ("B3 modified version unchecked", 'if isinstance(v, bool) or not isinstance(v, int):\n            raise ShapeError("modified', 'if False:\n            raise ShapeError("modified'),
 ("N1 push unmapped aborts batch", "        except MappingError as exc:                      # this record", "        except ZeroDivisionError as exc:                      # this record"),
 ("N1 unmapped not drift", '"remote-deleted", "unmapped")', '"remote-deleted")'),
 ("N3 record file default mode", "os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)", "os.open(self.path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o666)"),
 ("N-dry dry-run checkpoints in push loop", "        if checkpoint and not dry_run:\n            checkpoint(state)\n    return report", "        if checkpoint:\n            checkpoint(state)\n    return report"),
]
def run(d):
    r = subprocess.run([sys.executable, "-B", "-m", "unittest", "test_helixtrack_sync"], cwd=d, capture_output=True, text=True,
                       env=dict(os.environ, PYTHONPATH=d, PYTHONDONTWRITEBYTECODE="1"))
    return r.returncode, r.stderr.strip().splitlines()[-1] if r.stderr.strip() else ""
def main():
    bad = 0
    with tempfile.TemporaryDirectory() as t:
        d = os.path.join(t, "w"); shutil.copytree(SRC, d, ignore=shutil.ignore_patterns("__pycache__"))
        rc, last = run(d); print("CONTROL (unmutated)", "PASS" if rc == 0 else "FAIL", last)
        if rc: return 1
        orig = open(os.path.join(d, "helixtrack_sync.py")).read()
        for name, a, b in M:
            if orig.count(a) < 1:
                print("MUTANT-NOT-APPLICABLE", name); bad += 1; continue
            open(os.path.join(d, "helixtrack_sync.py"), "w").write(orig.replace(a, b, 1))
            rc, last = run(d)
            print("KILLED  " if rc else "SURVIVED", name, last)
            bad += 0 if rc else 1
        open(os.path.join(d, "helixtrack_sync.py"), "w").write(orig)
    print("survivors/not-applicable:", bad)
    return 1 if bad else 0
sys.exit(main())
