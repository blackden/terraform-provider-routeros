package routeros

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// fakeClient is an in-memory Client for unit tests without a router.
// Reads return the items stored for the URL path (filtered by `?.id=`/`name=`-style
// queries), writes are recorded in `requests`.
type fakeClient struct {
	transport TransportType
	items     map[string][]MikrotikItem
	requests  []fakeRequest
}

type fakeRequest struct {
	method crudMethod
	path   string
	item   MikrotikItem
}

func newFakeClient(items map[string][]MikrotikItem) *fakeClient {
	return &fakeClient{transport: TransportREST, items: items}
}

func (c *fakeClient) GetExtraParams() *ExtraParams { return &ExtraParams{SuppressSysODelWarn: true} }

func (c *fakeClient) GetTransport() TransportType { return c.transport }

func (c *fakeClient) SendRequest(method crudMethod, url *URL, item MikrotikItem, result interface{}) error {
	if method != crudRead {
		c.requests = append(c.requests, fakeRequest{method: method, path: url.Path, item: item})
		if r, ok := result.(*MikrotikItem); ok && r != nil {
			*r = MikrotikItem{}
		}
		return nil
	}

	stored := c.items[url.Path]
	switch r := result.(type) {
	case *MikrotikItem:
		// Singleton menu: REST returns one object.
		if len(stored) > 0 {
			*r = stored[0]
		}
	case *[]MikrotikItem:
		var out []MikrotikItem
		for _, it := range stored {
			if fakeMatch(it, url.Query) {
				out = append(out, it)
			}
		}
		*r = out
	default:
		return fmt.Errorf("fakeClient: unsupported result type %T", result)
	}
	return nil
}

// fakeMatch understands the query forms used by ReadItems/ReadItemsFiltered:
// "?.id=*1" (REST and API), "name=x" (REST filter) and "?=name=x" (API filter).
func fakeMatch(it MikrotikItem, query []string) bool {
	for _, q := range query {
		q = strings.TrimPrefix(strings.TrimPrefix(q, "?="), "?")
		k, v, ok := strings.Cut(q, "=")
		if !ok {
			continue
		}
		if it[k] != v {
			// Missing boolean flags read as "false".
			if !(v == "false" && it[k] == "") {
				return false
			}
		}
	}
	return true
}

// lastWrite returns the item of the last non-read request.
func (c *fakeClient) lastWrite() (fakeRequest, bool) {
	if len(c.requests) == 0 {
		return fakeRequest{}, false
	}
	return c.requests[len(c.requests)-1], true
}

// fakeApply plans and applies a create of resource r with the given configuration
// (attributes that are not listed are null), the same way the SDK does it for
// `terraform apply`: DiffSuppressFunc and GetRawConfig see the real configuration.
func fakeApply(t *testing.T, r *schema.Resource, cfg map[string]cty.Value, meta interface{}) (*terraform.InstanceState, diag.Diagnostics) {
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
	for name := range cfg {
		if _, ok := attrs[name]; !ok || !ty.HasAttribute(name) {
			t.Fatalf("fakeApply: unknown attribute %v", name)
		}
	}
	val := cty.ObjectVal(attrs)
	conf := terraform.NewResourceConfigShimmed(val, r.CoreConfigSchema())
	conf.CtyValue = val
	diff, err := r.SimpleDiff(context.Background(), nil, conf, meta)
	if err != nil {
		t.Fatalf("fakeApply: diff: %v", err)
	}
	if diff == nil {
		diff = &terraform.InstanceDiff{Attributes: map[string]*terraform.ResourceAttrDiff{}}
	}
	diff.RawConfig = val
	return r.Apply(context.Background(), nil, diff, meta)
}

// fakeRead refreshes a resource with the given id and prior state attributes, the way
// `terraform plan` does it. Diagnostics include warnings such as "Field ... not found
// in the schema".
func fakeRead(t *testing.T, r *schema.Resource, id string, attrs map[string]string, meta interface{}) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()
	if attrs == nil {
		attrs = map[string]string{}
	}
	attrs["id"] = id
	return r.RefreshWithoutUpgrade(context.Background(), &terraform.InstanceState{ID: id, Attributes: attrs}, meta)
}

// diagSummaries returns the summaries of all diagnostics (errors and warnings).
func diagSummaries(diags diag.Diagnostics) []string {
	var out []string
	for _, d := range diags {
		out = append(out, d.Summary)
	}
	return out
}
