/* SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> */
/* SPDX-License-Identifier: GPL-3.0-only */
/* The page's one script. It loads in the head, inline or as dashboard.js,
   and sets the saved theme at once, before the first paint; the rest waits
   for the parsed page, where each feature is one small init function. It
   never writes a style attribute, which a Content-Security-Policy without
   'unsafe-inline' would block: values go through the CSSOM
   (element.style), which no policy blocks. Everything is inside one
   closure, so the page adds no globals. */
(function () {
  "use strict";
  var root = document.documentElement;
  var themeKey = "passmcp-graph-theme";
  var byId = function (id) { return document.getElementById(id); };
  var each = function (sel, fn) { document.querySelectorAll(sel).forEach(fn); };
  function plural(n, one, many) { return n + " " + (n === 1 ? one : many); }

  root.classList.remove("no-js");
  (function () {
    var t = "system";
    try { t = localStorage.getItem(themeKey) || "system"; } catch (e) {}
    if (t === "light" || t === "dark") { root.setAttribute("data-theme", t); }
  })();

  /* Stagger: an element's place in an animated sequence is its data-i.
     The stylesheet maps the first few; this covers every one, however
     many nodes and edges the topology draws. */
  function initStagger() {
    each("[data-i]", function (el) { el.style.setProperty("--i", el.getAttribute("data-i")); });
  }

  /* Theme: System, Light or Dark, remembered in the browser. */
  function initTheme() {
    var buttons = document.querySelectorAll("[data-set-theme]");
    function mark() {
      var current = root.getAttribute("data-theme") || "system";
      buttons.forEach(function (b) { b.setAttribute("aria-pressed", String(b.getAttribute("data-set-theme") === current)); });
    }
    buttons.forEach(function (b) {
      b.addEventListener("click", function () {
        var t = b.getAttribute("data-set-theme");
        if (t === "system") { root.removeAttribute("data-theme"); } else { root.setAttribute("data-theme", t); }
        try { localStorage.setItem(themeKey, t); } catch (e) {}
        mark();
      });
    });
    mark();
  }

  /* Filters: one state scopes the topology, lists and tables. It lives in
     the history entry, not the address, so reload and back restore it and
     nothing typed into the search reaches a URL. */
  var blank = function () { return { q: "", st: [], bh: [], cli: "", ctx: false, all: false }; };
  var state = blank();
  var items = [];
  function has(tags, t) { return (" " + tags + " ").indexOf(" " + t + " ") >= 0; }
  function active() { return !!(state.q || state.cli || state.st.length || state.bh.length); }
  function anyTag(tags, list) { return !list.length || list.some(function (t) { return has(tags, t); }); }
  function matches(el) {
    var tags = el.getAttribute("data-tags") || "";
    if (state.cli && !has(tags, state.cli)) { return false; }
    if (!anyTag(tags, state.st) || !anyTag(tags, state.bh)) { return false; }
    var text = el.getAttribute("data-text") || "";
    return state.q.toLowerCase().split(/\s+/).every(function (w) { return !w || text.indexOf(w) >= 0; });
  }
  function chipLabel(sel) { var b = document.querySelector(sel); return b ? b.textContent.trim() : ""; }
  function selection() {
    var parts = [];
    if (state.cli) { parts.push(chipLabel('.chips button[data-pick="' + state.cli + '"]')); }
    state.st.forEach(function (t) { parts.push(chipLabel('[data-st="' + t + '"]')); });
    state.bh.forEach(function (t) { parts.push(chipLabel('[data-bh="' + t + '"]')); });
    if (state.q) { parts.push("“" + state.q + "”"); }
    return parts.join(" · ");
  }

  /* count is how many items of each kind there are, and how many show. */
  function count() {
    var c = {};
    ["server", "client", "path", "finding", "gap", "identity"].forEach(function (k) { c[k] = { all: 0, shown: 0 }; });
    items.forEach(function (el) {
      var k = el.getAttribute("data-kind");
      if (!k) { return; }
      c[k] = c[k] || { all: 0, shown: 0 };
      c[k].all++;
      if (!el.classList.contains("out")) { c[k].shown++; }
    });
    return c;
  }
  function describe(c) {
    var att = { all: c.finding.all + c.gap.all, shown: c.finding.shown + c.gap.shown };
    return [
      c.server.shown + " of " + plural(c.server.all, "server", "servers"),
      c.client.shown + " of " + plural(c.client.all, "client", "clients"),
      c.path.shown + " of " + plural(c.path.all, "path", "paths"),
      att.shown + " of " + plural(att.all, "attention item", "attention items") + " (" + plural(c.finding.shown, "finding", "findings") + ", " + plural(c.gap.shown, "evidence gap", "evidence gaps") + ")",
      c.identity.shown + " of " + plural(c.identity.all, "identity", "identities")
    ].join(" · ");
  }

  /* reproduceGroups is the query language's where clause for the filters,
     and what it cannot express. */
  function reproduceGroups() {
    var groups = [], gaps = [];
    if (state.cli) {
      var pick = document.querySelector('.chips button[data-pick="' + state.cli + '"]');
      var cq = pick ? pick.getAttribute("data-q") : "";
      if (cq) { groups.push(cq); } else { gaps.push("this client's configuration path cannot be quoted in the query language"); }
    }
    [state.st, state.bh].forEach(function (list) {
      var alts = list.map(function (t) { var c = document.querySelector('[data-st="' + t + '"], [data-bh="' + t + '"]'); return c ? c.getAttribute("data-q") : ""; });
      if (alts.length === 1) { groups.push(alts[0]); }
      if (alts.length > 1) { groups.push("(" + alts.join(" or ") + ")"); }
    });
    if (state.q) { gaps.push("the search text has no query equivalent"); }
    return { groups: groups, gaps: gaps };
  }
  function reproLabel(exact) {
    if (!active()) { return "Query all configured paths:"; }
    return exact ? "Reproduce this view:" : "Closest command:";
  }
  function reproduce() {
    var box = document.querySelector(".repro");
    if (!box) { return; }
    var r = reproduceGroups();
    var query = box.getAttribute("data-base") + (r.groups.length ? " where " + r.groups.join(" and ") : "");
    byId("repro").textContent = 'passmcp-graph query "' + query + '"';
    byId("reprolabel").textContent = reproLabel(!r.gaps.length);
    byId("repronote").textContent = (r.gaps.length ? "Not exact: " + r.gaps.join("; ") + ". " : "") + "Re-reads the local graph; collects no new evidence.";
  }

  /* attentionText is the attention count's sentence, for the selection or
     the whole snapshot. */
  function attentionText(c, on, f, g) {
    if (!on) {
      return plural(f + g, "attention item", "attention items") + ": " + plural(f, "finding", "findings") + " · " + plural(g, "evidence gap", "evidence gaps") + ".";
    }
    return plural(c.finding.shown + c.gap.shown, "attention item", "attention items") + " in this selection: " + plural(c.finding.shown, "finding", "findings") + " · " +
      plural(c.gap.shown, "evidence gap", "evidence gaps") + ". " + plural(f + g, "attention item", "attention items") + " in the whole snapshot.";
  }
  function showScope(c, on) {
    byId("scope").textContent = on ? "Showing " + describe(c) + ". Totals and overview marked whole snapshot are unchanged." : "Showing the whole snapshot.";
    byId("selpill").hidden = !on;
    byId("selwhat").textContent = selection();
    each("[data-scope-tag]", function (t) { t.textContent = on ? "Current selection" : "Whole snapshot"; t.classList.toggle("sel", on); });
    var ac = byId("attcount");
    if (ac) { ac.textContent = attentionText(c, on, +ac.getAttribute("data-findings"), +ac.getAttribute("data-gaps")); }
    each(".idsel", function (el) {
      var cl = (el.getAttribute("data-clients") || "").trim().split(/\s+/).filter(Boolean);
      el.textContent = state.cli ? " · " + plural(cl.indexOf(state.cli) >= 0 ? 1 : 0, "client", "clients") + " in this selection, " + el.getAttribute("data-total") + " in the whole snapshot" : "";
    });
  }
  function showControls(on) {
    byId("topo").classList.toggle("ctx", state.ctx);
    byId("ctx").checked = state.ctx;
    byId("ctxnote").hidden = !(state.ctx && on);
    byId("evtable").classList.toggle("core", !state.all);
    byId("allcols").checked = state.all;
    each("[data-st]", function (b) { b.setAttribute("aria-pressed", String(state.st.indexOf(b.getAttribute("data-st")) >= 0)); });
    each("[data-bh]", function (b) { b.setAttribute("aria-pressed", String(state.bh.indexOf(b.getAttribute("data-bh")) >= 0)); });
    each("button[data-pick]", function (b) { b.setAttribute("aria-pressed", String(b.getAttribute("data-pick") === state.cli)); });
  }
  function apply() {
    items.forEach(function (el) { el.classList.toggle("out", !matches(el)); });
    var on = active();
    showScope(count(), on);
    showControls(on);
    reproduce();
    try { history.replaceState(Object.assign({}, history.state, { pg: state }), ""); } catch (e) {}
  }
  function toggle(list, v) { var i = list.indexOf(v); if (i >= 0) { list.splice(i, 1); } else { list.push(v); } }
  function clear() {
    var keep = { ctx: state.ctx, all: state.all };
    state = Object.assign(blank(), keep);
    byId("q").value = "";
    apply();
  }
  function initFilters() {
    try { if (history.state && history.state.pg) { state = Object.assign(blank(), history.state.pg); } } catch (e) {}
    items = document.querySelectorAll("[data-tags]");
    var q = byId("q");
    each("[data-st]", function (b) { b.addEventListener("click", function () { toggle(state.st, b.getAttribute("data-st")); apply(); }); });
    each("[data-bh]", function (b) { b.addEventListener("click", function () { toggle(state.bh, b.getAttribute("data-bh")); apply(); }); });
    each("[data-pick]", function (el) {
      el.addEventListener("click", function () {
        var v = el.getAttribute("data-pick");
        state.cli = (state.cli === v && el.tagName !== "BUTTON") ? "" : v;
        apply();
      });
    });
    q.value = state.q;
    q.addEventListener("input", function () { state.q = q.value; apply(); });
    byId("clear").addEventListener("click", clear);
    byId("selclear").addEventListener("click", function () { clear(); q.focus(); });
    byId("ctx").addEventListener("change", function (e) { state.ctx = e.target.checked; apply(); });
    byId("allcols").addEventListener("change", function (e) { state.all = e.target.checked; apply(); });
    apply();
  }

  /* Details: a fragment such as #srv-2 opens that panel. A link from
     another panel, or from the banner, records where it came from, and
     the opened panel offers a way back that keeps the filters. */
  var detailId = /^(srv|cli|idn|fnd|gap)-[0-9]+$/;
  var origin = null;
  function label(el) {
    var t = el.querySelector(".t, summary strong, summary .mono, h2");
    var text = (t ? t.textContent : el.id).replace(/\s+/g, " ").trim();
    return text.length > 70 ? text.slice(0, 69) + "…" : text;
  }
  function recordOrigin(e) {
    var a = e.target.closest && e.target.closest('a[href^="#"]');
    if (!a) { return; }
    var from = a.closest("details[id], #top");
    origin = from ? { id: from.id, label: label(from), y: window.scrollY } : null;
  }
  function goBack(o) {
    return function () {
      history.back();
      setTimeout(function () {
        var t = byId(o.id);
        if (!t) { return; }
        if (t.tagName === "DETAILS") { t.open = true; }
        window.scrollTo(0, o.y);
        var f = t.querySelector("summary, .button, a");
        if (f) { f.focus({ preventScroll: true }); }
      }, 60);
    };
  }
  function offerBack(el, id) {
    var back = el.querySelector("[data-back]");
    if (!back || !origin || origin.id === id) { return; }
    back.hidden = false;
    back.textContent = "← Back to " + origin.label;
    back.onclick = goBack(origin);
  }
  function openFromHash() {
    var id = location.hash.slice(1);
    each("[data-back]", function (b) { b.hidden = true; });
    if (!detailId.test(id)) { return; }
    var el = byId(id);
    if (!el || el.tagName !== "DETAILS") { return; }
    el.open = true;
    offerBack(el, id);
    var s = el.querySelector("summary");
    if (s) { s.focus({ preventScroll: true }); }
    el.scrollIntoView({ block: "start" });
    apply();
  }
  function initDetails() {
    document.addEventListener("click", recordOrigin);
    window.addEventListener("hashchange", openFromHash);
    each("details[id]", function (d) {
      d.addEventListener("toggle", function () {
        if (d.open && location.hash !== "#" + d.id) { try { history.replaceState(history.state, "", "#" + d.id); } catch (e) {} }
      });
    });
    openFromHash();
  }

  /* Banner actions and copying */
  function copier(b, text) {
    var was = b.textContent;
    var done = function () { b.textContent = "Copied"; setTimeout(function () { b.textContent = was; }, 1600); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text()).then(done, function () { b.textContent = "Select the command to copy it"; });
    } else {
      b.textContent = "Select the command to copy it";
    }
  }
  function initActions() {
    each("[data-show-path]", function (b) {
      b.addEventListener("click", function () {
        state.cli = b.getAttribute("data-show-path");
        apply();
        byId("topology").scrollIntoView({ block: "start" });
      });
    });
    each("[data-copy]", function (b) { b.addEventListener("click", function () { copier(b, function () { return b.getAttribute("data-copy"); }); }); });
    each("[data-copy-from]", function (b) { b.addEventListener("click", function () { copier(b, function () { return byId(b.getAttribute("data-copy-from")).textContent; }); }); });
  }

  /* Command centre: severity tabs, the search shortcut, print, the
     share-safe export and the mitigation guide's highlight. */
  function initSeverityTabs() {
    each(".tab[data-sev]", function (t) {
      t.addEventListener("click", function () {
        var sev = t.getAttribute("data-sev");
        each(".tab[data-sev]", function (o) { o.setAttribute("aria-pressed", String(o === t)); });
        each("[data-sevrow]", function (row) { row.classList.toggle("sevout", !!sev && row.getAttribute("data-sevrow") !== sev); });
      });
    });
  }
  function onShortcut(e) {
    var typing = /^(INPUT|TEXTAREA|SELECT)$/.test((e.target && e.target.tagName) || "");
    var cmdK = e.key === "k" && (e.metaKey || e.ctrlKey);
    if (cmdK || (e.key === "/" && !typing)) { e.preventDefault(); byId("q").focus(); byId("q").select(); }
  }
  function exportJSON() {
    var el = byId("export");
    var text = el ? JSON.parse(el.textContent) : "";
    if (!text) { return; }
    var a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([text], { type: "application/json" }));
    a.download = "passmcp-graph-share-safe.json";
    document.body.appendChild(a); a.click(); a.remove();
    setTimeout(function () { URL.revokeObjectURL(a.href); }, 1000);
  }
  function initCommandCentre() {
    initSeverityTabs();
    document.addEventListener("keydown", onShortcut);
    each("[data-print]", function (b) { b.addEventListener("click", function () { window.print(); }); });
    each("[data-export]", function (b) { b.addEventListener("click", exportJSON); });
    each("[data-guide]", function (a) {
      a.addEventListener("click", function () {
        var m = byId("mitigation");
        m.classList.add("lit");
        setTimeout(function () { m.classList.remove("lit"); }, 1600);
      });
    });
  }

  /* Counters count up to their figure unless the system asks for
     reduced motion. */
  function countUp(el) {
    var text = el.textContent.trim();
    var end = parseFloat(text);
    if (!isFinite(end) || String(end) !== text) { return; }
    var start = null;
    function step(ts) {
      if (start === null) { start = ts; }
      var k = Math.min(1, (ts - start) / 1300);
      el.textContent = String(Math.round(end * (1 - Math.pow(1 - k, 3))));
      if (k < 1) { requestAnimationFrame(step); }
    }
    el.textContent = "0";
    requestAnimationFrame(step);
  }
  function initCounters() {
    var still = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (!still && window.requestAnimationFrame) { each("[data-count]", countUp); }
  }

  /* Navigation highlight: the sidebar marks the section in view. */
  function initNavHighlight() {
    var links = document.querySelectorAll(".nav a");
    if (!("IntersectionObserver" in window)) { return; }
    var seen = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) { return; }
        links.forEach(function (a) { a.classList.toggle("on", a.getAttribute("href") === "#" + e.target.id); });
      });
    }, { rootMargin: "-40% 0px -55% 0px" });
    each("main > section[id], main .cols > section[id]", function (s) { seen.observe(s); });
  }

  /* The navigation drawer: below 64rem the sidebar is a disclosure under
     the top bar. Following a link or pressing Escape closes it. Without the
     script, the no-js class keeps the sidebar open as a list of links. */
  function initMenu() {
    var toggleBtn = byId("navToggle"), menu = byId("navMenu");
    if (!toggleBtn || !menu) { return; }
    function setMenu(open) {
      toggleBtn.setAttribute("aria-expanded", String(open));
      menu.setAttribute("data-open", String(open));
    }
    setMenu(false);
    toggleBtn.addEventListener("click", function () { setMenu(toggleBtn.getAttribute("aria-expanded") !== "true"); });
    menu.addEventListener("click", function (e) { if (e.target.closest("a")) { setMenu(false); } });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && toggleBtn.getAttribute("aria-expanded") === "true") { setMenu(false); toggleBtn.focus(); }
    });
  }

  /* Flows: one per client and server entry, from its dot to the ring, in
     the entry's status colour. Each is drawn in one of the two empty columns
     beside the ring, never over text, and measured from the laid-out page,
     so they meet the dots at every width; redrawn when the map resizes.
     Below 64rem the columns are not shown and no flow is drawn. */
  var svgNS = "http://www.w3.org/2000/svg";
  function curve(x1, y1, x2, y2) {
    var mid = (x1 + x2) / 2;
    return "M" + x1 + " " + y1 + " C" + mid + " " + y1 + " " + mid + " " + y2 + " " + x2 + " " + y2;
  }
  function drawGap(svg, nodes, ring, listOnLeft) {
    while (svg.firstChild) { svg.removeChild(svg.firstChild); }
    var box = svg.getBoundingClientRect();
    if (!box.width || !box.height) { return; }
    svg.setAttribute("viewBox", "0 0 " + box.width + " " + box.height);
    var cy = ring.top + ring.height / 2 - box.top;
    nodes.forEach(function (node, i) {
      var r = node.getBoundingClientRect();
      var ny = r.top + r.height / 2 - box.top;
      var ry = cy + (i - (nodes.length - 1) / 2) * ring.height * 0.12;
      var status = /\bst-([a-z]+)/.exec(node.getAttribute("class") || "");
      var path = document.createElementNS(svgNS, "path");
      path.setAttribute("d", listOnLeft ? curve(0, ny, box.width, ry) : curve(0, ry, box.width, ny));
      path.setAttribute("class", "cc-flow" + (status ? " st-" + status[1] : ""));
      svg.appendChild(path);
    });
  }
  function drawFlows(map) {
    var dial = map.querySelector(".cc-ring-dial");
    var gapIn = map.querySelector(".cc-flows-in"), gapOut = map.querySelector(".cc-flows-out");
    if (!dial || !gapIn || !gapOut) { return; }
    var ring = dial.getBoundingClientRect();
    drawGap(gapIn, [].slice.call(map.querySelectorAll(".cc-list-in .cc-node")), ring, true);
    drawGap(gapOut, [].slice.call(map.querySelectorAll(".cc-list-out .cc-node")), ring, false);
  }
  function initFlows() {
    var map = document.querySelector(".cc-map");
    if (!map) { return; }
    var draw = function () { drawFlows(map); };
    draw();
    if ("ResizeObserver" in window) { new ResizeObserver(draw).observe(map); } else { window.addEventListener("resize", draw); }
    window.addEventListener("load", draw);
  }

  document.addEventListener("DOMContentLoaded", function () {
    initStagger();
    initTheme();
    initFilters();
    initDetails();
    initActions();
    initCommandCentre();
    initCounters();
    initNavHighlight();
    initMenu();
    initFlows();
  });
})();
