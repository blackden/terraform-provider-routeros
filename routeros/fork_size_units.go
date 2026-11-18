package routeros

import (
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// sizeKiB parses a RouterOS size (`2048`, `2048KiB`, `2MiB`, `1GiB`) into KiB.
// A number without a suffix is KiB. ok is false for non-numeric values (`unlimited`, `none`).
func sizeKiB(s string) (v *big.Int, ok bool) {
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "KiB"):
		s = strings.TrimSuffix(s, "KiB")
	case strings.HasSuffix(s, "MiB"):
		s, mult = strings.TrimSuffix(s, "MiB"), 1024
	case strings.HasSuffix(s, "GiB"):
		s, mult = strings.TrimSuffix(s, "GiB"), 1024*1024
	}
	v, ok = new(big.Int).SetString(s, 10)
	if !ok || v.Sign() < 0 {
		return nil, false
	}
	return v.Mul(v, big.NewInt(mult)), true
}

// SizeKiBEqual is a DiffSuppressFunc for size fields that RouterOS reports in KiB without a suffix
// but accepts (and the user may write) with KiB/MiB/GiB. It suppresses the diff when the new value
// is empty (not set by the user) or both values denote the same size. Non-numeric values
// (`unlimited`, `none`) are compared as strings.
func SizeKiBEqual(_, old, new string, _ *schema.ResourceData) bool {
	if new == "" || old == new {
		return true
	}
	o, okOld := sizeKiB(old)
	n, okNew := sizeKiB(new)
	return okOld && okNew && o.Cmp(n) == 0
}
