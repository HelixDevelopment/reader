import json, subprocess, sys, tempfile, os, unittest
HERE = os.path.dirname(os.path.abspath(__file__))
def run(rows, *args):
    with tempfile.NamedTemporaryFile("w", suffix=".log", delete=False) as f:
        f.write("not json\n"); [f.write(json.dumps(r) + "\n") for r in rows]
    try:
        p = subprocess.run([sys.executable, "-I", os.path.join(HERE, "check_h3.py"), f.name, *args], capture_output=True, text=True)
        return p.returncode, json.loads(p.stdout)
    finally:
        os.unlink(f.name)
A = lambda proto, enc="br", path="/v1/formats": {"msg": "access", "proto": proto, "encoding": enc, "path": path}
class T(unittest.TestCase):
    def test_h3_pass(self): self.assertEqual(run([A("HTTP/3.0")] * 3, "/v1/formats", "HTTP/3.0", "br")[0], 0)
    def test_h2_is_fail_when_h3_expected(self): self.assertEqual(run([A("HTTP/2.0")], "/v1/formats", "HTTP/3.0", "br")[0], 1)
    def test_mixed_protocols_fail(self): self.assertEqual(run([A("HTTP/3.0"), A("HTTP/2.0")], "/v1/formats", "HTTP/3.0", "br")[0], 1)
    def test_wrong_encoding_fail(self): self.assertEqual(run([A("HTTP/3.0", "gzip")], "/v1/formats", "HTTP/3.0", "br")[0], 1)
    def test_no_lines_is_blind_not_pass(self): self.assertEqual(run([A("HTTP/3.0", path="/other")], "/v1/formats", "HTTP/3.0", "br")[0], 2)
    def test_min_count(self): self.assertEqual(run([A("HTTP/3.0")], "/v1/formats", "HTTP/3.0", "br", "2")[0], 2)
    def test_red_polarity_http11(self): self.assertEqual(run([A("HTTP/1.1", "gzip")], "/v1/formats", "HTTP/1.1", "any")[0], 0)
if __name__ == "__main__": unittest.main()
