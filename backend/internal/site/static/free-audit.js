// The free SEO and AEO audit page: runs the check for ?url= (set by this
// page's form or the landing page's) and shows the server-rendered report.
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
  if (!form || !input || !progress || !result) return;
  var button = form.querySelector("button");
  var steps = progress.querySelectorAll("li");
  var timer = 0;
  var running = false;

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

  function run(address) {
    address = address.trim();
    if (!address || running) return;
    running = true;
    button.disabled = true;
    result.innerHTML = "";
    showProgress(address.replace(/^https?:\/\//i, "").split("/")[0]);
    progress.scrollIntoView({ behavior: "smooth", block: "center" });
    fetch("/free-seo-audit/report?url=" + encodeURIComponent(address), {
      headers: { Accept: "text/html" },
      credentials: "same-origin",
    })
      .then(function (response) {
        return response.text();
      })
      .catch(function () {
        return '<div class="fa-card fa-error" role="alert"><p>Could not reach Uplumo. Check your connection and try again.</p></div>';
      })
      .then(function (html) {
        stopProgress();
        // The report is HTML rendered by our own server with escaped values.
        result.innerHTML = html;
        result.scrollIntoView({ behavior: "smooth", block: "start" });
        running = false;
        button.disabled = false;
      });
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    var address = input.value.trim();
    if (!address) return;
    var url = new URL(window.location.href);
    url.searchParams.set("url", address);
    window.history.pushState(null, "", url);
    run(address);
  });

  var initial = new URLSearchParams(window.location.search).get("url");
  if (initial) {
    input.value = initial;
    run(initial);
  }
})();
