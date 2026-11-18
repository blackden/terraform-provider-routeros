package routeros

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Schema helpers of this provider fork.

var reTunnelHostname = regexp.MustCompile(`^(?i:[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?)(?:\.(?i:[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?))*\.?$`)

// ValidateTunnelRemoteAddress accepts what RouterOS 7.23/7.24 takes as `remote-address` of
// GRE, IPIP and EoIP (`/console inspect`: Address[@VRF], Address = IPv4 | Hostname): an IP
// address or a DNS name, optionally followed by @vrf.
func ValidateTunnelRemoteAddress(v interface{}, k string) (ws []string, errs []error) {
	s, ok := v.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected type of %q to be string", k)}
	}
	addr := s
	if i := strings.LastIndex(s, "@"); i > 0 {
		addr = s[:i]
		if s[i+1:] == "" {
			return nil, []error{fmt.Errorf("%q: empty VRF name in %q", k, s)}
		}
	}
	if net.ParseIP(addr) != nil {
		return nil, nil
	}
	if len(addr) > 253 || !reTunnelHostname.MatchString(addr) || strings.Trim(addr, "0123456789.") == "" {
		return nil, []error{fmt.Errorf("expected %q to be an IP address or a DNS name, got %q", k, s)}
	}
	return nil, nil
}

var (
	// PropRemoteAddressHostRw is remote-address of GRE/IPIP/EoIP: an IP address or a DNS name.
	PropRemoteAddressHostRw = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Description: "Address of the remote end of the tunnel: an IP address or a DNS name (RouterOS resolves " +
			"the name itself, the resolved address is in `current_remote_address`).",
		ValidateFunc:     ValidateTunnelRemoteAddress,
		DiffSuppressFunc: AlwaysPresentNotUserProvided,
	}
	// PropCurrentRemoteAddressRo is the address the tunnel currently uses (resolved remote-address).
	PropCurrentRemoteAddressRo = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "Address the tunnel currently sends to (`remote_address` resolved by RouterOS).",
	}
)
