package routeros

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testInterfaceEthernetMinVersion = "7.12"
const testInterfaceEthernetAddress = "routeros_interface_ethernet.test"
const testInterfaceEthernetImportAddress = "routeros_interface_ethernet.importtest"
const expectedIdForInterface3 = "*3"

func TestAccInterfaceEthernetTest_basic(t *testing.T) {
	if !testCheckMinVersion(t, testInterfaceEthernetMinVersion) {
		t.Logf("Test skipped, the minimum required version is %v", testInterfaceEthernetMinVersion)
		return
	}

	for _, name := range testNames {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
					testSetTransportEnv(t, name)
				},
				ProviderFactories: testAccProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccInterfaceEthernetConfig(),
						Check: resource.ComposeTestCheckFunc(
							testResourcePrimaryInstanceId(testInterfaceEthernetAddress),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "name", "terraform"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "mtu", "9000"),
							// auto_negotiation = true + advertise || auto_negotiation = false + speed
							// resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "advertise", "10000M-full"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "arp", "disabled"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "auto_negotiation", "false"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "tx_flow_control", "auto"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "rx_flow_control", "auto"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "mdix_enable", "false"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "sfp_shutdown_temperature", "60"),
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "speed", "10G-baseT"),

							// read only properties. #slave and #switch are not returned from the virtual switch
							// so we can add assertions.
							resource.TestCheckResourceAttr(testInterfaceEthernetAddress, "running", "true"),
						),
					},
				},
			})
		})
	}
}

func TestAccInterfaceEthernetTest_import(t *testing.T) {
	for _, name := range testNames {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
					testSetTransportEnv(t, name)
				},
				ProviderFactories: testAccProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:        testAccInterfaceEthernetImportConfig(),
						ResourceName:  testInterfaceEthernetImportAddress,
						ImportStateId: "*3",
						ImportState:   true,
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 {
								return fmt.Errorf("more than 1 states received, only one interface expected")
							}
							state := states[0]
							if state.ID != expectedIdForInterface3 {
								return fmt.Errorf("expected id in the state don't match %v vs %v", state.ID, expectedIdForInterface3)
							}
							return nil
						},
					},
				},
			})
		})
	}
}

func testAccInterfaceEthernetConfig() string {
	return providerConfig + `

resource "routeros_interface_ethernet" "test" {
  factory_name              = "ether2"
  name                      = "terraform"
  mtu                       = "9000"
//   advertise                 = "10000M-full"
  arp                       = "disabled"
  auto_negotiation          = false
  tx_flow_control           = "auto"
  rx_flow_control           = "auto"
//   full_duplex  	            = "true"
  mdix_enable               = false
  sfp_shutdown_temperature  = 60
  speed                     = "10G-baseT"
}
`
}

func testAccInterfaceEthernetImportConfig() string {
	return providerConfig + `

resource "routeros_interface_ethernet" "importtest" {
  factory_name              = "ether3"
  name                      = "ether3"
  speed                     = "1Gbps" 
}
`
}
