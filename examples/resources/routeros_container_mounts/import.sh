#The ID can be found via API or the terminal
#The command for the terminal is -> :put [/container/mounts get [print show-ids]]
terraform import routeros_container_mounts.caddyfile "*1"
#Or you can import a resource using one of its attributes (it must match exactly one entry)
terraform import routeros_container_mounts.caddyfile "dst=/etc/caddy/Caddyfile"
