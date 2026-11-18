package routeros

import (
	"reflect"
	"testing"
)

func TestDriftObjects_AddBefore(t *testing.T) {
	var do driftObjects
	do.Add("7.18", "/container", "tag", "repo")
	do.AddBefore("7.20", "/container", "envlists", "envlist")
	do.AddBefore("7.21", "/container", "mountlists", "mounts")
	do.Add("7.21", "/other", "a", "b")

	tests := []struct {
		ros     string
		reverse bool
		want    map[string]string
	}{
		{"7.17", false, map[string]string{"envlists": "envlist", "mountlists": "mounts"}},
		{"7.19.2", false, map[string]string{"tag": "repo", "envlists": "envlist", "mountlists": "mounts"}},
		{"7.20", false, map[string]string{"tag": "repo", "mountlists": "mounts"}},
		{"7.20.8", true, map[string]string{"repo": "tag", "mounts": "mountlists"}},
		{"7.21", false, map[string]string{"tag": "repo"}},
		{"7.23.8", true, map[string]string{"repo": "tag"}},
	}
	for _, tt := range tests {
		if got := do.GetDriftMap(tt.ros, "/container", tt.reverse); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("GetDriftMap(%v, reverse=%v) = %v, want %v", tt.ros, tt.reverse, got, tt.want)
		}
	}
}

// The generated drift table: on 7.21+ the container attributes carry their MikroTik names,
// older routers get the old parameter names.
func TestDriftAttributeSlice_Container(t *testing.T) {
	tests := []struct {
		ros, path string
		want      map[string]string
	}{
		{"7.19", "/container", map[string]string{"tag": "repo", "envlists": "envlist", "mountlists": "mounts"}},
		{"7.23.8", "/container", map[string]string{"tag": "repo"}},
		{"7.19", "/container/config", map[string]string{"memory_high": "ram-high"}},
		{"7.23.8", "/container/config", map[string]string{}},
		{"7.19", "/container/envs", map[string]string{"list": "name"}},
		{"7.23.8", "/container/envs", map[string]string{}},
		{"7.20", "/container/mounts", map[string]string{"list": "name"}},
		{"7.23.8", "/container/mounts", map[string]string{}},
	}
	for _, tt := range tests {
		if got := driftAttributeSlice.GetDriftMap(tt.ros, tt.path, false); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("GetDriftMap(%v, %v) = %v, want %v", tt.ros, tt.path, got, tt.want)
		}
	}
}

func TestContainerState(t *testing.T) {
	tests := []struct {
		item MikrotikItem
		want string
	}{
		{MikrotikItem{"status": "stopping"}, "stopping"},
		{MikrotikItem{"running": "true"}, "running"},
		{MikrotikItem{"running": "false", "stopped": "true"}, "stopped"},
		{MikrotikItem{"extracting": "true"}, "extracting"},
		{MikrotikItem{}, ""},
	}
	for _, tt := range tests {
		if got := containerState(tt.item); got != tt.want {
			t.Errorf("containerState(%v) = %q, want %q", tt.item, got, tt.want)
		}
	}
}
