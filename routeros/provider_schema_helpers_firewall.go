package routeros

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Firewall schema helpers.

// AddressListTimeoutEqual compares address-list-timeout of firewall rules: a value that is not in
// the configuration is no diff, `none-dynamic`/`none-static` are words, the rest is a time interval
// ("00:10:00" = "10m").
var AddressListTimeoutEqual schema.SchemaDiffSuppressFunc = func(k, old, new string, d *schema.ResourceData) bool {
	if old == new {
		return true
	}
	if new == "" {
		return AlwaysPresentNotUserProvided(k, old, new, d)
	}
	if old == "" {
		return false
	}
	for _, w := range []string{"none-dynamic", "none-static"} {
		if old == w || new == w {
			return false
		}
	}
	o, err1 := ParseDuration(old, time.Second)
	n, err2 := ParseDuration(new, time.Second)
	return err1 == nil && err2 == nil && o == n
}
