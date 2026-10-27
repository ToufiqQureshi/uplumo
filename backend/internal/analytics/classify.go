package analytics

import (
	"net/url"
	"slices"
	"strings"
)

// Channels a pageview is attributed to. channelInternal marks a
// pageview reached from the site itself (a click between pages); it is
// stored so pageview counts stay complete, but it is not an acquisition
// channel and the summary leaves it out.
const (
	channelAI       = "ai"
	channelSearch   = "search"
	channelSocial   = "social"
	channelPaid     = "paid"
	channelEmail    = "email"
	channelDirect   = "direct"
	channelReferral = "referral"
	channelInternal = "internal"
)

// AI assistants a visit can come from.
const (
	aiChatGPT    = "chatgpt"
	aiPerplexity = "perplexity"
	aiGemini     = "gemini"
	aiClaude     = "claude"
	aiCopilot    = "copilot"
	aiOther      = "other_ai"
)

// Devices, from the user agent and the screen width.
const (
	deviceDesktop = "desktop"
	deviceMobile  = "mobile"
	deviceTablet  = "tablet"
)

// aiDomains maps assistant hosts to the source they count as. A host
// matches its entry or any subdomain of it (www.perplexity.ai).
var aiDomains = map[string]string{
	"chatgpt.com":             aiChatGPT,
	"chat.openai.com":         aiChatGPT,
	"perplexity.ai":           aiPerplexity,
	"gemini.google.com":       aiGemini,
	"bard.google.com":         aiGemini,
	"claude.ai":               aiClaude,
	"copilot.microsoft.com":   aiCopilot,
	"copilot.cloud.microsoft": aiCopilot,
	"chat.deepseek.com":       aiOther,
	"chat.mistral.ai":         aiOther,
	"meta.ai":                 aiOther,
	"grok.com":                aiOther,
	"poe.com":                 aiOther,
	"you.com":                 aiOther,
	"phind.com":               aiOther,
	"chat.qwen.ai":            aiOther,
}

// aiTags maps bare utm_source values that assistants and their users put in
// links ("utm_source=chatgpt") to the source they count as.
var aiTags = map[string]string{
	"chatgpt":    aiChatGPT,
	"openai":     aiChatGPT,
	"perplexity": aiPerplexity,
	"gemini":     aiGemini,
	"bard":       aiGemini,
	"claude":     aiClaude,
	"copilot":    aiCopilot,
}

// brandChannels maps a brand label to its channel. Search engines and some
// networks run one host per country (google.co.in, yandex.ru), so any host
// with the brand as a whole label matches: www.google.co.in, l.facebook.com.
// A search brand only counts when the labels before it are search front ends
// (searchPrefixes): accounts.google.com is a sign-in hop and docs.google.com
// a referral, not a search. The same table classifies bare utm_source values
// such as "google".
var brandChannels = map[string]string{
	"google":     channelSearch,
	"bing":       channelSearch,
	"yahoo":      channelSearch,
	"yandex":     channelSearch,
	"baidu":      channelSearch,
	"duckduckgo": channelSearch,
	"ecosia":     channelSearch,
	"naver":      channelSearch,
	"seznam":     channelSearch,
	"facebook":   channelSocial,
	"instagram":  channelSocial,
	"linkedin":   channelSocial,
	"reddit":     channelSocial,
	"pinterest":  channelSocial,
	"youtube":    channelSocial,
	"tiktok":     channelSocial,
	"twitter":    channelSocial,
	"quora":      channelSocial,
	"whatsapp":   channelSocial,
	"telegram":   channelSocial,
}

// searchPrefixes are the labels allowed before a search brand: mobile and
// search front ends (m.bing.com, in.search.yahoo.com, html.duckduckgo.com,
// news.google.com). Two-letter country labels (cn.bing.com) are allowed too.
var searchPrefixes = map[string]bool{"m": true, "search": true, "html": true, "lite": true, "news": true}

// passThroughDomains are sign-in and payment pages a visitor passes through
// and returns from in the same visit (Google sign-in, Razorpay checkout).
// Counting them as a source would turn every login or payment into a new
// "search" or "referral" visit, so they are treated like internal navigation,
// as GA4's unwanted-referral list does, but without manual setup.
var passThroughDomains = []string{
	"accounts.google.com",
	"accounts.youtube.com",
	"login.microsoftonline.com",
	"login.live.com",
	"appleid.apple.com",
	"razorpay.com",
	"rzp.io",
	"checkout.stripe.com",
	"hooks.stripe.com",
	"paypal.com",
	"payu.in",
	"cashfree.com",
	"securegw.paytm.in",
	"mercury.phonepe.com",
	"ccavenue.com",
	"instamojo.com",
	"billdesk.com",
}

