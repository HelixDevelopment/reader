# Phase: CLARIFY — resolved doubts (each with research basis)

1. **Go vs Rust for the core?** RESOLVED: Go per operator mandate. Risk: WASM size and
   page-render hot loops; mitigation: rendering stays pluggable; WASM via TinyGo if
   binary size >6MB. Constitution R1 records the decision + reversal gate (measured).
2. **How do platforms without raw sockets (Web) use FTP/SMB?** RESOLVED: Helix Bridge
   helper/self-host proxy (docs/29), stated honestly in UI. No bluff about web parity.
3. **CRDT library in Go?** Evaluated: no mature Go Automerge binding → implement
   domain-narrow CRDT (LWW registers + add-wins sets over HLC) in `core/sync`, because our
   document types are small and closed (docs/10). Avoids cgo-in-WASM problems.
4. **EPUB parsing in Go?** Use golang.org/x/net/html tokenizer + own OPF/OCF layer; FXL via
   viewport metadata. PDF: delegate to platform MuPDF via FFI (GPL-safe dynamic linking) —
   open item OWED-1 flagged in analyze.
5. **Kindle MyClippings fuzzy matching threshold?** 0.85 auto-match, else user picker
   (spec'd doc 35). False-merge risk handled by user confirmation.
6. **Identification provider order per language?** OpenLibrary→GoogleBooks for Latin;
   Douban first for CJK; configurable chain (docs/26).
7. **E2E + server search conflict?** E2E on → server search disabled; client-side index
   only. Stated in settings (honest degradation, no hidden trade-off).
8. **containers submodule role?** It provides VM/simulator/emulator orchestration; our CI
   matrix definition lives in `.helix/matrix.yaml` consumed by it (schema TODO OWED-2:
   containers' own format must be adopted, not invented).
