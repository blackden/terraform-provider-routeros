package routeros

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

/*
  {
    ".id": "*3",
    "arch": "amd64",
    "interface": "veth1",
    "mounts": "",
    "name": "76e7fc0c-e2c0-4b8c-b3b0-4c985c496a90",
    "os": "linux",
    "repo": "registry-1.docker.io/adguard/adguardhome:latest",
    "root-dir": "sata4-part1/docker/adg",
    "status": "stopped",
    "workdir": "/opt/adguardhome/work"
  }
*/

// https://help.mikrotik.com/docs/display/ROS/Container#Container-Properties
func ResourceContainer() *schema.Resource {
	resSchema := map[string]*schema.Schema{
		MetaResourcePath: PropResourcePath("/container"),
		MetaId:           PropId(Id),
		// State flags (running ... starting_with_healthcheck) are read by resRead/containerState;
		// default_* (image defaults), env_current, container_size, restart_count, healthcheck_status are
		// read-only values RouterOS 7.23/7.24 prints with `print detail`.
		MetaSkipFields: PropSkipFields("running", "starting", "stopped", "passed_devs", "stopping", "extracting", "config_json",
			"healthy", "unhealthy", "starting_with_healthcheck",
			"container_size", "env_current", "healthcheck_status", "restart_count",
			"default_cmd", "default_dns", "default_entrypoint", "default_shell", "default_stop_signal", "default_user",
			"default_workdir", "default_healthcheck_cmd", "default_healthcheck_interval", "default_healthcheck_retries",
			"default_healthcheck_start_interval", "default_healthcheck_start_period", "default_healthcheck_timeout"),

		"arch": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The architecture of the container image",
		},
		"auto_restart_interval": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Specify an interval at which Container will be restarted on Container failure.",
			DiffSuppressFunc: TimeEqual,
		},
		"check_certificate": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Enables trust chain validation from local certificate store.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"cmd": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The main purpose of a CMD is to provide defaults for an executing container. These defaults can include an executable, or they can omit the executable, in which case you must specify an ENTRYPOINT instruction as well.",
		},
		"cpu_list": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"cpu_usage": {
			Type:        schema.TypeFloat,
			Computed:    true,
			Description: "Current CPU usage percentage",
		},
		KeyComment: PropCommentRw,
		"devices": {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "Passes through physical device to the container.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"dns": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Set custom DNS servers",
		},
		"domain_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Container NIS domain name",
		},
		"entrypoint": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "An ENTRYPOINT allows to specify executable to run when starting container. Example: /bin/sh",
		},
		"env": {
			Type:        schema.TypeSet,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "list of environmental variables (in the form `key=val`) to be used with container",
		},
		"envlists": {
			Type:     schema.TypeSet,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
			Description: "Lists of environment variables (configured under /container/envs) to be used with the container. " +
				"RouterOS before 7.20 has a single `envlist` parameter, the provider maps the attribute to it.",
		},
		"file": {
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "container *tar.gz tarball if the container is imported from a file",
			ExactlyOneOf: []string{"file", "remote_image"},
		},
		"hostname": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Container host name",
		},
		"hosts": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"image_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "SHA of image in use",
		},
		"ignore_remote_image_change": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Do not track changes of the remote image (RouterOS 7.2x).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"interface": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "veth interface to be used with the container",
		},
		"layer_dir": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Override container config layer dir",
		},
		"layers": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "List of layer dir names for this container image",
		},
		"logging": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "if set to yes, all container-generated output will be shown in the RouterOS log",
		},
		"memory_current": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Current RAM usage by the container.",
		},
		"memory_high": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "RAM usage limit in bytes for a specific container (string value).",
			DiffSuppressFunc: BytesEqual,
		},
		"memory_max": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "Max RAM usage limit in bytes for a specific container (string value). The container process " +
				"will be terminated if the limit is smaller than the container memory usage.",
			DiffSuppressFunc: BytesEqual,
		},
		"mount": {
			Type:        schema.TypeSet,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "Mounts (in the form `/src:/mnt:rw`) to be used with this container",
		},

		"mountlists": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Description: "Mount lists from /container/mounts/ sub-menu to be used with this container. " +
				"RouterOS before 7.21 has a `mounts` parameter instead, the provider maps the attribute to it.",
		},
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Assign a name to the container",
		},
		"os": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The OS of the container image",
		},
		"privileged": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Run the container in privileged mode (RouterOS 7.2x).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"remote_image": {
			Type:         schema.TypeString,
			Optional:     true,
			ForceNew:     true,
			Description:  "The container image name to be installed if an external registry is used (configured under /container/config set registry-url=...)",
			ExactlyOneOf: []string{"file", "remote_image"},
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				if old == "" {
					return false
				}

				if AlwaysPresentNotUserProvided(k, old, new, d) {
					return true
				}

				// Checking the presence of a tag:
				// ~ remote_image = "traefik/whoami:latest" -> "traefik/whoami" # forces replacement
				return old == new || old == new+":latest"
			},
		},
		"restart_interval": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Delay before restarting the container by `restart_policy`.",
			DiffSuppressFunc: TimeEqual,
		},
		"restart_max_count": {
			Type:             schema.TypeInt,
			Optional:         true,
			Description:      "Maximum number of restarts by `restart_policy` (0 is no limit).",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"restart_policy": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Restart the container when it stops: `always`, `no` or `on-failure`.",
			ValidateFunc:     validation.StringInSlice([]string{"always", "no", "on-failure"}, false),
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"root_dir": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Used to save container store outside main memory",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"running": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Container state.",
			Default:     true,
		},
		"shm_size": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Size of /dev/shm of the container.",
			DiffSuppressFunc: BytesEqual,
		},
		"start_on_boot": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Start the container on boot",
		},
		"status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The status of the container",
		},
		"stop_on_unhealthy": {
			Type:             schema.TypeBool,
			Optional:         true,
			Description:      "Stop the container when its health check fails.",
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"stop_signal": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			Description: "Signal to stop the container. The RouterOS 7.24 CLI reference uses the `15-SIGTERM` form, " +
				"some older versions accept only the number (`15`, upstream issue #969); both forms are treated as equal.",
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				if AlwaysPresentNotUserProvided(k, old, new, d) {
					return true
				}
				// "15" == "15-SIGTERM"
				num := func(v string) string { n, _, _ := strings.Cut(v, "-"); return n }
				return old != "" && new != "" && num(old) == num(new)
			},
		},
		"stop_time": {
			Type:             schema.TypeString,
			Optional:         true,
			DiffSuppressFunc: AlwaysPresentNotUserProvided,
		},
		"swap_max": {
			Type:             schema.TypeString,
			Optional:         true,
			Description:      "Max swap usage limit for the container (RouterOS 7.24 changelog: container swap limits).",
			DiffSuppressFunc: BytesEqual,
		},
		"tag": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The tag of the container image",
		},
		"user": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Sets the username used",
		},
		"workdir": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The working directory for cmd entrypoint",
		},
	}

	resRead := func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		metadata := GetMetadata(resSchema)

		res, err := ReadItems(&ItemId{metadata.IdType, d.Id()}, metadata.Path, m.(Client))
		if err != nil {
			ColorizedDebug(ctx, fmt.Sprintf(ErrorMsgGet, err))
			return diag.FromErr(err)
		}

		// Resource not found.
		if len(*res) == 0 {
			d.SetId("")
			return nil
		}

		item := (*res)[0]
		d.SetId(item.GetID(metadata.IdType))

		diags := MikrotikResourceDataToTerraform(item, resSchema, d)
		if diags.HasError() {
			return diags
		}

		// The `running` field is excluded from the default read (MetaSkipFields), so the state is
		// synchronized here. RouterOS returns the container state in the `running` field ("true"/"false"),
		// older versions returned a `status` field instead.
		d.Set("running", containerRunning(item))
		d.Set("status", containerState(item))

		tag, ok := d.Get("tag").(string)
		if ok && tag != "" {
			// Get Registry URL.
			res, err := ReadItems(nil, "/container/config", m.(Client))
			if err != nil {
				ColorizedDebug(ctx, fmt.Sprintf(ErrorMsgPut, err))
				return diag.FromErr(err)
			}

			if len(*res) == 0 {
				return diag.Diagnostics{
					diag.Diagnostic{
						Severity: diag.Error,
						Summary:  "Failed to retrieve the URL of the container registry, the response is empty",
					},
				}
			}

			registryUrl, ok := (*res)[0]["registry-url"]
			if !ok {
				// Key name for an unspecified registry.
				registryUrl, ok = (*res)[0]["assumed-registry-url"]
				if !ok {
					return diag.Diagnostics{
						diag.Diagnostic{
							Severity: diag.Error,
							Summary:  "The `registry-url` was not found in the response",
						},
					}
				}
			}

			u, err := url.Parse(registryUrl)
			if err != nil {
				return diag.FromErr(err)
			}

			// Remove http(s); host:port; path...
			for _, item := range []string{u.Scheme, u.Host, u.Path} {
				tag = strings.TrimPrefix(tag, item)
			}
			// Remove (:////)adguard/adguardhome:latest
			tag = strings.TrimLeft(tag, ":/")

			d.Set("remote_image", strings.TrimPrefix(tag, registryUrl))
		}

		return nil
	}

	resCreate := func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		// Run DefaultCreate.
		diags := ResourceCreate(ctx, resSchema, d, m)
		if diags.HasError() {
			return diags
		}

		if d.Get("running").(bool) {
			startContainer(ctx, resSchema, d, m)
		}

		return ResourceRead(ctx, resSchema, d, m)
	}

	resUpdate := func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		stopContainer(ctx, resSchema, d, m)

		// Run DefaultUpdate.
		diags := ResourceUpdate(ctx, resSchema, d, m)
		if diags.HasError() {
			return diags
		}
		if d.Get("running").(bool) {
			startContainer(ctx, resSchema, d, m)
		}

		return ResourceRead(ctx, resSchema, d, m)
	}

	resDelete := func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		// Stop container
		stopContainer(ctx, resSchema, d, m)

		// Run DefaultDelete.
		return ResourceDelete(ctx, resSchema, d, m)
	}

	return &schema.Resource{
		CreateContext: resCreate,
		ReadContext:   resRead,
		UpdateContext: resUpdate,
		DeleteContext: resDelete,

		Importer: &schema.ResourceImporter{
			StateContext: ImportStateCustomContext(resSchema),
		},

		Schema: resSchema,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},
	}
}

