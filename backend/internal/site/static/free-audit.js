// The free SEO and AEO audit page. The form and the two tabs run the check
// for ?url= (set by this page's form, or by the landing page's buttons) and
// show the server-rendered report. Switching tabs after a report is shown
// loads the other engine's report from the cache (cached=1): it never runs a
// fresh check of the site.
(function () {
  // Download PDF prints the report. Every check is opened first so the
  // printed page has the details, and closed again afterwards.
  var opened = [];
  window.addEventListener("beforeprint", function () {
    opened = [];
    document.querySelectorAll("#audit-result details").forEach(function (detail) {
      if (!detail.open) {
        detail.open = true;
        opened.push(detail);
      }
    });
  });
  window.addEventListener("afterprint", function () {
    opened.forEach(function (detail) {
      detail.open = false;
    });
    opened = [];
  });
  document.addEventListener("click", function (event) {
    var target = event.target;
    if (target && target.closest && target.closest("[data-print-report]")) {
      window.print();
    }
  });

  var form = document.getElementById("audit-form");
  var input = document.getElementById("audit-url");
  var progress = document.getElementById("audit-progress");
  var result = document.getElementById("audit-result");
  var tabs = document.querySelectorAll(".fa-tab");
  if (!form || !input || !progress || !result) return;
  var button = form.querySelector("button");
  var steps = progress.querySelectorAll("li");
  var timer = 0;
  var running = false;
  var kind = "seo";
  var address = "";
  // The HTML of each report shown for the current address, by kind.
  var reports = {};

  function currentKind() {
    var selected = document.querySelector(".fa-tab[aria-selected='true']");
    return selected ? selected.getAttribute("data-kind") : "seo";
  }

  function setTab(next) {
    kind = next;
    tabs.forEach(function (tab) {
      tab.setAttribute("aria-selected", tab.getAttribute("data-kind") === next ? "true" : "false");
    });
    form.setAttribute("action", next === "aeo" ? "/free-aeo-audit" : "/free-seo-audit");
    button.textContent = "Run " + (next === "aeo" ? "AEO" : "SEO") + " audit →";
  }

  function showURL(replace) {
    var url = new URL(window.location.href);
    if (address) url.searchParams.set("url", address);
    url.searchParams.set("type", kind);
    if (replace) {
      window.history.replaceState(null, "", url);
    } else {
      window.history.pushState(null, "", url);
    }
  }

  function showProgress(host) {
    document.getElementById("audit-progress-host").textContent = host;
    progress.hidden = false;
    var step = 0;
    function tick() {
      for (var i = 0; i < steps.length; i++) {
        steps[i].className = i < step ? "done" : i === step ? "on" : "";
      }
      // The last step keeps spinning until the report arrives.
      if (step < steps.length - 1) step++;
    }
    tick();
    timer = window.setInterval(tick, 1600);
  }

  function stopProgress() {
    window.clearInterval(timer);
    progress.hidden = true;
  }

  function show(html) {
    // The report is HTML rendered by our own server with escaped values.
    result.innerHTML = html;
    result.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  // fetchReport loads the report of one kind as {status, html}. cachedOnly asks
  // the server to read the cache only, so a tab switch never counts as a fresh
  // check; a 404 then means the cache no longer holds the page.
  function fetchReport(target, cachedOnly) {
    var query = "/free-seo-audit/report?url=" + encodeURIComponent(address) + "&type=" + target;
    if (cachedOnly) query += "&cached=1";
    return fetch(query, {
      headers: { Accept: "text/html" },
      credentials: "same-origin",
    }).then(function (response) {
      return response.text().then(function (html) {
        return { status: response.status, html: html };
      });
    });
  }

  function run(target) {
    if (!address || running) return;
    running = true;
    button.disabled = true;
    result.innerHTML = "";
    showProgress(address.replace(/^https?:\/\//i, "").split("/")[0]);
    progress.scrollIntoView({ behavior: "smooth", block: "center" });
    fetchReport(target, false)
      .catch(function () {
        return { status: 0, html: '<div class="fa-card fa-error" role="alert"><p>Could not reach Uplumo. Check your connection and try again.</p></div>' };
      })
      .then(function (reply) {
        stopProgress();
        reports[target] = reply.html;
        show(reply.html);
        running = false;
        button.disabled = false;
      });
  }

  // selectTab switches the engine. With a report already shown for this
  // address, the other engine's report comes from the cache (no fresh check).
  function selectTab(target) {
    setTab(target);
    if (!address) return;
    showURL(true);
    if (reports[target]) {
      show(reports[target]);
      return;
    }
    if (!Object.keys(reports).length) return;
    fetchReport(target, true)
      .catch(function () {
        return { status: 0, html: "" };
      })
      .then(function (reply) {
        if (reply.status === 404) {
          // The cache no longer holds the page: the visitor asked to see it,
          // so check it again.
          run(target);
          return;
        }
        if (reply.status === 0) return;
        reports[target] = reply.html;
        show(reply.html);
      });
  }

  tabs.forEach(function (tab) {
    tab.addEventListener("click", function (event) {
      // Without JavaScript the link opens the other page.
      event.preventDefault();
      selectTab(tab.getAttribute("data-kind"));
    });
  });

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    var typed = input.value.trim();
    if (!typed) return;
    if (typed !== address) reports = {};
    address = typed;
    kind = currentKind();
    showURL(false);
    run(kind);
  });

  var params = new URLSearchParams(window.location.search);
  var initial = params.get("url");
  var initialKind = params.get("type");
  setTab(initialKind === "aeo" || initialKind === "seo" ? initialKind : currentKind());
  if (initial) {
    input.value = initial;
    address = initial.trim();
    kind = currentKind();
    run(kind);
  }
})();
