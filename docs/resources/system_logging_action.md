# routeros_system_logging_action (Resource)




<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of an action.
- `target` (String) Storage facility or target of log messages.

### Optional

- `bsd_syslog` (Boolean) Whether to use bsd-syslog as defined in RFC 3164.
- `disk_file_count` (Number) Specifies number of files used to store log messages, applicable only if `action=disk`.
- `disk_file_name` (String) Name of the file used to store log messages, applicable only if `action=disk`.
- `disk_lines_per_file` (Number) Specifies maximum size of file in lines, applicable only if `action=disk`.
- `disk_stop_on_full` (Boolean) Whether to stop to save log messages to disk after the specified disk-lines-per-file and disk-file-count number is reached, applicable only if `action=disk`.
- `email_start_tls` (Boolean) Whether to use tls when sending email, applicable only if `action=email`.
- `email_to` (String) Email address where logs are sent, applicable only if `action=email`.
- `memory_lines` (Number) Number of records in local memory buffer, applicable only if `action=memory`.
- `memory_stop_on_full` (Boolean) Whether to stop to save log messages in local buffer after the specified memory-lines number is reached.
- `remember` (Boolean) Whether to keep log messages, which have not yet been displayed in console, applicable if `action=echo`.
- `remote` (String) Remote logging server's IP/IPv6 address, applicable if `action=remote`.
- `remote_port` (Number) Remote logging server's UDP port, applicable if `action=remote`.
- `src_address` (String) Source address used when sending packets to remote server, applicable if `action=remote`.
- `syslog_facility` (String) SYSLOG facility, applicable if `action=remote`.
- `syslog_severity` (String) Severity level indicator defined in RFC 3164, applicable if `action=remote`.
- `syslog_time_format` (String) SYSLOG time format (`bsd-syslog` or `iso8601`).

### Read-Only

- `default` (Boolean) This is a default action.
- `id` (String) The ID of this resource.

