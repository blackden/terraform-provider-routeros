package routeros

import (
	"strings"
	"testing"
)

// Unit tests (no router): RouterOS 7.24 container fields and health check states.

func noSchemaWarnings(t *testing.T, summaries []string) {
	t.Helper()
	for _, s := range summaries {
		if strings.Contains(s, "not found in the schema") {
			t.Errorf("unexpected warning: %v", s)
		}
	}
}

func TestContainerConfig_SwapMax(t *testing.T) {
	r := ResourceContainerConfig()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	// RouterOS 7.24: `'swap_max': 'unlimited'` was a "not found in the schema" warning.
	c := newFakeClient(map[string][]MikrotikItem{
		"/container/config": {{
			"layer-dir": "/containers/layers", "memory-current": "67108864", "memory-high": "268435456",
			"memory-max": "536870912", "registry-url": "https://registry.example.invalid", "swap-current": "0",
			"swap-max": "unlimited", "tmpdir": "/containers/tmp", "username": "",
		}},
	})
	st, diags := fakeRead(t, r, "container.config", nil, c)
	noSchemaWarnings(t, diagSummaries(diags))
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["swap_max"] != "unlimited" {
		t.Errorf("swap_max = %q", st.Attributes["swap_max"])
	}
}

func TestContainer_Fields724(t *testing.T) {
	r := ResourceContainer()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	for _, f := range []string{"privileged", "swap_max", "shm_size", "restart_policy", "restart_interval",
		"restart_max_count", "stop_on_unhealthy", "ignore_remote_image_change"} {
		if s := r.Schema[f]; s == nil || !s.Optional {
			t.Errorf("%v must be an argument", f)
		}
	}
	if _, errs := r.Schema["restart_policy"].ValidateFunc("on-failure", "restart_policy"); len(errs) > 0 {
		t.Errorf("on-failure rejected: %v", errs)
	}

	// Shaped like `print detail` on RouterOS 7.24; example values, env omitted.
	item := MikrotikItem{
		".id": "*5", "arch": "amd64", "check-certificate": "true", "container-size": "52428800",
		"cpu-list": "", "cpu-usage": "3", "default-cmd": "/app/run", "default-dns": "192.0.2.53",
		"default-entrypoint": "", "default-healthcheck-cmd": "CMD", "default-healthcheck-interval": "00:01:00",
		"default-healthcheck-retries": "3", "default-healthcheck-start-interval": "00:00:05",
		"default-healthcheck-start-period": "00:00:00", "default-healthcheck-timeout": "00:00:05",
		"default-shell": "/bin/sh -c", "default-stop-signal": "15-SIGTERM", "default-user": "1000:1000",
		"default-workdir": "/app", "env-current": "LANG=C.UTF-8", "envlists": "test-ct",
		"healthcheck-status": "healthy", "healthy": "true", "hostname": "", "ignore-remote-image-change": "false",
		"image-id": "0123abcd", "interface": "test-veth", "layer-dir": "/containers/layers", "logging": "true",
		"memory-current": "33554432", "memory-high": "134217728", "memory-max": "268435456",
		"mountlists": "test-ct-data", "name": "test-ct", "os": "linux", "privileged": "false",
		"remote-image": "example/sample-app:1.0.0", "restart-count": "0", "restart-interval": "00:00:00",
		"restart-max-count": "0", "restart-policy": "no", "root-dir": "/test-ct/root",
		"shm-size": "67108864", "start-on-boot": "true", "stop-on-unhealthy": "false", "stop-time": "00:00:10",
		"swap-max": "0", "tag": "registry.example.invalid/example/sample-app:1.0.0",
	}
	c := newFakeClient(map[string][]MikrotikItem{
		"/container":        {item},
		"/container/config": {{"registry-url": "https://registry.example.invalid"}},
	})
	st, diags := fakeRead(t, r, "*5", nil, c)
	noSchemaWarnings(t, diagSummaries(diags))
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["running"] != "true" {
		t.Errorf("healthy container: running = %q", st.Attributes["running"])
	}
	if st.Attributes["privileged"] != "false" || st.Attributes["restart_policy"] != "no" {
		t.Errorf("privileged = %q, restart_policy = %q", st.Attributes["privileged"], st.Attributes["restart_policy"])
	}
}

func TestContainerState_HealthCheck(t *testing.T) {
	for _, tt := range []struct {
		item    MikrotikItem
		state   string
		running bool
	}{
		{MikrotikItem{"healthy": "true"}, "running", true},                   // 7.24, H - HEALTHY
		{MikrotikItem{"starting-with-healthcheck": "true"}, "running", true}, // 7.23, C
		{MikrotikItem{"unhealthy": "true"}, "running", true},
		{MikrotikItem{"running": "true"}, "running", true},
		{MikrotikItem{"stopped": "true"}, "stopped", false},
		{MikrotikItem{"status": "running"}, "running", true},
		{MikrotikItem{"status": "stopped"}, "stopped", false},
		{MikrotikItem{}, "", false},
	} {
		if got := containerState(tt.item); got != tt.state {
			t.Errorf("containerState(%v) = %q, want %q", tt.item, got, tt.state)
		}
		if got := containerRunning(tt.item); got != tt.running {
			t.Errorf("containerRunning(%v) = %v, want %v", tt.item, got, tt.running)
		}
	}
}
