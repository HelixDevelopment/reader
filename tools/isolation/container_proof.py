#!/usr/bin/env python3
"""container_proof.py NAME OUTFILE - wait for container NAME, then write the isolation-relevant `podman inspect` facts as JSON
(env VALUES are never recorded, only names). Run in the background by run_container.sh -p."""
import json, subprocess, sys, time
name, out = sys.argv[1:3]
for _ in range(150):
    r = subprocess.run(["podman", "inspect", name], capture_output=True, text=True)
    if r.returncode == 0 and r.stdout.strip() not in ("", "[]"):
        break
    time.sleep(0.2)
else:
    sys.exit(1)
i = json.loads(r.stdout)[0]; h = i["HostConfig"]; c = i["Config"]
proof = {"name": i["Name"], "image": i["ImageName"], "created": i["Created"],
         "network_mode": h["NetworkMode"], "privileged": h["Privileged"], "cap_drop": h["CapDrop"], "cap_add": h["CapAdd"],
         "security_opt": h["SecurityOpt"], "memory": h["Memory"], "memory_swap": h["MemorySwap"], "pids_limit": h["PidsLimit"],
         "oom_score_adj": h.get("OomScoreAdj"), "userns_mode": h["UsernsMode"], "lifetime_timeout_s": c.get("Timeout"),
         "binds": h["Binds"], "devices": h.get("Devices"), "ports": h.get("PortBindings"),
         "env_names": sorted(e.split("=", 1)[0] for e in c["Env"]), "labels": c["Labels"], "entrypoint": c["Entrypoint"], "cmd": c["Cmd"]}
tmp = out + ".tmp"
open(tmp, "w").write(json.dumps(proof, indent=2) + "\n")
import os; os.replace(tmp, out)
