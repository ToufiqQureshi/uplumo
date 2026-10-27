// Uplumo first-party tracker. No cookies or third-party requests; the server
// stores only site-scoped ID hashes. Browser recognition is explicitly opt-in.
(function () {
  var script = document.currentScript;
  var nav = navigator;
  var site = script && script.getAttribute("data-site");
  if (!site || nav.doNotTrack === "1" || window.doNotTrack === "1" || nav.globalPrivacyControl) return;
  // Owners exclude their own browser by opening any page with
  // ?uplumo_ignore=1 (and undo it with ?uplumo_ignore=0).
  try {
    var ignore = new URLSearchParams(location.search).get("uplumo_ignore");
    if (ignore === "1") localStorage.setItem("uplumo-ignore", "1");
    else if (ignore === "0") localStorage.removeItem("uplumo-ignore");
    if (localStorage.getItem("uplumo-ignore") === "1") return;
  } catch (_) { /* Blocked storage cannot hold the opt-out; keep counting. */ }

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
  var browserKey = "uplumo-browser:" + site;
  var recognition = script.getAttribute("data-recognition") === "true";
  var collectionAllowed = true;
  var browserTTL = 180 * 24 * 60 * 60 * 1000;

  function privacyBlocked() {
    return nav.doNotTrack === "1" || window.doNotTrack === "1" || nav.globalPrivacyControl;
  }

  function browserID() {
    if (!recognition || !collectionAllowed || privacyBlocked()) return "";
    try {
      var stored = JSON.parse(localStorage.getItem(browserKey) || "null");
      var now = Date.now();
      if (!stored || !/^[0-9a-f]{32}$/.test(stored.id || "") ||
          !Number.isFinite(stored.created) || stored.created > now || now - stored.created >= browserTTL) {
        if (!window.crypto || !window.crypto.getRandomValues) return "";
        var secureID = id(true);
        if (!secureID) return "";
        stored = { id: secureID, created: now };
        localStorage.setItem(browserKey, JSON.stringify(stored));
      }
      return stored.id;
    } catch (_) { return ""; }
  }

  // A tab opened by another page starts with a copy of the opener's
  // sessionStorage. Drop the copied session on that first page only: later
  // pages and reloads in the same tab keep window.opener but are one visit.
  var navEntry = performance.getEntriesByType && performance.getEntriesByType("navigation")[0];
  if (window.opener && history.length === 1 && !(navEntry && navEntry.type === "reload")) {
    try { sessionStorage.removeItem(sessionKey); } catch (_) { storageAvailable = false; }
  }

  function id(requireSecure) {
    var bytes = new Uint8Array(16);
    try {
      if (!window.crypto || !window.crypto.getRandomValues) throw new Error("secure random unavailable");
      window.crypto.getRandomValues(bytes);
    } catch (_) {
      if (requireSecure) return "";
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
    if (!collectionAllowed || privacyBlocked()) return;
    var body = JSON.stringify(data);
    if (!(nav.sendBeacon && nav.sendBeacon(endpoint, body))) {
      fetch(endpoint, { method: "POST", body: body, keepalive: true, mode: "no-cors" }).catch(function () {});
    }
  }

  function payload(extra) {
    var data = { k: site, u: pageURL, r: pageReferrer, w: screen.width, s: session ? session.id : currentSession(), p: pageviewID };
    var browser = browserID();
    if (browser) data.b = browser;
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
    if (!collectionAllowed || privacyBlocked()) return;
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

  function track(name, extra) {
    if (!collectionAllowed || privacyBlocked() || !pageviewID) return;
    name = String(name || "").trim().slice(0, 64);
    if (!name) return;
    var oldSession = session && session.id;
    var activeSession = currentSession();
    if (oldSession && oldSession !== activeSession) startPage("", true);
    var data = { n: name, i: id() };
    if (extra) {
      data.c = true;
      if (extra.destination) data.x = extra.destination;
    }
    post(payload(data));
  }

  // contactEvent names a tap on a phone, SMS, email or WhatsApp link. These
  // are a small business's enquiries, and GA4 does not record them on its
  // own. The number or address itself is never sent.
  function contactEvent(href) {
    var h = String(href || "").trim().toLowerCase();
    if (h.indexOf("tel:") === 0) return "call_click";
    if (h.indexOf("sms:") === 0) return "sms_click";
    if (h.indexOf("mailto:") === 0) return "email_click";
    if (h.indexOf("whatsapp:") === 0) return "whatsapp_click";
    try {
      var url = new URL(href, location.href);
      var host = url.hostname.replace(/^www\./, "");
      if (host === "wa.me" || host === "api.whatsapp.com" || host === "web.whatsapp.com" ||
          host === "whatsapp.com" && url.pathname.indexOf("/send") === 0) return "whatsapp_click";
    } catch (_) { /* Not a URL. */ }
    return "";
  }

  // File types GA4's enhanced measurement counts as downloads.
  var fileTypes = /\.(pdf|xlsx?|docx?|txt|rtf|csv|exe|key|pp[st]x?|7z|pkg|rar|gz|zip|avi|mov|mp4|mpe?g|wmv|midi?|mp3|wav|wma)$/i;

  addEventListener("click", function (e) {
    var target = e.target && (e.target.closest ? e.target : e.target.parentElement);
    if (!target || !target.closest) return;
    var el = target.closest("[data-uplumo-event]");
    var link = target.closest("a[href]");
    var raw = el && el.getAttribute("data-uplumo-destination") || link && link.getAttribute("href");
    var destination = "";
    var external = false;
    var download = false;
    try {
      var url = new URL(raw || "", location.href);
      if (raw && /^https?:$/.test(url.protocol) && !url.username && !url.password) {
        external = url.hostname !== location.hostname;
        download = fileTypes.test(url.pathname);
        destination = url.origin + url.pathname;
        if (destination.length > 2048) destination = "";
      }
    } catch (_) { /* Missing or unsupported targets stay unknown. */ }
    var contact = link && contactEvent(link.getAttribute("href"));
    // An owner's tag names the click; otherwise contact, download, outbound.
    if (el) track(el.getAttribute("data-uplumo-event"), { destination: destination });
    else if (contact) track(contact, {});
    else if (download && destination) track("file_download", { destination: destination });
    else if (external && destination) track("outbound_click", { destination: destination });
  }, true);

  // A submitted form is an enquiry attempt, like GA4's form_submit, except
  // that site search forms are left out, and forms whose button is already
  // tagged are left to that tag so one submission counts once. The submit
  // event only fires after the browser's own validation passes; it does not
  // prove the server accepted the form. Field values are never read.
  function searchForm(form) {
    if (form.closest("[role=search]") || form.querySelector("input[type=search]")) return true;
    if ((form.getAttribute("method") || "get").toLowerCase() !== "get") return false;
    return !!form.querySelector("[name=q],[name=s],[name=search],[name=query],[name=keyword]");
  }
  addEventListener("submit", function (e) {
    var form = e.target;
    if (!form || form.nodeName !== "FORM" || form.hasAttribute("data-uplumo-ignore") ||
        form.closest("[data-uplumo-event]") || form.querySelector("[data-uplumo-event]") || searchForm(form)) return;
    track("form_submit");
  }, true);
  window.uplumo = function (name) { track(name); };
  // Site owners call this from their consent manager. Revocation stops all
  // future collection and clears our browser/session identifiers on this origin.
  window.uplumoConsent = function (granted) {
    if (granted !== true && granted !== false) return;
    recognition = granted;
    collectionAllowed = granted;
    if (!granted) {
      try { localStorage.removeItem(browserKey); } catch (_) { /* Storage may be blocked. */ }
      try { sessionStorage.removeItem(sessionKey); } catch (_) { /* Storage may be blocked. */ }
      session = null;
      pageviewID = "";
      last = "";
    } else {
      initialize();
    }
  };
  addEventListener("storage", function (e) {
    if (e.key === browserKey && e.newValue === null) {
      collectionAllowed = false;
      recognition = false;
      session = null;
      pageviewID = "";
      last = "";
      try { sessionStorage.removeItem(sessionKey); } catch (_) { /* Storage may be blocked. */ }
    }
  });

  var pushState = history.pushState;
  history.pushState = function () {
    var from = last;
    pushState.apply(this, arguments);
    startPage(from, false);
  };
  addEventListener("popstate", function () { startPage(last, false); });
  document.addEventListener("visibilitychange", function () {
    if (!collectionAllowed || privacyBlocked()) return;
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
    if (document.visibilityState !== "visible" || !collectionAllowed || privacyBlocked()) return;
    var oldSession = session && session.id;
    var activeSession = currentSession();
    if (oldSession && oldSession !== activeSession) startPage("", true);
    flushEngagement();
  }, 15000);

  function initialize() {
    // Web Locks serialize first recognition across same-origin tabs. Browsers
    // without that API use the default anonymous mode rather than race IDs.
    if (recognition) {
      if (!nav.locks || !nav.locks.request) recognition = false;
      else {
        nav.locks.request(browserKey, function () {
          browserID();
          startPage(document.referrer, false);
        }).catch(function () { recognition = false; startPage(document.referrer, false); });
        return;
      }
    }
    startPage(document.referrer, false);
  }
  initialize();
})();
