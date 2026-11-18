resource "routeros_container_mounts" "caddyfile" {
  list = "caddy"
  src  = "/usb1-part1/containers/caddy/Caddyfile"
  dst  = "/etc/caddy/Caddyfile"
}
