# The entries are built into RouterOS; the resource only changes them.
resource "routeros_ip_firewall_service_port" "sip" {
  name     = "sip"
  disabled = true
}
