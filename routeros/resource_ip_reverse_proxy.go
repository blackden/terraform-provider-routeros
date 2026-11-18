package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  /ip/reverse-proxy, RouterOS 7.2x. Field set per the 7.25 CLI reference
  (https://manual.mikrotik.com/docs/7.25/cli-reference/ip/reverse-proxy/):
  sni, ip-address, port, certificate, vrf; flags X (disabled), D (dynamic).
  TODO: capture a real REST response from a router (`/rest/ip/reverse-proxy`) and paste it here.
*/

// https://manual.mikrotik.com/docs/7.25/cli-reference/ip/reverse-proxy/
func ResourceIpReverseProxy() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/ip/reverse-proxy"),
		MetaId:           PropId(Id),

		"certificate": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Certificate the router presents to clients of this rule. With `none`, the certificate of " +
				"the `reverse-proxy` service in /ip/service is used; if that is also `none`, the TLS handshake fails.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		// TODO: `comment` is not listed in the 7.25 CLI reference; RouterOS menus usually accept it.
		// Verify on a router (`/ip/reverse-proxy/add ... comment=x`) before relying on it.
		KeyComment:  PropCommentRw,
		KeyDisabled: PropDisabledRw,
		KeyDynamic:  PropDynamicRo,
		"ip_address": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "IPv4 or IPv6 address of the server the requests are forwarded to (always as plain HTTP).",
			ValidateFunc: validation.IsIPAddress,
		},
		"port": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "TCP port of the server. The router always connects with plain HTTP, also when the port is 443.",
			ValidateFunc: validation.IntBetween(1, 65535),
		},
		"sni": {
			Type:     schema.TypeString,
			Required: true,
			Description: "Host name the rule serves. It is compared with the SNI the client sends and must match it " +
				"exactly; a wildcard such as `*.example.com` matches nothing.",
		},
		KeyVrf: PropVrfRw,
	}

	return &schema.Resource{
		Description: "Reverse proxy rules (`/ip/reverse-proxy`): each rule forwards the HTTPS requests for one host " +
			"name to an HTTP server. The listener is the `reverse-proxy` service in `/ip/service` (see " +
			"`routeros_ip_service`), its port is open only while at least one enabled rule exists.",
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
