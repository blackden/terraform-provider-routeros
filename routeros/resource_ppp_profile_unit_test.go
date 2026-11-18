package routeros

import (
	"strings"
	"testing"
)

// Unit test (no router): RouterOS 7 (7.21 and 7.24) uses `required` for
// use-encryption/use-ipv6/use-mpls; `require` stays valid for existing configurations.
func TestPPPProfile_UseRequired(t *testing.T) {
	r := ResourcePPPProfile()
	for _, f := range []string{"use_encryption", "use_ipv6", "use_mpls"} {
		for _, v := range []string{"yes", "no", "default", "required", "require"} {
			if _, errs := r.Schema[f].ValidateFunc(v, f); len(errs) > 0 {
				t.Errorf("%v = %q rejected: %v", f, v, errs)
			}
		}
		if _, errs := r.Schema[f].ValidateFunc("always", f); len(errs) == 0 {
			t.Errorf("%v = always accepted", f)
		}
	}

	c := newFakeClient(map[string][]MikrotikItem{
		"/ppp/profile": {{".id": "*2", "name": "vpn", "use-encryption": "required", "use-ipv6": "yes",
			"use-mpls": "default", "use-compression": "default", "use-upnp": "default", "change-tcp-mss": "yes",
			"only-one": "default", "default": "false"}},
	})
	st, diags := fakeRead(t, r, "*2", nil, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() || st.Attributes["use_encryption"] != "required" {
		t.Errorf("read: %v, use_encryption = %q", diags, st.Attributes["use_encryption"])
	}
}
