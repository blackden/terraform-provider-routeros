package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  RouterOS 7.24, `/system package update print` (example values):
  {
    "channel": "long-term",
    "check-certificate": "yes",
    "installed-version": "7.24.6",
    "ip-version": "auto",
    "latest-version": "7.24.6",
    "mode": "https",
    "status": "System is already up to date"
  }
  `latest-version` is present only after a successful check-for-updates.
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/328142/Packages#Packages-Updatingpackages
func ResourceSystemPackageUpdate() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/system/package/update"),
		MetaId:           PropId(Id),

		"channel": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Release channel the router checks for updates (`development`, `long-term`, `stable`, " +
				"`testing`). Changing the channel does not install anything by itself.",
			ValidateFunc:     validation.StringInSlice([]string{"development", "long-term", "stable", "testing"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"check_certificate": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Verify the certificate of the update server (`mode = \"https\"`).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"installed_version": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Currently installed RouterOS version.",
		},
		"ip_version": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "IP protocol used to reach the update server (`auto`, `ipv4`, `ipv6`).",
			ValidateFunc:     validation.StringInSlice([]string{"auto", "ipv4", "ipv6"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"latest_version": {
			Type:     schema.TypeString,
			Computed: true,
			Description: "Latest version available in the selected channel. Known only after a successful " +
				"`/system package update check-for-updates`.",
		},
		"mode": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Protocol used to download the update information and packages (`http`, `https`).",
			ValidateFunc:     validation.StringInSlice([]string{"http", "https"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Result of the last update check (as printed by RouterOS).",
		},
	}

	return &schema.Resource{
		Description: "Settings of `/system package update`: the update channel and how the router reaches " +
			"the MikroTik update servers. The resource only changes settings; it never checks for, " +
			"downloads or installs updates. Deleting it only removes it from the state.",

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
