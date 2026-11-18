package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// https://help.mikrotik.com/docs/display/ROS/Container#Container-Addenvironmentvariablesandmounts(optional)
func ResourceContainerMounts() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/container/mounts"),
		MetaId:           PropId(Id),

		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID for the mount",
		},
		KeyDisabled: PropDisabledRw,
		"list": {
			Type:     schema.TypeString,
			Required: true,
			Description: "Mount list this mount is for. RouterOS before 7.21 calls the parameter `name`, " +
				"the provider maps the attribute to it.",
		},
		"mode": {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			Description:      "Mount mode (RouterOS 7.24 CLI reference: `rw`, `ro`, `rw,noexec`, `ro,noexec`).",
			ValidateFunc:     validation.StringInSlice([]string{"rw", "ro", "rw,noexec", "ro,noexec"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		// TODO: `read-only` comes from PR #910 (7.21),
		// the 7.24 CLI reference has `mode` instead. Which one does 7.23 accept? Check on a router with
		// `/container/mounts/print detail` before using either.
		"read_only": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Whether to mount read-only (RouterOS 7.21-7.2x, replaced by `mode` later).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"src": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Specifies source path of the mount, which points to a RouterOS location",
		},
		"dst": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Specifies destination path of the mount, which points to defined location in container",
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
