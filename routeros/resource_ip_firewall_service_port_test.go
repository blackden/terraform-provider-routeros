package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit test (no router): /ip/firewall/service-port entries are changed by name, never created.
func TestIpFirewallServicePort(t *testing.T) {
	r := ResourceIpFirewallServicePort()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	c := newFakeClient(map[string][]MikrotikItem{
		"/ip/firewall/service-port": {
			{".id": "*1", "name": "ftp", "ports": "21", "disabled": "false", "invalid": "false"},
			{".id": "*5", "name": "sip", "ports": "5060,5061", "disabled": "false", "invalid": "false",
				"sip-direct-media": "true", "sip-timeout": "1h"},
		},
	})
	st, diags := fakeApply(t, r, map[string]cty.Value{"name": cty.StringVal("sip"), "disabled": cty.True}, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	w, _ := c.lastWrite()
	if w.method != crudUpdate || w.path != "/ip/firewall/service-port/*5" || w.item["disabled"] != "yes" {
		t.Errorf("request: %v %v %v", w.method, w.path, w.item)
	}
	if _, ok := w.item["name"]; ok {
		t.Errorf("name must not be sent (set has no such argument): %v", w.item)
	}
	if st.ID != "*5" || st.Attributes["ports.#"] != "2" || st.Attributes["sip_timeout"] != "1h" {
		t.Errorf("state: %v %v", st.ID, st.Attributes)
	}

	if _, diags := fakeApply(t, r, map[string]cty.Value{"name": cty.StringVal("gre")}, c); !diags.HasError() {
		t.Error("missing entry: no error")
	}

	// Import by name keeps `name` in the state (it is read back), so the config does not force a replacement.
	st, diags = fakeRead(t, r, "*1", nil, c)
	if diags.HasError() || st.Attributes["name"] != "ftp" || st.Attributes["ports.#"] != "1" {
		t.Errorf("read: %v %v", diags, st.Attributes)
	}
}
