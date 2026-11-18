package routeros

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// Unit tests (no router): /system/package/update is a singleton with four settings.

func TestSystemPackageUpdate_Schema(t *testing.T) {
	r := ResourceSystemPackageUpdate()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("InternalValidate: %v", err)
	}
	if got := r.Schema[MetaResourcePath].Default; got != "/system/package/update" {
		t.Errorf("resource path = %v", got)
	}
	for _, f := range []string{"channel", "check_certificate", "ip_version", "mode"} {
		if s := r.Schema[f]; s == nil || !s.Optional || s.Computed {
			t.Errorf("%v must be an optional argument", f)
		}
	}
	for _, f := range []string{"installed_version", "latest_version", "status"} {
		if s := r.Schema[f]; s == nil || !s.Computed || s.Optional {
			t.Errorf("%v must be read-only", f)
		}
	}
	for v, ok := range map[string]bool{"long-term": true, "stable": true, "testing": true, "development": true, "lts": false} {
		_, errs := r.Schema["channel"].ValidateFunc(v, "channel")
		if (len(errs) == 0) != ok {
			t.Errorf("channel = %q: errors %v", v, errs)
		}
	}
}

func TestSystemPackageUpdate_CreateSetsOnlyConfigured(t *testing.T) {
	r := ResourceSystemPackageUpdate()
	c := newFakeClient(map[string][]MikrotikItem{
		"/system/package/update": {{
			"channel": "long-term", "check-certificate": "yes", "installed-version": "7.24.6",
			"ip-version": "auto", "latest-version": "7.24.6", "mode": "https",
			"status": "System is already up to date",
		}},
	})
	st, diags := fakeApply(t, r, map[string]cty.Value{"channel": cty.StringVal("long-term")}, c)
	if diags.HasError() {
		t.Fatalf("create: %v", diags)
	}
	w, ok := c.lastWrite()
	if !ok {
		t.Fatal("no write request")
	}
	if w.path != "/system/package/update/set" {
		t.Errorf("path = %v", w.path)
	}
	if w.item["channel"] != "long-term" {
		t.Errorf("channel = %q", w.item["channel"])
	}
	for _, k := range []string{"mode", "ip-version", "check-certificate", "installed-version", "latest-version", "status"} {
		if _, ok := w.item[k]; ok {
			t.Errorf("unexpected field %v in the set request: %v", k, w.item)
		}
	}
	if st.ID != "system.package.update" {
		t.Errorf("id = %v", st.ID)
	}
	if got := st.Attributes["installed_version"]; got != "7.24.6" {
		t.Errorf("installed_version = %v", got)
	}
	if got := st.Attributes["check_certificate"]; got != "true" {
		t.Errorf("check_certificate = %v", got)
	}
}
