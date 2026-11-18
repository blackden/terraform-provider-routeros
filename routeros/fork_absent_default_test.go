package routeros

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Unit tests (no router): fields RouterOS leaves out of the response.

func TestDiffSuppressAbsentDefault(t *testing.T) {
	r := &schema.Resource{Schema: map[string]*schema.Schema{"x": {Type: schema.TypeString, Optional: true}}}
	existing := r.TestResourceData()
	existing.SetId("*1")
	created := r.TestResourceData() // no id: create

	cases := []struct {
		def, old, new string
		d             *schema.ResourceData
		suppress      bool
	}{
		{"true", "", "true", existing, true},       // connect absent, default planned
		{"true", "", "true", nil, true},            //
		{"179", "0", "179", existing, true},        // local.port read back as 0
		{"none", "", "none", existing, true},       // add_path_out
		{"true", "false", "true", existing, false}, // the router has connect=no
		{"true", "", "false", existing, false},     // the user sets connect=false
		{"179", "0", "1179", existing, false},      // the user sets a port
		{"179", "1179", "179", existing, false},    // back to the default
		{"none", "all", "none", existing, false},   //
		{"true", "", "true", created, false},       // create: no response yet
	}
	for _, tc := range cases {
		if got := DiffSuppressAbsentDefault(tc.def)("k", tc.old, tc.new, tc.d); got != tc.suppress {
			t.Errorf("def %q, %q -> %q (create %v): suppress = %v, want %v",
				tc.def, tc.old, tc.new, tc.d != nil && tc.d.Id() == "", got, tc.suppress)
		}
	}
}

func TestForkPayloadKey(t *testing.T) {
	drift := map[string]string{"address_families": "afi"}
	for attr, want := range map[string]string{
		"connect":                    "connect",
		"add_path_out":               "add-path-out",
		"address_families":           "afi",
		"local.0.ttl":                "local.ttl",
		"output.0.default_originate": "output.default-originate",
	} {
		if got := forkPayloadKey(attr, drift); got != want {
			t.Errorf("%v: %q, want %q", attr, got, want)
		}
	}
}

func TestValidationEmptyOr(t *testing.T) {
	f := ValidationEmptyOr(ValidationMultiValInSlice([]string{"static", "connected"}, false, false))
	for v, ok := range map[string]bool{"": true, "static": true, "static,connected": true, "bogus": false, "static,": false} {
		if diags := f(v, cty.Path{}); diags.HasError() == ok {
			t.Errorf("%q: errors %v", v, diags)
		}
	}
}
