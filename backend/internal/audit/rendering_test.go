package audit

import "testing"

func TestIsAuditRenderingAllowed(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"context key does not supply a renderer", map[string]string{"CONTEXT_API_KEY": "k"}, false},
		{"browser flag does not supply a renderer", map[string]string{"AUDIT_BROWSER_RENDERING": "true"}, false},
		{"nothing enabled", map[string]string{}, false},
		{"alternate flag cannot supply a renderer", map[string]string{"AUDIT_BROWSER_RENDERING": "1"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := func(key string) string { return tc.env[key] }
			if got := IsAuditRenderingAllowed(env); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if IsAuditRenderingAllowed(nil) {
		t.Fatal("a nil env cannot enable rendering")
	}
}
