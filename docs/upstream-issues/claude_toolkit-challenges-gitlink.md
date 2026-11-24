# DRAFT (not filed): claude_toolkit carries gitlinks with no `.gitmodules` mapping, breaking recursive submodule init

Target: `vasic-digital/claude_toolkit` (pinned here at `718cd256ea191871971dd46c09eda852438094ae`, vendored via `HelixDevelopment/constitution` `submodules/claude_toolkit`).
Drafted 2026-11-23 from the helix_reader checkout. Nothing has been filed.

## Summary

The tree of `claude_toolkit@718cd25` contains five gitlinks, but `.gitmodules` maps only two (`submodules/go`, `submodules/ccusage`).
Three own-org gitlinks are stale leftovers of the flat-dependency policy declared in the file's own header. Any
`git submodule update --init --recursive` that descends into claude_toolkit aborts with `fatal`.

## Reproducer

```
git clone <helix_reader or any project vendoring the constitution> && cd helix_reader
git -C submodules/constitution submodule update --init --recursive
```
Captured output (2026-11-23):
```
fatal: No url found for submodule path 'submodules/claude_toolkit/submodules/challenges' in .gitmodules
fatal: Failed to recurse into submodule path 'submodules/claude_toolkit'
```
Gitlinks in the tree (`git -C submodules/constitution/submodules/claude_toolkit ls-tree HEAD submodules/`):
```
160000 commit 757b457f91028948f2cef22deeecf70f38fddfd8	submodules/ccusage      (mapped)
160000 commit 8a352b971d07ff6b3cda192f7ad6322eebef348a	submodules/challenges   (UNMAPPED)
160000 commit 4a8f04e05f3535d77c48f69b89687f9fcc896fbf	submodules/containers   (UNMAPPED)
160000 commit 35455b81d3dcd923beec25b13489abbc8414221f	submodules/go           (mapped)
160000 commit c20ac6c1ba069cc9a72dacb8044b7b193d3dde81	submodules/superspec    (UNMAPPED)
```
`.gitmodules` header (verbatim): "claude_toolkit tracks NO own-org submodule (vasic-digital/*, HelixDevelopment/*): constitution, containers, challenges, superspec, llms_verifier (LLMsVerifier) and claude_code_router (claude-code-router) live FLAT in the parent project and resolve via scripts/resolve_deps.sh (single source of truth). Gate: scripts/tests/test_gitmodules_no_ownorg.sh."

## Root cause (static analysis; the commit that introduced the gitlinks was not bisected, UNCONFIRMED)

The `.gitmodules` entries for challenges/containers/superspec were removed (policy: own-org deps live flat in the parent)
but the three gitlink entries were left in the tree. A gitlink without a `.gitmodules` URL cannot be initialised, and
`--recursive` treats that as fatal. The gate `test_gitmodules_no_ownorg.sh` checks `.gitmodules` text, not the tree's gitlinks,
so it passes while the tree still pins own-org commits (also a latent violation of the single-source-of-truth rule and, for
`containers@4a8f04e`, a second pin that differs from the parent's `containers@5ea870a`).

## Proposed fixes (either)

A. Preferred: `git rm --cached submodules/challenges submodules/containers submodules/superspec` in claude_toolkit, commit, and extend `test_gitmodules_no_ownorg.sh` to also fail on any gitlink (`git ls-tree -r HEAD | awk '$1==160000'`) whose path is absent from `.gitmodules`. Bump the pin in the constitution.
B. Map them in `.gitmodules` (restores recursion but contradicts the stated flat-dependency policy and keeps duplicate pins).

## Workaround used here

Never use `--recursive` across claude_toolkit. Initialise per path and non-recursively:
`git -C submodules/constitution submodule update --init -- submodules/claude_toolkit` (claude_toolkit's own nested `go`/`ccusage` are not needed by consumers). The constitution's own hub-structure gates (`CM-HUB-ENGINE-NO-OWNORG-NESTING`) pass on this layout because they read `.gitmodules`, which is the same blind spot as above.
