// Status of the Chromium-sandbox observation. It is a RECORDED fact, never a PASS unless the sandbox was actually observed ON
// (renderers with seccomp mode 2 and no --no-sandbox). OFF + not expected -> WARN (visible, non-failing); OFF + expected -> FAIL.
export function sandboxStatus(sandboxed, expected) {
  if (sandboxed) return 'PASS'
  return expected ? 'FAIL' : 'WARN'
}
