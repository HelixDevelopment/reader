# HTTP/3 + Brotli for the Helix Reader clients (Stream A, phase 1)

Server: Go + Gin, TCP http.Server and quic-go http3.Server on the same port, `Alt-Svc: h3=":PORT"; ma=86400`, Brotli via the compression middleware.
Evidence rule: the SERVER access log (HELIX_ACCESS_LOG=true, `proto=` and `encoding=`) is the verdict, judged by qa/h3/check_h3.py (PASS/FAIL/BLIND).
Lab: tools/isolation/h3_lab.sh (rootless podman `--internal` network; UDP/QUIC between containers; no host display, no --net=host).

| Client | Stack | Result |
|---|---|---|
| Web (Chromium) | browser h3 | LIVE: HTTP/3.0 + br when QUIC is forced; with a private CA and no force flag Chromium stays on HTTP/2.0 + br (Alt-Svc discovery refused). Production needs a publicly trusted cert and UDP/PORT open. |
| Desktop (Electron) | Chromium session (`session.fromPartition('helix-api').fetch`) instead of Node fetch | LIVE: HTTP/3.0 + br. RED baseline (PR #18 shell, Node fetch): HTTP/1.1. Private-CA lab needs `origin-to-force-quic-on` + `ignore-certificate-errors-spki-list` (clients/desktop/lib/transport.js). |
| Android | Cronet (cronet-bundled 500.1.0, embedded) via WebViewClient.shouldInterceptRequest | Code done; JVM unit test (ApiRequestPolicyTest) committed, executed only by a Gradle run that was not part of the review (UNCONFIRMED here); APK builds. Live on emulator: transport works, Brotli decoded (br), but negotiated HTTP/2 not h3 (see below). UNCONFIRMED for h3. |
| KOReader | Lua, no QUIC | Honest limit: HTTP/2 or 1.1 + Brotli only (server falls back transparently). |

## Android findings
- Cronet needs `ACCESS_NETWORK_STATE` (SecurityException crash otherwise) - fixed in the manifest.
- Emulator guest resolves `helix-server` on the lab network; TLS with the per-run CA works through a lab-only network-security-config.
- ROOT CAUSE (Cronet net-log, evidence evidence/h3-cronet/android/cronet_netlog.json): UDP reaches the server and the QUIC handshake starts (forced QUIC option works, 
  ALT_SVC_FOUND, QUIC_SESSION created, server Initial/Handshake packets received), then Cronet closes the session with
  `TLS handshake failure 46: certificate unknown ... CERTIFICATE_VERIFY_FAILED` (net_error -356) and falls back to h2 over TCP, where the same CA is trusted
  through the network-security-config. So the QUIC path of Chromium's verifier rejects the private-CA certificate. Putting the CA into the emulator system store
  (tmpfs bind over the conscrypt apex cacerts in the zygote namespaces, LAB_SYSTEM_CA=1) did not change the result in one attempt: UNCONFIRMED whether a correct
  system-store install, or Chromium's known-root requirement for QUIC, is the reason. Desktop/web pass only because Chromium accepts the SPKI-pin flag; Cronet has no
  public equivalent. Android h3 is therefore verified only up to the QUIC handshake; production (public CA) is expected to work but is UNCONFIRMED here.
- Next candidates: a publicly-trusted-style chain in the lab (CA preinstalled as a real system cert via writable-system emulator), Cronet `setExperimentalOptions` host-resolver rules / known-root override if any exists in 500.1.0.

## Test policy (Android)
`CronetLiveTest`: `cronetTransportServesBrotliOverH2OrH3` always asserts (decoded Brotli JSON, h2 or h3); `cronetTransportSpeaksHttp3` is an honest
SKIP-with-reason while the private-CA QUIC verify failure (net_error -356) persists, and a hard assertion with `-e expectH3 1` (public certificate).
Follow-up to file: writable-system emulator / known-root approach for Cronet QUIC in the lab.
