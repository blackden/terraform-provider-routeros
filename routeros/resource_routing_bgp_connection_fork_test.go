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
			// An update carries only the changed fields (TestBgpConnectionUpdateOnlyChanged).
			bgpPayloadEqual(t, item, map[string]string{"comment": "test changed"})

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

// Raw REST responses of RouterOS 7.24.
var bgpConnectionTemplated = map[string]MikrotikItem{
	// test-bgp-tpl-out with templates=test-tpl. GET returns the effective values: as, hold-time,
	// keepalive-time, multihop, nexthop-choice, input.accept-nlri, input.allow-as, output.default-prepend
	// and output.no-early-cut come from the template (the connection's export does not list them).
	"*B": {
		".id": "*B", "as": "64498", "comment": "test-tpl-comment", "disabled": "true", "hold-time": "4m", "inactive": "true",
		"input.accept-nlri": "networks", "input.allow-as": "3", "input.filter": "filter-in",
		"instance": "bgp-instance-1", "keepalive-time": "1m", "local.address": "192.0.2.100", "local.role": "ebgp",
		"multihop": "true", "name": "test-bgp-tpl-out", "nexthop-choice": "force-self", "output.default-prepend": "1",
		"output.filter-chain": "discard", "output.network": "networks", "output.no-client-to-client-reflection": "true",
		"output.no-early-cut": "true", "remote.address": "192.0.2.2/32", "remote.as": "64502", "routing-table": "main",
		"templates": "test-tpl",
	},
	// test-bgp-u after `input { limit_process_routes_ipv4 = 80000 }` was applied.
	"*A": {
		".id": "*A", "as": "64496", "comment": "test-u1", "disabled": "true", "inactive": "true",
		"input.limit-process-routes-ipv4": "80000", "instance": "bgp-instance-1", "local.address": "",
		"local.role": "ebgp", "name": "test-bgp-u", "remote.address": "192.0.2.1", "remote.allowed-as": "",
		"remote.as": "64497", "routing-table": "main",
	},
}

func bgpWithoutBlock(cfg cty.Value, block string) cty.Value {
	attrs := cfg.AsValueMap()
	attrs[block] = cty.ListValEmpty(attrs[block].Type().ElementType())
	return cty.ObjectVal(attrs)
}

func bgpPayloadEqual(t *testing.T, got MikrotikItem, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("payload %v, want exactly %v", got, want)
		return
	}
	for k, v := range want {
		if gv, ok := got[k]; !ok || gv != v {
			t.Errorf("payload %v, want exactly %v", got, want)
			return
		}
	}
}

