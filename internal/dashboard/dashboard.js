/* SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> */
/* SPDX-License-Identifier: GPL-3.0-only */
/* The page's one script. It loads in the head, inline or as dashboard.js,
   and sets the saved theme at once, before the first paint; the rest waits
   for the parsed page. It never writes a style attribute, which a
   Content-Security-Policy without 'unsafe-inline' would block: values go
   through the CSSOM (element.style), which no policy blocks. */
(function () {
  var t = "system";
  try { t = localStorage.getItem("passmcp-graph-theme") || "system"; } catch (e) {}
  if (t === "light" || t === "dark") { document.documentElement.setAttribute("data-theme", t); }
})();

document.addEventListener("DOMContentLoaded", function () {
  var root = document.documentElement;

  /* Stagger: an element's place in an animated sequence is its data-i.
     The stylesheet maps the first few; this covers every one, however
     many nodes and edges the topology draws. */
  document.querySelectorAll("[data-i]").forEach(function (el) { el.style.setProperty("--i", el.getAttribute("data-i")); });

  /* Theme */
  var themeKey = "passmcp-graph-theme";
  var themeButtons = document.querySelectorAll("[data-set-theme]");
  function markTheme() {
    var current = root.getAttribute("data-theme") || "system";
    themeButtons.forEach(function (b) { b.setAttribute("aria-pressed", String(b.getAttribute("data-set-theme") === current)); });
  }
  themeButtons.forEach(function (b) {
    b.addEventListener("click", function () {
      var t = b.getAttribute("data-set-theme");
      if (t === "system") { root.removeAttribute("data-theme"); } else { root.setAttribute("data-theme", t); }
      try { localStorage.setItem(themeKey, t); } catch (e) {}
      markTheme();
    });
  });
  markTheme();

  /* Filters: one state scopes the topology, lists and tables. It lives in
     the history entry, not the address, so reload and back restore it and
     nothing typed into the search reaches a URL. */
  var blank = function () { return { q: "", st: [], bh: [], cli: "", ctx: false, all: false }; };
  var state = blank();
  try { if (history.state && history.state.pg) { state = Object.assign(blank(), history.state.pg); } } catch (e) {}
  var q = document.getElementById("q");
  var items = document.querySelectorAll("[data-tags]");
  var byId = function (id) { return document.getElementById(id); };
  function has(tags, t) { return (" " + tags + " ").indexOf(" " + t + " ") >= 0; }
  function active() { return !!(state.q || state.cli || state.st.length || state.bh.length); }
  function matches(el) {
    var tags = el.getAttribute("data-tags") || "";
    if (state.cli && !has(tags, state.cli)) { return false; }
    if (state.st.length && !state.st.some(function (t) { return has(tags, t); })) { return false; }
    if (state.bh.length && !state.bh.some(function (t) { return has(tags, t); })) { return false; }
    var text = el.getAttribute("data-text") || "";
    return state.q.toLowerCase().split(/\s+/).every(function (w) { return !w || text.indexOf(w) >= 0; });
  }
  function plural(n, one, many) { return n + " " + (n === 1 ? one : many); }
  function chipLabel(sel) { var b = document.querySelector(sel); return b ? b.textContent.trim() : ""; }
  function selection() {
    var parts = [];
    if (state.cli) { parts.push(chipLabel('.chips button[data-pick="' + state.cli + '"]')); }
    state.st.forEach(function (t) { parts.push(chipLabel('[data-st="' + t + '"]')); });
    state.bh.forEach(function (t) { parts.push(chipLabel('[data-bh="' + t + '"]')); });
    if (state.q) { parts.push("“" + state.q + "”"); }
    return parts.join(" · ");
  }
  function count() {
    var c = {};
    items.forEach(function (el) {
      var k = el.getAttribute("data-kind");
      if (!k) { return; }
      c[k] = c[k] || { all: 0, shown: 0 };
      c[k].all++;
      if (!el.classList.contains("out")) { c[k].shown++; }
    });
    ["server", "client", "path", "finding", "gap", "identity"].forEach(function (k) { c[k] = c[k] || { all: 0, shown: 0 }; });
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
  function reproduce() {
    var box = document.querySelector(".repro");
    if (!box) { return; }
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
    var query = box.getAttribute("data-base") + (groups.length ? " where " + groups.join(" and ") : "");
    byId("repro").textContent = 'passmcp-graph query "' + query + '"';
    byId("reprolabel").textContent = active() && !gaps.length ? "Reproduce this view:" : (active() ? "Closest command:" : "Query all configured paths:");
    byId("repronote").textContent = (gaps.length ? "Not exact: " + gaps.join("; ") + ". " : "") + "Re-reads the local graph; collects no new evidence.";
  }
  function apply() {
    items.forEach(function (el) { el.classList.toggle("out", !matches(el)); });
    var c = count(), on = active();
    byId("scope").textContent = on ? "Showing " + describe(c) + ". Totals and overview marked whole snapshot are unchanged." : "Showing the whole snapshot.";
    byId("selpill").hidden = !on;
    byId("selwhat").textContent = selection();
    document.querySelectorAll("[data-scope-tag]").forEach(function (t) { t.textContent = on ? "Current selection" : "Whole snapshot"; t.classList.toggle("sel", on); });
    var ac = byId("attcount");
    if (ac) {
      var f = +ac.getAttribute("data-findings"), g = +ac.getAttribute("data-gaps");
      ac.textContent = on
        ? plural(c.finding.shown + c.gap.shown, "attention item", "attention items") + " in this selection: " + plural(c.finding.shown, "finding", "findings") + " · " + plural(c.gap.shown, "evidence gap", "evidence gaps") + ". " + plural(f + g, "attention item", "attention items") + " in the whole snapshot."
        : plural(f + g, "attention item", "attention items") + ": " + plural(f, "finding", "findings") + " · " + plural(g, "evidence gap", "evidence gaps") + ".";
    }
    document.querySelectorAll(".idsel").forEach(function (el) {
      var cl = (el.getAttribute("data-clients") || "").trim().split(/\s+/).filter(Boolean);
      el.textContent = state.cli ? " · " + plural(cl.indexOf(state.cli) >= 0 ? 1 : 0, "client", "clients") + " in this selection, " + el.getAttribute("data-total") + " in the whole snapshot" : "";
    });
    byId("topo").classList.toggle("ctx", state.ctx);
    byId("ctx").checked = state.ctx;
    byId("ctxnote").hidden = !(state.ctx && on);
    byId("evtable").classList.toggle("core", !state.all);
    byId("allcols").checked = state.all;
    document.querySelectorAll("[data-st]").forEach(function (b) { b.setAttribute("aria-pressed", String(state.st.indexOf(b.getAttribute("data-st")) >= 0)); });
    document.querySelectorAll("[data-bh]").forEach(function (b) { b.setAttribute("aria-pressed", String(state.bh.indexOf(b.getAttribute("data-bh")) >= 0)); });
    document.querySelectorAll("button[data-pick]").forEach(function (b) { b.setAttribute("aria-pressed", String(b.getAttribute("data-pick") === state.cli)); });
    reproduce();
    try { history.replaceState(Object.assign({}, history.state, { pg: state }), ""); } catch (e) {}
  }
  function toggle(list, v) { var i = list.indexOf(v); if (i >= 0) { list.splice(i, 1); } else { list.push(v); } }
  document.querySelectorAll("[data-st]").forEach(function (b) { b.addEventListener("click", function () { toggle(state.st, b.getAttribute("data-st")); apply(); }); });
  document.querySelectorAll("[data-bh]").forEach(function (b) { b.addEventListener("click", function () { toggle(state.bh, b.getAttribute("data-bh")); apply(); }); });
  document.querySelectorAll("[data-pick]").forEach(function (el) {
    el.addEventListener("click", function () {
      var v = el.getAttribute("data-pick");
      state.cli = (state.cli === v && el.tagName !== "BUTTON") ? "" : v;
      apply();
    });
  });
  q.value = state.q;
  q.addEventListener("input", function () { state.q = q.value; apply(); });
  function clear() { var keep = { ctx: state.ctx, all: state.all }; state = Object.assign(blank(), keep); q.value = ""; apply(); }
  byId("clear").addEventListener("click", clear);
  byId("selclear").addEventListener("click", function () { clear(); q.focus(); });
  byId("ctx").addEventListener("change", function (e) { state.ctx = e.target.checked; apply(); });
  byId("allcols").addEventListener("change", function (e) { state.all = e.target.checked; apply(); });
  apply();

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
  document.addEventListener("click", function (e) {
    var a = e.target.closest && e.target.closest('a[href^="#"]');
    if (!a) { return; }
    var from = a.closest("details[id], #top");
    origin = from ? { id: from.id, label: label(from), y: window.scrollY } : null;
  });
  function openFromHash() {
    var id = location.hash.slice(1);
    document.querySelectorAll("[data-back]").forEach(function (b) { b.hidden = true; });
    if (!detailId.test(id)) { return; }
    var el = byId(id);
    if (!el || el.tagName !== "DETAILS") { return; }
    el.open = true;
    var back = el.querySelector("[data-back]");
    if (back && origin && origin.id !== id) {
      back.hidden = false;
      back.textContent = "← Back to " + origin.label;
      back.onclick = (function (o) {
        return function () {
          history.back();
          setTimeout(function () {
            var t = byId(o.id);
            if (!t) { return; }
            if (t.tagName === "DETAILS") { t.open = true; }
            window.scrollTo(0, o.y);
            var f = t.querySelector("summary, .btn, a");
            if (f) { f.focus({ preventScroll: true }); }
          }, 60);
        };
      })(origin);
    }
    var s = el.querySelector("summary");
    if (s) { s.focus({ preventScroll: true }); }
    el.scrollIntoView({ block: "start" });
    apply();
  }
  window.addEventListener("hashchange", openFromHash);
  document.querySelectorAll("details[id]").forEach(function (d) {
    d.addEventListener("toggle", function () {
      if (d.open && location.hash !== "#" + d.id) { try { history.replaceState(history.state, "", "#" + d.id); } catch (e) {} }
    });
  });
  openFromHash();

  /* Banner actions and copying */
  document.querySelectorAll("[data-show-path]").forEach(function (b) {
    b.addEventListener("click", function () {
      state.cli = b.getAttribute("data-show-path");
      apply();
      byId("topology").scrollIntoView({ block: "start" });
    });
  });
  function copier(b, text) {
    var was = b.textContent;
    var done = function () { b.textContent = "Copied"; setTimeout(function () { b.textContent = was; }, 1600); };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text()).then(done, function () { b.textContent = "Select the command to copy it"; });
    } else {
      b.textContent = "Select the command to copy it";
    }
  }
  document.querySelectorAll("[data-copy]").forEach(function (b) { b.addEventListener("click", function () { copier(b, function () { return b.getAttribute("data-copy"); }); }); });
  document.querySelectorAll("[data-copy-from]").forEach(function (b) { b.addEventListener("click", function () { copier(b, function () { return byId(b.getAttribute("data-copy-from")).textContent; }); }); });

  /* Command centre */
  document.querySelectorAll(".tab[data-sev]").forEach(function (t) {
    t.addEventListener("click", function () {
      var sev = t.getAttribute("data-sev");
      document.querySelectorAll(".tab[data-sev]").forEach(function (o) { o.setAttribute("aria-pressed", String(o === t)); });
      document.querySelectorAll("[data-sevrow]").forEach(function (row) { row.classList.toggle("sevout", !!sev && row.getAttribute("data-sevrow") !== sev); });
    });
  });
  document.addEventListener("keydown", function (e) {
    var typing = /^(INPUT|TEXTAREA|SELECT)$/.test((e.target && e.target.tagName) || "");
    if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typing)) { e.preventDefault(); q.focus(); q.select(); }
  });
  document.querySelectorAll("[data-print]").forEach(function (b) { b.addEventListener("click", function () { window.print(); }); });
  document.querySelectorAll("[data-export]").forEach(function (b) {
    b.addEventListener("click", function () {
      var el = byId("export");
      var text = el ? JSON.parse(el.textContent) : "";
      if (!text) { return; }
      var a = document.createElement("a");
      a.href = URL.createObjectURL(new Blob([text], { type: "application/json" }));
      a.download = "passmcp-graph-share-safe.json";
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(function () { URL.revokeObjectURL(a.href); }, 1000);
    });
  });
  document.querySelectorAll("[data-guide]").forEach(function (a) {
    a.addEventListener("click", function () {
      var m = byId("mitigation");
      m.classList.add("lit");
      setTimeout(function () { m.classList.remove("lit"); }, 1600);
    });
  });

  /* Flows: measured from the laid-out page, so they meet the dots and the
     ring at every width; redrawn when the hero resizes. */
  var svgNS = "http://www.w3.org/2000/svg";
  function drawFlows() {
    var hero = byId("overview"), svg = byId("flows");
    if (!hero || !svg) { return; }
    while (svg.firstChild) { svg.removeChild(svg.firstChild); }
    if (getComputedStyle(svg).display === "none") { return; }
    var hb = hero.getBoundingClientRect(), rb = hero.querySelector(".ring").getBoundingClientRect();
    var cx = rb.left + rb.width / 2 - hb.left, cy = rb.top + rb.height / 2 - hb.top, R = rb.width * 0.42;
    svg.setAttribute("viewBox", "0 0 " + hb.width + " " + hb.height);
    hero.querySelectorAll(".fdot").forEach(function (dot, i) {
      var b = dot.getBoundingClientRect();
      var x = b.left + b.width / 2 - hb.left, y = b.top + b.height / 2 - hb.top;
      var left = dot.getAttribute("data-flow") === "c";
      var tx = left ? cx - R : cx + R, ty = cy + (y - cy) * 0.22;
      var dx = (tx - x) * 0.55;
      var d = "M" + x + "," + y + " C" + (x + dx) + "," + y + " " + (tx - dx) + "," + ty + " " + tx + "," + ty;
      var colour = getComputedStyle(dot).backgroundColor;
      ["glow", "fl", "run"].forEach(function (cls) {
        var p = document.createElementNS(svgNS, "path");
        p.setAttribute("d", d);
        p.setAttribute("class", cls);
        if (cls === "run") { p.setAttribute("pathLength", "23.4"); p.style.animationDelay = (-i * 0.45) + "s"; } else { p.style.stroke = colour; }
        svg.appendChild(p);
      });
    });
  }
  drawFlows();
  if ("ResizeObserver" in window) { new ResizeObserver(function () { drawFlows(); }).observe(byId("overview")); }
  window.addEventListener("load", drawFlows);

  /* Counters */
  var still = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (!still && window.requestAnimationFrame) {
    document.querySelectorAll("[data-count]").forEach(function (el) {
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
    });
  }

  /* Navigation highlight */
  var links = document.querySelectorAll(".nav a, .tabbar a");
  if ("IntersectionObserver" in window) {
    var seen = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) { return; }
        links.forEach(function (a) { a.classList.toggle("on", a.getAttribute("href") === "#" + e.target.id); });
      });
    }, { rootMargin: "-40% 0px -55% 0px" });
    document.querySelectorAll("main > section[id], main .cols > section[id]").forEach(function (s) { seen.observe(s); });
  }
});