// appChannels maps Android app referrers (android-app://<package>/) to the
// channel the app belongs to. Gmail and the Google app send these instead of
// a web referrer, so without this a newsletter click read as direct.
var appChannels = map[string]string{
	"com.google.android.gm":                   channelEmail,
	"com.microsoft.office.outlook":            channelEmail,
	"com.google.android.googlequicksearchbox": channelSearch,
	"com.google.android.apps.searchlite":      channelSearch,
	"com.facebook.katana":                     channelSocial,
	"com.facebook.lite":                       channelSocial,
	"com.facebook.orca":                       channelSocial,
	"com.instagram.android":                   channelSocial,
	"com.linkedin.android":                    channelSocial,
	"com.twitter.android":                     channelSocial,
	"com.reddit.frontpage":                    channelSocial,
	"com.pinterest":                           channelSocial,
	"com.google.android.youtube":              channelSocial,
	"com.whatsapp":                            channelSocial,
	"org.telegram.messenger":                  channelSocial,
}

// domainChannels lists hosts whose name is not a brand label, and webmail
// hosts that would otherwise match a search brand. A host matches its entry
// or any subdomain of it, and these entries win over brandChannels.
var domainChannels = map[string]string{
	"mail.google.com":       channelEmail,
	"inbox.google.com":      channelEmail,
	"outlook.live.com":      channelEmail,
	"outlook.office.com":    channelEmail,
	"outlook.office365.com": channelEmail,
	"mail.yahoo.com":        channelEmail,
	"mail.zoho.com":         channelEmail,
	"mail.zoho.in":          channelEmail,
	"mail.proton.me":        channelEmail,
	"mail.yandex.ru":        channelEmail,
	"search.brave.com":      channelSearch,
	"startpage.com":         channelSearch,
	"qwant.com":             channelSearch,
	"kagi.com":              channelSearch,
	"t.co":                  channelSocial,
	"x.com":                 channelSocial,
	"fb.com":                channelSocial,
	"lnkd.in":               channelSocial,
	"threads.net":           channelSocial,
	"bsky.app":              channelSocial,
	"news.ycombinator.com":  channelSocial,
	"t.me":                  channelSocial,
	"wa.me":                 channelSocial,
}

// source is where a pageview came from.
type source struct {
	Channel      string
	AISource     string // set only for channelAI
	ReferrerHost string // the external referrer's host, empty for direct and internal
}

// classify attributes a pageview of page, reached from referrer, to a
// channel. domain is the project's domain, empty when the project has none.
//
// The heuristic, in order:
//  1. A referrer on the page's own host, or on the project's domain or a
//     subdomain of it, is internal navigation. utm_source is ignored then,
//     because SPAs keep the landing page's query string as users click on.
//     So is a sign-in or payment page the visitor returns from
//     (passThroughDomains).
//  2. A referrer on a known AI assistant host is that assistant.
//  3. A utm_source naming an assistant ("chatgpt.com", which ChatGPT appends
//     to every link it cites, or "perplexity") is that assistant. Many
//     assistants strip the referrer, so this recovers AI visits that would
//     otherwise read as direct.
//  4. A paid/email utm_medium or a recognized click-id parameter classifies
//     paid/email traffic. Only the parameter name is used; its value is dropped.
//  5. A referrer on a search engine, webmail or social network is that
//     channel; so is a known Android app referrer
//     (android-app://com.google.android.gm/ is Gmail, so email).
//  6. Any other parseable web referrer host is a referral.
//  7. With no usable referrer, a utm_source naming a search engine or social
//     network is that channel (in-app browsers often send no referrer).
//  8. Everything else is direct.
//
// Hosts compare case-insensitively without a leading "www.".
func classify(page *url.URL, referrer, domain string) source {
	pageHost := normalizeHost(page.Hostname())
	refHost, appPackage := "", ""
	if ref, err := url.Parse(strings.TrimSpace(referrer)); err == nil {
		switch ref.Scheme {
		case "http", "https":
			refHost = normalizeHost(ref.Hostname())
		case "android-app":
			appPackage = strings.ToLower(ref.Hostname())
		}
	}
	if refHost != "" && (refHost == pageHost || (domain != "" && withinDomain(refHost, domain)) || passThrough(refHost)) {
		return source{Channel: channelInternal}
	}

	utm := normalizeHost(page.Query().Get("utm_source"))
	if ai := aiSourceOf(refHost); ai != "" {
		return source{Channel: channelAI, AISource: ai, ReferrerHost: refHost}
	}
	if ai := aiSourceOf(utm); ai != "" {
		return source{Channel: channelAI, AISource: ai, ReferrerHost: refHost}
	}
	medium := strings.ToLower(strings.TrimSpace(page.Query().Get("utm_medium")))
	if ch := mediumChannel(medium, utm); ch != "" {
		return source{Channel: ch, ReferrerHost: refHost}
	}
	if clickSource := adClickSource(page); clickSource != "" {
		if clickSource == "Meta click (auto-tagged)" {
			return source{Channel: channelSocial, ReferrerHost: refHost}
		}
		return source{Channel: channelPaid, ReferrerHost: refHost}
	}
	if refHost != "" {
		if ch := channelOf(refHost); ch != "" {
			return source{Channel: ch, ReferrerHost: refHost}
		}
		return source{Channel: channelReferral, ReferrerHost: refHost}
	}
	if ch, ok := appChannels[appPackage]; ok {
		return source{Channel: ch, ReferrerHost: appPackage}
	}
	if ch := channelOf(utm); ch != "" {
		return source{Channel: ch}
	}
	return source{Channel: channelDirect}
}

