package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// https://help.mikrotik.com/docs/display/ROS/Container#Container-Containerconfiguration
func ResourceContainerConfig() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/container/config"),
		MetaId:           PropId(Name),
		MetaSkipFields:   PropSkipFields("assumed_registry_url", "memory_current", "swap_current"),

		"registry_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "External registry url from where the container will be downloaded.",
		},
		"username": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Specifies the username for authentication (starting from ROS 7.8)",
		},
		"password": {
			Type:        schema.TypeString,
			Sensitive:   true,
			Optional:    true,
			Description: "Specifies the password for authentication (starting from ROS 7.8)",
		},
		"memory_high": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "RAM usage limit for all containers. RouterOS before 7.20 calls the parameter `ram-high`, " +
				"the provider maps the attribute to it. The attribute was named `ram_high` in the upstream provider.",
			DiffSuppressFunc: BytesEqual,
		},
		"memory_max": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Max RAM usage limit for all containers (RouterOS 7.20+).",
			DiffSuppressFunc: BytesEqual,
		},
		"swap_max": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Max swap usage limit for all containers (RouterOS 7.24: `unlimited` or a size, " +
				"container swap limits).",
			DiffSuppressFunc: BytesEqual,
		},
		"tmpdir": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Container extraction directory.",
		},
		"layer_dir": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Container layers directory.",
		},
	}

	return &schema.Resource{
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
