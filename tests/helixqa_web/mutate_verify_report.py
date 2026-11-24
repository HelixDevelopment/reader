#!/usr/bin/env python3
"""HRD-038 mutation matrix for verify_report.py (temp copy; working tree untouched)."""
import os, sys
ROOT = os.path.realpath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "../.."))
sys.path.insert(0, os.path.join(ROOT, "scripts"))
import mutlib
V = "tests/helixqa_web/verify_report.py"
M = [
 (V, "M1 symlinked report allowed", "os.path.islink(rp) or not os.path.isfile(rp)", "not os.path.isfile(rp)"),
 (V, "M2 unreadable report ignored", 'return [f"report.json unreadable: {e}"]', "return []"),
 (V, "M3 empty cases allowed", "if not isinstance(cases, list) or not cases:", "if False:"),
 (V, "M4 duplicate ids allowed", "if len(set(ids)) != len(ids):", "if False:"),
 (V, "M5 attempts not required", 'or s["attempts"] < 1:', 'or False:'),
 (V, "M6 bool attempts accepted", " or isinstance(s.get(\"attempts\"), bool)", ""),
 (V, "M7 page state not required", "if not pg or [pg", "if False and [pg"),
 (V, "M8 unsafe name allowed", 'if not f or not SAFE.match(f) or "/" in f:', "if not f:"),
 (V, "M9 symlink screenshot allowed", "if os.path.islink(p) or not os.path.isfile(p):", "if not os.path.isfile(p):"),
 (V, "M10 sha not compared", 'if sha(p) != s.get("screenshot_sha256"):', "if False:"),
 (V, "M11 totals not compared", 'if (rep.get("pass"), rep.get("fail"), rep.get("skip")) != (n_pass, n_fail, n_skip):', "if False:"),
 (V, "M12 FAIL/SKIP tolerated", "if n_fail or n_skip:", "if False:"),
 (V, "M13 unpinned allowed", "if not isinstance(ec, int) or not isinstance(es, int):", "if False:"),
 (V, "M14 case pin ignored", "if len(cases) != ec:", "if False:"),
 (V, "M15 step pin ignored", "if n_steps != es or n_pass != es:", "if False:"),
 (V, "M16 pin_errors ignored", 'if rep.get("pin_errors"):', "if False:"),
 (V, "M17 stray png ignored", 'if f.endswith(".png") and f not in referenced:', "if False:"),
 (V, "M18 exit always 0", "return 1 if errs else 0", "return 0"),
]
sys.exit(mutlib.run([V, "tests/test_verify_report.py", "scripts/mutlib.py"], M, [sys.executable, "-m", "unittest", "test_verify_report"], "tests"))
