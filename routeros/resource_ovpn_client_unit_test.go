package routeros

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Unit test (no router): an ovpn-client on RouterOS 7.23/7.24 is read without
// "Field 'tls_crypt' not found in the schema" and the flags land in the state.
func TestOpenVPNClient_ReadTlsFlags(t *testing.T) {
	r := ResourceOpenVPNClient()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	for _, f := range []string{"tls_auth", "tls_crypt"} {
		if s := r.Schema[f]; s == nil || !s.Computed || s.Optional {
			t.Errorf("%v must be read-only", f)
		}
	}
	if s := r.Schema["disconnect_notify"]; s == nil || !s.Optional {
		t.Error("disconnect_notify must be an argument")
	}

	c := newFakeClient(map[string][]MikrotikItem{
		"/interface/ovpn-client": {{
			".id": "*1", "add-default-route": "false", "auth": "sha256", "certificate": "client-cert",
			"cipher": "aes256-gcm", "connect-to": "192.0.2.1", "disabled": "false", "disconnect-notify": "yes",
			"hw-crypto": "false", "mac-address": "02:00:00:00:00:01", "max-mtu": "1500", "mode": "ethernet",
			"name": "ovpn-test", "port": "1194", "profile": "default", "protocol": "tcp", "route-nopull": "true",
			"running": "true", "tls-auth": "false", "tls-crypt": "true", "tls-version": "any",
			"use-peer-dns": "true", "user": "ovpnuser", "verify-server-certificate": "false",
		}},
	})
	st, diags := fakeRead(t, r, "*1", nil, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["tls_crypt"] != "true" || st.Attributes["tls_auth"] != "false" {
		t.Errorf("tls_crypt = %q, tls_auth = %q", st.Attributes["tls_crypt"], st.Attributes["tls_auth"])
	}
	if st.Attributes["disconnect_notify"] != "true" {
		t.Errorf("disconnect_notify = %q", st.Attributes["disconnect_notify"])
	}
}

// RouterOS returns a write-only password as "*****". Without `password` in the
// configuration there must be no diff; a password set in the configuration must
// still produce one.
func TestOpenVPNClient_PasswordMaskDiff(t *testing.T) {
	r := ResourceOpenVPNClient()
	c := newFakeClient(map[string][]MikrotikItem{
		"/interface/ovpn-client": {{
			".id": "*5D", "connect-to": "192.0.2.10", "disabled": "true", "name": "test-ovpn",
			"password": "*****", "user": "u",
		}},
	})
	st, diags := fakeRead(t, r, "*5D", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["password"] != "*****" {
		t.Fatalf("password in state = %q, want the mask", st.Attributes["password"])
	}

	// The same way as fakeApply: the diff sees the real configuration (GetRawConfig).
	diffFor := func(cfg map[string]cty.Value) bool {
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
		d, err := r.SimpleDiff(context.Background(), st, conf, c)
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		if d == nil {
			return false
		}
		_, ok := d.Attributes["password"]
		return ok
	}
	base := map[string]cty.Value{
		"name": cty.StringVal("test-ovpn"), "connect_to": cty.StringVal("192.0.2.10"), "user": cty.StringVal("u"),
	}
	if diffFor(base) {
		t.Error("password absent from the configuration must not produce a diff")
	}
	withPw := map[string]cty.Value{}
	for k, v := range base {
		withPw[k] = v
	}
	withPw["password"] = cty.StringVal("x")
	if !diffFor(withPw) {
		t.Error("password = \"x\" in the configuration must produce a diff")
	}
}
