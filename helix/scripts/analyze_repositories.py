#!/usr/bin/env python3
"""Build deterministic, revision-pinned static evidence for research clones.

This is an evidence indexer, not a support detector. Keyword evidence is explicitly
classified by file role and never promoted to a feature/support claim without review.
No source lines or file contents are copied into evidence output.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import gzip
import hashlib
import json
import os
import re
import subprocess
from collections import Counter, defaultdict
from datetime import datetime, timezone
from pathlib import Path

MAX_TEXT_BYTES = 1_048_576
MAX_EVIDENCE_PER_FEATURE = 40

FEATURE_PATTERNS = {
    "opds": r"\bOPDS(?:[-_ ]?(?:1\.2|2\.0|PSE))?\b|application/atom\+xml|application/opds\+json",
    "opds_pse": r"OPDS[-_ ]?PSE|page[-_ ]?streaming|pageStreaming",
    "opensearch": r"OpenSearch|application/opensearchdescription\+xml",
    "readium": r"Readium|RWPM|webpub|publication\.manifest",
    "lcp_drm": r"\bLCP\b|LCPL|LCPA|LCPDF|License Document|DRM|rights\.xml",
    "daisy_accessibility": r"\bDAISY\b|ncc\.html|DTBook|Accessibility|screen reader|aria-",
    "koreader_sync": r"KOSync|KOReader sync|koreader-sync|/syncs/progress|quick sync",
    "kobo_sync": r"Kobo sync|kobo.*sync|/v1/library|/v1/books|KoboAffiliate",
    "annotations": r"annotation|highlight|bookmark|text clipping|CFI|epubcfi",
    "notes_export": r"export.*(?:notes|highlights|annotations)|(?:notes|highlights|annotations).*export|markdown export",
    "tts_audio": r"text.to.speech|\bTTS\b|audiobook|podcast|chapter mark|audio chapter",
    "ai_llm": r"\bLLM\b|OpenAI|Anthropic|Ollama|language model|RAG|embedding|pgvector|vector search|semantic search",
    "metadata_enrichment": r"metadata|ISBN|Goodreads|Google Books|OpenLibrary|Douban|metadata provider|fetch.*metadata",
    "metadata_review": r"metadata.*(?:review|approve|lock|merge)|(?:review|approve|lock|merge).*metadata|dry.run",
    "deduplication": r"duplicate|dedup|SHA-?256|content hash|checksum|fingerprint|perceptual hash",
    "watch_import": r"watch(?:ed|ing)? (?:folder|director)|file watcher|fsnotify|BookDrop|Book Dock|auto.?import|inotify",
    "collections_smart": r"smart shelf|smart collection|reading list|collection|bookshelf|shelves|series",
    "progress_sync": r"reading progress|progress sync|sync.*progress|position sync|vector clock|furthest",
    "multiuser_auth": r"multi.user|OIDC|OpenID Connect|OAuth|JWT|authentication|authorization|per.user|single sign.on|\bSSO\b",
    "permissions": r"role.based|\bRBAC\b|permission|access control|per.library|library.*share|user.*role",
    "local_library": r"local (?:library|files|storage)|filesystem|in.place|directory scan|folder scan|file browser",
    "webdav": r"WebDAV|webdav",
    "network_sources": r"\bSFTP\b|\bFTPS?\b|\bSMB\b|\bNFS\b|\bS3\b|rclone|cloud storage|network share",
    "http_protocols": r"HTTP/3|HTTP/2|QUIC|Brotli|Content-Range|Accept-Ranges|Range request|Alt-Svc",
    "compression_cache": r"Brotli|gzip|cache|CDN|thumbnail|cover cache|ETag|Last-Modified",
    "database": r"PostgreSQL|SQLite|MySQL|MariaDB|Redis|MongoDB|DuckDB|database migration|sqlx|gorm|drizzle",
    "search": r"full.text search|FTS5|Elasticsearch|Meilisearch|Typesense|search index|fuzzy search",
    "backup_restore": r"backup|restore|snapshot|disaster recovery|export library",
    "reader_typography": r"hyphenation|font family|custom font|line.height|line spacing|font.size|typography|sepia|night mode|dark mode|theme",
    "dictionary_language": r"dictionary|StarDict|Yomitan|translation|Anki|word lookup|language learning|furigana",
    "comics_manga": r"comic|manga|webtoon|CBZ|CBR|CB7|CBT|double page|reading direction|right.to.left",
    "pdf_fixed_layout": r"MuPDF|PDF\.js|pdfjs|DjVu|fixed.layout|reflow|crop margin|column detection",
    "conversion": r"ebook-convert|conversion|converter|transcode|EPUB.*(?:FB2|TXT|PDF)|FB2.*EPUB|KEPUB",
    "security": r"TLS|HTTPS|certificate|constant.time|CSRF|CORS|CSP|Content-Security-Policy|SSRF|symlink|path traversal|secret|credential",
    "deployment": r"Docker|docker-compose|Kubernetes|Helm chart|systemd|NixOS|github actions|CI/CD|healthcheck|readiness",
    "testing": r"unit test|integration test|fuzz test|benchmark|Playwright|Vitest|pytest|go test|cargo test|mutation test",
}

FORMAT_PATTERNS = {
    "epub": r"\.epub3?\b|\bEPUB\s*[23]?\b",
    "fb2": r"\.fb2(?:\.zip)?\b|\bFB2(?:\.ZIP)?\b|FictionBook",
    "fb3": r"\.fb3\b|\bFB3\b",
    "mobi_azw": r"\.mobi\b|\.azw3?\b|\bMOBI\b|\bAZW3?\b|PalmDOC",
    "pdf": r"\.pdf\b|\bPDF\b|application/pdf",
    "djvu": r"\.djvu?\b|\bDjVu\b",
    "cbz": r"\.cbz\b|\bCBZ\b",
    "cbr": r"\.cbr\b|\bCBR\b",
    "cb7_cbt_cba": r"\.cb7\b|\.cbt\b|\.cba\b|\bCB7\b|\bCBT\b|\bCBA\b",
    "txt": r"\.txt\b|plain text",
    "html_xhtml": r"\.x?html\b|XHTML|HTML5",
    "doc_docx": r"\.docx?\b|\bDOCX\b|wordprocessingml",
    "odt_rtf": r"\.odt\b|\.rtf\b|\bODT\b|\bRTF\b",
    "markdown": r"\.md\b|Markdown",
    "webpub_lpf": r"\.webpub\b|\.lpf\b|WebPub|LPF|Readium Web Publication",
    "audiobook": r"\.audiobook\b|\.lcpa\b|\.lcpaudiobook\b|Audiobook Manifest",
    "divina": r"\.divina\b|DiViNa",
    "daisy": r"\.daisy\b|\bDAISY\b|ncc\.html|DTBook",
    "lcp_protected": r"\.lcpl\b|\.lcpdf\b|\.lcpa\b|\.lcpaudiobook\b|\bLCP\b",
    "opf": r"\.opf\b|package document|Open Packaging Format",
    "xtc_xtch": r"\.xtch?\b|\bXTCH?\b",
}

LANG_BY_EXT = {
    ".go": "Go", ".rs": "Rust", ".py": "Python", ".ts": "TypeScript", ".tsx": "TypeScript React",
    ".js": "JavaScript", ".jsx": "JavaScript React", ".kt": "Kotlin", ".kts": "Kotlin",
    ".java": "Java", ".swift": "Swift", ".cs": "C#", ".c": "C", ".h": "C/C++ header",
    ".cc": "C++", ".cpp": "C++", ".cxx": "C++", ".hpp": "C++ header", ".lua": "Lua",
    ".dart": "Dart", ".php": "PHP", ".rb": "Ruby", ".scala": "Scala", ".vue": "Vue",
    ".svelte": "Svelte", ".css": "CSS", ".scss": "SCSS", ".html": "HTML", ".xml": "XML",
    ".sql": "SQL", ".sh": "Shell", ".ps1": "PowerShell", ".m": "Objective-C", ".mm": "Objective-C++",
}

BINARY_EXT = set(".7z aac avi avif bin bmp cb7 cba cbr cbt cbz class db dll dmg doc docx epub exe flac flv gif gz ico iso jar jpeg jpg lcpl lcpdf m4a m4b mobi mov mp3 mp4 odp ods odt ogg otf pdf png ppm rar rpm so sqlite tar ttf wav webm webp woff woff2 xls xlsx xz zip".split())
SECRET_NAME = re.compile(r"(?:^|/)(?:\.env(?:\.|$)|.*(?:credential|secret|private[-_]?key).*)|\.(?:pem|p12|pfx|key)$", re.I)
DOC_PATH = re.compile(r"(?:^|/)(?:docs?|documentation|wiki|examples?)(?:/|$)|(?:^|/)(?:README|CHANGELOG|CONTRIBUTING|SECURITY|FAQ|LICENSE|COPYING|NOTICE|AUTHORS)(?:\.|$)", re.I)
TEST_PATH = re.compile(r"(?:^|/)(?:tests?|__tests__|specs?|testdata|fixtures?)(?:/|$)|(?:^|/).*(?:_test|\.test|\.spec|Test|Tests)\.[A-Za-z0-9]+$", re.I)
CONFIG_PATH = re.compile(r"(?:^|/)(?:\.github|\.gitlab|deploy|deployment|docker|k8s|helm|nix)(?:/|$)|(?:Dockerfile|docker-compose|package\.json|go\.mod|Cargo\.toml|pyproject\.toml|requirements.*\.txt|pubspec\.yaml|.*\.csproj|.*\.sln)$", re.I)


def run_git(repo: Path, *args: str) -> str:
    return subprocess.check_output(["git", "-C", str(repo), *args], text=True, errors="replace").strip()


def role_for(path: str) -> str:
    if TEST_PATH.search(path): return "test"
    if DOC_PATH.search(path): return "documentation"
    if CONFIG_PATH.search(path): return "configuration"
    return "source"


def classify_license(text: str) -> list[str]:
    found = []
    tests = [
        ("AGPL-3.0", r"GNU AFFERO GENERAL PUBLIC LICENSE|AGPL-3\.0"),
        ("GPL-3.0", r"GNU GENERAL PUBLIC LICENSE[\s\S]{0,100}Version 3|GPL-3\.0"),
        ("GPL-2.0", r"GNU GENERAL PUBLIC LICENSE[\s\S]{0,100}Version 2|GPL-2\.0"),
        ("LGPL", r"GNU LESSER GENERAL PUBLIC LICENSE|LGPL"),
        ("Apache-2.0", r"Apache License[\s\S]{0,80}Version 2\.0|Apache-2\.0"),
        ("MIT", r"MIT License|Permission is hereby granted, free of charge"),
        ("BSD-3-Clause", r"Redistribution and use in source and binary forms[\s\S]{0,1200}Neither the name"),
        ("BSD-2-Clause", r"Redistribution and use in source and binary forms"),
        ("MPL-2.0", r"Mozilla Public License[\s\S]{0,80}2\.0|MPL-2\.0"),
        ("OFL-1.1", r"SIL OPEN FONT LICENSE|OFL-1\.1"),
        ("CC0", r"CC0|Public Domain Dedication"),
        ("Unlicense", r"This is free and unencumbered software released into the public domain"),
    ]
    for name, pattern in tests:
        if re.search(pattern, text, re.I): found.append(name)
    return found


def text_candidate(path: str, size: int) -> bool:
    ext = Path(path).suffix.lower()
    return ext not in BINARY_EXT and not SECRET_NAME.search(path) and 0 < size <= MAX_TEXT_BYTES


def scan_repo(kind: str, slug: str, url: str, repo: Path, out_root: Path) -> dict:
    revision = run_git(repo, "rev-parse", "HEAD")
    tree = run_git(repo, "rev-parse", "HEAD^{tree}")
    origin = run_git(repo, "remote", "get-url", "origin")
    raw = subprocess.check_output(["git", "-C", str(repo), "ls-tree", "-rz", "-l", "HEAD"])
    records = []
    for item in raw.split(b"\0"):
        if not item: continue
        meta, path_b = item.split(b"\t", 1)
        mode, obj_type, blob, size = meta.decode().split()
        records.append({"path": path_b.decode("utf-8", "surrogateescape"), "mode": mode, "type": obj_type, "blob": blob, "size": -1 if size == "-" else int(size)})
    records.sort(key=lambda r: r["path"])
    languages = Counter()
    feature_hits = defaultdict(list)
    feature_roles = defaultdict(Counter)
    format_hits = defaultdict(list)
    format_roles = defaultdict(Counter)
    license_files, licenses = [], set()
    readmes = []
    scanned_files = scanned_bytes = skipped_large = skipped_binary = skipped_secret = 0
    evidence_path = out_root / "file_evidence" / f"{kind}-{slug}.jsonl.gz"
    evidence_path.parent.mkdir(parents=True, exist_ok=True)
    with gzip.open(evidence_path, "wt", encoding="utf-8", newline="\n") as evidence:
        for rec in records:
            path, size = rec["path"], rec["size"]
            ext = Path(path).suffix.lower()
            lang = LANG_BY_EXT.get(ext)
            if lang: languages[lang] += 1
            base = Path(path).name.lower()
            rec_out = dict(rec)
            rec_out["role"] = role_for(path)
            rec_out["language"] = lang
            if base.startswith(("readme", "read_me")): readmes.append(path)
            if base.startswith(("license", "licence", "copying", "notice")):
                license_files.append(path)
                try:
                    text = (repo / path).read_bytes()[:131072].decode("utf-8", "replace")
                    licenses.update(classify_license(text))
                except OSError: pass
            if rec["type"] != "blob" or rec["mode"] == "120000":
                rec_out["content_scan"] = "not_regular_blob"
            elif SECRET_NAME.search(path):
                skipped_secret += 1; rec_out["content_scan"] = "skipped_sensitive_name"
            elif size > MAX_TEXT_BYTES:
                skipped_large += 1; rec_out["content_scan"] = "skipped_size_limit"
            elif ext in BINARY_EXT or size == 0:
                skipped_binary += 1; rec_out["content_scan"] = "skipped_binary_or_empty"
            else:
                try:
                    data = (repo / path).read_bytes()
                    if b"\0" in data[:8192]:
                        skipped_binary += 1; rec_out["content_scan"] = "skipped_nul_byte"
                    else:
                        text = data.decode("utf-8", "replace")
                        scanned_files += 1; scanned_bytes += len(data)
                        rec_out["content_scan"] = "scanned"
                        rec_out["sha256"] = hashlib.sha256(data).hexdigest()
                        file_features = []
                        for name, pattern in FEATURE_PATTERNS.items():
                            matches = list(re.finditer(pattern, text, re.I))
                            if not matches: continue
                            file_features.append(name)
                            role = role_for(path)
                            feature_roles[name][role] += len(matches)
                            for match in matches[:3]:
                                if len(feature_hits[name]) < MAX_EVIDENCE_PER_FEATURE:
                                    feature_hits[name].append({"path": path, "line": text.count("\n", 0, match.start()) + 1, "role": role, "matched_term": match.group(0)[:100]})
                        file_formats = []
                        for name, pattern in FORMAT_PATTERNS.items():
                            matches = list(re.finditer(pattern, text, re.I))
                            if not matches: continue
                            file_formats.append(name)
                            role = role_for(path)
                            format_roles[name][role] += len(matches)
                            for match in matches[:2]:
                                if len(format_hits[name]) < MAX_EVIDENCE_PER_FEATURE:
                                    format_hits[name].append({"path": path, "line": text.count("\n", 0, match.start()) + 1, "role": role, "matched_term": match.group(0)[:100]})
                        rec_out["feature_terms"] = sorted(file_features)
                        rec_out["format_terms"] = sorted(file_formats)
                except OSError as exc:
                    rec_out["content_scan"] = f"read_error_{type(exc).__name__}"
            evidence.write(json.dumps(rec_out, ensure_ascii=False, sort_keys=True) + "\n")
    features = {name: {"roles": dict(sorted(feature_roles[name].items())), "evidence": feature_hits[name]} for name in sorted(feature_hits)}
    formats = {name: {"roles": dict(sorted(format_roles[name].items())), "evidence": format_hits[name]} for name in sorted(format_hits)}
    result = {
        "schema_version": 1, "kind": kind, "slug": slug, "inventory_url": url, "origin_url": origin,
        "revision": revision, "tree": tree, "analysis_status": "automated_static_index_complete",
        "manual_review_status": "pending", "support_claims_allowed": False,
        "method": {"tracked_tree": "git ls-tree -rz -l HEAD", "max_text_bytes_per_file": MAX_TEXT_BYTES, "source_lines_retained": False, "keyword_evidence_is_not_support_proof": True},
        "files": {"tracked": len(records), "scanned_text": scanned_files, "scanned_bytes": scanned_bytes, "skipped_large": skipped_large, "skipped_binary_or_empty": skipped_binary, "skipped_sensitive_name": skipped_secret},
        "languages_by_file_count": dict(languages.most_common()), "readme_files": sorted(readmes),
        "license_review": {"status": "heuristic_requires_legal_review", "candidate_notices": sorted(licenses), "notice_files": sorted(license_files)},
        "feature_keyword_evidence": features, "format_keyword_evidence": formats,
        "file_evidence_gzip": str(evidence_path.relative_to(out_root)),
    }
    analysis_dir = out_root / f"{kind}_analysis"
    analysis_dir.mkdir(parents=True, exist_ok=True)
    (analysis_dir / f"{slug}.json").write_text(json.dumps(result, indent=2, ensure_ascii=False, sort_keys=True) + "\n")
    return result


def render_markdown(r: dict) -> str:
    lines = [f"# {r['slug']} — automated static evidence", "", f"- Inventory URL: {r['inventory_url']}", f"- Origin: {r['origin_url']}", f"- Revision: `{r['revision']}`", f"- Tree: `{r['tree']}`", f"- Analysis status: **{r['analysis_status']}**; manual review: **{r['manual_review_status']}**.", "- Keyword hits are leads for code review, not proof of working support.", "", "## Tree and languages", "", f"Tracked files: {r['files']['tracked']}. Text files scanned: {r['files']['scanned_text']} ({r['files']['scanned_bytes']} bytes). Files skipped by size, binary classification, or sensitive-name policy remain indexed in the compressed file ledger.", ""]
    lines += [f"- {lang}: {count} files" for lang, count in r["languages_by_file_count"].items()]
    lines += ["", "## License review", "", f"Status: {r['license_review']['status']}. Candidate notices: {', '.join(r['license_review']['candidate_notices']) or 'none identified'}."]
    lines += [f"- `{p}`" for p in r["license_review"]["notice_files"]]
    for section, key in [("Feature evidence", "feature_keyword_evidence"), ("Format evidence", "format_keyword_evidence")]:
        lines += ["", f"## {section}", ""]
        if not r[key]: lines.append("No indexed keyword evidence.")
        for name, data in r[key].items():
            lines += [f"### {name}", "", f"Role-weighted hits: `{json.dumps(data['roles'], sort_keys=True)}`.", ""]
            for e in data["evidence"][:5]: lines.append(f"- `{e['path']}:{e['line']}` ({e['role']}; matched `{e['matched_term']}`)")
            if len(data["evidence"]) > 5: lines.append(f"- Additional evidence retained in JSON: {len(data['evidence']) - 5} records.")
            lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--output", default="/mnt/agents/output/helix/research")
    ap.add_argument("--workers", type=int, default=3)
    ap.add_argument("--only", default="")
    args = ap.parse_args()
    out = Path(args.output); out.mkdir(parents=True, exist_ok=True)
    jobs = []
    base = Path("/mnt/agents/research/git-repos")
    backend_aliases = {"stumpapp-stump": "stump", "advplyr-audiobookshelf": "audiobookshelf", "bookorbit-bookorbit": "bookorbit", "grimmory-tools-grimmory": "grimmory", "kareadita-kavita": "kavita", "seblucas-cops": "cops"}
    for kind, inv in [("client", out / "repository_inventory.json"), ("backend", out / "backend_inventory.json")]:
        for r in json.loads(inv.read_text())["repositories"]:
            slug = r.get("slug") or re.sub(r"[^a-z0-9]+", "-", r["owner_repo"].lower()).strip("-")
            if args.only and slug not in args.only.split(","): continue
            repo = base / kind / slug
            if kind == "backend" and slug in backend_aliases:
                repo = base / "client" / backend_aliases[slug]
            jobs.append((kind, slug, r.get("canonical") or r["url"], repo))
    results, unavailable = [], []
    with concurrent.futures.ProcessPoolExecutor(max_workers=args.workers) as pool:
        futures = {}
        for kind, slug, url, repo in jobs:
            if subprocess.run(["git", "-C", str(repo), "rev-parse", "HEAD"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode != 0:
                unavailable.append({"kind": kind, "slug": slug, "url": url, "status": "valid_git_clone_unavailable"}); continue
            futures[pool.submit(scan_repo, kind, slug, url, repo, out)] = slug
        for future in concurrent.futures.as_completed(futures):
            slug = futures[future]
            try:
                r = future.result(); results.append(r); print(f"INDEXED {r['kind']} {slug} {r['files']['tracked']} files", flush=True)
            except Exception as exc:
                unavailable.append({"kind": kind, "slug": slug, "status": "analysis_error", "error": type(exc).__name__, "message": str(exc)}); print(f"ERROR {slug} {exc}", flush=True)
    results.sort(key=lambda r: (r["kind"], r["slug"]))
    for r in results:
        (out / f"{r['kind']}_analysis" / f"{r['slug']}.md").write_text(render_markdown(r))
    summary = {"schema_version": 1, "generated_at": datetime.now(timezone.utc).isoformat(), "status": "partial" if unavailable else "automated_index_complete", "manual_review_status": "pending", "repositories": [{"kind": r["kind"], "slug": r["slug"], "revision": r["revision"], "tree": r["tree"], "tracked_files": r["files"]["tracked"], "analysis_status": r["analysis_status"]} for r in results], "unavailable": sorted(unavailable, key=lambda r: (r["kind"], r["slug"]))}
    (out / "analysis_summary.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"indexed": len(results), "unavailable": len(unavailable), "status": summary["status"]}))
    return 0 if results else 1

if __name__ == "__main__": raise SystemExit(main())
