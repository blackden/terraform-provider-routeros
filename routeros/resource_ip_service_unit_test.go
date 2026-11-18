package routeros

import (
	"context"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestIpService_NumbersValidation(t *testing.T) {
	validate := ResourceIpService().Schema["numbers"].ValidateDiagFunc
	for _, name := range []string{"reverse-proxy", "www-ssl", "ssh"} {
		if diags := validate(name, cty.GetAttrPath("numbers")); diags.HasError() {
			t.Errorf("numbers = %q: unexpected error %v", name, diags)
		}
	}
	if diags := validate("socks", cty.GetAttrPath("numbers")); !diags.HasError() {
		t.Error(`numbers = "socks": expected a validation error`)
	}
}

// Import by `*N` or by name must fill `numbers`, otherwise the next plan shows `+ numbers`
// (imported services used to show a perpetual update).
func TestIpService_ImportFillsNumbers(t *testing.T) {
	r := ResourceIpService()
	c := newFakeClient(map[string][]MikrotikItem{
		"/ip/service": {
			{".id": "*1", "name": "telnet", "port": "23", "disabled": "true", "dynamic": "false"},
			{".id": "*3", "name": "ssh", "port": "22", "disabled": "false", "dynamic": "false", "address": ""},
			{".id": "*8", "name": "ssh", "port": "22", "dynamic": "true", "connection": "true"},
		},
	})
	for _, importId := range []string{"*3", "ssh", "name=ssh"} {
		d := r.Data(nil)
		d.SetId(importId)
		imported, err := r.Importer.StateContext(context.Background(), d, c)
		if err != nil || len(imported) != 1 {
			t.Fatalf("import %q: %v (%d)", importId, err, len(imported))
		}
		st, diags := fakeRead(t, r, imported[0].Id(), nil, c)
		if diags.HasError() {
			t.Fatalf("read %q: %v", importId, diags)
		}
		if st.ID != "*3" || st.Attributes["numbers"] != "ssh" {
			t.Errorf("import %q: id = %q, numbers = %q", importId, st.ID, st.Attributes["numbers"])
		}
	}
}