// An update sends only what changed. GET returns the values inherited from `templates`,
// a full-object PATCH wrote them onto the connection (a comment change pinned as, hold-time,
// input.allow-as... and wrote input.* = "" explicitly), breaking inheritance from the template.
func TestBgpConnectionUpdateOnlyChanged(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	c := newFakeClient(map[string][]MikrotikItem{"/routing/bgp/connection": {bgpConnectionTemplated["*B"]}})
	st, diags := fakeRead(t, r, "*B", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	cfg := bgpGeneratedConfig(t, r, st)
	if d := bgpDiff(t, r, st, cfg, c); d != nil && !d.Empty() {
		t.Fatalf("second plan is not empty: %v", d.Attributes)
	}

	attrs := cfg.AsValueMap()
	attrs["comment"] = cty.StringVal("test-u1")
	changed := cty.ObjectVal(attrs)
	bgpPayloadEqual(t, bgpFakeUpdate(t, r, st, changed, c), map[string]string{"comment": "test-u1"})

	limit := bgpSetNested(t, cfg, "input", "limit_process_routes_ipv4", cty.NumberIntVal(80000))
	bgpPayloadEqual(t, bgpFakeUpdate(t, r, st, limit, c), map[string]string{"input.limit-process-routes-ipv4": "80000"})

	// An override equal to the schema Default of a value inherited from the template is a change
	// and is sent (GET returns the inherited nexthop-choice=force-self, not an absent field).
	attrs = cfg.AsValueMap()
	attrs["nexthop_choice"] = cty.StringVal("default")
	bgpPayloadEqual(t, bgpFakeUpdate(t, r, st, cty.ObjectVal(attrs), c), map[string]string{"nexthop-choice": "default"})

	// Other fields removed from the configuration are sent with their zero value (REST ignores
	// `!comment`).
	attrs = cfg.AsValueMap()
	attrs["comment"] = cty.NullVal(cty.String)
	bgpPayloadEqual(t, bgpFakeUpdate(t, r, st, cty.ObjectVal(attrs), c), map[string]string{"comment": ""})

	// affinity and default_originate changed to the zero value are unset; ttl is not (REST ignores
	// `!local.ttl`, RouterOS 7.24.6 cannot clear it): 0 is sent and RouterOS reports the error.
	set := bgpSetNested(t, bgpSetNested(t, cfg, "local", "ttl", cty.NumberIntVal(0)), "input", "affinity", cty.StringVal(""))
	set = bgpSetNested(t, set, "output", "default_originate", cty.StringVal(""))
	stSet, _ := fakeRead(t, r, "*B", nil, newFakeClient(map[string][]MikrotikItem{"/routing/bgp/connection": {
		func() MikrotikItem {
			it := MikrotikItem{}
			for k, v := range bgpConnectionTemplated["*B"] {
				it[k] = v
			}
			it["local.ttl"], it["input.affinity"], it["output.default-originate"] = "64", "alone", "always"
			return it
		}(),
	}}))
	bgpPayloadEqual(t, bgpFakeUpdate(t, r, stSet, set, c),
		map[string]string{"local.ttl": "0", "!input.affinity": "", "!output.default-originate": ""})
}

// Removing the limit unsets it (RouterOS: `unset` removes the key, `0` is stored as an explicit value
// whose meaning is not documented). Removing the whole block used to panic in the serializer
// (index out of range [0] with length 0).
func TestBgpConnectionRemoveLimit(t *testing.T) {
	r := ResourceRoutingBgpConnection()
	unset := map[string]string{"!input.limit-process-routes-ipv4": ""}
	c := newFakeClient(map[string][]MikrotikItem{"/routing/bgp/connection": {bgpConnectionTemplated["*A"]}})
	st, diags := fakeRead(t, r, "*A", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	cfg := bgpGeneratedConfig(t, r, st)
	if st.Attributes["input.0.limit_process_routes_ipv4"] != "80000" {
		t.Fatalf("state limit = %q", st.Attributes["input.0.limit_process_routes_ipv4"])
	}

	t.Run("block removed", func(t *testing.T) {
		bgpPayloadEqual(t, bgpFakeUpdate(t, r, st, bgpWithoutBlock(cfg, "input"), c), unset)
	})
	t.Run("field removed", func(t *testing.T) {
		in := r.CoreConfigSchema().ImpliedType().AttributeType("input").ElementType()
		bgpPayloadEqual(t, bgpFakeUpdate(t, r, st,
			bgpSetNested(t, cfg, "input", "limit_process_routes_ipv4", cty.NullVal(in.AttributeType("limit_process_routes_ipv4"))), c), unset)
	})
	t.Run("set to 0", func(t *testing.T) {
		bgpPayloadEqual(t, bgpFakeUpdate(t, r, st,
			bgpSetNested(t, cfg, "input", "limit_process_routes_ipv4", cty.NumberIntVal(0)), c), unset)
	})
}

// The serializer does not panic when a nested block is removed from the configuration (upstream
// resources, DefaultUpdate): changed fields of the removed block are sent with their zero values,
// the way a removed top-level attribute is.
func TestSerializeRemovedNestedBlock(t *testing.T) {
	r := ResourceRoutingBgpTemplate()
	c := newFakeClient(map[string][]MikrotikItem{"/routing/bgp/template": {
		{".id": "*5", "as": "64498", "name": "test-tpl", "input.allow-as": "3"},
	}})
	st, diags := fakeRead(t, r, "*5", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	cfg := bgpGeneratedConfig(t, r, st)
	item := bgpFakeUpdate(t, r, st, bgpWithoutBlock(cfg, "input"), c)
	if item["input.allow-as"] != "0" {
		t.Errorf("input.allow-as = %q, want \"0\": %v", item["input.allow-as"], item)
	}
}
