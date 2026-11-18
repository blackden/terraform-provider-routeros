# blackden/routeros: a fork of terraform-provider-routeros

This is a fork of [terraform-routeros/terraform-provider-routeros](https://github.com/terraform-routeros/terraform-provider-routeros).
It is `main` of the upstream provider plus a few open upstream pull requests plus
additional fixes and resources. It is published so that the fixes are usable
while they make their way upstream. The fork is not meant to replace the upstream
project, and the changes are offered back to it.

## Why

The last upstream release is v1.99.1 (2026-03-08); upstream `main` has not moved
much since. Several fixes needed for RouterOS 7.21 to 7.24 (7.24 is the current
long-term branch) exist only as open pull requests, and some resources and
attributes are missing. This fork
collects them in one place and adds the rest.

## Supported RouterOS versions

RouterOS 7 over the REST API (the legacy API works as upstream). The provider
adapts to the router version through the drift table
`routeros/mikrotik_resource_drift.yaml`, so older 7.x releases keep working with
the current schema where the table has an entry. The unit tests run against
7.23 and 7.24 (`ROS_VERSION=7.23 go test ./...` and `ROS_VERSION=7.24 ...`).
Menus that depend on specific hardware (wifi, switch, LTE and similar) are
provided as is, without being tested against hardware.

## Upstream pull requests merged

Authorship of the original commits is preserved.

| PR | Author | What it does |
|---|---|---|
| [#910](https://github.com/terraform-routeros/terraform-provider-routeros/pull/910) Fix container mounts and misc. new attrs in ROS v7.21 | Oliver Ford | `/container` attributes of RouterOS 7.21: `envlists`, `mountlists`, inline `env` and `mount`, `name`, `cpu_list`, `hosts` and others |
| [#983](https://github.com/terraform-routeros/terraform-provider-routeros/pull/983) fix(ip_service): handle dynamic per-connection rows in RouterOS 7.20+ | Sol Astrius | `/ip service` is addressed by `.id` and dynamic rows of open connections are ignored |
| [#992](https://github.com/terraform-routeros/terraform-provider-routeros/pull/992) feat: add stable RouterOS 7.22 schema support | Levi Zitting | fields added in 7.22, MLAG management through bridge attributes |
| [#1015](https://github.com/terraform-routeros/terraform-provider-routeros/pull/1015) fix(container): read running state from the running field, not status | Egor Zaitsev | container `running` is read from the `running` field (`status` is gone in 7.24) |
| [#1009](https://github.com/terraform-routeros/terraform-provider-routeros/pull/1009) fix: Fields not found in the schema (7.24.1) | Aaron Morgan | drift handling and skipped fields for RouterOS 7.24 |

Fixes for upstream issues
[#966](https://github.com/terraform-routeros/terraform-provider-routeros/issues/966)
(`client_allowed_address`) and
[#969](https://github.com/terraform-routeros/terraform-provider-routeros/issues/969)
(`stop_signal` without a default value) are also included.

## Added in the fork

New resources:

- `routeros_ip_reverse_proxy` (`/ip/reverse-proxy`)
- `routeros_ip_proxy` (`/ip/proxy`)
- `routeros_ip_service_webserver` (`/ip/service/webserver`)
- `routeros_ip_firewall_service_port` (`/ip/firewall/service-port`)
- `routeros_ovpn_server_instance` (list of named OpenVPN servers, RouterOS 7.17+)
- `routeros_system_package_update` (`/system/package/update`, channel and mode only)
- `routeros_system_certificate_settings` (`/certificate/settings`)
- `routeros_tool_romon` (`/tool/romon`)

Fixes and changes to existing resources:

- `routeros_ip_service`: the `reverse-proxy` service; state with the old
  name-based id is migrated to `*N` instead of being dropped; import fills `numbers`.
- `routeros_container*`: RouterOS 7.21+ attribute names (with drift entries for
  older versions), 7.24 fields, container state read from flags,
  start/stop by `numbers` with error checking, no replacement when `remote_image` changes.
- `routeros_container_config`: `memory_high` and `memory_max`.
- `routeros_ovpn_client`: `tls_auth` and `tls_crypt` (read-only), `disconnect_notify`
  (effective with `protocol = udp` only).
- `routeros_gre`, `routeros_ipip`, `routeros_eoip`: `remote_address` accepts a DNS name;
  new read-only `current_remote_address`.
- `routeros_ppp_profile`: `use_encryption`, `use_ipv6`, `use_mpls` accept `required`.
- `routeros_ip_firewall_mangle`: no default for `address_list_timeout` and `log`.
- `routeros_ip_proxy`: `max_cache_object_size` and `max_cache_size` are compared by size
  (`"2MiB"` equals `"2048"`).
- `routeros_ovpn_client`, `routeros_container_config`: the masked value `*****`
  of `password` produces no diff and is never sent on update.
- All resources (serializer): a masked value (`*****`) of any `Sensitive` string
  attribute is sent only when the configuration holds exactly that value, and
  removing a nested block from the configuration no longer panics (the changed
  fields of the block are sent with their zero values, as for a removed
  top-level attribute).
- `routeros_routing_bgp_connection`: import and `-generate-config-out` work without errors;
  update sends only changed fields.
- `routeros_interface_veth`: `dhcp_address` is read-only; `routeros_routing_rule`:
  `vrf` and `realm` produce no diff while unset.

The full list is in the git history (`git log 0d8c069..`).

### Passwords and the `sensitive` policy

RouterOS returns write-only secrets masked (`*****`) when the RouterOS user of
the provider lacks the `sensitive` policy. Then a `password` absent from the
configuration (or set to `""`) is not managed: the router keeps its value, the
mask is never sent back, and the password cannot be cleared through the
configuration. With the `sensitive` policy RouterOS returns the real password:
a `password` absent from the configuration is then a planned change, and apply
clears the password on the router.

## Migrating from the upstream provider

In `required_providers`:

```hcl
terraform {
  required_providers {
    routeros = {
      source  = "blackden/routeros"
      version = "~> 1.100"
    }
  }
}
```

Then re-initialise and move the existing state to the new source address:

```sh
tofu init -upgrade
tofu state replace-provider registry.opentofu.org/terraform-routeros/routeros registry.opentofu.org/blackden/routeros
tofu plan
```

(With Terraform use `registry.terraform.io/...` addresses.) Resource and
attribute names are the same as upstream, except for the container attributes
renamed to follow RouterOS 7.21+ (see above). Run `plan` before `apply` and
check the output.

## Versions

The fork follows the upstream version line. `1.100.0` is the first fork release;
it is based on upstream `main` at commit `0d8c069` and also contains the upstream `routeros_routing_id` change, which is not released upstream. Releases are tagged `vX.Y.Z`;
the version of the current build is in `FORK_VERSION`. Once upstream releases these changes,
the fork will be rebased onto the upstream release.

Fork version numbers do not correspond to upstream releases with the same number.
Upstream `main` already has an unreleased `1.100.0` changelog entry; a future upstream
`1.100.0` (or any later upstream version) is a different release from the fork's
`1.100.0`.

## Building

```sh
fork/build.sh            # tests and builds into dist/
fork/build.sh install    # also installs into a local filesystem mirror
```

Variables: `SKIP_TESTS=1`, `FORK_PLATFORMS`, `FORK_MIRROR`, `FORK_SOURCE`
(defaults to `registry.opentofu.org/blackden/routeros`).

## License

The same as upstream, see `LICENSE.md`. This project is not affiliated with MikroTik.
