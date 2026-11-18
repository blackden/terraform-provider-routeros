package routeros

import (
	"context"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Unit test (no router): an imported mangle rule without add-*-to-address-list,
// for which RouterOS returns neither address-list-timeout nor log, plans no changes.
func TestIpFirewallMangle_NoDefaultDiff(t *testing.T) {
	r := ResourceIPFirewallMangle()
	c := newFakeClient(map[string][]MikrotikItem{
		"/ip/firewall/mangle": {{
			".id": "*1", "action": "mark-routing", "chain": "output", "comment": "test",
			"disabled": "false", "dst-address": "198.51.100.0/24", "dynamic": "false", "invalid": "false",
			"new-routing-mark": "to-isp2", "passthrough": "false",
		}},
	})
	st, diags := fakeRead(t, r, "*1", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}

	plan := func(cfg map[string]cty.Value) *terraform.InstanceDiff {
		ty := r.CoreConfigSchema().ImpliedType()
		attrs := map[string]cty.Value{}
		for name, at := range ty.AttributeTypes() {
			if v, ok := cfg[name]; ok {
				attrs[name] = v
			} else {
				attrs[name] = cty.NullVal(at)
			}
		}
		val := cty.ObjectVal(attrs)
		conf := terraform.NewResourceConfigShimmed(val, r.CoreConfigSchema())
		conf.CtyValue = val
		st.RawConfig = val
		diff, err := r.SimpleDiff(context.Background(), st, conf, c)
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		return diff
	}
	base := map[string]cty.Value{
		"action": cty.StringVal("mark-routing"), "chain": cty.StringVal("output"),
		"comment": cty.StringVal("test"), "dst_address": cty.StringVal("198.51.100.0/24"),
		"new_routing_mark": cty.StringVal("to-isp2"), "passthrough": cty.False,
	}
	if diff := plan(base); diff != nil && len(diff.Attributes) > 0 {
		for k, a := range diff.Attributes {
			t.Errorf("unexpected diff %v: %q -> %q", k, a.Old, a.New)
		}
	}

	// A configured timeout is still compared.
	withTimeout := map[string]cty.Value{"address_list_timeout": cty.StringVal("10m")}
	for k, v := range base {
		withTimeout[k] = v
	}
	if diff := plan(withTimeout); diff == nil || diff.Attributes["address_list_timeout"] == nil {
		t.Error("address_list_timeout = 10m: no diff against an empty value")
	}
}

func TestAddressListTimeoutEqual(t *testing.T) {
	for _, tt := range []struct {
		old, new string
		eq       bool
	}{
		{"10m", "10m", true},
		{"00:10:00", "10m", true},
		{"none-dynamic", "none-dynamic", true},
		{"none-dynamic", "none-static", false},
		{"none-dynamic", "10m", false},
		{"", "10m", false},
		{"1d", "10m", false},
	} {
		d := ResourceIPFirewallMangle().TestResourceData()
		if got := AddressListTimeoutEqual("address_list_timeout", tt.old, tt.new, d); got != tt.eq {
			t.Errorf("%q vs %q: %v, want %v", tt.old, tt.new, got, tt.eq)
		}
	}
}
