package routeros

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Unit tests (no router): /container start/stop on RouterOS 7.24.6.

// ctClient is a fakeClient with one container that behaves like RouterOS 7.24.6: `pulls` reads return
// the `downloading/extracting` flag, /container/start and /container/stop change the flags and reject
// the `number` argument with the REST error of 7.24.6.
type ctClient struct {
	*fakeClient
	pulls int
}

func newCtClient(ct MikrotikItem, pulls int) *ctClient {
	return &ctClient{fakeClient: newFakeClient(map[string][]MikrotikItem{"/container": {ct}}), pulls: pulls}
}

func (c *ctClient) ct() MikrotikItem { return c.items["/container"][0] }

func (c *ctClient) setFlag(flag string) {
	for _, f := range []string{"running", "stopped", "downloading/extracting"} {
		delete(c.ct(), f)
	}
	c.ct()[flag] = "true"
}

func (c *ctClient) SendRequest(method crudMethod, url *URL, item MikrotikItem, result interface{}) error {
	switch method {
	case crudRead:
		if url.Path == "/container" {
			if c.pulls > 0 {
				c.pulls--
				c.setFlag("downloading/extracting")
			} else if c.ct()["downloading/extracting"] == "true" {
				c.setFlag("stopped")
			}
		}
	case crudStart, crudStop:
		if _, ok := item["number"]; ok {
			c.requests = append(c.requests, fakeRequest{method: method, path: url.Path, item: item})
			return fmt.Errorf("POST '%v' returned response code: 400, message: 'Bad Request', details: "+
				"'unknown parameter number'", url.Path)
		}
		if item["numbers"] == c.ct()[".id"] {
			if method == crudStart {
				c.setFlag("running")
			} else {
				c.setFlag("stopped")
			}
		}
	}

	err := c.fakeClient.SendRequest(method, url, item, result)
	if method == crudCreate {
		// PUT /rest/container returns the new item.
		for k, v := range c.ct() {
			(*result.(*MikrotikItem))[k] = v
		}
	}
	return err
}

func (c *ctClient) commands() []string {
	var out []string
	for _, r := range c.requests {
		switch r.method {
		case crudStart, crudStop, crudUpdate, crudCreate:
			out = append(out, fmt.Sprintf("%v %v", r.path, r.item["numbers"]))
		}
	}
	return out
}

// ctItem is a REST item of a stopped busybox container on RouterOS 7.24.6, shortened.
func ctItem() MikrotikItem {
	return MikrotikItem{
		".id": "*3", "arch": "amd64", "cmd": "/bin/sleep 86400", "comment": "test-ct",
		"interface": "test-veth", "logging": "false", "name": "test-ct", "os": "linux",
		"remote-image": "library/busybox:latest", "root-dir": "/test-ct/root", "start-on-boot": "false",
		"stop-time": "10s", "stopped": "true", "tag": "registry-1.docker.io/library/busybox:latest",
	}
}

func ctConfig(running bool) map[string]cty.Value {
	return map[string]cty.Value{
		"name":         cty.StringVal("test-ct"),
		"interface":    cty.StringVal("test-veth"),
		"remote_image": cty.StringVal("library/busybox:latest"),
		"root_dir":     cty.StringVal("/test-ct/root"),
		"cmd":          cty.StringVal("/bin/sleep 86400"),
		"comment":      cty.StringVal("test-ct"),
		"running":      cty.BoolVal(running),
	}
}

// fakePlanApply plans the configuration against the prior state and applies the diff, like
// `tofu apply` of a changed configuration. It returns the diff (nil: no changes).
func fakePlanApply(t *testing.T, r *schema.Resource, prior *terraform.InstanceState, cfg map[string]cty.Value,
	meta interface{}) (*terraform.InstanceDiff, *terraform.InstanceState, diag.Diagnostics) {
	t.Helper()
	ty := r.CoreConfigSchema().ImpliedType()
	attrs := map[string]cty.Value{}
	for name, at := range ty.AttributeTypes() {
		if v, ok := cfg[name]; ok {
			attrs[name] = v
		} else {
			attrs[name] = cty.NullVal(at)
		}
	}
	val := cty.ObjectVal(attrs)
	conf := terraform.NewResourceConfigShimmed(val, r.CoreConfigSchema())
	conf.CtyValue = val
	diff, err := r.SimpleDiff(context.Background(), prior, conf, meta)
	if err != nil {
		t.Fatalf("fakePlanApply: diff: %v", err)
	}
	if diff == nil || diff.Empty() {
		return nil, prior, nil
	}
	diff.RawConfig = val
	st, diags := r.Apply(context.Background(), prior, diff, meta)
	return diff, st, diags
}

func TestContainerState_Flags724(t *testing.T) {
	for _, tc := range []struct {
		item MikrotikItem
		want string
	}{
		{MikrotikItem{"downloading/extracting": "true"}, "pulling"},
		{MikrotikItem{"download/extract failed": "true"}, "failed"},
		{MikrotikItem{"download/extract failed": "false", "stopped": "true"}, "stopped"},
		{MikrotikItem{"running": "true"}, "running"},
		{MikrotikItem{"status": "running"}, "running"},
	} {
		if got := containerState(tc.item); got != tc.want {
			t.Errorf("containerState(%v) = %q, want %q", tc.item, got, tc.want)
		}
	}
}

