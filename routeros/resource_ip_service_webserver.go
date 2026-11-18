package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

/*
  RouterOS 7.23/7.24, `/ip service webserver print` (example values):
  {
    "acme-plain": "true", "crl-plain": "true", "graphs-plain": "false", "graphs-secure": "true",
    "index-plain": "false", "index-secure": "true", "rest-plain": "true", "rest-secure": "true",
    "scep-plain": "true", "webfig-plain": "false", "webfig-secure": "true"
  }
  *-plain is what the `www` service (HTTP) serves, *-secure - `www-ssl` (HTTPS).
*/

// ResourceIpServiceWebserver: /ip/service/webserver, what the www and www-ssl services serve.
func ResourceIpServiceWebserver() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/ip/service/webserver"),
		MetaId:           PropId(Id),

		"acme_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve ACME HTTP-01 challenge responses over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"crl_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve certificate revocation lists over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"graphs_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve graphs (`/tool graphing`) over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"graphs_secure": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve graphs (`/tool graphing`) over `www-ssl` (HTTPS).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"index_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve the start page over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"index_secure": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve the start page over `www-ssl` (HTTPS).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"rest_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve the REST API over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"rest_secure": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve the REST API over `www-ssl` (HTTPS).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"scep_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve the SCEP server over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"webfig_plain": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve WebFig over `www` (HTTP).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"webfig_secure": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Serve WebFig over `www-ssl` (HTTPS).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
	}

	return &schema.Resource{
		Description: "Settings of `/ip service webserver`: which applications the `www` (HTTP, `*_plain`) and " +
			"`www-ssl` (HTTPS, `*_secure`) services serve. Deleting the resource only removes it from the state.",

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
