resource "routeros_container_envs" "busybox_tz" {
  list  = "busybox"
  key   = "TZ"
  value = "UTC"
}

resource "routeros_container_mounts" "busybox_www" {
  list = "busybox"
  src  = "/usb1-part1/containers/busybox/www"
  dst  = "/www"
}

resource "routeros_container" "busybox" {
  remote_image  = "library/busybox:1.35.0"
  cmd           = "/bin/httpd -f -p 8080 -h /www"
  interface     = routeros_interface_veth.busybox.name
  logging       = true
  root_dir      = "/usb1-part1/containers/busybox/root"
  start_on_boot = true
  envlists      = [routeros_container_envs.busybox_tz.list]
  mountlists    = [routeros_container_mounts.busybox_www.list]
  memory_high   = "64M"
  memory_max    = "128M"
}
