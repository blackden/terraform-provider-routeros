package routeros

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testIpReverseProxy = "routeros_ip_reverse_proxy.test"

// TODO: the first RouterOS version with /ip/reverse-proxy is not documented; 7.23 is
// the oldest one seen.
const testIpReverseProxyMinVersion = "7.23"

func TestAccIpReverseProxyTest_basic(t *testing.T) {
	if !testCheckMinVersion(t, testIpReverseProxyMinVersion) {
		t.Logf("Test skipped, the minimum required version is %v", testIpReverseProxyMinVersion)
		return
	}

	t.Parallel()
	for _, name := range testNames {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
					testSetTransportEnv(t, name)
				},
				ProviderFactories: testAccProviderFactories,
				CheckDestroy:      testCheckResourceDestroy("/ip/reverse-proxy", "routeros_ip_reverse_proxy"),
				Steps: []resource.TestStep{
					{
						Config: testAccIpReverseProxyConfig(),
						Check: resource.ComposeTestCheckFunc(
							testResourcePrimaryInstanceId(testIpReverseProxy),
							resource.TestCheckResourceAttr(testIpReverseProxy, "sni", "test.example.com"),
							resource.TestCheckResourceAttr(testIpReverseProxy, "ip_address", "127.0.0.1"),
							resource.TestCheckResourceAttr(testIpReverseProxy, "port", "8080"),
						),
					},
				},
			})
		})
	}
}

func testAccIpReverseProxyConfig() string {
	return fmt.Sprintf(`%v

resource "routeros_ip_reverse_proxy" "test" {
  sni        = "test.example.com"
  ip_address = "127.0.0.1"
  port       = 8080
  disabled   = true
}
`, providerConfig)
}

// Unit test (no router): schema shape and validation.
func TestIpReverseProxy_Schema(t *testing.T) {
	r := ResourceIpReverseProxy()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	for _, f := range []string{"sni", "ip_address", "port"} {
		if !r.Schema[f].Required {
			t.Errorf("%v must be required", f)
		}
	}
	if got := r.Schema[MetaResourcePath].Default; got != "/ip/reverse-proxy" {
		t.Errorf("resource path = %v", got)
	}
	if _, errs := r.Schema["ip_address"].ValidateFunc("2001:db8::1", "ip_address"); len(errs) > 0 {
		t.Errorf("IPv6 backend rejected: %v", errs)
	}
	if _, errs := r.Schema["ip_address"].ValidateFunc("backend.local", "ip_address"); len(errs) == 0 {
		t.Error("host name accepted as ip_address")
	}
}
