package routeros

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  RouterOS 7.23/7.24, `/ip firewall service-port print`: a fixed list of connection
  tracking helpers that can only be changed, not added or removed.
  [
    {".id": "*1", "name": "ftp", "ports": "21", "disabled": "false", "invalid": "false"},
    {".id": "*5", "name": "sip", "ports": "5060,5061", "sip-direct-media": "true", "sip-timeout": "1h", ...},
    ...
  ]
  Names: ftp, tftp, irc, h323, sip, pptp, rtsp, udplite, dccp, sctp.
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/328227/Services#Services-ServicePorts
func ResourceIpFirewallServicePort() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/ip/firewall/service-port"),
		MetaId:           PropId(Id),

		KeyDisabled: PropDisabledRw,
		"invalid": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "RouterOS marks the helper invalid (`I` flag).",
		},
		KeyName: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			Description: "Name of the helper (`ftp`, `tftp`, `irc`, `h323`, `sip`, `pptp`, `rtsp`, `udplite`, " +
				"`dccp`, `sctp`). The entry must exist on the router.",
		},
		"ports": {
			Type:             schema.TypeSet,
			Optional:         true,
			Elem:             &schema.Schema{Type: schema.TypeInt, ValidateFunc: validation.IntBetween(1, 65535)},
			Description:      "Ports the helper listens on (only for helpers that have ports).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"sip_direct_media": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "SIP: allow media streams directly between the endpoints.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"sip_timeout": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "SIP: timeout of the SIP connection tracking entries.",
			DiffSuppressFunc: TimeEqual,
		},
	}

	read := DefaultRead(resSchema)

	// The entry always exists: create is `set` of the entry found by name, delete only forgets it.
	createUpdate := func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		if d.Id() == "" {
			name := d.Get(KeyName).(string)
			res, err := ReadItemsFiltered([]string{"name=" + name}, resSchema[MetaResourcePath].Default.(string), m.(Client))
			if err != nil {
				return diag.FromErr(err)
			}
			if len(*res) != 1 {
				return diag.FromErr(fmt.Errorf("ip firewall service-port %q: found %d entries, want 1", name, len(*res)))
			}
			d.SetId((*res)[0].GetID(Id))
		}
		// `name` identifies the entry, `set` has no such argument.
		item, metadata := TerraformResourceDataToMikrotik(resSchema, d)
		delete(item, "name")
		if _, err := UpdateItem(&ItemId{Id, d.Id()}, metadata.Path, item, m.(Client)); err != nil {
			return diag.FromErr(err)
		}
		return read(ctx, d, m)
	}

	return &schema.Resource{
		Description: "Settings of one entry of `/ip firewall service-port` (connection tracking helpers). " +
			"The entries are built into RouterOS: creating the resource changes the existing entry, deleting " +
			"it only removes it from the state.",

		CreateContext: createUpdate,
		ReadContext:   read,
		UpdateContext: createUpdate,
		DeleteContext: DefaultSystemDelete(resSchema),

		Importer: &schema.ResourceImporter{
			StateContext: ImportStateCustomContext(resSchema),
		},

		Schema: resSchema,
	}
}
