# Test isolation policy and architecture

Status: policy binding since 2026-11-23 (operator hard rule, stated twice); hardened after the independent isolation
review (findings F1-F6, evidence `evidence/campaign-2026-11-23/isolated/hardening/`).

## 1. The rule

Automated tests must NEVER directly access or affect the logged-in user's desktop, display, windows, input devices,
clipboard, browser or apps, on any host. Every autonomous GUI / E2E / emulator test runs INSIDE an isolated rootless
container built on the Containers submodule (`submodules/containers`) and driven through HelixQA (`submodules/helixqa`,
banks in `qa/helixqa/`). No credential of the operator (sudo password, tokens) is ever used. Other projects' containers
(`lava-*`, `helixagent-*`, `helixcode-*`, `boba-*`, `qbittorrent`, `workshop-*`, `helixqual-*`) are never touched; ours are
named `hrdiso-*` (tests: `hrdisorev-*`).

**The container is the boundary. The guard (section 4) is a lint**: it catches honest mistakes early and has a documented
residual; it is not what keeps the host safe.

## 2. What runs where

| Layer | Runs on host | Runs in the container |
|---|---|---|
| helix-server (Go, no GUI) | yes, loopback, a free port picked per run | no |
| Logging proxy (`tools/isolation/access_proxy.py`) | yes, on a UNIX SOCKET `<state>/sock/h.sock` (mode 0600, dir 0700) | no |
| redis (disposable, `hrdiso-redis-<runid>`, 128 MB, 1 h lifetime, per run) | rootless container, loopback on a free port picked per run (server only) | n/a |
| Web client e2e (Playwright + Chrome for Testing) | no | yes (`localhost/helix-gui-test:1`) |
| KOReader GUI (SDL on private Xvfb, xdotool, screenshots) | no | yes |
| Electron desktop client e2e | no | yes |
| HelixQA HTTP banks (`helixqa http`) | no (static binary built on host, executed in container) | yes |
| HelixQA web bank browser cases (HLX-WEB-001/002) | no | yes, by `tests/helixqa_web/run_web_bank.sh` (HRD-038, separate work, container name `hrd038-web` is outside the `hrdiso-` namespace: reported, not changed) |
| Android emulator | no | NOT covered: the wrapper was removed from this PR (section 6); no isolated Android run exists yet |

The host only ever runs non-GUI processes (server, proxy, redis, `podman`). Drivers such as xdotool, Xvfb, openbox,
ImageMagick and Chromium exist only inside the image.

## 3. Architecture

```
host (logged-in desktop: never touched)
 |  helix-server 127.0.0.1:<free port>  <- access_proxy (route allow-list)  unix:<state>/sock/h.sock   (access.log + KOSync bodies, redacted)
 |  redis hrdiso-redis-<runid> (podman, 127.0.0.1:<free port>, server only)
 |
 +-- podman run --network none   (tools/isolation/run_container.sh)
       localhost/helix-gui-test:1  (submodules/containers/images/gui-test/)
       /run/helixsock/h.sock  <- the ONLY thing handed in (bind of <state>/sock)
       qa/isolation/inside/launch.sh: runtime assertions, then node sockbridge.js 127.0.0.1:38191 -> h.sock, then the test
       entrypoint: private Xvfb :99 (-nolisten tcp), optional openbox
```

**Network (finding F1).** The first design used podman's `pasta --map-host-loopback`, which exposes EVERY host loopback
service (adb 5037, postgres 8432/55432, ollama 11434, cups 631, ssh 22, ...) and the internet to the container (RED log:
`hardening/F1_RED.log`). Now the container runs with `--network none`: it has only `lo`. The one path to the test server is
a unix socket bind-mounted from the host and bridged to the container's own `127.0.0.1:38191` by a 10-line in-container
node bridge. Options evaluated: (a) `pasta -T <port>` forwards host-to-container, not the direction needed, and without
`--map-host-loopback` pasta cannot reach host loopback at all; (b) per-port nft rules need root; (c) `--network none` +
unix socket works rootless, is deny-by-default and also blocks all outbound internet. Chosen: (c). The permanent gate test
`tools/isolation/tests/test_network_isolation.py` probes from inside a container every TCP port a host process listens on
(loopback and wildcard), the host's own IPs, the pasta gateway names, `1.1.1.1:443` and a canary listener: nothing is
reachable, and with `-b` only `127.0.0.1:38191` opens and really reaches the upstream through the proxy. Outbound
internet is DENIED by default for every suite (none of them needs it; the browsers/KOReader are bind-mounted). The Android
emulator container (section 6) is a separate recipe with its own published adb port and no host-loopback map.

