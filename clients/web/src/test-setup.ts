import '@testing-library/jest-dom/vitest'

// jsdom lacks Blob.text() and object URLs; real browsers (and the Playwright run) have both.
if (!Blob.prototype.text) {
  Blob.prototype.text = function (this: Blob) {
    return new Promise<string>((resolve, reject) => {
      const r = new FileReader()
      r.onload = () => resolve(String(r.result))
      r.onerror = () => reject(r.error)
      r.readAsText(this)
    })
  }
}
if (!URL.createObjectURL) URL.createObjectURL = () => 'blob:test'
if (!URL.revokeObjectURL) URL.revokeObjectURL = () => {}

// Node 26 ships an experimental global localStorage that shadows jsdom's and is
// unusable without --localstorage-file; replace both with in-memory stores.
function memoryStorage(): Storage {
  const m = new Map<string, string>()
  return {
    get length() { return m.size },
    clear: () => m.clear(),
    getItem: (k) => (m.has(k) ? m.get(k)! : null),
    key: (i) => Array.from(m.keys())[i] ?? null,
    removeItem: (k) => { m.delete(k) },
    setItem: (k, v) => { m.set(k, String(v)) },
  }
}
for (const name of ['localStorage', 'sessionStorage'] as const) {
  Object.defineProperty(globalThis, name, { value: memoryStorage(), configurable: true })
}
