package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit test (no router): /ip/service/webserver singleton.
func TestIpServiceWebserver(t *testing.T) {
	r := ResourceIpServiceWebserver()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	sample := MikrotikItem{
		"acme-plain": "true", "crl-plain": "true", "graphs-plain": "false", "graphs-secure": "true",
		"index-plain": "false", "index-secure": "true", "rest-plain": "true", "rest-secure": "true",
		"scep-plain": "true", "webfig-plain": "false", "webfig-secure": "true",
	}
	for k := range sample {
		if _, ok := r.Schema[KebabToSnake(k)]; !ok {
			t.Errorf("missing field %v", k)
		}
	}
	c := newFakeClient(map[string][]MikrotikItem{"/ip/service/webserver": {sample}})
	// Example: REST on www, WebFig and graphs off.
	st, diags := fakeApply(t, r, map[string]cty.Value{
		"webfig_plain": cty.False, "graphs_plain": cty.False, "rest_plain": cty.True,
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
	if w.path != "/ip/service/webserver/set" || len(w.item) != 3 ||
		w.item["webfig-plain"] != "no" || w.item["graphs-plain"] != "no" || w.item["rest-plain"] != "yes" {
		t.Errorf("set request: %v %v", w.path, w.item)
	}
	if st.ID != "ip.service.webserver" || st.Attributes["rest_secure"] != "true" {
		t.Errorf("state: %v", st.Attributes)
	}
}
