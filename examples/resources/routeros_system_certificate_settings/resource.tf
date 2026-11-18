resource "routeros_system_certificate_settings" "this" {
  builtin_trust_store = ["fetch", "dns"]
}
