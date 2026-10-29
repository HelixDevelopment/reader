#!/usr/bin/env python3
"""Deterministic Playwright checks for the static Helix product prototype."""
import json
import re
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:4173/"
OUT = Path(sys.argv[2]) if len(sys.argv) > 2 else Path("/mnt/agents/output/helix/evidence/ui")
OUT.mkdir(parents=True, exist_ok=True)
VIEWPORTS = [("phone", 375, 812), ("tablet", 768, 1024), ("desktop", 1024, 900), ("wide", 1440, 1000)]


def visible_button(page, text):
    return page.locator("button:visible", has_text=text).first


def nav_button(page, width, desktop_text, mobile_text=None):
    """Scope navigation to the layout that can actually receive input."""
    if width > 700:
        return page.locator(".sidebar .nav-item", has_text=desktop_text).first
    return page.locator(".mobile-nav button", has_text=mobile_text or desktop_text).first


def run_view(browser, name, width, height):
    context = browser.new_context(viewport={"width": width, "height": height}, reduced_motion="reduce")
    page = context.new_page()
    console_errors, missing, checks = [], [], []

    def check(check_id, passed, evidence):
        checks.append({"id": check_id, "status": "PASS" if passed else "FAIL", "evidence": evidence})

    page.on("console", lambda msg: console_errors.append({"text": msg.text, "location": msg.location}) if msg.type == "error" else None)
    page.on("response", lambda response: missing.append(f"{response.status} {response.url}") if response.status == 404 else None)
    page.goto(BASE, wait_until="networkidle")
    page.screenshot(path=str(OUT / f"{name}-library.png"), full_page=True)
    check("viewport-meta", page.locator('meta[name="viewport"]').count() == 1, "one viewport declaration")
    dimensions = page.evaluate("({scroll:document.documentElement.scrollWidth,client:document.documentElement.clientWidth})")
    check("no-horizontal-overflow", dimensions["scroll"] <= dimensions["client"], dimensions)
    disclosure_text = "No backend connected" if width > 700 else "Sample data only · no backend connected"
    disclosure = page.get_by_text(disclosure_text, exact=True).first
    check("sample-data-disclosure", disclosure.is_visible(), f"backend state disclosed and visible: {disclosure_text}")

    search = page.get_by_label("Search library")
    search.fill("Orbital")
    check("search-filter", page.locator(".book-card").count() == 1 and visible_button(page, "Orbital Gardens").count() >= 1, "six books reduced to one")
    search.fill("")
    visible_button(page, "Reset filters").count()  # no implicit action; preserve flow evidence only
    visible_button(page, "Resume chapter").click()
    before = page.locator(".reading-page h1").inner_text()
    visible_button(page, "Next").click()
    after = page.locator(".reading-page h1").inner_text()
    check("reader-next-delta", before != after and "Chapter 8" in after, {"before": before, "after": after})
    size_before = page.locator(".reading-page").evaluate("el=>getComputedStyle(el).fontSize")
    page.get_by_label("Appearance", exact=True).click()
    page.get_by_label("Decrease text size").click()
    size_after = page.locator(".reading-page").evaluate("el=>getComputedStyle(el).fontSize")
    check("reader-font-delta", size_before != size_after, {"before": size_before, "after": size_after})
    page.get_by_label("Close panel").click()
    visible_button(page, "Save a passage").click()
    check("save-passage", page.locator(".reader-panel .note").count() >= 2, "passage added to notes panel")
    page.get_by_label("Close panel").click()
    page.locator(".reader-topbar .back-button").click()
    check("book-detail", page.locator(".detail-copy h1").inner_text() == "The Glass Meridian", "reader returns to selected edition")
    page.locator(".detail-page > .back-button").click()

    nav_button(page, width, "Metadata review", "Review").click()
    count_before = page.locator(".review-item").count()
    page.locator(".review-item").first.get_by_role("button", name="Accept proposal").click()
    accepted = page.locator(".review-item.accepted").count()
    check("metadata-decision-delta", count_before == 3 and accepted == 1, {"items": count_before, "accepted": accepted})

    nav_button(page, width, "Sources").click()
    visible_button(page, "Add source").click()
    page.get_by_role("button", name="Add to sample workspace").click()
    check("source-validation", page.get_by_text("Enter a descriptive source name.").is_visible(), "empty source rejected")
    page.get_by_label("Source name").fill("Test OPDS")
    page.get_by_label("Connector").select_option("OPDS 1.2")
    page.get_by_role("button", name="Add to sample workspace").click()
    check("source-add-delta", page.get_by_text("Test OPDS", exact=True).is_visible(), "validated source appears")
    page.get_by_text("Test OPDS", exact=True).locator("xpath=ancestor::article").get_by_role("button", name="Scan").click()
    check("source-scan-state", page.get_by_text("Scan queued in prototype", exact=True).is_visible(), "scan action changes explicit prototype state")

    nav_button(page, width, "Synchronization").click()
    visible_button(page, "Keep furthest position").click()
    check("sync-conflict-delta", page.get_by_text("Conflict resolved", exact=True).is_visible(), "47% position selected")

    nav_button(page, width, "Reading settings", "Settings").click()
    page.locator(".theme-card.sepia").click()
    check("theme-delta", page.locator(".app").get_attribute("data-theme") == "sepia", "sepia theme applied")
    page.screenshot(path=str(OUT / f"{name}-settings.png"), full_page=True)
    check("console-errors", not console_errors, console_errors)
    check("missing-assets", not missing, missing)
    context.close()
    return {"viewport": {"name": name, "width": width, "height": height}, "checks": checks}


def main():
    results = []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True, executable_path="/usr/bin/chromium", args=["--no-sandbox"])
        try:
            for args in VIEWPORTS:
                try:
                    results.append(run_view(browser, *args))
                except Exception as exc:
                    name, width, height = args
                    results.append({
                        "viewport": {"name": name, "width": width, "height": height},
                        "checks": [{
                            "id": "viewport-run",
                            "status": "FAIL",
                            "evidence": {"exception": type(exc).__name__, "message": str(exc)},
                        }],
                    })
        finally:
            browser.close()
    failed = [c for r in results for c in r["checks"] if c["status"] == "FAIL"]
    report = {"schema_version": 1, "base_url": BASE, "viewports": results, "summary": {"pass": sum(c["status"] == "PASS" for r in results for c in r["checks"]), "fail": len(failed), "skip": 0}}
    (OUT / "ui-results.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report["summary"]))
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
