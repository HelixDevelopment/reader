"""Unit + mutation tests for assert_host_untouched. Run: python3 -I -m unittest discover -s tools/isolation/guard -v"""
import os, subprocess, sys, tempfile, unittest, pathlib
HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import assert_host_untouched as g  # noqa: E402

FORBIDDEN = {
    "xdotool": "xdotool key Return\n",
    "ydotool": "ydotool click 0xC0\n",
    "wtype": "wtype hello\n",
    "uinput": "open('/dev/uinput','wb')\n",
    "scrcpy": "scrcpy --serial emu\n",
    "emulator-windowed": "emulator -avd Pixel_6 -no-audio\n",
    "display-0": "DISPLAY=:0 firefox\n",
    "wayland-env": "export WAYLAND_DISPLAY=wayland-0\n",
    "grim": "grim shot.png\n",
    "gnome-screenshot": "gnome-screenshot -f a.png\n",
    "import-root": "import -window root a.png\n",
    "pw-headless-false": "chromium.launch({ headless: false })\n",
    "pw-headed": "npx playwright test --headed\n",
    "electron": "npx electron .\n",
    "electron-pkgjson": '"start": "electron ."\n',
    "electron-bin": "node_modules/.bin/electron main.js\n",
}

def scan_tree(files):
    with tempfile.TemporaryDirectory() as d:
        for name, body in files.items():
            p = pathlib.Path(d, name); p.parent.mkdir(parents=True, exist_ok=True); p.write_text(body)
        return g.scan(d)

class Guard(unittest.TestCase):
    def test_clean_tree_passes(self):
        self.assertEqual(scan_tree({"a.sh": "echo hi\n", "b.mjs": "chromium.launch({ headless: true })\n",
                                    "c.sh": "emulator -avd x -no-window -no-audio\n"}), [])

    def test_every_forbidden_pattern_is_caught(self):  # mutation check
        for label, body in FORBIDDEN.items():
            with self.subTest(label=label):
                self.assertTrue(scan_tree({"tools/x.sh": "#!/bin/sh\n" + body}), f"{label} not detected")

    def test_marker_allowlists_container_only_code(self):
        body = "#!/bin/sh\n# isolation: runs-inside-container\nxdotool key Return\n"
        self.assertEqual(scan_tree({"qa/isolation/inside/x.sh": body}), [])

    def test_marker_not_honoured_deep_in_file(self):
        body = "#!/bin/sh\n" + "\n" * 40 + "# isolation: runs-inside-container\nxdotool key a\n"
        self.assertTrue(scan_tree({"x.sh": body}))

    def test_allowlisted_dir_requires_no_marker(self):
        self.assertEqual(scan_tree({"submodules/containers/images/gui-test/e.sh": "xdotool key a\n"}), [])

    def test_agent_prompt_is_scanned(self):
        self.assertTrue(scan_tree({".claude/agents/qa.md": "Run xdotool to click\n"}))

    def test_plain_docs_not_scanned(self):
        self.assertEqual(scan_tree({"docs/x.md": "never use xdotool\n"}), [])

    def test_containerised_android_emulator_is_allowed_pattern(self):
        body = "podman run -d --device /dev/kvm --userns=keep-id img emulator -avd default -no-window -no-audio\n"
        # the emulator/device/kvm pattern itself raises no rule; since round 3 every runtime call outside a pinned waiver is the class finding, nothing else
        self.assertEqual([h for h in scan_tree({"tools/isolation/a.sh": body}) if h[2] != "container-outside-isolation"], [])

    def test_cli_exit_codes(self):
        with tempfile.TemporaryDirectory() as d:
            pathlib.Path(d, "ok.sh").write_text("echo ok\n")
            r = subprocess.run([sys.executable, "-I", str(HERE / "assert_host_untouched.py"), d], capture_output=True)
            self.assertEqual(r.returncode, 0, r.stderr)
            pathlib.Path(d, "bad.sh").write_text("xdotool key a\n")
            r = subprocess.run([sys.executable, "-I", str(HERE / "assert_host_untouched.py"), d], capture_output=True)
            self.assertEqual(r.returncode, 1)
            self.assertIn(b"bad.sh:1", r.stdout)

class RunIsolated(unittest.TestCase):
    SH = str(HERE / "run_isolated.sh")
    def run_it(self, env, *cmd):
        return subprocess.run([self.SH, *cmd], env=env, capture_output=True, text=True)
    def test_env_is_sanitised(self):
        env = {"PATH": os.environ["PATH"], "DISPLAY": ":0", "WAYLAND_DISPLAY": "w0", "XAUTHORITY": "/x",
               "DBUS_SESSION_BUS_ADDRESS": "unix:path=/x", "XDG_RUNTIME_DIR": "/run/user/1000", "FOO": "bar"}
        r = self.run_it(env, "--", "env")
        self.assertEqual(r.returncode, 0, r.stderr)
        for k in ("DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR", "FOO"):
            self.assertNotIn(k + "=", r.stdout)
    def test_refuses_when_display_required(self):
        r = self.run_it({"PATH": os.environ["PATH"]}, "--require-display", "--", "true")
        self.assertEqual(r.returncode, 64)
    def test_refuses_command_that_needs_host_display(self):
        r = self.run_it({"PATH": os.environ["PATH"]}, "--", "xdotool", "key", "a")
        self.assertEqual(r.returncode, 64)
    def test_broad_allow_pattern_cannot_leak_session_vars(self):
        env = {"PATH": os.environ["PATH"], "DBUS_SESSION_BUS_ADDRESS": "unix:path=/x", "HELIX_A": "1"}
        r = subprocess.run([self.SH, "--allow", "DBUS_*", "--", "env"], env=env, capture_output=True, text=True)
        self.assertNotIn("DBUS_SESSION", r.stdout)
    def test_allowlisted_var_passes(self):
        r = self.run_it({"PATH": os.environ["PATH"], "HELIX_BASE_URL": "http://x"}, "--", "env")
        self.assertIn("HELIX_BASE_URL=http://x", r.stdout)

if __name__ == "__main__":
    unittest.main()
