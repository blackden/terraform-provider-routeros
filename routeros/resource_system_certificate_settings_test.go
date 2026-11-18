package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit test (no router): /certificate/settings singleton.
func TestSystemCertificateSettings(t *testing.T) {
	r := ResourceSystemCertificateSettings()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	if got := r.Schema[MetaResourcePath].Default; got != "/certificate/settings" {
		t.Errorf("resource path = %v", got)
	}
	c := newFakeClient(map[string][]MikrotikItem{
		"/certificate/settings": {{
			"builtin-trust-store": "fetch,dns", "crl-download": "false", "crl-store": "ram", "crl-use": "false",
			"current-defaults": "fetch,mqtt,email,netwatch,container,lora,wiliot,dns,www,reverse-proxy",
		}},
	})
	// A router with builtin-trust-store=fetch,dns.
	st, diags := fakeApply(t, r, map[string]cty.Value{
		"builtin_trust_store": cty.SetVal([]cty.Value{cty.StringVal("fetch"), cty.StringVal("dns")}),
	}, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	w, _ := c.lastWrite()
	if w.path != "/certificate/settings/set" || len(w.item) != 1 {
		t.Fatalf("set request: %v %v", w.path, w.item)
	}
	if got := strings.Split(w.item["builtin-trust-store"], ","); len(got) != 2 {
		t.Errorf("builtin-trust-store = %q", w.item["builtin-trust-store"])
	}
	if st.ID != "certificate.settings" || st.Attributes["builtin_trust_store.#"] != "2" || st.Attributes["crl_store"] != "ram" {
		t.Errorf("state: %v", st.Attributes)
	}
}
