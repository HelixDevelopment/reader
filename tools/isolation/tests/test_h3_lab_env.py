"""h3_lab.sh run -e must apply the same host display/session env denylist as run_container.sh (gate-logic test: a podman
stub on PATH records the argv; the denylist decision is made by the script before any podman call)."""
import os, stat, subprocess, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
LAB = os.path.join(os.path.dirname(HERE), "h3_lab.sh")


def run_lab(*envs):
    with tempfile.TemporaryDirectory() as d:
        st = os.path.join(d, "st"); os.makedirs(os.path.join(st, "lab"))
        open(os.path.join(st, "rid"), "w").write("abcd\n")
        bindir = os.path.join(d, "bin"); os.makedirs(bindir)
        stub = os.path.join(bindir, "podman")
        open(stub, "w").write('#!/bin/sh\necho "PODMAN-CALLED $*"\n')
        os.chmod(stub, os.stat(stub).st_mode | stat.S_IXUSR)
        args = [LAB, "run", st, "hrdiso-h3-envtest"]
        for e in envs:
            args += ["-e", e]
        args += ["--", "true"]
        env = {"PATH": bindir + ":" + os.environ["PATH"], "HOME": d}
        p = subprocess.run(args, capture_output=True, text=True, env=env)
        return p.returncode, p.stdout, p.stderr


class T(unittest.TestCase):
    def test_denied_names_refused_before_podman(self):
        for e in ["DISPLAY=:0", "WAYLAND_DISPLAY=w", "DBUS_SESSION_BUS_ADDRESS=x", "SSH_AUTH_SOCK=/s", "XDG_RUNTIME_DIR=/r",
                  "GTK_THEME=x", "XAUTHORITY=/x", "PULSE_SERVER=x"]:
            rc, out, err = run_lab(e)
            self.assertEqual(rc, 64, e); self.assertNotIn("PODMAN-CALLED", out, e)

    def test_bare_name_refused(self):
        rc, out, _ = run_lab("HOME"); self.assertEqual(rc, 64); self.assertNotIn("PODMAN-CALLED", out)

    def test_invalid_name_refused(self):
        rc, out, _ = run_lab("a-b=1"); self.assertEqual(rc, 64); self.assertNotIn("PODMAN-CALLED", out)

    def test_plain_lab_var_allowed(self):
        rc, out, _ = run_lab("HELIX_LAB_URL=https://helix-server:8443")
        self.assertEqual(rc, 0); self.assertIn("PODMAN-CALLED", out); self.assertIn("-e HELIX_LAB_URL=https://helix-server:8443", out)


if __name__ == "__main__":
    unittest.main()
