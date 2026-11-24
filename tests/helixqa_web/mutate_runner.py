#!/usr/bin/env python3
"""HRD-038 mutation matrix for runner.go: every mutant must be KILLED. Runs on a temp copy; the working tree is never modified."""
import os, sys
HERE = os.path.dirname(os.path.abspath(__file__)); ROOT = os.path.realpath(os.path.join(HERE, "../.."))
sys.path.insert(0, os.path.join(ROOT, "scripts"))
import mutlib
R = "tests/helixqa_web/runner.go"
M = [
 (R, "M1 OK ignores skips", "r.Fail == 0 && r.Skip == 0 &&", "r.Fail == 0 && true &&"),
 (R, "M2 _skip step executes", "case st.Skip:", "case false:"),
 (R, "M3 non-playwright step not skipped", "case typ != testbank.ActionTypePlaywright:", "case false:"),
 (R, "M4 executor Blocked counts as FAIL not SKIP", "case res.Skipped || res.Blocked:", "case res.Skipped:"),
 (R, "M5 FAIL does not stop the case", 'if sr.Verdict == "FAIL" {\n\t\t\t\tbreak', 'if false {\n\t\t\t\tbreak'),
 (R, "M6 every verb retried", 'return strings.HasPrefix(v, "assertvisible ") || strings.HasPrefix(v, "assertnotvisible ")', "return true"),
 (R, "M7 retry window unbounded", "time.Now().Before(deadline)", "true"),
 (R, "M8 OK ignores passes", "r.Skip == 0 && r.Pass > 0", "r.Skip == 0 && true"),
 (R, "M9 no retry at all", "if !assertVerb(value) {", "if true {"),
 (R, "M10 skip counted as pass", "default:\n\t\t\t\trep.Skip++", "default:\n\t\t\t\trep.Pass++"),
 (R, "M11 screenshot capture error ignored", "err != nil || len(png) == 0", "false"),
 (R, "M12 screenshot save error ignored", "err != nil || p == \"\" || h == \"\"", "false"),
 (R, "M13 missing shooter allowed", "o.Shoot == nil || o.Save == nil", "false"),
 (R, "M14 pins ignored (bank shrink)", "if len(rep.Cases) != ec {", "if false {"),
 (R, "M15 step pin ignored", "if steps != es {", "if false {"),
 (R, "M16 pass pin ignored", "if rep.Pass != es {", "if false {"),
 (R, "M17 unpinned bank allowed", "if !o.AllowUnpinned {", "if false {"),
 (R, "M18 PinErrors ignored by OK", " && len(r.PinErrors) == 0", ""),
 (R, "M19 platform filter ignored", "if !platformOK(tc, o.Platform) {", "if false {"),
 (R, "M20 attempts not recorded", "attempts++", ""),
 (R, "M21 viewport not enforced", "o.ExpectViewport != [2]int{} &&", "false &&"),
 (R, "M22 page-state failure ignored", 'sr.Verdict, sr.Message = "FAIL", "page state probe failed: "+err.Error()', "_ = err"),
 (R, "M24 duplicate case ids allowed", "if seen[tc.ID] || (tc.ID != \"\" && seen[safeName(tc.ID)]) {", "if false {"),
 (R, "M25 sanitised-id collision allowed", '(tc.ID != "" && seen[safeName(tc.ID)])', "false"),
 (R, "M26 unsafe names not sanitised", "s = unsafeName.ReplaceAllString(s, \"_\")", ""),
 (R, "M27 dotdot allowed", 's = strings.ReplaceAll(s, "..", "_")', ""),
 (R, "M28 empty name allowed", 'if s == "" || s == "." {', "if false {"),
 (R, "M29 pins not carried into the report", "rep.ExpectedCases, rep.ExpectedSteps = ec, es", ""),
 (R, "M23 page state not recorded", "sr.Page = &ps", "_ = ps"),
]

def prepare(tmp):
    d = os.path.join(tmp, "tests/helixqa_web")
    subs = ["helixqa", "challenges", "containers", "doc_processor", "llm_orchestrator", "llm_provider", "security", "vision_engine", "llms_verifier/llm-verifier"]
    with open(os.path.join(d, "go.work"), "w") as f:
        f.write("go 1.26\n\nuse (\n\t.\n" + "".join(f"\t{ROOT}/submodules/{s}\n" for s in subs) + ")\n")
    for n in ("go.work.sum",):
        p = os.path.join(ROOT, "tests/helixqa_web", n)
        if os.path.exists(p):
            import shutil; shutil.copy(p, d)

sys.exit(mutlib.run(["tests/helixqa_web", "scripts/mutlib.py"], M, ["go", "test", "-count=1", "-timeout", "25s", "."], "tests/helixqa_web",
                    env={"HELIX_WEB_BANK": f"{ROOT}/qa/helixqa/banks/helix-web.yaml"}, prepare=prepare, timeout=120))
