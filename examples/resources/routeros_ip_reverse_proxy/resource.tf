# The listener: port and default certificate of the reverse-proxy service.
resource "routeros_ip_service" "reverse_proxy" {
  numbers     = "reverse-proxy"
  port        = 443
  certificate = "wildcard.example.com"
  disabled    = false
}

resource "routeros_ip_reverse_proxy" "app" {
  sni        = "app.example.com"
  ip_address = "192.0.2.10"
  port       = 8080
  comment    = "app container"
}
