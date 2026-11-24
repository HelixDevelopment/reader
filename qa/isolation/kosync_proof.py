#!/usr/bin/env python3
"""kosync_proof.py <kosync_bodies.jsonl> <out.json> - machine proof of the KOSync claims from the SERVER-SIDE bodies
(recorded by the logging proxy, secrets redacted): A's first push carries page 4/9, B's pull response returns that same
progress, B's push carries 6/9, A's later pull returns B's progress."""
import json, sys
rows = [json.loads(l) for l in open(sys.argv[1]) if l.strip()]
puts = [r for r in rows if r["method"] == "PUT" and r["path"] == "/syncs/progress"]
gets = [r for r in rows if r["method"] == "GET" and r["path"].startswith("/syncs/progress/") and r["status"] == 200 and r.get("response_body")]
gets_nonempty = [g for g in gets if isinstance(g["response_body"], dict) and g["response_body"].get("percentage") is not None]
def pct(r, side): return r[side]["percentage"]
checks = []
def chk(name, ok, ev): checks.append({"check": name, "status": "PASS" if ok else "FAIL", "evidence": ev})
chk("at least 2 progress pushes and 2 non-empty pulls recorded", len(puts) >= 2 and len(gets_nonempty) >= 2, {"puts": len(puts), "pulls_nonempty": len(gets_nonempty)})
if len(puts) >= 2 and len(gets_nonempty) >= 2:
    a_push, b_push = puts[0], puts[1]
    b_pull, a_pull = gets_nonempty[0], gets_nonempty[-1]
    chk("A pushed page 4/9 (0.4444)", abs(pct(a_push, "request_body") - 4 / 9) < 0.005, {"request": a_push["request_body"]})
    chk("B's pull response equals A's pushed progress/percentage", b_pull["response_body"].get("progress") == a_push["request_body"].get("progress")
        and abs(pct(b_pull, "response_body") - pct(a_push, "request_body")) < 1e-6, {"response": b_pull["response_body"]})
    chk("B pushed page 6/9 (0.6667)", abs(pct(b_push, "request_body") - 6 / 9) < 0.005, {"request": b_push["request_body"]})
    chk("A's later pull returns B's progress", a_pull["response_body"].get("progress") == b_push["request_body"].get("progress"), {"response": a_pull["response_body"]})
    chk("device ids differ between A and B", a_push["request_body"].get("device_id") != b_push["request_body"].get("device_id"), {})
out = {"source": sys.argv[1], "verdict": "PASS" if all(c["status"] == "PASS" for c in checks) else "FAIL", "checks": checks}
json.dump(out, open(sys.argv[2], "w"), indent=2); print(out["verdict"])
sys.exit(0 if out["verdict"] == "PASS" else 1)
