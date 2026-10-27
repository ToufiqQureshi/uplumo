package analytics

import (
	"slices"
	"testing"
)

func TestIsEnquiry(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// The tracker's own contact events.
		{"whatsapp_click", true},
		{"call_click", true},
		{"sms_click", true},
		{"email_click", true},
		{"form_submit", true},
		// Owner-tagged contact buttons in any style.
		{"WhatsApp", true},
		{"WhatsAppClick", true},
		{"wa_button", true},
		{"Call Now", true},
		{"click-to-call", true},
		{"phone_tap", true},
		{"SMSButton", true},
		{"contact_form", true},
		{"Enquiry", true},
		{"send-inquiry", true},
		{"get_quote", true},
		{"BookDemo", true},
		{"booking_confirmed", true},
		{"lead", true},
		// Words that only contain an enquiry word are not enquiries.
		{"facebook_click", false},
		{"ebook_download", false},
		{"platform_menu", false},
		{"hotel_gallery", false},
		{"callout_close", false},
		{"recall", false},
		{"performance_tab", false},
		// Contact elements that were only seen or failed.
		{"form_start", false},
		{"contact_page_view", false},
		{"booking_error", false},
		{"whatsapp_open", false},
		// Sales steps and navigation are not enquiries.
		{"signup_click", false},
		{"sign_up", false},
		{"subscription_checkout_agency", false},
		{"purchase", false},
		{"pricing_open", false},
		{"outbound_click", false},
		{"", false},
		{"   ", false},
	}
	for _, tt := range tests {
		if got := isEnquiry(tt.name); got != tt.want {
			t.Errorf("isEnquiry(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNameWords(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"BookDemo-2", []string{"book", "demo"}},
		{"SMSButton", []string{"sms", "button"}},
		{"whatsapp_click", []string{"whatsapp", "click"}},
		{"Call Now", []string{"call", "now"}},
		{"संपर्क form", []string{"संपर्क", "form"}},
		{"", nil},
	}
	for _, tt := range tests {
		if got := nameWords(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("nameWords(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
