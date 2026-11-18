package routeros

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Unit tests (no router): `tofu plan -generate-config-out` for an imported
// routeros_routing_bgp_connection failed its own validation. RouterOS leaves unset fields of the
// nested blocks out of the response, SDKv2 puts zero values into the state (ttl 0,
// default_originate "", redistribute "") and generated configuration carries them over.

// Raw `GET /rest/routing/bgp/connection?name=...` responses of RouterOS 7.24.
var bgpConnectionRaw = map[string]MikrotikItem{
	// Disabled test connection: no output.* fields at all.
	"*5": {
		".id": "*5", "as": "64496", "comment": "test", "disabled": "true", "inactive": "true",
		"instance": "bgp-instance-1", "local.role": "ebgp", "name": "test-bgp",
		"remote.address": "192.0.2.1", "remote.as": "64497", "routing-table": "main",
	},
	// test-bgp-out: an output block without default-originate and redistribute, an input block.
	"*7": {
		".id": "*7", "as": "64500", "comment": "test", "disabled": "true", "hold-time": "4m",
		"inactive": "true", "input.filter": "filter-in", "instance": "bgp-instance-1", "keepalive-time": "4m",
		"local.role": "ebgp", "multihop": "true", "name": "test-bgp-out", "output.filter-chain": "discard",
		"output.network": "networks", "output.no-client-to-client-reflection": "true",
		"remote.address": "192.0.2.1/32", "remote.as": "64501", "routing-table": "main",
	},
}

// bgpGeneratedConfig builds the configuration that `-generate-config-out` writes from a state:
// arguments with their state values; read-only, deprecated attributes and fork metadata left out
// (null), as OpenTofu 1.13 does.
func bgpGeneratedConfig(t *testing.T, r *schema.Resource, st *terraform.InstanceState) cty.Value {
	t.Helper()
	ty := r.CoreConfigSchema().ImpliedType()
	val, err := st.AttrsAsObjectValue(ty)
	if err != nil {
		t.Fatalf("state -> cty: %v", err)
	}
	attrs := val.AsValueMap()
	for name, s := range r.Schema {
		if strings.HasPrefix(name, "___") || (s.Computed && !s.Optional) || s.Deprecated != "" {
			attrs[name] = cty.NullVal(ty.AttributeType(name))
			continue
		}
		elem, ok := s.Elem.(*schema.Resource)
		if !ok || attrs[name].IsNull() || attrs[name].LengthInt() == 0 {
			continue
		}
		blocks := attrs[name].AsValueSlice()
		for i, b := range blocks {
			fields := b.AsValueMap()
			for f, fs := range elem.Schema {
				if (fs.Computed && !fs.Optional) || fs.Deprecated != "" {
					fields[f] = cty.NullVal(b.Type().AttributeType(f))
				}
			}
			blocks[i] = cty.ObjectVal(fields)
		}
		attrs[name] = cty.ListVal(blocks)
	}
	attrs["id"] = cty.NullVal(cty.String)
	return cty.ObjectVal(attrs)
}

// bgpSetNested replaces block.0.field in a configuration value.
func bgpSetNested(t *testing.T, cfg cty.Value, block, field string, v cty.Value) cty.Value {
	t.Helper()
	attrs := cfg.AsValueMap()
	blocks := attrs[block].AsValueSlice()
	if len(blocks) != 1 {
		t.Fatalf("%v: %d blocks in the configuration", block, len(blocks))
	}
	b := blocks[0].AsValueMap()
	b[field] = v
	attrs[block] = cty.ListVal([]cty.Value{cty.ObjectVal(b)})
	return cty.ObjectVal(attrs)
}

type bgpField struct {
	block, field string
	v            cty.Value
}

func bgpDiff(t *testing.T, r *schema.Resource, st *terraform.InstanceState, cfg cty.Value, meta interface{}) *terraform.InstanceDiff {
	t.Helper()
	conf := terraform.NewResourceConfigShimmed(cfg, r.CoreConfigSchema())
	conf.CtyValue = cfg
	d, err := r.SimpleDiff(context.Background(), st, conf, meta)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	return d
}

