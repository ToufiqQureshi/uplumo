// Owner-approved analytics for uplumo.com; shared by public pages and the app.
(function () {
  function initialize() {
    if (
      location.hostname !== "uplumo.com" &&
      location.hostname !== "www.uplumo.com"
    )
      return;
    if (
      window.__uplumoAnalyticsLoaded ||
      navigator.doNotTrack === "1" ||
      window.doNotTrack === "1" ||
      navigator.globalPrivacyControl
    )
      return;
    // Authentication links can contain credentials. Do not load analytics there.
    if (
      /^\/(?:sign-in|sign-up|login|verify-email|reset-password|forgot-password|accept-invitation|oauth-consent|api)(?:\/|$)/.test(
        location.pathname,
      )
    )
      return;
    window.__uplumoAnalyticsLoaded = true;

    window.dataLayer = window.dataLayer || [];
    window.gtag =
      window.gtag ||
      function () {
        window.dataLayer.push(arguments);
      };
    window.gtag("js", new Date());
    window.gtag("config", "G-NTP32DNMN2", {
      allow_google_signals: false,
      allow_ad_personalization_signals: false,
    });

    var google = document.createElement("script");
    google.async = true;
    google.src = "https://www.googletagmanager.com/gtag/js?id=G-NTP32DNMN2";
    document.head.appendChild(google);

    var uplumo = document.createElement("script");
    uplumo.defer = true;
    uplumo.setAttribute("data-site", "DDVAYTG2IADAYZQSEM4GMU6HFA");
    uplumo.src = "https://uplumo.com/t.js";
    document.head.appendChild(uplumo);

    // Uplumo's tracker reads the same named attributes. Do not record form values
    // or claim that a checkout click means a successful payment.
    document.addEventListener("click", function (event) {
      var element =
        event.target &&
        event.target.closest &&
        event.target.closest("[data-uplumo-event]");
      if (element && !element.disabled) {
        window.gtag("event", element.getAttribute("data-uplumo-event"));
      }
    });
  }
  window.addEventListener("uplumo:navigation", initialize);
  initialize();
})();