func adClickSource(page *url.URL) string {
	for _, tag := range []struct{ parameter, source string }{
		{"gclid", "Google Ads (auto-tagged)"},
		{"gbraid", "Google Ads (auto-tagged)"},
		{"wbraid", "Google Ads (auto-tagged)"},
		{"msclkid", "Microsoft Ads (auto-tagged)"},
		{"li_fat_id", "LinkedIn (auto-tagged)"},
		{"fbclid", "Meta click (auto-tagged)"},
	} {
		if page.Query().Get(tag.parameter) != "" {
			return tag.source
		}
	}
	return ""
}

// mediumChannel applies GA4's default-channel medium rules
// (support.google.com/analytics/answer/9756891) to a campaign-tagged visit:
// paid mediums (anything containing "cp", ppc, retargeting, paid*, display
// formats, affiliate), email in the medium or source, social and organic
// mediums. It returns "" when the medium names no channel.
func mediumChannel(medium, utmSource string) string {
	switch {
	case medium == "":
		if isEmailTag(utmSource) {
			return channelEmail
		}
		return ""
	case strings.Contains(medium, "cp"), strings.HasPrefix(medium, "paid"),
		slices.Contains([]string{"ppc", "retargeting", "display", "banner", "expandable", "interstitial", "affiliate", "sponsored"}, medium):
		return channelPaid
	case isEmailTag(medium), medium == "newsletter", isEmailTag(utmSource):
		return channelEmail
	case slices.Contains([]string{"social", "social-network", "social-media", "sm", "social network", "social media"}, medium):
		return channelSocial
	case medium == "organic":
		return channelSearch
	}
	return ""
}

func isEmailTag(tag string) bool {
	return slices.Contains([]string{"email", "e-mail", "e_mail", "e mail"}, tag)
}

// aiSourceOf returns the assistant host or utm tag names, or "".
func aiSourceOf(host string) string {
	if host == "" {
		return ""
	}
	if ai, ok := aiTags[host]; ok {
		return ai
	}
	for d, ai := range aiDomains {
		if withinDomain(host, d) {
			return ai
		}
	}
	return ""
}

// channelOf returns the search, email or social channel of host, or "".
func channelOf(host string) string {
	if host == "" {
		return ""
	}
	for d, ch := range domainChannels {
		if withinDomain(host, d) {
			return ch
		}
	}
	labels := strings.Split(host, ".")
	for i, label := range labels {
		ch, ok := brandChannels[label]
		if !ok {
			continue
		}
		if ch == channelSearch && slices.ContainsFunc(labels[:i], notSearchFrontEnd) {
			return ""
		}
		return ch
	}
	return ""
}

// notSearchFrontEnd reports whether a label before a search brand marks a
// different product (accounts, docs) rather than a search front end.
func notSearchFrontEnd(label string) bool {
	return !searchPrefixes[label] && len(label) != 2
}

// passThrough reports whether host is a sign-in or payment page.
func passThrough(host string) bool {
	return slices.ContainsFunc(passThroughDomains, func(d string) bool { return withinDomain(host, d) })
}

// withinDomain reports whether host is domain or a subdomain of it.
func withinDomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// normalizeHost lowercases host and drops a trailing dot and a leading
// "www.", the form legacy projects store their domain in.
func normalizeHost(host string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return strings.TrimPrefix(host, "www.")
}

// Screen widths below these are phones and tablets.
const (
	maxMobileWidth = 767
	maxTabletWidth = 1024
)

// deviceOf classifies the visitor's device. The user agent decides when it
// says: "Mobi" is the cross-browser mobile marker, and iPads and Android
// devices without "Mobi" are tablets. iPadOS sends a desktop Safari user
// agent, so a touch-sized screen width decides otherwise.
func deviceOf(userAgent string, screenWidth int) string {
	switch {
	case strings.Contains(userAgent, "Mobi"):
		return deviceMobile
	case strings.Contains(userAgent, "iPad"), strings.Contains(userAgent, "Tablet"), strings.Contains(userAgent, "Android"):
		return deviceTablet
	case screenWidth > 0 && screenWidth <= maxMobileWidth:
		return deviceMobile
	case screenWidth > maxMobileWidth && screenWidth < maxTabletWidth:
		return deviceTablet
	default:
		return deviceDesktop
	}
}

// isBot reports whether userAgent belongs to a crawler or headless browser.
// Crawlers such as Googlebot run JavaScript and would fire the tracker.
func isBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, marker := range []string{"bot", "crawler", "spider", "headless", "lighthouse", "slurp"} {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return userAgent == ""
}