// bgpFakeUpdate plans and applies cfg over the state st and returns the payload sent to RouterOS.
func bgpFakeUpdate(t *testing.T, r *schema.Resource, st *terraform.InstanceState, cfg cty.Value, c *fakeClient) MikrotikItem {
	t.Helper()
	d := bgpDiff(t, r, st, cfg, c)
	if d == nil || d.Empty() {
		t.Fatalf("update: empty diff")
	}
	d.RawConfig = cfg
	n := len(c.requests)
	if _, diags := r.Apply(context.Background(), st, d, c); diags.HasError() {
		t.Fatalf("update: %v", diags)
	}
	if len(c.requests) == n {
		t.Fatalf("update: nothing sent")
	}
	w, _ := c.lastWrite()
	if w.method != crudUpdate {
		t.Fatalf("update: method %v", w.method)
	}
	return w.item
}

func bgpValidate(r *schema.Resource, cfg cty.Value) diag.Diagnostics {
	conf := terraform.NewResourceConfigShimmed(cfg, r.CoreConfigSchema())
	conf.CtyValue = cfg
	return r.Validate(conf)
}

func TestBgpConnectionImportZeroFields(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	c := newFakeClient(map[string][]MikrotikItem{
		"/routing/bgp/connection": {bgpConnectionRaw["*5"], bgpConnectionRaw["*7"]},
	})

	for _, id := range []string{"*5", "*7"} {
		t.Run(id, func(t *testing.T) {
			st, diags := fakeRead(t, r, id, nil, c)
			noSchemaWarnings(t, diagSummaries(diags))
			if diags.HasError() {
				t.Fatalf("read: %v", diags)
			}
			// What the router left out is a zero value in the state.
			for _, k := range []string{"local.0.ttl", "remote.0.ttl"} {
				if st.Attributes[k] != "0" {
					t.Errorf("%v = %q, want \"0\"", k, st.Attributes[k])
				}
			}
			if id == "*7" && st.Attributes["output.0.default_originate"] != "" {
				t.Errorf("output.0.default_originate = %q", st.Attributes["output.0.default_originate"])
			}

			// Generated configuration passes the provider's validation...
			cfg := bgpGeneratedConfig(t, r, st)
			if diags := bgpValidate(r, cfg); diags.HasError() {
				t.Fatalf("generated configuration is rejected: %v", diagSummaries(diags))
			}
			// ...and the next plan is empty.
			if d := bgpDiff(t, r, st, cfg, c); d != nil && !d.Empty() {
				for k, a := range d.Attributes {
					t.Errorf("second plan: %v: %q -> %q", k, a.Old, a.New)
				}
			}

			// What the user writes is still checked.
			bad := []bgpField{
				{"local", "ttl", cty.NumberIntVal(300)},
				{"remote", "ttl", cty.NumberIntVal(256)},
				{"local", "ttl", cty.NumberIntVal(-1)},
			}
			if id == "*7" {
				bad = append(bad,
					bgpField{"output", "default_originate", cty.StringVal("sometimes")},
					bgpField{"output", "redistribute", cty.StringVal("static,bogus")},
				)
			}
			for _, b := range bad {
				if diags := bgpValidate(r, bgpSetNested(t, cfg, b.block, b.field, b.v)); !diags.HasError() {
					t.Errorf("%v.%v = %v: no validation error", b.block, b.field, b.v.GoString())
				}
			}
			good := []bgpField{
				{"local", "ttl", cty.NumberIntVal(1)},
				{"remote", "ttl", cty.NumberIntVal(255)},
			}
			if id == "*7" {
				good = append(good,
					bgpField{"output", "default_originate", cty.StringVal("if-installed")},
					bgpField{"output", "redistribute", cty.StringVal("connected,static")},
				)
			}
			for _, g := range good {
				if diags := bgpValidate(r, bgpSetNested(t, cfg, g.block, g.field, g.v)); diags.HasError() {
					t.Errorf("%v.%v = %v: %v", g.block, g.field, g.v.GoString(), diagSummaries(diags))
				}
			}
		})
	}
}

// An unset value read back as 0/"" is not a change, a real value against it is.
func TestBgpConnectionZeroFieldsDiffSuppress(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	nested := func(block, field string) *schema.Schema {
		return r.Schema[block].Elem.(*schema.Resource).Schema[field]
	}
	cases := []struct {
		block, field, old, new string
		suppress               bool
	}{
		{"local", "ttl", "", "0", true},
		{"local", "ttl", "0", "", true},
		{"local", "ttl", "0", "0", true},
		{"local", "ttl", "", "64", false},
		{"local", "ttl", "0", "64", false},
		{"local", "ttl", "64", "0", false},
		{"local", "ttl", "64", "", false},
		{"remote", "ttl", "", "0", true},
		{"remote", "ttl", "1", "0", false},
		{"output", "default_originate", "", "", true},
		{"output", "default_originate", "", "never", false},
		{"output", "default_originate", "always", "", false},
	}
	for _, tc := range cases {
		s := nested(tc.block, tc.field)
		if s.DiffSuppressFunc == nil {
			t.Fatalf("%v.%v: no DiffSuppressFunc", tc.block, tc.field)
		}
		k := tc.block + ".0." + tc.field
		if got := s.DiffSuppressFunc(k, tc.old, tc.new, nil); got != tc.suppress {
			t.Errorf("%v %q -> %q: suppress = %v, want %v", k, tc.old, tc.new, got, tc.suppress)
		}
	}
}