// Creating with running = true used to send nothing (the wait saw "" for `downloading/extracting`),
// and the state had running = true for a stopped container.
func TestContainer_CreateRunningStarts(t *testing.T) {
	r := ResourceContainer()
	c := newCtClient(ctItem(), 2)
	st, diags := fakeApply(t, r, ctConfig(true), c)
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	want := []string{"/container ", "/container/start *3"}
	if got := c.commands(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	if st.Attributes["running"] != "true" || st.Attributes["status"] != "running" {
		t.Errorf("running = %q, status = %q", st.Attributes["running"], st.Attributes["status"])
	}
	// remote-image as RouterOS returns it, not the tag without the registry.
	if st.Attributes["remote_image"] != "library/busybox:latest" {
		t.Errorf("remote_image = %q", st.Attributes["remote_image"])
	}
}

func TestContainer_CreateStopped(t *testing.T) {
	r := ResourceContainer()
	c := newCtClient(ctItem(), 1)
	st, diags := fakeApply(t, r, ctConfig(false), c)
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	if got := c.commands(); len(got) != 1 {
		t.Errorf("requests = %v, want only create", got)
	}
	if st.Attributes["running"] != "false" {
		t.Errorf("running = %q", st.Attributes["running"])
	}
}

func TestContainer_CreateFailedDownload(t *testing.T) {
	r := ResourceContainer()
	ct := ctItem()
	delete(ct, "stopped")
	ct["download/extract failed"] = "true"
	c := newCtClient(ct, 0)
	_, diags := fakeApply(t, r, ctConfig(true), c)
	if !diags.HasError() {
		t.Fatalf("a failed download must be an error")
	}
	if got := c.commands(); len(got) != 1 {
		t.Errorf("requests = %v, want only create", got)
	}
}

func TestContainer_RunningToggle(t *testing.T) {
	r := ResourceContainer()
	c := newCtClient(ctItem(), 0)
	prior, diags := fakeRead(t, r, "*3", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}

	// The read state with the same configuration: no changes, in particular no remote_image replacement.
	if diff, _, _ := fakePlanApply(t, r, prior, ctConfig(false), c); diff != nil {
		t.Fatalf("unexpected diff: %v", diff.Attributes)
	}

	// false -> true: only /container/start, no PATCH (a PATCH with remote-image downloads the image again).
	diff, st, diags := fakePlanApply(t, r, prior, ctConfig(true), c)
	if diags.HasError() {
		t.Fatalf("start: %v", diags)
	}
	if diff == nil || diff.RequiresNew() {
		t.Fatalf("want an in-place update, diff = %v", diff)
	}
	want := []string{"/container/start *3"}
	if got := c.commands(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	if st.Attributes["running"] != "true" || c.ct()["running"] != "true" {
		t.Errorf("running = %q, flags %v", st.Attributes["running"], c.ct())
	}

	// true -> false: only /container/stop.
	c.requests = nil
	_, st, diags = fakePlanApply(t, r, st, ctConfig(false), c)
	if diags.HasError() {
		t.Fatalf("stop: %v", diags)
	}
	want = []string{"/container/stop *3"}
	if got := c.commands(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	if st.Attributes["running"] != "false" || c.ct()["stopped"] != "true" {
		t.Errorf("running = %q, flags %v", st.Attributes["running"], c.ct())
	}
}

func TestContainer_UpdateRunning(t *testing.T) {
	r := ResourceContainer()
	ct := ctItem()
	delete(ct, "stopped")
	ct["running"] = "true"
	c := newCtClient(ct, 0)
	prior, diags := fakeRead(t, r, "*3", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}

	// A changed argument of a running container: stop, PATCH without remote-image, start.
	cfg := ctConfig(true)
	cfg["comment"] = cty.StringVal("test-ct2")
	_, _, diags = fakePlanApply(t, r, prior, cfg, c)
	if diags.HasError() {
		t.Fatalf("update: %v", diags)
	}
	want := []string{"/container/stop *3", "/container/*3 ", "/container/start *3"}
	if got := c.commands(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("requests = %v, want %v", got, want)
	}
	for _, req := range c.requests {
		if req.method == crudUpdate {
			if _, ok := req.item["remote-image"]; ok {
				t.Errorf("PATCH with remote-image: %v", req.item)
			}
			if req.item["comment"] != "test-ct2" {
				t.Errorf("PATCH comment = %q", req.item["comment"])
			}
		}
	}
}

// The `number` argument (as sent by upstream) is rejected by RouterOS 7.24.6; the error must not be
// swallowed.
func TestContainer_StartError(t *testing.T) {
	r := ResourceContainer()
	c := newCtClient(ctItem(), 0)
	d := r.TestResourceData()
	d.SetId("*3")
	d.Set("running", true)
	if err := c.SendRequest(crudStart, &URL{Path: "/container/start"}, MikrotikItem{"number": "*3"}, nil); err == nil {
		t.Fatalf("ctClient must reject number")
	}
	c.requests = nil
	if diags := startContainer(context.Background(), r.Schema, d, c); diags.HasError() {
		t.Fatalf("start: %v", diags)
	}
	req, ok := c.lastWrite()
	if _, has := req.item["number"]; !ok || has || req.item["numbers"] != "*3" {
		t.Errorf("start request = %v", req.item)
	}
}

// Old RouterOS without `remote-image` in the response: remote_image from the tag without the registry.
func TestContainer_RemoteImageFromTag(t *testing.T) {
	r := ResourceContainer()
	ct := ctItem()
	delete(ct, "remote-image")
	ct["tag"] = "registry.example.invalid/example/sample-app:2.1.1"
	c := newFakeClient(map[string][]MikrotikItem{
		"/container":        {ct},
		"/container/config": {{"registry-url": "https://registry.example.invalid"}},
	})
	st, diags := fakeRead(t, r, "*3", nil, c)
	if diags.HasError() {
		t.Fatalf("read: %v", diags)
	}
	if st.Attributes["remote_image"] != "example/sample-app:2.1.1" {
		t.Errorf("remote_image = %q", st.Attributes["remote_image"])
	}
}
