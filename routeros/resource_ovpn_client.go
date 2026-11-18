package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  {
    ".id": "*5C",
    "add-default-route": "false",
    "auth": "sha1",
    "certificate": "none",
    "cipher": "blowfish128",
  * "connect-to": "192.168.1.1",
    "disabled": "false",
    "disconnect-notify": "yes",
    "hw-crypto": "false",
    "mac-address": "02:E7:60:C6:40:EE",
    "max-mtu": "1500",
    "mode": "ip",
    "name": "ovpn-out1",
    "password": "",
    "port": "1194",
    "profile": "default",
    "protocol": "tcp",
    "route-nopull": "false",
    "running": "false",
    "tls-auth": "false",
    "tls-crypt": "true",
    "tls-version": "any",
    "use-peer-dns": "yes",
  * "user": "aaa",
    "verify-server-certificate": "false"
  }

  tls-auth/tls-crypt are read-only flags (print: Ta - TLS-AUTH, Tc - TLS-CRYPT; RouterOS 7.23 and 7.24):
  the key comes from `import-ovpn-configuration`, `set` has no such arguments. disconnect-notify is
  an argument of `set` (yes/no) on the same versions. With protocol=tcp RouterOS 7.24.6 accepts
  disconnect-notify in add/set but does not return it in print/get.
  import-ovpn-configuration on 7.24.6: in a test the <tls-crypt> block was imported only when the
  file had `key-direction`; without it the import succeeded and tls-crypt stayed unset. Whether
  `key-direction` alone is enough was not checked.
*/

// https://help.mikrotik.com/docs/display/ROS/OpenVPN
func ResourceOpenVPNClient() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/interface/ovpn-client"),
		MetaId:           PropId(Id),

		"add_default_route": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Whether to add OVPN remote address as a default route.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"auth": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Authentication methods that the server will accept.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateDiagFunc: ValidationMultiValInSlice([]string{"md5", "sha1", "null", "sha256", "sha512"}, false, false),
		},
		"certificate": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Name of the client certificate.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cipher": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      `Allowed ciphers.`,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateDiagFunc: ValidationMultiValInSlice([]string{
				"null", "aes128-cbc", "aes128-gcm", "aes192-cbc", "aes192-gcm", "aes256-cbc", "aes256-gcm", "blowfish128",
				// Backward compatibility with ROS v7.7
				"aes128", "aes192", "aes256",
			}, false, false),
		},
		KeyComment: PropCommentRw,
		"connect_to": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Remote address of the OVPN server.",
		},
		KeyDisabled: PropDisabledRw,
		"disconnect_notify": {
			Type:     schema.TypeBool,
			Optional: true,
			Description: "Send the server a notification when the client disconnects. Only with `protocol = udp`: " +
				"with `tcp` RouterOS accepts the value but does not return it.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"hw_crypto": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "",
		},
		KeyMacAddress: PropMacAddressRw(`Mac address of OVPN interface. Will be automatically generated if not specified.`, false),
		"max_mtu": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Maximum Transmission Unit. Max packet size that the OVPN interface will be able to send without packet fragmentation.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateFunc:     validation.IntBetween(64, 65535),
		},
		"mode": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Layer3 or layer2 tunnel mode (alternatively tun, tap)",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateFunc:     validation.StringInSlice([]string{"ip", "ethernet"}, false),
		},
		KeyName: PropName("Descriptive name of the interface."),
		"password": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Password used for authentication. " +
				"RouterOS returns the password masked (`*****`) when the provider's RouterOS user lacks the `sensitive` policy: then an absent or empty password is not managed, the router keeps its value, the mask is never sent back, and the password cannot be cleared through the configuration. With the `sensitive` policy RouterOS returns the real password: an absent or empty password is then a planned change of `password` (to null or `\"\"`, shown as `(sensitive value)`), and apply clears the password on the router.",
			Sensitive:        true,
			DiffSuppressFunc: MaskedSecretNotInConfig,
		},
		"port": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Port to connect to.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateFunc:     validation.IntBetween(1, 65535),
		},
		"profile": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Specifies which PPP profile configuration will be used when establishing the tunnel.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"protocol": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Indicates the protocol to use when connecting with the remote endpoint.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateFunc:     validation.StringInSlice([]string{"tcp", "udp"}, false),
		},
		"route_nopull": {
			Type:     schema.TypeBool,
			Optional: true,
			Description: "Specifies whether to allow the OVPN server to add routes to the OVPN client instance " +
				"routing table.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyRunning: PropRunningRo,
		"tls_auth": {
			Type:     schema.TypeBool,
			Computed: true,
			Description: "Whether the control channel uses a tls-auth key (set by `import-ovpn-configuration`, " +
				"read-only).",
		},
		"tls_crypt": {
			Type:     schema.TypeBool,
			Computed: true,
			Description: "Whether the control channel uses a tls-crypt key (set by `import-ovpn-configuration`, " +
				"read-only).",
		},
		"tls_version": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Specifies which TLS versions to allow.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
			ValidateFunc:     validation.StringInSlice([]string{"any", "only-1.2"}, false),
		},
		"use_peer_dns": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Whether to add DNS servers provided by the OVPN server to IP/DNS configuration.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"user": {
			Type:             schema.TypeString,
			Required:         true,
			Description:      "User name used for authentication.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"verify_server_certificate": {
			Type:     schema.TypeBool,
			Optional: true,
			Description: `Checks the certificates CN or SAN against the "connect-to" parameter. The IP or ` +
				`hostname must be present in the server's certificate.`,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
	}

	return &schema.Resource{
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
