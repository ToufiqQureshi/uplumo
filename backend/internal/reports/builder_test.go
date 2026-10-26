package reports

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuilderDefinitionIsPrivateBoundedAndRoundTrips(t *testing.T) {
	store := newMemoryStore()
	svc := &Service{Store: store}
	in := SaveInput{ProjectID: "p", OrganizationID: "o", Title: "Builder", Summary: "Snapshot", HTML: "<html><body>Visible totals</body></html>", Definition: json.RawMessage(`{"version":1,"datasets":[{"private":"customer rows"}]}`)}
	saved, err := svc.Save(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	item, _, err := svc.GetWithHTML(t.Context(), "p", saved.ReportID)
	if err != nil || string(item.Definition) != string(in.Definition) {
		t.Fatalf("definition did not round trip: %s, %v", item.Definition, err)
	}
	encoded, err := json.Marshal(item)
	if err != nil || strings.Contains(string(encoded), "customer rows") {
		t.Fatalf("definition exposed in metadata: %s, %v", encoded, err)
	}
	if item.SizeBytes != len(in.HTML)+len(in.Definition) {
		t.Fatal("private data not charged against storage quota")
	}
	if _, _, err := svc.GetWithHTML(context.Background(), "other-project", saved.ReportID); err == nil {
		t.Fatal("cross-project read allowed")
	}
	in.ReportID = saved.ReportID
	for _, bad := range []string{`[]`, `{"version":2}`, `{"version":1`, `{"version":1,"data":"` + strings.Repeat("x", MaxHTMLBytes) + `"}`} {
		in.Definition = json.RawMessage(bad)
		if _, err := svc.Save(t.Context(), in); err == nil {
			t.Fatalf("accepted invalid definition: %.40s", bad)
		}
	}
	if string(store.items[saved.ReportID].Definition) != string(item.Definition) {
		t.Fatal("failed save destroyed existing definition")
	}
}
