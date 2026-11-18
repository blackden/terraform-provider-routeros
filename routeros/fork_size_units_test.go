package routeros

import "testing"

func TestSizeKiBEqual(t *testing.T) {
	cases := []struct {
		old, new string
		want     bool
	}{
		{"2048", "2048KiB", true},
		{"2048", "2MiB", true},
		{"2048", "2048", true},
		{"unlimited", "unlimited", true},
		{"2048", "4096", false},
		{"2048", "2048MiB", false},
		{"2048", "", true},
		{"unlimited", "2048", false},
		{"1048576", "1GiB", true},
		{"none", "none", true},
		{"none", "2048", false},
	}
	for _, c := range cases {
		if got := SizeKiBEqual("k", c.old, c.new, nil); got != c.want {
			t.Errorf("SizeKiBEqual(%q, %q) = %v, want %v", c.old, c.new, got, c.want)
		}
	}
}
