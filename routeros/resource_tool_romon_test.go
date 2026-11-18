package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit test (no router): /tool/romon singleton; RouterOS `id` is `romon_id` in the schema.
func TestToolRomon(t *testing.T) {
	r := ResourceToolRomon()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	if !r.Schema["secrets"].Sensitive {
		t.Error("secrets must be sensitive")
	}
	c := newFakeClient(map[string][]MikrotikItem{
		"/tool/romon": {{"enabled": "false", "id": "00:00:00:00:00:00", "secrets": ""}},
	})
	// A router with RoMON disabled.
	st, diags := fakeApply(t, r, map[string]cty.Value{"enabled": cty.False}, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	w, _ := c.lastWrite()
	if w.path != "/tool/romon/set" || w.item["enabled"] != "no" || len(w.item) != 1 {
		t.Errorf("set request: %v %v", w.path, w.item)
	}
	if st.ID != "tool.romon" || st.Attributes["romon_id"] != "00:00:00:00:00:00" || st.Attributes["enabled"] != "false" {
		t.Errorf("state: %v", st.Attributes)
	}

	// romon_id is written as `id`.
	c = newFakeClient(map[string][]MikrotikItem{"/tool/romon": {{"enabled": "true", "id": "02:00:00:00:00:01"}}})
	if _, diags := fakeApply(t, r, map[string]cty.Value{"romon_id": cty.StringVal("02:00:00:00:00:01")}, c); diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	if w, _ := c.lastWrite(); w.item["id"] != "02:00:00:00:00:01" {
		t.Errorf("set request: %v", w.item)
	}
}
