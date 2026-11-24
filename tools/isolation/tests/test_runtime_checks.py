"""F2 runtime gate: the in-container launch.sh refuses a container that carries host display/session state."""
import pathlib, subprocess, sys, unittest
ROOT = pathlib.Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "tools/isolation")); from rawrun import run_direct, run_raw     # R6-B1: container output only ever comes back through a Raw (sanitized str surfaces)
IMG = "localhost/helix-gui-test:1"
def direct(extra, cmd=("true",)):
    # Deliberately bypasses run_container.sh (which would refuse these env vars) to prove the RUNTIME check on its own.
    return run_direct(["podman", "run", "--rm", "--network", "none", "--name", "hrdiso-rt", *extra,
                       "-v", f"{ROOT}/qa/isolation/inside:/opt/isoinside:ro", IMG, "sh", "/opt/isoinside/launch.sh", "--", *cmd], timeout=90)
class Runtime(unittest.TestCase):
    def test_clean_container_passes(self):
        r = direct([]); self.assertEqual(r.returncode, 0, r.stderr)
    def test_each_poisoned_env_is_refused(self):
        for k, v in (("WAYLAND" + "_DISPLAY", "wayland-0"), ("XAUTH" + "ORITY", "/x"), ("DBUS_SESSION_BUS_ADDRESS", "unix:path=/x"),
                     ("XDG_RUNTIME_DIR", "/run/user/1000"), ("SSH_AUTH_SOCK", "/s")):
            with self.subTest(var=k):
                r = direct(["-e", f"{k}={v}"])
                self.assertEqual(r.returncode, 65, (k, r.stderr)); self.assertIn("isolation violation", r.stderr)
    def test_host_display_value_cannot_survive_entrypoint(self):
        # the image entrypoint pins the container-private :99 whatever was passed in
        r = direct(["-e", "DIS" + "PLAY=:0"], ("sh", "-c", "echo D=$DISPLAY"))
        self.assertEqual(r.returncode, 0, r.stderr); self.assertIn("D=:99", r.stdout)
    def test_private_display_value_is_allowed(self):
        self.assertEqual(direct(["-e", "DIS" + "PLAY=:99"]).returncode, 0)
    def test_extra_network_interface_refused(self):
        r = run_direct(["podman", "run", "--rm", "--name", "hrdiso-rt2", "--network", "pasta", "-v", f"{ROOT}/qa/isolation/inside:/opt/isoinside:ro",
                        IMG, "sh", "/opt/isoinside/launch.sh", "--", "true"], timeout=90)
        self.assertEqual(r.returncode, 65, r.stderr)
    def test_container_env_via_wrapper_has_no_host_session(self):
        r = run_raw(["-n", "hrdiso-rt3"], ["env"], timeout=90)
        for k in ("WAYLAND", "XAUTHORITY", "DBUS", "XDG_RUNTIME_DIR", "SSH_AUTH_SOCK"):
            self.assertNotIn(k, r.stdout)
if __name__ == "__main__":
    unittest.main()
