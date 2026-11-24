# qa/lcp-interop (HRD-046)

Interop of `core/pkg/lcp` against a THIRD-PARTY provider: the Readium reference License Server, Status Server and
`lcpencrypt` (github.com/readium/readium-lcp-server), run as real processes inside a rootless, network-less container
(`tools/isolation/run_container.sh`, `--network none`, caps dropped). Nothing here touches the host desktop (docs/test-isolation.md).
Without `HELIX_LCP_INTEROP_LCP_URL` the Go test skips (an honest SKIP, never a pass).

Scope: a library consumer only. Not verified: the real EDRLab/Readium root (HRD-122), renew (provider config has `renew: false`),
the server/OPDS path, a third-party reader opening a Helix artifact.

## Files

| File | Role |
|---|---|
| `run_interop.sh` | host driver: random credentials, container run, host-side secret scan, sealing; exit status = verdict |
| `inside.sh` | runs inside the container: starts the provider, encrypts the sample EPUB, runs the Helix test, scans `/out` for secrets |
| `core/pkg/lcp/interop_readium_test.go` | the interop test (`TestInteropReadiumLCPServer`) |

## Prerequisites (an `<iso>` work dir, ABSOLUTE path)

`<iso>/bin` : `lcpserver lsdserver lcpencrypt lcp.test`; `<iso>/in` : `sample.epub cert-edrlab-test.pem` (+ `lorem.epub` for the mutant).

1. Provider binaries (CGO on, module download needs network; do this step only, then go back to `--network none`):
   `git clone https://github.com/readium/readium-lcp-server.git src && git -C src checkout 32570b69d02948801f161e7923648dcc0bbfb178`, then in
   `docker.io/library/golang@sha256:79ecd0a7157c7a4cf3b281ced8465350591bf54fedab540e75374a003e1fde7e` run
   `CGO_ENABLED=1 go build -o /out/lcpserver ./lcpserver` (likewise `./lsdserver`, `./lcpencrypt`) with `src` mounted at the working dir.
2. Helix test binary, no network:
   `podman run --rm --network none --cap-drop ALL --security-opt no-new-privileges -e CGO_ENABLED=0 -e GOPROXY=off -e GOFLAGS=-mod=mod -e GOCACHE=/gc -v <repo>/core:/src:ro -v <out>:/out -v <gocache>:/gc -w /src docker.io/library/golang@sha256:0fd028db9bdd4f01657bc8047c209b04d8ce19b853d776bd6f1e76a41729b936 go test -c -o /out/lcp.test ./pkg/lcp/`
   (the module cache in `<gocache>` must already hold the dependencies).
3. Inputs, all from `readium-lcp-server@32570b69` `test/`: `samples/sample.epub` (sha256 9c516119872123fe61ca8311b525b980a08c52fcb1710777fc10f87c33caf3c7),
   `samples/lorem.epub` (sha256 291086e08da8671a702f64f3b5594113f0684f076f3957080a2c9ed7321bcdef), `cert/cert-edrlab-test.pem`
   (sha256 874afaf5bd6b8d2262d541ea867dce31452e3cd63b98699797741e9298738f83). The hashes were compared with the copies used for the evidence.

## Run

    qa/lcp-interop/run_interop.sh <iso> evidence/lcp-interop-runN                       # expected exit 0
    qa/lcp-interop/run_interop.sh <iso> evidence/lcp-interop-mutant-wrong-oracle lorem.epub   # expected exit 1 (wrong plaintext oracle)

The driver archives an existing evidence dir (never deletes it), runs the container, scans the result for the content-encryption-key
marker and for the credential values, and only then seals the directory with `tools/isolation/seal_evidence.sh` (writes `SHA256SUMS`).
A failed scan exits 3 and seals nothing; do not commit that directory.

## Secrets

`lcpencrypt` must NOT run with `-verbose`: that prints the content-encryption-key. The first evidence commit of PR #33 (be88c4b, still
reachable through `refs/pull/33/head`) did exactly that, and four ephemeral test content keys are therefore PUBLIC by disclosure. They
protect only the throw-away test ciphertext and are not retained; history is not rewritten (never force-push, 11.4.113).
