package routeros

import (
	"context"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestMaskedSecretNotInConfig(t *testing.T) {
	cases := []struct {
		old, new string
		want     bool
	}{
		{"*****", "", true},
		{"", "", true},
		{"*****", "secret1", false},
		{"", "secret1", false},
		{"old", "new", false},
		{"old", "", false},
	}
	for _, c := range cases {
		if got := MaskedSecretNotInConfig("password", c.old, c.new, nil); got != c.want {
			t.Errorf("MaskedSecretNotInConfig(%q, %q) = %v, want %v", c.old, c.new, got, c.want)
		}
	}
}

// fakeUpdate plans and applies an update of an existing resource (state from fakeRead)
// with the given configuration, the way the SDK does it for `terraform apply`.
func fakeUpdate(t *testing.T, r *schema.Resource, st *terraform.InstanceState, cfg map[string]cty.Value, meta interface{}) {
	t.Helper()
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
	diff, err := r.SimpleDiff(context.Background(), st, conf, meta)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if diff == nil {
		t.Fatal("expected a diff")
	}
	diff.RawConfig = val
	if _, diags := r.Apply(context.Background(), st, diff, meta); diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
}

// The mask read from RouterOS is never written back when another field changes
// (the real password on the router would be overwritten with "*****"); a password
// set in the configuration is sent.
func TestMaskedSecret_NotSentOnUpdate(t *testing.T) {
	cases := []struct {
		name     string
		res      *schema.Resource
		path     string
		item     MikrotikItem
		cfg      map[string]cty.Value
		changeFn string
	}{
		{
			"ovpn_client", ResourceOpenVPNClient(), "/interface/ovpn-client",
			MikrotikItem{".id": "*5D", "connect-to": "192.0.2.10", "disabled": "true", "name": "t", "password": "*****", "user": "u", "comment": "old"},
			map[string]cty.Value{"name": cty.StringVal("t"), "connect_to": cty.StringVal("192.0.2.10"), "user": cty.StringVal("u"), "disabled": cty.True, "comment": cty.StringVal("new")},
			"*5D",
		},
		{
			"container_config", ResourceContainerConfig(), "/container/config",
			MikrotikItem{".id": "*0", "password": "*****", "tmpdir": "/old"},
			map[string]cty.Value{"tmpdir": cty.StringVal("/new")},
			".",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeClient(map[string][]MikrotikItem{tc.path: {tc.item}})
			st, diags := fakeRead(t, tc.res, tc.changeFn, nil, c)
			if diags.HasError() {
				t.Fatalf("read: %v", diags)
			}
			if st.Attributes["password"] != "*****" {
				t.Fatalf("state password = %q", st.Attributes["password"])
			}
			fakeUpdate(t, tc.res, st, tc.cfg, c)
			w, ok := c.lastWrite()
			if !ok {
				t.Fatal("no write request")
			}
			if _, has := w.item["password"]; has {
				t.Errorf("payload must not contain password: %v", w.item)
			}

			cfg := map[string]cty.Value{"password": cty.StringVal("x")}
			for k, v := range tc.cfg {
				cfg[k] = v
			}
			st, _ = fakeRead(t, tc.res, tc.changeFn, nil, c)
			fakeUpdate(t, tc.res, st, cfg, c)
			w, _ = c.lastWrite()
			if w.item["password"] != "x" {
				t.Errorf("payload must contain password=x: %v", w.item)
			}
		})
	}
}
