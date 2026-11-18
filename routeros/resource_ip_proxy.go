package routeros

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  RouterOS 7.23/7.24, `/ip proxy print`:
  {
    "always-from-cache": "false", "anonymous": "false", "cache-administrator": "webmaster",
    "cache-hit-dscp": "4", "cache-on-disk": "false", "cache-path": "web-proxy", "enabled": "false",
    "max-cache-object-size": "2048", "max-cache-size": "unlimited", "max-client-connections": "600",
    "max-fresh-time": "3d00:00:00", "max-server-connections": "600", "parent-proxy": "::",
    "parent-proxy-port": "0", "port": "8080", "serialize-connections": "false", "src-address": "::"
  }
*/

// https://help.mikrotik.com/docs/spaces/ROS/pages/132350014/Proxy
func ResourceIpProxy() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/ip/proxy"),
		MetaId:           PropId(Id),

		"always_from_cache": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Ignore client refresh requests if the content is considered fresh.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"anonymous": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Do not pass the client IP address to the remote server.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cache_administrator": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Administrator e-mail shown on proxy error pages.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cache_hit_dscp": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "DSCP value for packets served from the cache.",
			ValidateFunc:     validation.IntBetween(0, 63),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cache_on_disk": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Keep the cache on disk instead of RAM.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cache_path": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Directory of the on-disk cache.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		KeyEnabled: PropEnabled("Enable the web proxy."),
		"max_cache_object_size": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Largest object to cache, KiB (or `unlimited`).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"max_cache_size": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Maximum cache size, KiB (`none`, `unlimited` or a number).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"max_client_connections": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Maximum number of client connections.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"max_fresh_time": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Maximum time an object without an explicit expiry is considered fresh.",
			DiffSuppressFunc: TimeEqual,
		},
		"max_server_connections": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Maximum number of connections to servers.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"parent_proxy": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Address of the parent proxy (`::` or `0.0.0.0` - none).",
			ValidateFunc:     validation.IsIPAddress,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"parent_proxy_port": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Port of the parent proxy (0 - none).",
			ValidateFunc:     validation.IntBetween(0, 65535),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"port": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "TCP port the proxy listens on.",
			ValidateFunc:     validation.IntBetween(1, 65535),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"serialize_connections": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Do not open several connections to the server for several clients at once.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"src_address": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Source address of connections to servers and the parent proxy (`::` - any).",
			ValidateFunc:     validation.IsIPAddress,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
	}

	return &schema.Resource{
		Description: "Settings of the web proxy (`/ip proxy`). Access rules (`/ip proxy access`) and the cache " +
			"rules are not covered. Deleting the resource only removes it from the state.",

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
