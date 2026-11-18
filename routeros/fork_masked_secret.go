package routeros

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// isMaskedSecret reports whether v is a non-empty string of asterisks, the way RouterOS
// returns write-only secrets.
func isMaskedSecret(v string) bool {
	return v != "" && strings.Trim(v, "*") == ""
}

// MaskedSecretNotInConfig is a DiffSuppressFunc for write-only secrets that RouterOS
// returns masked ("*****") or empty. It suppresses the diff only when the attribute is
// absent from the configuration (new == "") and the stored value is empty or consists
// of asterisks only. A secret the user did set (new != "") always produces a diff.
//
// An absent or empty secret in the configuration means "not managed": the router keeps
// its value, and the secret cannot be cleared through the configuration. The serializer
// (TerraformResourceDataToMikrotik) does not send such a masked value back either.
func MaskedSecretNotInConfig(_, old, new string, _ *schema.ResourceData) bool {
	return new == "" && strings.Trim(old, "*") == ""
}