**Proxy route policy (N1).** The unix socket is not a free pass to the host server: `access_proxy.py` is default-DENY with a
per-policy method+path allow-list. Allowed (GET/HEAD): `/healthz`, `/readyz`, `/opds/...`, `/opds-pse/stream/<id>/<n>`,
`/v1/works[/<id>]`, `/v1/editions/<id>`, `/v1/files/<id>/content|license`, `/v1/covers/<id>`, `/v1/formats`, `/v1/protocols`,
`/syncs/progress/<doc>`, `/users/auth`; plus `PUT /syncs/progress` and `POST /users/create` (KOSync). Everything else - in
particular `POST /v1/sources`, `/v1/scans`, any `/v1/admin/...`, local-root or webdav source registration - is answered
`403 {"error":"isolation proxy: route not allowed"}` and never reaches the server (logged `DENIED`). Path tricks (encoded
slashes, backslashes, NULs, `//`, dot segments) are denied; chunked bodies get 411, bad/negative `Content-Length` 400, more than
8 MiB 413, handler exceptions 502 (logged, never fatal, nothing leaked). Policy `helixqa` additionally allows `GET /v1/scans`
and `POST /v1/sources` ONLY with the empty JSON body `{}` (that is what the bank's negative cases send); the bank's hard-coded
literal bearer is rewritten by the proxy to the real admin token on those two routes only. The admin token is RANDOM per run, is
passed to the proxy through its environment, and never enters a container; setup (registering the fixture source, scanning)
is done host-side by `start_server.sh` before any container starts. There is no fixed admin literal in `tools/isolation` or
`qa/isolation` any more except the bank's own fixture string used as the proxy's mapping KEY. Tests:
`tests/test_proxy_policy.py` (unit, real proxy on unix sockets) and `tests/test_network_isolation.py` (from inside a container,
through the bridge: `POST /v1/sources` with a webdav endpoint pointing at a host canary, `POST /v1/scans`, a local-root source are
refused, the canary records 0 hits, no source exists afterwards - also against the REAL server when `HELIX_SERVER_BIN` is set).
RED: `hardening/N1_RED.log`.

**No host display/session.** No X11/Wayland socket, `/dev/input`, D-Bus or `XDG_RUNTIME_DIR` is mounted; `DISPLAY` is never
passed (the entrypoint pins the container-private `:99`). `run_container.sh` refuses such env/mounts (exit 64) and the
in-container `launch.sh` asserts at RUNTIME that none exists (exit 65): no WAYLAND/XAUTHORITY/DBUS/XDG_RUNTIME_DIR/
SSH_AUTH_SOCK variables, no foreign X socket, no `/dev/input|uinput|dri`, nothing in `/run/user`, no display/dbus mount in
`mountinfo`, only the `lo` interface. Tested by `tests/test_runtime_checks.py` (it deliberately bypasses the wrapper to
prove the runtime check alone).

**Bind allow-list (F3, N5).** `tools/isolation/bindcheck.py` (called for every `-r/-w` and for the `-b` bridge): the source's
resolved real path (symlinks followed) must lie inside the repo `clients/`, `qa/`, `tools/isolation/`,
`submodules/containers/images/`, the Playwright cache `~/.cache/ms-playwright`, `/usr/lib/koreader` (all read-only) or under
`ISO_WORK` (read-write only for its sub-directories) or `evidence/` sub-directories (read-write, never the root itself).
Always refused: `$HOME`, `/`, `/tmp`, `/dev/shm`, `/var/tmp`, system dirs, any ancestor of `$HOME`, anything inside - or an ancestor
of - `~/.ssh .config .gnupg .aws .kube .docker .mozilla .local .password-store .claude*` and similar, sources containing `:` `,`
or control characters (mount-option injection). Destinations must be absolute and canonical (`normpath`-equal: no `..`, `//`,
`.`, trailing `/`, charset `[A-Za-z0-9_./-]`), below `/work /opt/ /out /prof /fx /pw /ms-playwright`, and may never be, lie
inside or cover `/opt/isoinside` or `/run/helixsock` (the latter only for the `-b` bridge itself); `run_container.sh` also
refuses duplicate destinations. `ISO_WORK` must be absolute, canonical, symlink-free, strictly inside
`~/.cache/claude-tmp` or `~/.cache/helix-iso`, and clear of the sensitive dirs. A read-write bind under `evidence/` is only
accepted for the run's OWN FRESH directory: nothing below it may be a `SHA256SUMS` seal or older than 6 h, so a sealed or stale
sibling is never writable; `tools/isolation/fresh_out.sh <dir>` (called by every runner) moves an existing output dir into
`~/.cache/claude-tmp/evidence-archive/` (never deletes) and recreates it empty. `bindcheck.py` prints the RESOLVED path and the
wrapper hands exactly that to podman (`-v "$real:$dst"`), so a symlink swapped after the check cannot redirect the mount.
Tests: `tests/test_bindcheck.py` (RED `hardening/N5_RED.log`, GREEN `hardening/N5_GREEN.log`).

**Lifetime (F5, N4).** Every container gets `--timeout` (default 1800 s; `-t`), an owner label (`org.helix.owner-pid`, or
`org.helix.owner-file` for redis), a lease label `org.helix.lease-ts`, and `tools/isolation/reap.sh`, which every start runs,
removes stale `hrdiso-*` / `hrdisorev-*` containers whose owner is gone (nothing else, ever). STARTUP GRACE: a container
younger than `REAP_GRACE` seconds (default 120; lease label, else creation time) is never reaped, and `start_server.sh` writes
its owner file (naming the live script) BEFORE it creates redis, so a parallel start can no longer have its brand-new redis
killed. Redis and the server are per run: container `hrdiso-redis-<runid>`, free loopback ports bound-checked per run,
recorded in `<state>/run.env` and `<state>/redis.name`; there is no shared fixed `hrdiso-redis`, so parallel suites never
`flushall` each other. The server and proxy run under `timeout 3600`, redis under `--timeout 3600`. Tests:
`tests/test_reaper.py` (kill -9 orphan reaped after the window and capped; live run not reaped; the startup RACE: owner file
missing inside the window is kept, then reaped; expired lease reaped; live owner kept; unique per-run ids and owner-file-first order).
RED: `hardening/N4_RED.log` (the old reaper killed a starting container).

**Resources.** memory = swap cap, `--cpus`, `--pids-limit 512`, `--oom-score-adj 500`, `--cap-drop ALL`,
`no-new-privileges`, `--userns=keep-id` (`submodules/containers/docs/RESOURCE_LIMITS.md`). One test container at a time.
`-p FILE` records the container's isolation facts (`podman inspect` subset; env VALUES never) next to the evidence
(`container_proof.json`).

## 4. Guard and gate (F2) - a lint

* Gate: `tools/isolation/assert_host_untouched.sh` (fast: lint tests, 265-case evader matrix, pinned-waiver tests, bind tests, proxy
  route-policy tests, repo scan) and `--full` (adds network isolation incl. the real-server N1 test, runtime checks, reaper and
  startup-race tests; needs rootless podman and the image).
* Scan set: tracked, untracked AND git-ignored files; extensions incl. `.gradle .kts .txt .html`; extension-less files WITH a
  shebang as code and WITHOUT one as data (data-safe rule subset: never silently skipped); any file with a shebang or the exec bit;
  Dockerfile/Containerfile/Makefile, `.claude/{agents,commands,skills}`, `prompts/`, `.git/hooks`. Directories named `build out
  dist .cache .gradle evidence venv target __pycache__` are no longer blind spots: their script-like files (script extension,
  shebang or exec bit) get the FULL rule set. Third-party `node_modules` trees: only script-like files, and only the HARD
  subset (input injection, clipboard, uinput path, portal/dbus, GUI-automation libs, container escapes); the count is printed
  (`third-party script files ...`). Binaries (NUL bytes) are skipped (residual). Symlinks are followed (inside the repo) or flagged
  (outside) and counted; `submodules/` is scanned host-side in report mode only (`--submodules`, report:
  `hardening/submodule_host_entrypoints_report.txt`; submodules are not edited).
* Rules (see `assert_host_untouched.py`): input injection and window/clipboard/screenshot tools, desktop launchers,
  uinput/evdev, scrcpy, `adb ... input`, GUI-automation libraries, session-bus/portal calls, x11grab/pipewire capture, ANY use of
  the display/session environment names (DISPLAY except the container-private `:99`, WAYLAND_DISPLAY, XAUTHORITY,
  XDG_RUNTIME_DIR, DBUS_*), X11/Wayland sockets, headed browsers, electron/chrome/chromium/firefox launches, windowed
  `emulator -avd` and `qemu` without `-display none|-nographic`, host-service probes (`localhost:5037` adb server), environment
  reconnaissance (`env | grep wayl`), and container escapes on continuation-joined lines matched on BOTH the raw and the
  quote-stripped view (`--net="host"`, `-v "/":/h`): ANY `--network/--net` other than `none` (so the old pasta
  `--map-host-loopback` mode is caught), `--map-host-loopback`, `allow_host_loopback`, `--privileged`, `--cap-add`, `--device`
  other than `/dev/kvm`, `--security-opt` other than `no-new-privileges` (`label=disable`, `seccomp=unconfined`), `--pid/--ipc/
  --userns/--uts host`, `--volumes-from`, `--mount` (except tmpfs/volume), `-v` with an absolute, `~`, `$HOME` or `..` source (`$PWD/..`),
  `podman kube play`/`play kube`, `docker compose up`, remote `-H`; references to inside-scripts from host files
  (`sh qa/isolation/inside/*`, globs, `cd qa/isolation/inside && ./x.sh`); and obfuscation: `base64 -d` (also into a variable),
  `source <(curl ...)`, `sh <<< "$X"`, `sh -c "$X"`, `curl | sh`, `eval "$X"`, `${X}tool` / `$(cmd)tool` command concatenation, glob and
  charclass command names that expand to a known tool (`xdoto?l`, `x[d]otool`, in shell files), `"a"+"b"`, `x''y`, `x"d"otool`, `x\y`.
* Allowlist, ONLY for code that runs inside a container image: the marker `isolation: runs-inside-container` in the first
  15 lines of a file under `qa/isolation/inside/` or `submodules/containers/images/` (ignored anywhere else). Such files
  must not be executable and may not be referenced by file name from host-side files (only the directory mount is allowed).
* Exemptions are PINNED (N3) in `tools/isolation/guard/waivers.txt`, one line `kind|path|sha256|rule|justification|expires`, exact
  paths only (no globs), expiry at most 366 days out: `self` (the lint's own NINE sources - the count is whatever `waivers.txt` lists, `pin_waivers.py list`: `assert_host_untouched.py`,
  `guard/test_assert_host_untouched.py`, `tests/test_guard_matrix.py`, `tests/test_runtime_checks.py`, `tests/test_bindcheck.py`,
  `tests/test_review_round1.py`, `tests/test_review_round2.py`, `tests/test_review_round3.py`; not scanned while the hash matches), `denylist` (the trailing `# isolation: denylist` annotation on a line that names a forbidden
  thing in order to refuse it is honoured ONLY in a pinned file under `tools/isolation/`; an unpinned marker is itself a finding
  `denylist-marker-unpinned` and exempts nothing), `waive` (findings of one rule in one pinned file: the spec-kit `eval` scripts,
  three third-party `xdg-open` shims). A changed pinned file (`waiver-pin-mismatch`), a deleted waiver (the file is scanned
  again and fails), an expired, orphaned, malformed or hand-edited entry (`waiver-expired|orphan|malformed|unrecorded`) FAILS the
  gate. Pins are created and updated only with `tools/isolation/guard/pin_waivers.py add|repin|remove`, which appends
  `utc|action|path|old|new|reason` to `waivers.history` and prints `RE-PINNED ... old -> new`; a pin whose hash has no history
  record fails. The gate prints a `NOTICE pinned waivers active: N (self/denylist/waive)` line. Tests: `tests/test_waivers.py`
  (RED/GREEN `hardening/N3_RED_GREEN.log`, `hardening/N3_GREEN.log`).
* Evader matrix: `tools/isolation/guard/mutants.b64` (base64 so this tree carries no forbidden literals; edit with
  `mutants_codec.py`): 265 cases = 253 REQUIRED (round 4 added 22 global-option / wider-runtime-family shapes and 4 Python list forms; earlier: 227 REQUIRED (all caught; round 3 added 14 runtime-family and 12 risky-flag cases; the 83 earlier ones, the 94 round-2 evaders and the round-1/round-2 review additions incl. 12 odd-flag cases of the class rule `container-outside-isolation`: skip-dir hiding,
  extensions, container escapes, inside-script references, obfuscation, recon) and 12 documented RESIDUAL cases. HONEST
  CLAIM: "caught" means caught by the lint; it does NOT mean the host is safe - the container is the boundary. The reviewer's own
  80 `isorev2_*` mutants were not available to this session (scratch not found); the new cases were re-created from the
  reviewer's list, class by class, so equality with the reviewer's set is UNCONFIRMED. REAL REMAINING (uncaught) classes: command
  names computed at run time (chr/hex/base64 held in data), command lines read from environment, files or config at run time,
  tools not in the rule list, code executed by third-party imports/binaries/`node_modules` (HARD subset only) and NUL-containing
  files, glob names in non-shell files, and anything a future tool does that no rule names. RED:
  `hardening/N2_RED.log` (`n2_red_baseline.py`, a RECONSTRUCTION of the round-1 behaviour, evaded 61 of the 94 new cases and none of the 83 earlier ones; the earlier F2 RED
  is `hardening/F2_RED_old_ruleset_vs_matrix.log`); GREEN: `hardening/N2_GREEN.log`.
* `run_isolated.sh -- <cmd>`: `env -i` allow-list (PATH HOME LANG LC_ALL TZ `HELIX_*` `QA_*` `HQA_*`), never forwards
  session variables, re-asserts the final environment (exit 65), refuses `--require-display` and host-GUI commands (64).
* `clients/desktop/package.json` `start` does not launch Electron on the host; it prints the instruction and exits 64.

## 5. How to run each suite

**Clean-clone prerequisites that are NOT in this repository state (stated plainly).** On a fresh clone of `main` with this
change merged, the runners cannot work until all of these land (each in its own PR): (1) `submodules/containers/images/gui-test`
does not exist in the containers submodule pin `22b49707`; it is untracked in the operator's checkout, so the image
`localhost/helix-gui-test:1` cannot be built from a clean clone and `launch.sh`'s assumptions about the image (Xvfb `:99`, empty
`/run/user`) cannot be reviewed from this repository; (2) `clients/` (web, desktop, android), (3) `qa/helixqa/` (banks, fixtures)
and (4) `evidence/clients/` are untracked in this PR; (5) `submodules/containers/docs/RESOURCE_LIMITS.md`, cited in section 3,
is not verified to exist in the pin. Behaviour without them is fail-closed, never a fallback to the host: a missing image makes
`run_container.sh` exit **66**; a missing `qa/helixqa` makes `run_helixqa_container.sh` stop with a non-zero status (the review observed 70 with a stub; the script itself sets no code, `python3` exits 2 for a missing file: exact code UNCONFIRMED); a refused
bind, name or env exits 64. The runner is therefore not usable end to end from a clean clone yet.

Waiver ownership: the three `waive` rows for `clients/{android,desktop,web}/node_modules/.../xdg-open` (third-party npm shims)
are NOT in this PR (an orphan waiver is fatal by design, and `clients/` is absent here). They belong to the PR that adds
`clients/` and must be added there with `pin_waivers.py add waive ...`.

Prerequisites (once): `podman build -t localhost/helix-gui-test:1 submodules/containers/images/gui-test`; a helix-server
binary (`HELIX_SERVER_BIN`); `export ISO_WORK=$HOME/.cache/claude-tmp/<dedicated-dir>` (state, fixtures, profiles; must lie strictly inside `~/.cache/claude-tmp` or `~/.cache/helix-iso`); Playwright
browsers in `~/.cache/ms-playwright`.

* Web (67 checks): `python3 -I qa/isolation/make_gui_fixtures.py web $ISO_WORK/fx-web`;
  `tools/isolation/start_server.sh start $HELIX_SERVER_BIN $ISO_WORK/st $ISO_WORK/fx-web`;
  `qa/isolation/run_web_live.sh $ISO_WORK/st <out>`; `tools/isolation/start_server.sh stop $ISO_WORK/st`.
* KOReader GUI: `qa/isolation/run_koreader_gui.sh <out>` (starts and stops its own server; also writes
  `kosync_bodies.jsonl` and the machine verdict `kosync_proof.json`).
* Desktop: `qa/isolation/run_desktop_live.sh $ISO_WORK/st <out>` (server as for web).
* HelixQA HTTP banks: `qa/isolation/run_helixqa_container.sh <out>` (`HELIXQA_BIN` = `CGO_ENABLED=0 go build ./cmd/helixqa`).
* HelixQA web bank: `tests/helixqa_web/run_web_bank.sh` (HRD-038, not run or modified here).
* Android: no runner in this PR (section 6).
* Evidence manifest: `tools/isolation/seal_evidence.sh <dir>` (atomic `SHA256SUMS`).

## 6. Android (canonical recipe)

Verified by another agent (`evidence/clients/android/run-20261123T211340Z/`): AVD `default` (API 34, google_apis, x86_64)
from `ghcr.io/vasic-digital/lava-android-emulator:api34-x86_64`, own rootless container, `--device /dev/kvm
--userns=keep-id`, `-no-window -no-audio`, no host display, `adb connect` to the mapped loopback port, a logging proxy for
server access. A script for that recipe was REMOVED from this PR (review round 1, finding I4): it ran outside the `--network none`
boundary (published adb port, host adb server), without `--timeout` and without the lease labels the reaper needs, so it could
not honestly be called isolated. Android emulator runs are therefore NOT covered by the guarantees of this document until a
wrapper with explicit, reviewed network allowance, `--timeout` and `org.helix.*` labels lands (tracked under HRD-022 / HRD-040).
The guard treats `emulator ... -no-window` as the allowed pattern.

## 7. HelixQA and the Vision oracle

`helixqa http` runs in the container (`run_helixqa_container.sh`: 34 cases, 32 PASS, 0 FAIL, 2 SKIP - the web cases,
because `helixqa http` never executes `playwright:` steps; they are executed by `tests/helixqa_web/run_web_bank.sh`).
The Vision model is not validated: any screenshot-based judgement is UNVERIFIED and no claim rests on it. Oracles used
here: DOM/accessibility assertions, HTTP bodies, server-side sink records (access log, KOSync request/response bodies,
redis), pixel probes of KOReader's rendered pages.

## 8. Known limits

* TOCTOU (review M2): `bindcheck.py` hands podman the RESOLVED path, which closes a swap of the leaf, but a parent component
  could still be replaced by a symlink between the check and podman's mount if a concurrent host-side writer exists. Not
  closed; mitigated only by `ISO_WORK` being a dedicated directory no other process should write.

### 8.1 Container-writable state the host later acts on or follows (review round 2)

Model: after a container has had a rw bind of a directory, everything in it is attacker-controlled (names, symlinks, hard links,
fifos, git pointer files, `.claude*` hooks). Rules, each with a test and a paired mutant in `tools/isolation/tests/test_review_round2.py`:

* Host-side access is symlink-safe: `tools/isolation/safe_io.py` (`put`, `copy`, `scan`, `first`) opens every component below the
  trusted base with `O_NOFOLLOW`, refuses non-regular and hard-linked files, and creates outputs with `unlink` + `O_EXCL|O_NOFOLLOW`.
  `container_proof.py` writes through it; `seal_evidence.sh` scans first and refuses (exit 3) a directory holding a link or special
  file; `qa/isolation/run_koreader_gui.sh` stops its container before any host read, builds every host log/derived file in a
  host-only directory (`$ISO_WORK/host-ko`, never bound) and publishes them with `safe_io put`. Enumerated host accesses to rw-bound
  paths: `run_koreader_gui.sh` (profiles `$PR`, out `$OUT`), `run_web_live.sh` / `run_desktop_live.sh` (only the `-p` proof, now via
  `safe_io`), `container_proof.py`, `seal_evidence.sh`, `fresh_out.sh` (moves a directory, never follows). `run_helixqa_container.sh`
  binds only read-only paths, so its `$OUT` is host-only. The socket directory bound with `-b` (`$ST/sock`) is read by the host only
  through the unix socket itself. RULE (round 3, pinned by `test_review_round3.SocketRule`): after a container start the host is
  never a client of `h.sock` - the directory is container-writable, so the container can replace the socket by its own listener;
  the only host use is the pre-start `curl` that creates the throwaway KOSync user, before `run_container.sh` runs.
* `bindcheck.py` refuses an `ISO_WORK`, and any rw source, that is or lies inside a git checkout/worktree (a `.git` directory or
  pointer file in the directory or an ancestor below the dedicated root), contains `.git`, `.claude*`, `.specify`, `.husky`,
  `.githooks`, or holds any symlink. Read-only binds are not affected.
* `run_container.sh`: `-m` 64m..16g (unit required, no leading zero), `-c` 0.10..8 (at most two decimals), `-t` 1..14400 s;
  `ICEAUTHORITY` joins the refused environment names.
* Lint class rule `container-outside-isolation`: any `podman|docker|nerdctl` `run|create|exec|cp` (round 3: and `podman-remote`, `buildah run`,
  `runc`, `lxc`, `apptainer`, `singularity`, `distrobox`, `toolbox`, `flatpak run`, `systemd-run`, `bwrap`, `nsenter`, `unshare`, `chroot`) is a
  finding unless a content-pinned `waive` row names the file (12 pre-existing service-container scripts and
  `run_koreader_gui.sh` = 13 rows in round 2; round 3 removed the blanket `tools/isolation/` exemption, so `run_container.sh`,
  `start_server.sh` and `tests/test_reaper.py` carry their own rows: 16 `container-outside-isolation` rows now). This replaces the endless per-flag list as the primary control for odd flags.
* Not run in this session: the container-gated tests (network isolation, runtime checks, reaper against real podman, canary probe) and
  every live run; the hermetic suites use a PATH `podman` stub.

* Chromium's own sandbox is OFF for the Electron run: with it on, Electron fails to launch in the unprivileged rootless
  container (`hardening/desktop_sandbox_attempt.log`; `--cap-add SYS_ADMIN` is not acceptable). Evidence therefore says
  "DOM/CSP/navigation behaviour verified with the Chromium sandbox disabled", never "sandboxed".
* The KOReader GUI is driven by coordinates calibrated against KOReader 2026.07-1 at 600x800; a first launch of a fresh
  profile needs a warm-up (the runner does it); KOReader is closed by SIGTERM and still pushes progress.
* The lint has residual classes (section 4); only the container is the boundary.
* Disk: the image is about 0.9 GB; browsers and KOReader are bind-mounted.

### 8.2 Post-run trust of container-written trees (review round 3)

Class: after the container ran, the host must treat every entry of every rw-bound tree as hostile BEFORE it seals, hashes, copies, commits or
executes anything. Rules, each with a test and a paired mutant in `tools/isolation/tests/test_review_round3.py`:

* ONE deny table (`safe_io.DENY_EXACT` / `DENY_PREFIX` / `deny_name`, case-insensitive) names the git / agent / tool control artifacts:
  `.git` (dir or file), `.gitattributes`, `.gitmodules`, `.gitignore`, `.claude*`, `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, `QWEN.md`, `.mcp.json`,
  `.envrc`, `.direnv`, `.vscode`, `.idea`, `.cursor*`, `.husky`, `.githooks`, `.specify`, `hooks`, `.hooks` and a few credential-config names. The pre-run
  `bindcheck.py` imports it (no copy) and the post-run `safe_io.py scan` uses it, so the two cannot drift. `scan` also refuses symlinks, fifos,
  devices, sockets and hard-linked files at any depth and never descends into a denied directory. A planted nested git repository with
  `core.fsmonitor` plus `.claude/settings.json` is refused by `seal_evidence.sh` (exit 3) with no SHA256SUMS written and no host `git`/`claude`
  executed (the test puts logging stubs first on PATH).
* Publish allow-list (corrected in round 4, see 8.3): the runners finish with `safe_io.py scan "$OUT" --allow` and `seal_evidence.sh` runs the same allow-list
  scan; the round-3 wording "`seal_evidence.sh` keeps the default scan" no longer holds. UNCONFIRMED: the allow-list was not exercised by a live
  web/desktop run (no container was started); a real run that writes another file type is refused, which is the intended fail-closed outcome.
* Pre-run rw sources are refused when they hold hard-linked files, fifos, devices or sockets as well as symlinks and denied names; the enclosing-checkout
  check now climbs to `/` (the dedicated roots and everything above them included); another runner's host-only directory (`$ISO_WORK/host-*`) is never
  bound in any mode; the tree walk's `SCAN_LIMIT` is checked per entry and is itself tested (M9).
* Proofs and verdicts live in a host-only directory: `run_container.sh -p FILE` refuses a FILE that is relative, not canonical, in a missing / linked
  directory or inside any rw-bound source; the runners pass `$ISO_WORK/host-ko/` or `$ST/host-proof/` and publish with `safe_io put` AFTER the container
  stopped. EXISTING committed proofs under `evidence/campaign-2026-11-23/isolated/{web,koreader,desktop}/container_proof.json` were written by the OLD flow
  (the proof inside the container-writable directory); they are frozen evidence and are NOT regenerated here. Anything produced by the new flow carries the
  host-only path in its `-p` argument. `run_helixqa_container.sh` binds no rw path, so its proof in `$OUT` is host-only already.
* Volume caps: `safe_io.py scan` refuses more than 20000 files, 4 GiB in total, one file above 2 GiB or more than 12 directory levels
  (`SAFE_IO_MAX_FILES|TOTAL|FILE|DEPTH` override; a malformed value is a refusal), and `run_container.sh` passes `--ulimit fsize=2147483648` so a
  single container-written file cannot exceed 2 GiB while the container runs. UNCONFIRMED / not closed: there is no run-time cap on the TOTAL bytes a
  container writes to a bind-mounted directory (rootless podman offers no portable quota for bind mounts; `--storage-opt size` applies to the image
  layer, not to binds); the total is bounded only by the post-run refusal and by the 512 MiB `/tmp` tmpfs.
* Lint: the runtime family is extended (see section 8.1), the blanket `tools/isolation/` exemption of the class rule is gone, and the escape flags
  `--rootfs`, `--env-host`, `--pod`, a relative `-v .:`, `--userns=host` and `unconfined` security options are escapes in every file, `tools/isolation/`
  included; matrix entries for each were added.
* Behavioural (not hash-pin) coverage of ordering and fatality: the harness runs the real runner scripts in a repo-shaped temp tree with podman /
  curl / server / `run_container.sh` stubs and ONE event log: the container is stopped before the first host `safe_io` read (M5); a failing final scan
  makes the runner exit 1 (M6); `safe_io` read size cap holds even when `fstat` under-reports (M4).

### 8.3 Allow-list instead of name list, sanitized container bytes (review round 4)

Class R4-B1 "container-planted control artifacts" was still open in round 3 because it was closed by naming members. It is inverted: the post-run scan of
a container-written tree (`safe_io.py scan DIR --allow`, used by `seal_evidence.sh` and by every runner) accepts only what is positively known to be safe.
Tests and paired mutants: `tools/isolation/tests/test_review_round4.py`.

* A file must have a name matching `[A-Za-z0-9_][A-Za-z0-9._-]{0,127}` (no leading dot or dash, no escape byte, newline, space, `;`), an extension in
  `png jpg jpeg json jsonl log txt csv mp4 webm gif webp`, link count 1 and be a regular file. A directory must match `[A-Za-z0-9_][A-Za-z0-9_-]*`
  (no dots at all), so every dot-directory (`.git`, `.github`, `.opencode`, `.kimi`, ...) is refused wholesale. `md` and `html` are NOT publishable:
  a container-authored `.md` can be an agent instruction file and a container-authored `.html` runs its script when the operator opens it in the host
  browser; no runner publishes either (the runners' output is png / log / json / jsonl / txt / mp4). Names that look like checksum manifests
  (`SHA256SUMS*`, `.SHA256SUMS*`, any case) are refused anywhere, so nothing can sit in a "sealed" directory outside the seal; the only host-written
  manifest is the top-level `SHA256SUMS` that `seal_evidence.sh` itself replaces atomically (`--host-seal`).
* A directory whose entry names make git accept it as a git directory (`HEAD` or `config` together with `objects` or `refs`, or `gitdir` / `commondir` /
  `worktrees` with those) is refused by every scan and by the pre-run bindcheck: an embedded BARE repository needs no `.git` name and would run its
  `core.fsmonitor` once any host git command ran inside it. The deny table (8.2) gained the non-dot agent names the review listed (`CLAUDE.local.md`,
  `AGENT.md`, `CRUSH.md`, `CONVENTIONS.md`, `copilot-instructions.md`, `opencode.json`, ...) and the dot prefixes `.github .opencode .crush .kimi .pi .roo
  .continue .junie .amazonq .kiro`; this list is a second line of defence only, the allow-list is the control. RESIDUAL (honest): a publishable-extension
  file with a tool-config name that is on no list (a `.json` that some agent auto-loads from a nested directory) passes; which agents auto-load which
  nested files is UNCONFIRMED, and the real mitigation is that the host never runs an agent, git or build tool inside an evidence directory.
* `seal_evidence.sh DIR` refuses (exit 3, no temp file, no SHA256SUMS) anything else. `seal_evidence.sh --host-authored DIR` is the explicit opt-in for a
  directory only the host ever wrote (docs, scripts, reports: the kinds-and-deny-table scan); no runner uses it and it must never be used on a directory
  a container could write. Evidence directories committed earlier that hold scripts or reports were sealed before this rule; they are frozen and are not
  re-sealed with the allow-list mode.
* Tainted trees are kept, not deleted: `fresh_out.sh` moves a previous output directory to `~/.cache/claude-tmp/evidence-archive` (dedicated root) and
  nothing prunes it. Retention policy: the operator removes an archive entry only after inspection, by hand, never with a tool that follows links; a
  tainted tree is never opened with an editor, agent or git client. The archive is outside the repository and is never committed.
* Container bytes never reach the operator's terminal or a host-written log raw. `tools/isolation/sanitize.py` is the one filter: it removes OSC / DCS /
  APC / PM / SOS / CSI / other ESC sequences (also unterminated ones), every C0 control except newline and tab (so CR, BEL, BS, ESC, NUL), DEL, the C1
  range, and bidirectional / line-separator controls, and prefixes EVERY line with `[container] `, so a forged `KOSYNC PROOF PASS` line is visibly
  container-origin. It is applied on: the attached `podman run` of `run_container.sh` (stdout and stderr merged through the filter; podman's exit status
  is preserved through `PIPESTATUS`; detached mode prints only the host-generated container id and keeps `exec`), every `podman exec` of
  `run_koreader_gui.sh` (function `D`), the proxy-log sink (`[proxy-log] ` prefix), the redis dump `redis_sink.txt`, the published `ko_A.log` / `ko_B.log`
  (copied through `safe_io` to the host-only directory first) and the kosync proof's printed text. `access_proxy.py` writes the request method, path
  and User-Agent percent-escaped (`esc`), so a request line cannot carry a sequence or forge a log line; `access.log` and `kosync_bodies.jsonl` are
  therefore already clean. UNCONFIRMED: whether the operator's outer terminal accepts OSC 52 (it was not tested because it would touch the desktop);
  the filter removes the bytes either way. The text a container prints is still only evidence of what it printed, not of what happened.
* `run_koreader_gui.sh` runs with `pipefail`; the KOSync proof's exit status is captured, a non-zero status prints `KOSYNC PROOF FAILED` into steps.log
  and makes the runner exit 1 after publishing (a crashing proof script too). Previously the status was `tee`'s and the failure branch was dead.
* Lint (round 4): global runtime options between the CLI and the subcommand (`podman --root X run`, `docker --context c run`, `docker -H h run`,
  `podman --remote exec`) are normalised away before the class and escape rules; `crun`, `incus`, `ctr`, `kubectl`, `firejail`, `toolbx`,
  `systemd-nspawn`, `machinectl`, `flatpak-spawn`, `multipass`, `limactl`, `vagrant`, `host-spawn`, `ch-run`, `podman start -a`, `container runlabel`,
  `pod create` and the Python / JS list form of the whole family are findings; 26 matrix entries were added. Waiver counts after this round:
  16 container-outside-isolation rows (unchanged, `run_container.sh` repinned), 9 `self` sources (round 3 had 8; the new one is `test_review_round4.py`).
