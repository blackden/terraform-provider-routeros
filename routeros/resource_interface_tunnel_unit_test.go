package routeros

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Unit tests (no router): remote-address of GRE/IPIP/EoIP may be a DNS name
// (RouterOS 7.23/7.24: Address[@VRF], Address = IPv4 | Hostname).

func TestValidateTunnelRemoteAddress(t *testing.T) {
	for v, ok := range map[string]bool{
		"198.51.100.7":         true,
		"router.example":       true,
		"peer.example.net.":    true,
		"peer-vpn":             true,
		"192.0.2.1@main":       true,
		"router.example@x":     true,
		"2001:db8::1":          true, // was accepted before (validation.IsIPAddress)
		"":                     false,
		"300.1.1.1":            false,
		"bad name.example.com": false,
		"-bad.example.com":     false,
		"router.example@":      false,
	} {
		_, errs := ValidateTunnelRemoteAddress(v, "remote_address")
		if (len(errs) == 0) != ok {
			t.Errorf("%q: errors %v, want ok=%v", v, errs, ok)
		}
	}
}

func TestInterfaceTunnels_RemoteHostname(t *testing.T) {
	for name, r := range map[string]*schema.Resource{
		"gre":  ResourceInterfaceGre(),
		"ipip": ResourceInterfaceIPIP(),
		"eoip": ResourceInterfaceEoip(),
	} {
		if err := r.InternalValidate(nil, true); err != nil {
			t.Fatalf("%v: InternalValidate: %v", name, err)
		}
		if s := r.Schema["current_remote_address"]; s == nil || !s.Computed || s.Optional {
			t.Errorf("%v: current_remote_address must be read-only", name)
		}
		if _, errs := r.Schema[KeyRemoteAddress].ValidateFunc("router.example", KeyRemoteAddress); len(errs) > 0 {
			t.Errorf("%v: DNS name rejected: %v", name, errs)
		}
	}
	gre := ResourceInterfaceGre()

	// gre1 with a DNS name in remote-address and current-remote-address resolved from it.
	c := newFakeClient(map[string][]MikrotikItem{
		"/interface/gre": {{
			".id": "*1", "actual-mtu": "1476", "allow-fast-path": "false", "clamp-tcp-mss": "true",
			"current-remote-address": "198.51.100.7", "disabled": "false", "dont-fragment": "no",
			"dscp": "inherit", "keepalive": "10s,10", "l2mtu": "65535", "local-address": "0.0.0.0", "mtu": "auto",
			"name": "gre1", "remote-address": "router.example", "running": "true",
		}},
	})
	st, diags := fakeRead(t, gre, "*1", nil, c)
	for _, s := range diagSummaries(diags) {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["remote_address"] != "router.example" || st.Attributes["current_remote_address"] != "198.51.100.7" {
		t.Errorf("remote_address = %q, current_remote_address = %q", st.Attributes["remote_address"], st.Attributes["current_remote_address"])
	}
}
