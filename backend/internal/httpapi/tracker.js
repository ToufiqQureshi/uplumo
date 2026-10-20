// Uplumo first-party tracker. No cookies, third-party requests or raw IDs sent to storage.
(function () {
  var script = document.currentScript;
  var nav = navigator;
  var site = script && script.getAttribute("data-site");
  if (!site || nav.doNotTrack === "1" || window.doNotTrack === "1" || nav.globalPrivacyControl) return;

  var endpoint = new URL("/collect", script.src).href;
  var sessionKey = "uplumo-session:" + site;
  var session = null;
  var storageAvailable = true;
  var last = "";
  var pageviewID = "";
  var pageURL = "";
  var pageReferrer = "";
  var engagedMS = 0;
  var tickAt = 0;
  // Whether the page was visible since tickAt. Checked instead of the current
  // state, because visibilitychange fires after the page is already hidden.
  var visibleSinceTick = false;
  var idleMS = 30 * 60 * 1000;
  var maxSessionMS = 24 * 60 * 60 * 1000;

  // A tab opened by another page starts with a copy of the opener's
  // sessionStorage. Drop the copied session on that first page only: later
  // pages and reloads in the same tab keep window.opener but are one visit.
  var navEntry = performance.getEntriesByType && performance.getEntriesByType("navigation")[0];
  if (window.opener && history.length === 1 && !(navEntry && navEntry.type === "reload")) {
    try { sessionStorage.removeItem(sessionKey); } catch (_) { storageAvailable = false; }
  }

  function id() {
    var bytes = new Uint8Array(16);
    try {
      if (!window.crypto || !window.crypto.getRandomValues) throw new Error("secure random unavailable");
      window.crypto.getRandomValues(bytes);
    } catch (_) {
      for (var i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
    }
    return Array.prototype.map.call(bytes, function (n) { return n.toString(16).padStart(2, "0"); }).join("");
  }

  function storeSession() {
    if (!storageAvailable) return;
    try {
      sessionStorage.setItem(sessionKey, JSON.stringify(session));
    } catch (_) {
      storageAvailable = false;
    }
  }

  function currentSession() {
    var now = Date.now();
    if (!session && storageAvailable) {
      try {
        session = JSON.parse(sessionStorage.getItem(sessionKey) || "null");
      } catch (_) {
        storageAvailable = false;
      }
    }
    if (!session || !/^[0-9a-f]{32}$/.test(session.id || "") ||
        !Number.isFinite(session.started) || !Number.isFinite(session.last) ||
        now - session.last > idleMS || now - session.started > maxSessionMS ||
        session.started > now + 60000 || session.last > now + 60000) {
      session = { id: id(), started: now, last: now };
    } else {
      session.last = now;
    }
    storeSession();
    return session.id;
  }

  function post(data) {
    var body = JSON.stringify(data);
    if (!(nav.sendBeacon && nav.sendBeacon(endpoint, body))) {
      fetch(endpoint, { method: "POST", body: body, keepalive: true, mode: "no-cors" }).catch(function () {});
    }
  }

  function payload(extra) {
    var data = { k: site, u: pageURL, r: pageReferrer, w: screen.width, s: session ? session.id : currentSession(), p: pageviewID };
    Object.keys(extra || {}).forEach(function (key) { data[key] = extra[key]; });
    return data;
  }

  function accrue() {
    var now = performance.now();
    if (tickAt && visibleSinceTick) engagedMS += Math.min(now - tickAt, 15000);
    tickAt = now;
    visibleSinceTick = document.visibilityState === "visible";
  }

  function flushEngagement() {
    accrue();
    if (engagedMS > 0 && pageviewID) post(payload({ d: Math.min(Math.floor(engagedMS), maxSessionMS) }));
  }

  function startPage(referrer, force) {
    var url = location.href.split("#")[0];
    if (url === last && !force) return;
    if (pageviewID) flushEngagement();
    currentSession();
    last = url;
    pageURL = url;
    pageReferrer = referrer || "";
    pageviewID = id();
    engagedMS = 0;
    tickAt = performance.now();
    visibleSinceTick = document.visibilityState === "visible";
    post(payload({ r: referrer || "" }));
  }

  function track(name) {
    name = String(name || "").trim().slice(0, 64);
    if (!name) return;
    var oldSession = session && session.id;
    var activeSession = currentSession();
    if (oldSession && oldSession !== activeSession) startPage("", true);
    post(payload({ n: name, i: id() }));
  }

  addEventListener("click", function (e) {
    var el = e.target && e.target.closest && e.target.closest("[data-uplumo-event]");
    if (el) track(el.getAttribute("data-uplumo-event"));
  }, true);
  window.uplumo = track;

  var pushState = history.pushState;
  history.pushState = function () {
    var from = last;
    pushState.apply(this, arguments);
    startPage(from, false);
  };
  addEventListener("popstate", function () { startPage(last, false); });
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "hidden") flushEngagement();
    else {
      var oldSession = session && session.id;
      var activeSession = currentSession();
      if (oldSession && oldSession !== activeSession) startPage("", true);
      tickAt = performance.now();
      visibleSinceTick = true;
    }
  });
  addEventListener("pagehide", flushEngagement);
  setInterval(function () {
    if (document.visibilityState !== "visible") return;
    var oldSession = session && session.id;
    var activeSession = currentSession();
    if (oldSession && oldSession !== activeSession) startPage("", true);
    flushEngagement();
  }, 15000);

  startPage(document.referrer, false);
})();