func startContainer(ctx context.Context, s map[string]*schema.Schema, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	stopStateConf := &retry.StateChangeConf{
		Pending: []string{"pulling", "extracting"},
		Target:  []string{"stopped"},
		Refresh: containerStateRefresh(s, d, m),
		Timeout: d.Timeout(schema.TimeoutCreate),
	}
	_, err := stopStateConf.WaitForStateContext(ctx)
	if err != nil {
		err = fmt.Errorf("error waiting for container instance (%s) to be pulled: %s", d.Id(), err)
		return diag.FromErr(err)
	}

	item := MikrotikItem{"number": d.Id()}

	// Start container
	var resUrl = &URL{
		Path: s[MetaResourcePath].Default.(string),
	}
	if m.(Client).GetTransport() == TransportREST {
		resUrl.Path += "/start"
	}

	err = m.(Client).SendRequest(crudStart, resUrl, item, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	startStateConf := &retry.StateChangeConf{
		Pending: []string{"stopped", "starting"},
		Target:  []string{"running"},
		Refresh: containerStateRefresh(s, d, m),
		Timeout: d.Timeout(schema.TimeoutCreate),
	}
	_, err = startStateConf.WaitForStateContext(ctx)
	if err != nil {
		err = fmt.Errorf("error waiting for container instance (%s) to be started: %s", d.Id(), err)
		return diag.FromErr(err)
	}

	return nil
}

func stopContainer(ctx context.Context, s map[string]*schema.Schema, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	item := MikrotikItem{"number": d.Id()}

	var resUrl = &URL{
		Path: s[MetaResourcePath].Default.(string),
	}
	if m.(Client).GetTransport() == TransportREST {
		resUrl.Path += "/stop"
	}

	err := m.(Client).SendRequest(crudStop, resUrl, item, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	stopStateConf := &retry.StateChangeConf{
		Pending: []string{"stopping"},
		Target:  []string{"stopped"},
		Refresh: containerStateRefresh(s, d, m),
		Timeout: d.Timeout(schema.TimeoutDelete),
	}
	_, err = stopStateConf.WaitForStateContext(ctx)
	if err != nil {
		err = fmt.Errorf("error waiting for container instance (%s) to be stopped: %s", d.Id(), err)
		return diag.FromErr(err)
	}
	return nil
}

// containerState returns the container state. Older RouterOS returns it in the `status` field,
// newer versions (7.24.1, upstream PR #1015) return boolean flags `stopped`, `running`, ... instead.
func containerState(item MikrotikItem) string {
	if status, ok := item["status"]; ok && status != "" {
		return status
	}
	// A container with a health check has no `running` flag. RouterOS 7.23.8 prints it with
	// C - STARTING-WITH-HEALTHCHECK, 7.24.6 with H - HEALTHY; the process
	// runs in all of these states, so they are "running" for the start/stop wait and the state.
	for _, state := range []string{"healthy", "unhealthy", "starting-with-healthcheck"} {
		if BoolFromMikrotikJSON(item[state]) {
			return "running"
		}
	}
	// TODO: the flag names are taken from MetaSkipFields and the 7.24 print flags
	// (S stopped, N starting, R running, T stopping, E extracting); verify the REST output on 7.23.
	for _, state := range []string{"running", "starting", "stopping", "stopped", "extracting", "pulling", "downloading"} {
		if BoolFromMikrotikJSON(item[state]) {
			if state == "downloading" {
				return "pulling"
			}
			return state
		}
	}
	return ""
}

// containerRunning reports whether the container process runs: the `running` flag (7.24.1+, PR #1015),
// the health check flags (healthy, unhealthy, starting-with-healthcheck) or `status` of older RouterOS.
func containerRunning(item MikrotikItem) bool {
	if BoolFromMikrotikJSON(item["running"]) {
		return true
	}
	return containerState(item) == "running"
}

func containerStateRefresh(s map[string]*schema.Schema, d *schema.ResourceData, m interface{}) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		metadata := GetMetadata(s)

		res, err := ReadItems(&ItemId{metadata.IdType, d.Id()}, metadata.Path, m.(Client))
		if err != nil {
			return nil, "", err
		}
		if res == nil || len(*res) == 0 {
			return nil, "", fmt.Errorf("container %s not found", d.Id())
		}

		return res, containerState((*res)[0]), nil
	}
}
