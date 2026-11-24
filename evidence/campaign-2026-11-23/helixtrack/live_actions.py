"""Helper used by run_live.sh: acts as a SECOND Core user (remote editor) / forces a stale-version write.
Credentials come only from the env file passed as argv[1]; nothing secret is printed."""
import json, os, sys
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "..", "tools", "helixtrack-sync"))
import helixtrack_sync as hs

envf, action, key, out = sys.argv[1:5]
env = {}
hs.load_env_file(envf, env)
rec = hs.Recorder(out)
c = hs.CoreClient(hs.urllib_transport(env["HT_BASE_URL"]), env["HT_USER"], env["HT_PASSWORD"], recorder=rec)
c.login()
pid = c.ensure_project(os.environ.get("HRD_PROJECT_KEY", "HRD"), "Helix Reader", create=False)
t = [t for t in c.list_tickets(pid) if t["title"].startswith("[%s]" % key)][0]
if action == "remote-edit":
    v = c.probe_version(t["id"])
    nv = c.modify_ticket(t["id"], v, status="review", description=t["description"] + "\n\nREMOTE EDIT by second user")
    print(json.dumps({"action": action, "key": key, "from_version": v, "to_version": nv}))
elif action == "remote-retitle":
    v = c.probe_version(t["id"])
    nv = c.modify_ticket(t["id"], v, title="retitled remotely without the registry key")
    print(json.dumps({"action": action, "key": key, "from_version": v, "to_version": nv}))
elif action == "remote-delete":
    c._do({"action": "remove", "object": "ticket", "data": {"id": t["id"]}})
    print(json.dumps({"action": action, "key": key, "deleted_ticket_id_prefix": t["id"][:8]}))
elif action == "stale-version":
    try:
        c.modify_ticket(t["id"], 1, title="must-not-apply")
        print(json.dumps({"action": action, "key": key, "result": "UNEXPECTED SUCCESS"}))
        sys.exit(1)
    except hs.VersionConflict as exc:
        print(json.dumps({"action": action, "key": key, "result": "VersionConflict", "current": exc.current}))
