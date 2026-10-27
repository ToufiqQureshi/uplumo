package analytics

import (
	"slices"
	"strings"
	"unicode"
)

// Event names the tracker sends on its own for contact intent (see
// httpapi/tracker.js): a tap on a tel:, sms:, mailto: or WhatsApp link, and a
// non-search form submission.
const (
	eventWhatsAppClick = "whatsapp_click"
	eventCallClick     = "call_click"
	eventSMSClick      = "sms_click"
	eventEmailClick    = "email_click"
	eventFormSubmit    = "form_submit"
)

// enquirySubstrings are safe to match anywhere in a name ("WhatsAppClick"),
// because no unrelated English word contains them.
var enquirySubstrings = []string{"whatsapp", "enquir", "inquir"}

// enquiryWords must be a whole word of the name: "book_demo" is an enquiry,
// "facebook_click" and "ebook_download" are not.
var enquiryWords = []string{
	"wa", "call", "callback", "phone", "tel", "telephone", "sms",
	"form", "contact", "lead", "leads", "quote", "book", "booking", "appointment",
}

// notEnquiryWords mark an event about a contact element that is not an
// enquiry itself: "form_start", "contact_page_view", "booking_error".
var notEnquiryWords = []string{"start", "view", "open", "close", "error", "abandon", "fail", "failed"}

// isEnquiry reports whether a custom event name means a visitor tried to
// reach the business: WhatsApp, call, SMS, email or form. Sign-ups,
// checkouts and purchases are sales steps, not enquiries, so they are not
// counted here; they stay visible as custom events.
func isEnquiry(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch lower {
	case eventWhatsAppClick, eventCallClick, eventSMSClick, eventEmailClick, eventFormSubmit:
		return true
	}
	words := nameWords(name)
	if slices.ContainsFunc(words, func(w string) bool { return slices.Contains(notEnquiryWords, w) }) {
		return false
	}
	if slices.ContainsFunc(enquirySubstrings, func(s string) bool { return strings.Contains(lower, s) }) {
		return true
	}
	return slices.ContainsFunc(words, func(w string) bool { return slices.Contains(enquiryWords, w) })
}

// nameWords splits an event name into lower-case words at punctuation,
// spaces, digits and camelCase boundaries: "BookDemo-2" is [book demo].
func nameWords(name string) []string {
	var words []string
	var word []rune
	flush := func() {
		if len(word) > 0 {
			words = append(words, strings.ToLower(string(word)))
			word = word[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		// Marks belong to the word: Devanagari vowel signs are marks.
		if !unicode.IsLetter(r) && !unicode.IsMark(r) {
			flush()
			continue
		}
		// A capital starts a word after a lower-case letter ("bookDemo"), or
		// ends an acronym before a lower-case letter ("SMSButton" -> SMS, Button).
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) ||
			unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
			flush()
		}
		word = append(word, r)
	}
	flush()
	return words
}
