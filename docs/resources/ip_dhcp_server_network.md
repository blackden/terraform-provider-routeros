# routeros_ip_dhcp_server_network (Resource)


## Example Usage
```terraform
resource "routeros_ip_dhcp_server_network" "dhcp_server_network" {
  address    = "10.0.0.0/24"
  gateway    = "10.0.0.1"
  dns_server = ["1.1.1.1"]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `address` (String) The network DHCP server(s) will lease addresses from.

### Optional

- `boot_file_name` (String) Boot filename.
- `caps_manager` (List of String) A list of IP addresses for one or more CAPsMAN system managers. DHCP Option 138 (capwap) will be used.
- `comment` (String)
- `dhcp_option` (List of String) Add additional DHCP options from the option list.
- `dhcp_option_set` (String) Add an additional set of DHCP options.
- `dns_none` (Boolean) If set, then DHCP Server will not pass dynamic DNS servers configured on the router to the DHCP clients if no DNS Server in DNS-server is set.
- `dns_server` (List of String) The DHCP client will use these as the default DNS servers. Two DNS servers can be specified to be used by the DHCP client as primary and secondary DNS servers.
- `domain` (String) The DHCP client will use this as the 'DNS domain' setting for the network adapter.
- `gateway` (String) The default gateway to be used by DHCP Client.
- `netmask` (Number) The actual network mask is to be used by the DHCP client. If set to '0' - netmask from network address will be used.
- `next_server` (String) The IP address of the next server to use in bootstrap.
- `ntp_server` (List of String) The DHCP client will use these as the default NTP servers. Two NTP servers can be specified to be used by the DHCP client as primary and secondary NTP servers
- `wins_server` (List of String) The Windows DHCP client will use these as the default WINS servers. Two WINS servers can be specified to be used by the DHCP client as primary and secondary WINS servers

### Read-Only

- `dynamic` (Boolean) Configuration item created by software, not by management interface. It is not exported, and cannot be directly modified.
- `id` (String) The ID of this resource.

## Import
Import is supported using the following syntax:
```shell
#The ID can be found via API or the terminal
#The command for the terminal is -> :put [/ip/dhcp-server/network get [print show-ids]]
terraform import routeros_ip_dhcp_server_network.dhcp_server_network "*0"
```
