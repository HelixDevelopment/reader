(function () {
  if (window.__hv) return true;
  const norm = (s) => (s || "").replace(/\s+/g, " ").trim();
  const lc = (s) => norm(s).toLowerCase();
  const isVisible = (el) => {
    const r = el.getBoundingClientRect();
    const st = getComputedStyle(el);
    return r.width > 0 && r.height > 0 && st.visibility !== "hidden";
  };
  const match = (hay, needle, exact) => (exact ? norm(hay) === norm(needle) : lc(hay).includes(lc(needle)));
  const SKIP = new Set(["SCRIPT", "STYLE", "HEAD", "META", "LINK", "TITLE", "NOSCRIPT"]);
  const uniq = (arr) => Array.from(new Set(arr));

  function inScope(root) {
    return root === document ? Array.from(document.querySelectorAll("*")) : Array.from(root.querySelectorAll("*"));
  }
  function textMatches(root, needle, exact) {
    const hits = inScope(root).filter((el) => !SKIP.has(el.tagName) && match(el.textContent, needle, exact));
    // the smallest elements containing the text: drop an element when a descendant also matches
    const set = new Set(hits);
    return hits.filter((el) => !Array.from(el.children).some((c) => set.has(c)));
  }
  function labelMatches(root, needle, exact) {
    const out = [];
    for (const el of inScope(root)) {
      const al = el.getAttribute("aria-label");
      if (al !== null && match(al, needle, exact)) out.push(el);
    }
    const doc = root === document ? document : root.ownerDocument;
    for (const lab of (root === document ? document : root).querySelectorAll("label")) {
      if (!match(lab.textContent, needle, exact)) continue;
      if (lab.control) out.push(lab.control);
      else if (lab.htmlFor) { const c = doc.getElementById(lab.htmlFor); if (c) out.push(c); }
    }
    for (const el of inScope(root)) {
      const by = el.getAttribute("aria-labelledby");
      if (!by) continue;
      const txt = by.split(/\s+/).map((id) => { const t = doc.getElementById(id); return t ? t.textContent : ""; }).join(" ");
      if (match(txt, needle, exact)) out.push(el);
    }
    return out;
  }
  function roleMatches(root, role, name, exact) {
    if (role !== "button") return [];
    return inScope(root).filter((el) => {
      const isBtn = el.tagName === "BUTTON" || el.getAttribute("role") === "button" ||
        (el.tagName === "INPUT" && ["button", "submit", "reset"].includes(el.type));
      if (!isBtn || !isVisible(el) || el.closest('[aria-hidden="true"]')) return false;
      const accessible = el.getAttribute("aria-label") || (el.tagName === "INPUT" ? el.value : el.textContent);
      return match(accessible, name, exact);
    });
  }
  function resolve(spec) {
    const roots = spec.scope ? resolve(spec.scope) : [document];
    let out = [];
    for (const root of roots) {
      let found = [];
      if (spec.kind === "css") {
        found = Array.from(root.querySelectorAll(spec.value));
        if (spec.hasText) found = found.filter((el) => match(el.textContent, spec.hasText, false));
      } else if (spec.kind === "text") found = textMatches(root, spec.value, !!spec.exact);
      else if (spec.kind === "label") found = labelMatches(root, spec.value, !!spec.exact);
      else if (spec.kind === "role") found = roleMatches(root, spec.role, spec.name, !!spec.exact);
      else throw new Error("unknown locator kind " + spec.kind);
      out = out.concat(found);
    }
    out = uniq(out);
    if (spec.visible) out = out.filter(isVisible);
    if (spec.ancestor) out = uniq(out.map((el) => el.closest(spec.ancestor)).filter(Boolean));
    if (spec.first) out = out.slice(0, 1);
    return out;
  }
  const first = (spec) => resolve(spec)[0] || null;
  const rectKey = (el) => { const r = el.getBoundingClientRect(); return [r.x, r.y, r.width, r.height].join(","); };
  const setNative = (el, value) => {
    const proto = el.tagName === "SELECT" ? HTMLSelectElement.prototype : el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value").set.call(el, value);
    el.dispatchEvent(new Event("input", { bubbles: true }));
    el.dispatchEvent(new Event("change", { bubbles: true }));
  };
  window.__hv = {
    count: (spec) => resolve(spec).length,
    visible: (spec) => { const el = first(spec); return el ? isVisible(el) : false; },
    // actionability for click/fill: present, visible, enabled, and (click) receiving events at its centre
    prepare: (spec, forClick) => {
      const el = first(spec);
      if (!el) return { ok: false, reason: "no element matches " + JSON.stringify(spec) };
      if (!isVisible(el)) return { ok: false, reason: "element is not visible" };
      if (el.disabled) return { ok: false, reason: "element is disabled" };
      el.scrollIntoView({ block: "center", inline: "center" });
      const key = rectKey(el);
      if (forClick) {
        const r = el.getBoundingClientRect();
        const top = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
        if (!top || !(top === el || el.contains(top))) return { ok: false, reason: "another element would receive the click", key };
        document.querySelectorAll("[data-hv-target]").forEach((n) => n.removeAttribute("data-hv-target"));
        el.setAttribute("data-hv-target", "1");
      }
      return { ok: true, key };
    },
    clear: () => document.querySelectorAll("[data-hv-target]").forEach((n) => n.removeAttribute("data-hv-target")),
    fill: (spec, value) => { const el = first(spec); el.focus(); setNative(el, value); return true; },
    select: (spec, label) => {
      const el = first(spec);
      const opt = Array.from(el.options).find((o) => o.label === label || o.text === label || o.value === label);
      if (!opt) return false;
      el.focus(); setNative(el, opt.value); return true;
    },
    text: (spec) => { const el = first(spec); return el ? el.innerText : null; },
    style: (spec, prop) => { const el = first(spec); return el ? getComputedStyle(el)[prop] : null; },
    attr: (spec, name) => { const el = first(spec); return el ? { v: el.getAttribute(name) } : null; },
    dims: () => ({ scroll: document.documentElement.scrollWidth, client: document.documentElement.clientWidth }),
  };
  return true;
})()
