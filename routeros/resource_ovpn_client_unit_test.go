package routeros

import (
	"strings"
	"testing"
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
