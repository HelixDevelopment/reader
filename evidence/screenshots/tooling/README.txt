HRD-076 tooling evidence. bindcheck_bridge_socket_RED.log: the new test (bridge directory holding its real unix socket h.sock) FAILS against the
unfixed tools/isolation/bindcheck.py (every -b bridge run was refused: "rw source refused: .../h.sock is a fifo/device/socket"). *_GREEN.log: the
same test file passes with the fix (the single socket <dir>/h.sock is accepted for --bridge only; a second socket and non-bridge use stay refused).
quickstart_run.txt: transcript of docs/quickstart.md steps 1-2 executed for real on host loopback (a non-GUI process, NOT in the container) with the exact commands, the port and path deviations, and the status codes (201, 202, 200). Not a screenshot; supports the quickstart claim only.
