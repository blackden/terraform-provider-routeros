resource "routeros_container_envs" "test_envs" {
  list  = "test_envs"
  key   = "TZ"
  value = "UTC"
}
