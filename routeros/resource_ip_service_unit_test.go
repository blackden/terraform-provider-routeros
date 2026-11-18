package routeros

import (
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
