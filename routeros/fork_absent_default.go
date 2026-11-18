package routeros

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Fields that RouterOS leaves out of the response.
//
// RouterOS 7.24 does not return arguments that have their default value (`connect`, `listen`,
// `local.port`...) or are not set at all (`local.ttl`). SDKv2 reads them back as "" (top-level) or
// as the zero value (fields of nested blocks), so an imported resource differs from a schema
// Default, and `-generate-config-out` writes zero values that RouterOS rejects on write.

// DiffSuppressAbsentDefault suppresses the diff between a field that RouterOS left out of the
// response (old "" or "0") and the schema Default def: the router has the default. Any value read
// from the router ("false" included) and any other planned value is a change. On create there is
// no response yet, the diff is kept.
func DiffSuppressAbsentDefault(def string) schema.SchemaDiffSuppressFunc {
	return func(k, old, new string, d *schema.ResourceData) bool {
		if d != nil && d.Id() == "" {
			return false
		}
		return (old == "" || old == "0") && new == def
	}
}

// DiffSuppressUnsetZero treats "" (attribute absent) and a zero value ("0" or "") as the same "not
// set" for fields without a Default (`local.ttl`, `output.default-originate`). A real value on
// either side is a change.
func DiffSuppressUnsetZero(k, old, new string, d *schema.ResourceData) bool {
	return forkIsZeroString(old) && forkIsZeroString(new)
}

// ValidationEmptyOr accepts "" (not set, see DiffSuppressUnsetZero) and passes any other value to f.
func ValidationEmptyOr(f schema.SchemaValidateDiagFunc) schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		if s, ok := i.(string); ok && s == "" {
			return nil
		}
		return f(i, p)
	}
}

// forkPayloadField is an argument that must not reach RouterOS with a value meaning "not set".
type forkPayloadField struct {
	// Attr is the attribute path: "connect", "local.0.ttl".
	Attr string
	// Default is the schema Default as a string. With a Default the field is dropped when its
	// value is the zero value read back for an absent field and the configuration leaves it out,
	// sets the default or (not for booleans) the zero value. Without a Default the zero value
	// itself is "not set" and is dropped unless it is a change.
	Default string
}

// forkDropUnsetFields removes from a create/update payload the fields that RouterOS left out of the
// response and the configuration does not ask to change (see forkPayloadField).
func forkDropUnsetFields(s map[string]*schema.Schema, d *schema.ResourceData, item MikrotikItem, fields []forkPayloadField) {
	drift := driftAttributeSlice.GetDriftMap(RouterOSVersion, s[MetaResourcePath].Default.(string), false)
	for _, f := range fields {
		if !forkIsZero(d.Get(f.Attr)) {
			continue
		}
		if f.Default == "" {
			if d.HasChange(f.Attr) {
				continue
			}
		} else {
			conf := forkRawConfigAt(d, f.Attr)
			if !conf.IsNull() && conf.IsKnown() {
				cs := forkCtyString(conf)
				_, isBool := d.Get(f.Attr).(bool)
				if cs != f.Default && (isBool || !forkIsZeroString(cs)) {
					continue
				}
			}
		}
		key := forkPayloadKey(f.Attr, drift)
		delete(item, key)
		delete(item, "!"+key)
	}
}

// forkPayloadKey converts an attribute path to the payload key: "local.0.ttl" -> "local.ttl",
// "address_families" -> "afi" (drift).
func forkPayloadKey(attr string, drift map[string]string) string {
	parts := strings.Split(attr, ".")
	if len(parts) == 1 {
		if mt, ok := drift[attr]; ok {
			return SnakeToKebab(mt)
		}
		return SnakeToKebab(attr)
	}
	var out []string
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err == nil {
			continue
		}
		out = append(out, p)
	}
	return SnakeToKebab(strings.Join(out, "."))
}

// forkRawConfigAt returns the configuration value at an attribute path, null if it is not there.
func forkRawConfigAt(d *schema.ResourceData, attr string) cty.Value {
	v := d.GetRawConfig()
	for _, p := range strings.Split(attr, ".") {
		if v.IsNull() || !v.IsKnown() {
			return cty.NullVal(cty.DynamicPseudoType)
		}
		if i, err := strconv.Atoi(p); err == nil {
			if !v.Type().IsListType() || v.LengthInt() <= i {
				return cty.NullVal(cty.DynamicPseudoType)
			}
			v = v.Index(cty.NumberIntVal(int64(i)))
			continue
		}
		if !v.Type().IsObjectType() || !v.Type().HasAttribute(p) {
			return cty.NullVal(cty.DynamicPseudoType)
		}
		v = v.GetAttr(p)
	}
	return v
}

func forkCtyString(v cty.Value) string {
	switch v.Type() {
	case cty.Bool:
		return strconv.FormatBool(v.True())
	case cty.Number:
		return v.AsBigFloat().Text('f', -1)
	case cty.String:
		return v.AsString()
	}
	return ""
}

func forkIsZeroString(s string) bool { return s == "" || s == "0" }

func forkIsZero(v interface{}) bool {
	switch v := v.(type) {
	case string:
		return v == ""
	case int:
		return v == 0
	case bool:
		return !v
	case float64:
		return v == 0
	}
	return false
}

// forkFilterClient drops unset fields from create/update payloads of one resource.
type forkFilterClient struct {
	Client
	s      map[string]*schema.Schema
	d      *schema.ResourceData
	fields []forkPayloadField
}

func (c *forkFilterClient) SendRequest(method crudMethod, url *URL, item MikrotikItem, result interface{}) error {
	if item != nil && (method == crudCreate || method == crudUpdate) {
		forkDropUnsetFields(c.s, c.d, item, c.fields)
	}
	return c.Client.SendRequest(method, url, item, result)
}

// ForkCreateDropUnset is DefaultCreate without the unset fields in the payload.
func ForkCreateDropUnset(s map[string]*schema.Schema, fields []forkPayloadField) schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		return ResourceCreate(ctx, s, d, &forkFilterClient{Client: m.(Client), s: s, d: d, fields: fields})
	}
}

// ForkUpdateDropUnset is DefaultUpdate without the unset fields in the payload.
func ForkUpdateDropUnset(s map[string]*schema.Schema, fields []forkPayloadField) schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		return ResourceUpdate(ctx, s, d, &forkFilterClient{Client: m.(Client), s: s, d: d, fields: fields})
	}
}
