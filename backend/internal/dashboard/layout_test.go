package dashboard

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	many := make([]Card, MaxCards+1)
	for i := range many {
		many[i] = Card{ID: "c" + strings.Repeat("x", i%39)}
	}
	for _, tc := range []struct {
		name  string
		cards []Card
		ok    bool
	}{
		{"empty resets", []Card{}, true},
		{"normal", []Card{{ID: "gsc"}, {ID: "ai_visibility", Hidden: true}}, true},
		{"longest id", []Card{{ID: "a" + strings.Repeat("b", 39)}}, true},
		{"id too long", []Card{{ID: "a" + strings.Repeat("b", 40)}}, false},
		{"uppercase", []Card{{ID: "GSC"}}, false},
		{"starts with digit", []Card{{ID: "1gsc"}}, false},
		{"empty id", []Card{{ID: ""}}, false},
		{"unicode", []Card{{ID: "gscé"}}, false},
		{"duplicate", []Card{{ID: "gsc"}, {ID: "gsc", Hidden: true}}, false},
		{"at the cap", many[:MaxCards], true},
		{"over the cap", many, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.cards)
			if tc.ok && err != nil {
				t.Fatalf("Validate = %v, want nil", err)
			}
			if !tc.ok && !errors.Is(err, ErrInvalidLayout) {
				t.Fatalf("Validate = %v, want ErrInvalidLayout", err)
			}
		})
	}
}
