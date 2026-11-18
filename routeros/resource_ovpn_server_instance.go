package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  /interface/ovpn-server/server, RouterOS 7.17+: a list of named OVPN servers
  (`add name=...`), no longer one system setting. On RouterOS 7.17+ the list
  is empty by default. `routeros_ovpn_server` treats the path as a singleton
  (DefaultSystem*) and cannot create anything there.

  Arguments of `add` on 7.23 (/console/inspect request=child):
  auth certificate cipher comment copy-from default-profile disabled
  enable-tun-ipv6 ipv6-prefix-len keepalive-timeout mac-address max-mtu mode
  name netmask port protocol push-routes push-routes-ipv6 redirect-gateway
  reneg-sec require-client-certificate tls-version tun-server-ipv6
  user-auth-method vrf
  Print flags: I - INACTIVE, X - DISABLED.
  TODO: paste a real REST response from a router that has a server.
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/2031655/OpenVPN
func ResourceOpenVPNServerInstance() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/interface/ovpn-server/server"),
		MetaId:           PropId(Id),

		"auth": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{"md5", "sha1", "null", "sha256", "sha512"}, false),
			},
			Description:      "Authentication (HMAC) methods that the server accepts.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"certificate": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Name of the certificate the server presents.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cipher": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					"null", "aes128-cbc", "aes128-gcm", "aes192-cbc", "aes192-gcm", "aes256-cbc", "aes256-gcm", "blowfish128",
				}, false),
			},
			Description:      "Allowed data channel ciphers.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyComment: PropCommentRw,
		"default_profile": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "PPP profile for users whose secret does not name one.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyDisabled: PropDisabledRw,
		"enable_tun_ipv6": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Whether IPv6 can be tunnelled in `ip` mode.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyInactive: PropInactiveRo,
		"ipv6_prefix_len": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Length of the IPv6 prefix of the server-side interface.",
			ValidateFunc:     validation.IntBetween(1, 128),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"keepalive_timeout": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Seconds of silence after which the router starts sending keepalives; a client that " +
				"stays silent for twice this time is disconnected. `disabled` turns keepalive off.",
			DiffSuppressFunc: TimeEqual,
		},
		"mac_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "MAC address of the server (generated when not set; used in `ethernet` mode).",
		},
		"max_mtu": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Largest packet the OVPN interface sends without fragmentation.",
			ValidateFunc:     validation.IntBetween(64, 65535),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"mode": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "`ip` (layer 3, tun) or `ethernet` (layer 2, tap).",
			ValidateFunc:     validation.StringInSlice([]string{"ip", "ethernet"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyName: PropName("Server name, e.g. `ovpn-server1`."),
		"netmask": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Prefix length pushed to the client with its address.",
			ValidateFunc:     validation.IntBetween(0, 32),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"port": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Listening port (TCP or UDP, see `protocol`).",
			ValidateFunc:     validation.IntBetween(1, 65535),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"protocol": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Transport: `tcp` or `udp`.",
			ValidateFunc:     validation.StringInSlice([]string{"tcp", "udp"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"push_routes": {
			Type:             schema.TypeSet,
			Optional:         true,
			Elem:             &schema.Schema{Type: schema.TypeString},
			Description:      "IPv4 routes pushed to the client, e.g. `192.168.0.0/24` or `192.168.0.0 255.255.255.0`.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"push_routes_ipv6": {
			Type:             schema.TypeSet,
			Optional:         true,
			Elem:             &schema.Schema{Type: schema.TypeString},
			Description:      "IPv6 routes pushed to the client.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"redirect_gateway": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{"def1", "disabled", "ipv6"}, false),
			},
			Description: "redirect-gateway flags sent to the client: `def1` (0.0.0.0/1 + 128.0.0.0/1 through the " +
				"tunnel), `ipv6` (2000::/4 + 3000::/4), or `disabled`.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"reneg_sec": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Renegotiate the data channel key every n seconds.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"require_client_certificate": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Accept only clients whose certificate is signed by the chain of the server certificate.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"tls_version": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Allowed TLS versions: `any` or `only-1.2`.",
			ValidateFunc:     validation.StringInSlice([]string{"any", "only-1.2"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"tun_server_ipv6": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "IPv6 prefix of the server-side interface in `ip` mode.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"user_auth_method": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "How the user name and password are checked (`pap`, `mschap2`).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyVrf: PropVrfRw,
	}

	return &schema.Resource{
		Description: "One named OVPN server (`/interface/ovpn-server/server`) on RouterOS 7.17+, where the menu is " +
			"a list. On older versions use `routeros_ovpn_server` (one system setting). Not to be confused with " +
			"`routeros_interface_ovpn_server` (static per-user interface bindings in `/interface/ovpn-server`).",
		CreateContext: DefaultCreate(resSchema),
		ReadContext:   DefaultRead(resSchema),
		UpdateContext: DefaultUpdate(resSchema),
		DeleteContext: DefaultDelete(resSchema),

		Importer: &schema.ResourceImporter{
			StateContext: ImportStateCustomContext(resSchema),
		},

		Schema: resSchema,
	}
}
