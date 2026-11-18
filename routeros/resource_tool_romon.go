package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  RouterOS 7.23/7.24, `/tool romon print`:
  {
    "enabled": "false",
    "id": "00:00:00:00:00:00",
    "secrets": ""
  }
  id 00:00:00:00:00:00 means "use a MAC address of the router".
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/8978569/RoMON
func ResourceToolRomon() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/tool/romon"),
		MetaId:           PropId(Id),
		// `id` is reserved by Terraform.
		MetaTransformSet: PropTransformSet("romon_id: id"),

		KeyEnabled: PropEnabled("Enable RoMON (Router Management Overlay Network)."),
		"romon_id": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			Description: "RoMON ID of the router (`id` in RouterOS, a MAC address); `00:00:00:00:00:00` makes " +
				"RouterOS use one of its interface MAC addresses.",
			ValidateFunc:     validation.IsMACAddress,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"secrets": {
			Type:      schema.TypeSet,
			Optional:  true,
			Sensitive: true,
			Elem:      &schema.Schema{Type: schema.TypeString},
			Description: "Secrets used to authenticate RoMON packets; an empty set disables " +
				"authentication.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
	}

	return &schema.Resource{
		Description: "Settings of `/tool romon`. Per-interface settings (`/tool romon port`) are not covered. " +
			"Deleting the resource only removes it from the state.",

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
