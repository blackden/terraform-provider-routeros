package routeros

import "testing"

// Unit test (no router): the 7.17+ list menu needs a name and list CRUD, not the
// singleton set/get of routeros_ovpn_server.
func TestOpenVPNServerInstance_Schema(t *testing.T) {
	r := ResourceOpenVPNServerInstance()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	if got := r.Schema[MetaResourcePath].Default; got != "/interface/ovpn-server/server" {
		t.Errorf("resource path = %v", got)
	}
	if !r.Schema[KeyName].Required {
		t.Error("name must be required")
	}
	if _, ok := r.Schema[MetaSkipFields]; ok {
		t.Error("no fields may be skipped: name/disabled/vrf/user_auth_method are real arguments here")
	}
	for _, f := range []string{KeyDisabled, KeyComment, KeyVrf, "user_auth_method", "push_routes_ipv6", "require_client_certificate"} {
		if _, ok := r.Schema[f]; !ok {
			t.Errorf("missing field %v", f)
		}
	}
	if _, ok := r.Schema[KeyEnabled]; ok {
		t.Error("list items use disabled, not enabled")
	}
	if _, errs := r.Schema["protocol"].ValidateFunc("udp", "protocol"); len(errs) > 0 {
		t.Errorf("udp rejected: %v", errs)
	}
}