// The generated configuration can be applied: an update sends neither the zero values of unset
// fields nor the defaults RouterOS left out; a value the user writes is sent.
func TestBgpConnectionUpdatePayload(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	unset := []string{"local.ttl", "remote.ttl", "output.default-originate", "output.redistribute",
		"add-path-out", "afi", "address-families", "connect", "listen", "nexthop-choice", "local.port", "remote.port",
		"input.affinity", "output.affinity", "input.limit-process-routes-ipv4", "input.limit-process-routes-ipv6",
		// deprecated, not in the generated configuration; RouterOS 7.24 rejects ignore-as-path-len=no
		"input.accept-unknown", "input.ignore-as-path-len"}

	for _, id := range []string{"*5", "*7"} {
		t.Run(id, func(t *testing.T) {
			c := newFakeClient(map[string][]MikrotikItem{
				"/routing/bgp/connection": {bgpConnectionRaw["*5"], bgpConnectionRaw["*7"]},
			})
			st, diags := fakeRead(t, r, id, nil, c)
			if diags.HasError() {
				t.Fatalf("read: %v", diags)
			}
			cfg := bgpGeneratedConfig(t, r, st)
			attrs := cfg.AsValueMap()
			attrs["comment"] = cty.StringVal("test changed")
			changed := cty.ObjectVal(attrs)

			item := bgpFakeUpdate(t, r, st, changed, c)
			t.Logf("payload: %v", item)
			if item["comment"] != "test changed" {
				t.Errorf("comment = %q", item["comment"])
			}
			for _, k := range unset {
				for _, kk := range []string{k, "!" + k} {
					if v, ok := item[kk]; ok {
						t.Errorf("%v = %q is sent", kk, v)
					}
				}
			}
			if id == "*7" && item["keepalive-time"] != "4m" {
				t.Errorf("keepalive-time = %q, want 4m", item["keepalive-time"])
			}

			// A real value is sent.
			withTTL := bgpSetNested(t, bgpSetNested(t, changed, "local", "ttl", cty.NumberIntVal(64)),
				"remote", "port", cty.NumberIntVal(1179))
			attrs = withTTL.AsValueMap()
			attrs["connect"] = cty.False
			item = bgpFakeUpdate(t, r, st, cty.ObjectVal(attrs), c)
			for k, want := range map[string]string{"local.ttl": "64", "remote.port": "1179", "connect": "no"} {
				if item[k] != want {
					t.Errorf("%v = %q, want %q", k, item[k], want)
				}
			}
		})
	}
}

// Create is unchanged: schema defaults are sent; a generated `ttl = 0` is not.
func TestBgpConnectionCreatePayload(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	c := newFakeClient(nil)
	lt := r.CoreConfigSchema().ImpliedType().AttributeType("local").ElementType()
	loc := map[string]cty.Value{}
	for n, at := range lt.AttributeTypes() {
		loc[n] = cty.NullVal(at)
	}
	loc["role"] = cty.StringVal("ebgp")
	loc["ttl"] = cty.NumberIntVal(0)
	fakeApply(t, r, map[string]cty.Value{"name": cty.StringVal("test"), "as": cty.StringVal("64500"),
		"instance": cty.StringVal("i"), "local": cty.ListVal([]cty.Value{cty.ObjectVal(loc)})}, c)
	w, ok := c.lastWrite()
	if !ok || w.method != crudCreate {
		t.Fatalf("no create: %v", c.requests)
	}
	for k, want := range map[string]string{"add-path-out": "none", "afi": "ip", "connect": "yes", "listen": "yes",
		"keepalive-time": "3m", "nexthop-choice": "default", "local.port": "179", "local.role": "ebgp"} {
		if w.item[k] != want {
			t.Errorf("%v = %q, want %q", k, w.item[k], want)
		}
	}
	if v, ok := w.item["local.ttl"]; ok {
		t.Errorf("local.ttl = %q is sent", v)
	}
}
