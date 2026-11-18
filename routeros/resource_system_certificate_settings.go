package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  RouterOS 7.24, `/certificate settings print`:
  {
    "builtin-trust-store": "default",
    "crl-download": "false",
    "crl-store": "ram",
    "crl-use": "false",
    "current-defaults": "fetch,mqtt,email,netwatch,container,lora,wiliot,dns,www,reverse-proxy"
  }
  `builtin-trust-store` can be a list, e.g. `fetch,dns`.
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/2555969/Certificates
func ResourceSystemCertificateSettings() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/certificate/settings"),
		MetaId:           PropId(Id),

		"builtin_trust_store": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Description: "Services that trust the built-in CA store of RouterOS: `all`, `default`, `untrusted` " +
				"or a list of services (`fetch`, `dns`, `container`, `www`, `reverse-proxy`, `ovpn`, ...; " +
				"`/console inspect request=completion path=certificate,settings,set,builtin-trust-store` " +
				"lists them for the RouterOS version).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"crl_download": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Download CRLs of the certificates automatically.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"crl_store": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Where downloaded CRLs are stored: `ram` or `system`.",
			ValidateFunc:     validation.StringInSlice([]string{"ram", "system"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"crl_use": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Check certificates against CRLs.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"current_defaults": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Services that `builtin_trust_store = [\"default\"]` stands for on this RouterOS version.",
		},
	}

	return &schema.Resource{
		Description: "Settings of `/certificate settings` (built-in trust store, CRL). Deleting the resource " +
			"only removes it from the state.",

		CreateContext: DefaultSystemCreate(resSchema),
		ReadContext:   DefaultSystemRead(resSchema),
		UpdateContext: DefaultSystemUpdate(resSchema),
		DeleteContext: DefaultSystemDelete(resSchema),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: resSchema,
	}
}
