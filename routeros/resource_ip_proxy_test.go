package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit test (no router): /ip/proxy singleton.
func TestIpProxy(t *testing.T) {
	r := ResourceIpProxy()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	sample := MikrotikItem{
		"always-from-cache": "false", "anonymous": "false", "cache-administrator": "webmaster",
		"cache-hit-dscp": "4", "cache-on-disk": "false", "cache-path": "web-proxy", "enabled": "false",
		"max-cache-object-size": "2048", "max-cache-size": "unlimited", "max-client-connections": "600",
		"max-fresh-time": "3d00:00:00", "max-server-connections": "600", "parent-proxy": "::",
		"parent-proxy-port": "0", "port": "8080", "serialize-connections": "false", "src-address": "::",
	}
	c := newFakeClient(map[string][]MikrotikItem{"/ip/proxy": {sample}})
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
	if w.path != "/ip/proxy/set" || len(w.item) != 1 || w.item["enabled"] != "no" {
		t.Errorf("set request: %v %v", w.path, w.item)
	}
	if st.ID != "ip.proxy" || st.Attributes["port"] != "8080" || st.Attributes["max_fresh_time"] != "3d00:00:00" {
		t.Errorf("state: %v", st.Attributes)
	}
}
