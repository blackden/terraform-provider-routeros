# Serve REST and WebFig only over www-ssl (HTTPS).
resource "routeros_ip_service_webserver" "this" {
  rest_plain    = false
  rest_secure   = true
  webfig_plain  = false
  webfig_secure = true
}
